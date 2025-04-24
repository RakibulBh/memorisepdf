package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/genai"
)

func (app *application) initiateSSE(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	log.Printf("[%s] SSE headers set", requestID)

	rc := http.NewResponseController(w)

	// Get task_id from URL query parameters
	taskID := r.URL.Query().Get("task_id")
	if taskID == "" {
		log.Printf("[%s] ERROR: task ID is required but not provided", requestID)
		fmt.Fprintf(w, "event: error\ndata: Task ID is required\n\n")
		rc.Flush()
		return
	}
	log.Printf("[%s] Processing task ID: %s", requestID, taskID)

	// Get channel for task
	c, exists := taskStore.Get(taskID)
	if !exists {
		log.Printf("[%s] ERROR: task not found: %s", requestID, taskID)
		fmt.Fprintf(w, "event: error\ndata: Task not found\n\n")
		rc.Flush()
		return
	}
	log.Printf("[%s] Task channel retrieved successfully", requestID)

	// Ensure cleanup even if client disconnects
	defer func() {
		log.Printf("[%s] Cleaning up task resources for %s", requestID, taskID)
		taskStore.Delete(taskID)
	}()

	// Listen for messages or client disconnection
	for {
		select {
		case msg, ok := <-c:
			if !ok {
				// Channel closed properly
				fmt.Fprintf(w, "event: finished\ndata: Task completed\n\n")
				rc.Flush()
			}
			// Send message with explicit flush
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", msg.Type, msg.Content)
			rc.Flush()
		case <-r.Context().Done():
			// Client disconnected
			log.Printf("Client disconnected")
			return
		}
	}
}

type presentationRequest struct {
	PresentationText string `json:"presentation_text"`
}

// Response structures
type PresentationResponse struct {
	Flashcards []Flashcard `json:"flashcards"`
	Quizzes    []Quiz      `json:"quizzes"`
}

type Flashcard struct {
	Term       string `json:"term"`
	Definition string `json:"definition"`
}

type Quiz struct {
	Question string   `json:"question"`
	Answers  []Answer `json:"answers"`
}

type Answer struct {
	Text        string `json:"text"`
	Correct     bool   `json:"correct"`
	Explanation string `json:"explanation"`
}

func (app *application) initiateProcessing(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()
	startTime := time.Now()
	log.Printf("[%s] Processing request started at %s", requestID, startTime.Format(time.RFC3339))

	var request presentationRequest
	err := app.readJSON(w, r, &request)
	if err != nil {
		log.Printf("[%s] ERROR reading JSON request: %v", requestID, err)
		app.badRequestError(w, err.Error())
		return
	}
	log.Printf("[%s] Request parsed successfully", requestID)

	// Validate input
	if strings.TrimSpace(request.PresentationText) == "" {
		log.Printf("[%s] ERROR: presentation text cannot be empty", requestID)
		app.badRequestError(w, "presentation text cannot be empty")
		return
	}

	textLength := len(request.PresentationText)
	log.Printf("[%s] Presentation text length: %d characters", requestID, textLength)

	if textLength > 50000 {
		log.Printf("[%s] ERROR: presentation text exceeds maximum length (%d > 50000)", requestID, textLength)
		app.badRequestError(w, "presentation text exceeds maximum length")
		return
	}

	// Create a task ID and channel
	taskID := uuid.New().String()
	ch := make(chan TaskMessage, 10) // Buffered channel to avoid blocking
	taskStore.Set(taskID, ch)

	log.Printf("[%s] Initiating processing task %s", requestID, taskID)

	// Start processing in a goroutine
	go app.generateContent(taskID, request.PresentationText, ch)

	// Respond with task ID
	duration := time.Since(startTime).Milliseconds()
	log.Printf("[%s] Task initiated in %dms, returning task_id to client", requestID, duration)
	app.writeJSON(w, http.StatusOK, "task initiated", map[string]string{
		"task_id": taskID,
	})
}

