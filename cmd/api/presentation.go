package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/genai"
)

func (app *application) initiateSSE(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Ensure the response can be flushed for streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		fmt.Println("streaming not supported")
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Get task_id from URL query parameters
	taskID := r.URL.Query().Get("task_id")
	if taskID == "" {
		fmt.Println("task ID is required")
		fmt.Fprintf(w, "event: error\ndata: Task ID is required\n\n")
		flusher.Flush()
		return
	}

	// Get channel for task
	c, exists := taskStore.Get(taskID)
	if !exists {
		fmt.Printf("task not found: %s\n", taskID)
		fmt.Fprintf(w, "event: error\ndata: Task not found\n\n")
		flusher.Flush()
		return
	}

	// Ensure cleanup even if client disconnects
	defer taskStore.Delete(taskID)

	// Listen for messages or client disconnection
	for {
		select {
		case msg, ok := <-c:
			if !ok {
				// Channel closed, task is complete
				fmt.Printf("task %s: channel closed\n", taskID)
				fmt.Fprintf(w, "event: finished\ndata: Task completed\n\n")
				flusher.Flush()
				return
			}
			// Send message with appropriate event type
			fmt.Printf("sending message [%s] for task %s: %s\n", msg.Type, taskID, msg.Content)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", msg.Type, msg.Content)
			flusher.Flush()
		case <-r.Context().Done():
			// Client disconnected
			fmt.Printf("client disconnected for task %s\n", taskID)
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
	var request presentationRequest
	err := app.readJSON(w, r, &request)
	if err != nil {
		fmt.Printf("error reading JSON request: %v\n", err)
		app.badRequestError(w, err.Error())
		return
	}

	// Validate input
	if strings.TrimSpace(request.PresentationText) == "" {
		fmt.Println("presentation text cannot be empty")
		app.badRequestError(w, "presentation text cannot be empty")
		return
	}

	if len(request.PresentationText) > 50000 {
		fmt.Println("presentation text exceeds maximum length")
		app.badRequestError(w, "presentation text exceeds maximum length")
		return
	}

	// Create a task ID and channel
	taskID := uuid.New().String()
	ch := make(chan TaskMessage, 10) // Buffered channel to avoid blocking
	taskStore.Set(taskID, ch)

	fmt.Printf("initiating processing task %s\n", taskID)

	// Start processing
	go app.generateContent(taskID, request.PresentationText, ch)

	// Respond with task ID
	app.writeJSON(w, http.StatusOK, "task initiated", map[string]string{
		"task_id": taskID,
	})
}

func (app *application) generateContent(taskID, presentationText string, c chan TaskMessage) {
	defer close(c)
	defer taskStore.Delete(taskID)

	ctx := context.Background()

	fmt.Printf("task %s: starting content generation\n", taskID)
	c <- TaskMessage{Type: "progress", Content: "Analyzing presentation content..."}

	// Call LLM
	result, err := app.llm.Models.GenerateContent(ctx, app.config.geimini.model, genai.Text(generatePresentationPrompt("presentation", presentationText)), &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		MaxOutputTokens:  int32(app.config.geimini.maxOutputTokens),
		Temperature:      genai.Ptr[float32](0.5),
	})
	if err != nil {
		fmt.Printf("task %s: error generating content: %v\n", taskID, err)
		c <- TaskMessage{Type: "error", Content: fmt.Sprintf("Error generating presentation: %v", err)}
		return
	}

	c <- TaskMessage{Type: "progress", Content: "Creating flashcards and quizzes..."}
	fmt.Printf("task %s: processing LLM response\n", taskID)

	// Parse the JSON response
	var response PresentationResponse
	jsonText := result.Text()
	fmt.Printf("task %s: raw LLM response: %s\n", taskID, jsonText)

	err = json.Unmarshal([]byte(jsonText), &response)
	if err != nil {
		fmt.Printf("task %s: error parsing JSON response: %v\n", taskID, err)
		c <- TaskMessage{Type: "error", Content: fmt.Sprintf("Error parsing response: %v", err)}
		return
	}

	// Validate response data
	if len(response.Flashcards) == 0 && len(response.Quizzes) == 0 {
		fmt.Printf("task %s: no flashcards or quizzes generated\n", taskID)
		c <- TaskMessage{Type: "error", Content: "No flashcards or quizzes could be generated from the presentation"}
		return
	}

	// Return the processed data
	encodedJSON := base64.StdEncoding.EncodeToString([]byte(jsonText))
	fmt.Printf("task %s: successfully generated %d flashcards and %d quizzes\n",
		taskID, len(response.Flashcards), len(response.Quizzes))
	c <- TaskMessage{Type: "finished", Content: encodedJSON}
}
