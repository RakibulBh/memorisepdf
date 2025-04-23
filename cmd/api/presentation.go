package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/google/uuid"
	"google.golang.org/genai"
)

// TaskMessage defines the structure of messages sent through the channel
type TaskMessage struct {
	Type    string // "progress", "finished", or "error"
	Content string // The message or result
}

// TaskStore with mutex for thread safety
type TaskStore struct {
	sync.RWMutex
	channels map[string]chan TaskMessage
}

// Global task store
var taskStore = &TaskStore{
	channels: make(map[string]chan TaskMessage),
}

func (ts *TaskStore) Set(taskID string, ch chan TaskMessage) {
	ts.Lock()
	ts.channels[taskID] = ch
	ts.Unlock()
}

func (ts *TaskStore) Get(taskID string) (chan TaskMessage, bool) {
	ts.RLock()
	ch, exists := ts.channels[taskID]
	ts.RUnlock()
	return ch, exists
}

func (ts *TaskStore) Delete(taskID string) {
	ts.Lock()
	delete(ts.channels, taskID)
	ts.Unlock()
}

func (app *application) initiateSSE(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Ensure the response can be flushed for streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Get task_id from URL query parameters
	taskID := r.URL.Query().Get("task_id")
	if taskID == "" {
		fmt.Fprintf(w, "event: error\ndata: Task ID is required\n\n")
		flusher.Flush()
		return
	}

	// Get channel for task
	c, exists := taskStore.Get(taskID)
	if !exists {
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
				fmt.Fprintf(w, "event: finished\ndata: Task completed\n\n")
				flusher.Flush()
				return
			}
			// Send message with appropriate event type
			fmt.Println("sending message", msg.Type, msg.Content)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", msg.Type, msg.Content)
			flusher.Flush()
		case <-r.Context().Done():
			// Client disconnected
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

	// Send progress update
	c <- TaskMessage{Type: "progress", Content: "Analyzing presentation content..."}

	// Call LLM
	result, err := app.llm.Models.GenerateContent(ctx, app.config.geimini.model, genai.Text(generatePresentationPrompt("presentation", presentationText)), &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		MaxOutputTokens:  4096,
		Temperature:      genai.Ptr[float32](0.5),
	})
	if err != nil {
		fmt.Println("Error generating presentation:", err)
		c <- TaskMessage{Type: "error", Content: fmt.Sprintf("Error generating presentation: %v", err)}
		return
	}

	// Process the result
	c <- TaskMessage{Type: "progress", Content: "Creating flashcards and quizzes..."}

	// Parse the JSON response
	var response PresentationResponse
	jsonText := result.Text()

	err = json.Unmarshal([]byte(jsonText), &response)
	if err != nil {
		fmt.Println("Error parsing response:", err)
		c <- TaskMessage{Type: "error", Content: fmt.Sprintf("Error parsing response: %v", err)}
		return
	}

	// Validate response data
	if len(response.Flashcards) == 0 && len(response.Quizzes) == 0 {
		fmt.Println("No flashcards or quizzes could be generated from the presentation")
		c <- TaskMessage{Type: "error", Content: "No flashcards or quizzes could be generated from the presentation"}
		return
	}

	// Return the processed data
	encodedJSON := base64.StdEncoding.EncodeToString([]byte(jsonText))
	c <- TaskMessage{Type: "finished", Content: encodedJSON}
}