func (app *application) generateContent(taskID, presentationText string, c chan TaskMessage) {
	startTime := time.Now()
	log.Printf("Task %s: Content generation started at %s", taskID, startTime.Format(time.RFC3339))

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Task %s: PANIC RECOVERED in generateContent: %v", taskID, r)
			c <- TaskMessage{Type: "error", Content: "Internal server error occurred"}
		}
		close(c)
		log.Printf("Task %s: channel closed, total processing time: %v", taskID, time.Since(startTime))
		taskStore.Delete(taskID)
	}()

	ctx := context.Background()

	log.Printf("Task %s: starting content generation", taskID)
	c <- TaskMessage{Type: "progress", Content: "Analyzing presentation content..."}

	// Set a timeout for the LLM call
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Call LLM with retry logic
	var result *genai.GenerateContentResponse
	var err error
	maxRetries := 2
	retryDelay := 1 * time.Second

	log.Printf("Task %s: calling LLM with model %s, max tokens %d",
		taskID, app.config.geimini.model, app.config.geimini.maxOutputTokens)

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("Task %s: retry attempt %d/%d after %v delay",
				taskID, attempt, maxRetries, retryDelay)
			time.Sleep(retryDelay)
			retryDelay *= 2 // Exponential backoff
		}

		callStartTime := time.Now()
		result, err = app.llm.Models.GenerateContent(
			ctxWithTimeout,
			app.config.geimini.model,
			genai.Text(generatePresentationPrompt("presentation", presentationText)),
			&genai.GenerateContentConfig{
				ResponseMIMEType: "application/json",
				MaxOutputTokens:  int32(app.config.geimini.maxOutputTokens),
				Temperature:      genai.Ptr[float32](0.5),
			},
		)

		callDuration := time.Since(callStartTime)
		if err == nil {
			log.Printf("Task %s: LLM call successful in %v on attempt %d",
				taskID, callDuration, attempt+1)
			break
		}

		log.Printf("Task %s: error generating content (attempt %d/%d): %v",
			taskID, attempt+1, maxRetries+1, err)

		if attempt == maxRetries {
			log.Printf("Task %s: all retry attempts failed", taskID)
			c <- TaskMessage{Type: "error", Content: fmt.Sprintf("Error generating presentation after %d attempts: %v", maxRetries+1, err)}
			return
		}
	}

	c <- TaskMessage{Type: "progress", Content: "Creating flashcards and quizzes..."}
	log.Printf("Task %s: processing LLM response", taskID)

	// Parse the JSON response
	var response PresentationResponse
	jsonText := result.Text()
	log.Printf("Task %s: raw LLM response length: %d bytes", taskID, len(jsonText))

	// Log a short preview of the response for debugging
	previewLength := 200
	if len(jsonText) > previewLength {
		log.Printf("Task %s: response preview: %s...", taskID, jsonText[:previewLength])
	} else {
		log.Printf("Task %s: response preview: %s", taskID, jsonText)
	}

	err = json.Unmarshal([]byte(jsonText), &response)
	if err != nil {
		log.Printf("Task %s: ERROR parsing JSON response: %v", taskID, err)
		log.Printf("Task %s: problematic JSON: %s", taskID, jsonText)
		c <- TaskMessage{Type: "error", Content: fmt.Sprintf("Error parsing response: %v", err)}
		return
	}

	// Validate response data
	if len(response.Flashcards) == 0 && len(response.Quizzes) == 0 {
		log.Printf("Task %s: no flashcards or quizzes generated", taskID)
		c <- TaskMessage{Type: "error", Content: "No flashcards or quizzes could be generated from the presentation"}
		return
	}

	// Return the processed data
	encodedJSON := base64.StdEncoding.EncodeToString([]byte(jsonText))
	log.Printf("Task %s: successfully generated %d flashcards and %d quizzes in %v",
		taskID, len(response.Flashcards), len(response.Quizzes), time.Since(startTime))
	c <- TaskMessage{Type: "finished", Content: encodedJSON}
}

// Helper function to generate a request ID for tracking
func generateRequestID() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}
