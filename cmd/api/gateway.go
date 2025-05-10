package main

import (
	"fmt"
	"log"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"
)

func (app *application) initiateSSE(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()

	// Set appropriate headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Get response controller for flushing
	rc := http.NewResponseController(w)

	// Check if task ID is provided
	taskID := r.URL.Query().Get("task_id")
	if taskID == "" {
		log.Printf("[%s] ERROR: task ID is required", requestID)
		fmt.Fprintf(w, "event: error\ndata: Task ID is required\n\n")
		rc.Flush()
		return
	}

	// Get the channel for this task
	c, exists := taskStore.Get(taskID)
	if !exists {
		log.Printf("[%s] ERROR: task not found: %s", requestID, taskID)
		fmt.Fprintf(w, "event: error\ndata: Task not found\n\n")
		rc.Flush()
		return
	}

	// Prevent race conditions with a mutex to track if the channel is already closed
	var channelClosed bool
	var mu sync.Mutex

	defer func() {
		mu.Lock()
		defer mu.Unlock()

		if !channelClosed {
			taskStore.Delete(taskID)
		}
	}()

	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Printf("[%s] ERROR: streaming not supported", requestID)
		fmt.Fprintf(w, "event: error\ndata: Streaming not supported\n\n")
		rc.Flush()
		return
	}

	fmt.Fprintf(w, "event: progress\ndata: Connection established\n\n")
	flusher.Flush()

	// Handle client disconnection
	clientGone := r.Context().Done()

	for {
		select {
		case <-clientGone:
			log.Printf("[%s] INFO: client disconnected", requestID)
			mu.Lock()
			channelClosed = true
			mu.Unlock()
			return
		case msg, ok := <-c:
			if !ok {
				log.Printf("[%s] INFO: channel closed", requestID)
				mu.Lock()
				channelClosed = true
				mu.Unlock()
				return
			}

			// Format and send the SSE message
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", msg.Type, msg.Content)
			flusher.Flush()

			// If this is the final message, we can return
			if msg.Type == "finished" || msg.Type == "error" {
				log.Printf("[%s] INFO: task completed with status: %s", requestID, msg.Type)
				mu.Lock()
				channelClosed = true
				mu.Unlock()
				return
			}
		}
	}
}

func (app *application) initiateProcessing(w http.ResponseWriter, r *http.Request) {
	fmt.Println("initiateProcessing")
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		app.badRequestError(w, "failed to read file")
		return
	}
	defer file.Close()

	requestID := generateRequestID()

	// Get client IP address
	clientIP := getIPFromRequest(r)
	log.Printf("[%s] Request from IP: %s", requestID, clientIP)

	// get the form data values
	difficulty := r.FormValue("difficulty")
	teachingStyle := r.FormValue("teaching_style")
	serviceType := r.FormValue("service_type")

	// validate service type
	if serviceType != "testme" && serviceType != "teachme" {
		log.Printf("[%s] ERROR: invalid service type", requestID)
		app.badRequestError(w, "invalid service type")
		return
	}

	// validate difficulty
	if !slices.Contains(difficultyLevels, difficulty) && serviceType == "testme" {
		log.Printf("[%s] ERROR: invalid difficulty", requestID)
		app.badRequestError(w, "invalid difficulty")
		return
	}

	// validate teaching style
	if !slices.Contains(teachingStyles, teachingStyle) && serviceType == "teachme" {
		log.Printf("[%s] ERROR: invalid teaching style", requestID)
		app.badRequestError(w, "invalid teaching style")
		return
	}

	// upload to tika
	presentationText, err := app.parseFile(&file)
	if err != nil {
		log.Printf("[%s] ERROR: failed to upload file: %v", requestID, err)
		app.badRequestError(w, "failed to upload file")
		return
	}

	// validate presentation text
	if len(presentationText) == 0 || len(presentationText) > 100000 {
		log.Printf("[%s] ERROR: length of presentation text is invalid", requestID)
		app.badRequestError(w, "presentation text cannot be empty or too long")
		return
	}

	taskID := uuid.New().String()
	ch := make(chan TaskMessage, 10)

	// Send an initial message with the client IP
	ipMessage := TaskMessage{
		Type:    "ip",
		Content: clientIP,
	}
	ch <- ipMessage

	taskStore.Set(taskID, ch)

	if serviceType == "testme" {
		go app.generateQuizzes(fileHeader, taskID, presentationText, difficulty, ch)
	} else {
		go app.generateTeachingCards(fileHeader, taskID, presentationText, teachingStyle, ch)
	}

	app.writeJSON(w, http.StatusOK, "task initiated", map[string]string{
		"task_id": taskID,
	})
}

func generateRequestID() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}
