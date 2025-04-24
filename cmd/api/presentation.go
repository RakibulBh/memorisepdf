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

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	rc := http.NewResponseController(w)

	taskID := r.URL.Query().Get("task_id")
	if taskID == "" {
		log.Printf("[%s] ERROR: task ID is required", requestID)
		fmt.Fprintf(w, "event: error\ndata: Task ID is required\n\n")
		rc.Flush()
		return
	}

	c, exists := taskStore.Get(taskID)
	if !exists {
		log.Printf("[%s] ERROR: task not found: %s", requestID, taskID)
		fmt.Fprintf(w, "event: error\ndata: Task not found\n\n")
		rc.Flush()
		return
	}

	defer func() {
		taskStore.Delete(taskID)
	}()

	for {
		select {
		case msg, ok := <-c:
			if !ok {
				fmt.Fprintf(w, "event: finished\ndata: Task completed\n\n")
				rc.Flush()
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", msg.Type, msg.Content)
			rc.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

type presentationRequest struct {
	PresentationText string `json:"presentation_text"`
}

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

	var request presentationRequest
	err := app.readJSON(w, r, &request)
	if err != nil {
		log.Printf("[%s] ERROR reading JSON: %v", requestID, err)
		app.badRequestError(w, err.Error())
		return
	}

	if strings.TrimSpace(request.PresentationText) == "" {
		log.Printf("[%s] ERROR: empty presentation text", requestID)
		app.badRequestError(w, "presentation text cannot be empty")
		return
	}

	if len(request.PresentationText) > 100000 {
		log.Printf("[%s] ERROR: text exceeds limit", requestID)
		app.badRequestError(w, "presentation text exceeds maximum length")
		return
	}

	taskID := uuid.New().String()
	ch := make(chan TaskMessage, 10)
	taskStore.Set(taskID, ch)

	go app.generateContent(taskID, request.PresentationText, ch)

	app.writeJSON(w, http.StatusOK, "task initiated", map[string]string{
		"task_id": taskID,
	})
}

func (app *application) generateContent(taskID, presentationText string, c chan TaskMessage) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Task %s: PANIC: %v", taskID, r)
			c <- TaskMessage{Type: "error", Content: "Internal server error"}
		}
		close(c)
		taskStore.Delete(taskID)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	c <- TaskMessage{Type: "progress", Content: "Analyzing presentation content..."}

	var result *genai.GenerateContentResponse
	var err error
	maxRetries := 2

	for attempt := 0; attempt <= maxRetries; attempt++ {
		result, err = app.llm.Models.GenerateContent(
			ctx,
			app.config.geimini.model,
			genai.Text(generatePresentationPrompt("presentation", presentationText)),
			&genai.GenerateContentConfig{
				ResponseMIMEType: "application/json",
				MaxOutputTokens:  int32(app.config.geimini.maxOutputTokens),
				Temperature:      genai.Ptr[float32](0.5),
			},
		)

		if err == nil {
			break
		}

		if attempt == maxRetries {
			log.Printf("Task %s: LLM failed after %d attempts: %v", taskID, maxRetries+1, err)
			c <- TaskMessage{Type: "error", Content: "Content generation failed"}
			return
		}
		time.Sleep(time.Duration(attempt+1) * time.Second)
	}

	c <- TaskMessage{Type: "progress", Content: "Creating flashcards and quizzes..."}

	jsonText := result.Text()
	var response PresentationResponse
	if err := json.Unmarshal([]byte(jsonText), &response); err != nil {
		log.Printf("Task %s: JSON parse error: %v\nData: %s", taskID, err, jsonText[:200])
		c <- TaskMessage{Type: "error", Content: "Invalid response format"}
		return
	}

	if len(response.Flashcards) == 0 && len(response.Quizzes) == 0 {
		log.Printf("Task %s: empty generation result", taskID)
		c <- TaskMessage{Type: "error", Content: "No content generated"}
		return
	}

	encodedJSON := base64.StdEncoding.EncodeToString([]byte(jsonText))
	c <- TaskMessage{Type: "finished", Content: encodedJSON}
}

func generateRequestID() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}
