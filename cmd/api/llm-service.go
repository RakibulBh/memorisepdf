package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"time"

	"google.golang.org/genai"
)

func (app *application) generateTeachingCards(taskID, presentationText, teachingStyle string, c chan TaskMessage) {
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
			genai.Text(generatePresentationPrompt("teachme", presentationText, nil, &teachingStyle)),
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

	c <- TaskMessage{Type: "progress", Content: "Creating teaching cards..."}

	jsonText := result.Text()

	// For debugging
	log.Printf("Task %s: Generated JSON response: %s", taskID, jsonText[:min(200, len(jsonText))])

	var response TeachingCardsResponse
	if err := json.Unmarshal([]byte(jsonText), &response); err != nil {
		log.Printf("Task %s: JSON parse error: %v\nData: %s", taskID, err, jsonText[:min(200, len(jsonText))])
		c <- TaskMessage{Type: "error", Content: "Invalid response format"}
		return
	}

	if len(response.TeachingCards) == 0 {
		log.Printf("Task %s: empty generation result", taskID)
		c <- TaskMessage{Type: "error", Content: "No content generated"}
		return
	}

	encodedJSON := base64.StdEncoding.EncodeToString([]byte(jsonText))
	c <- TaskMessage{Type: "finished", Content: encodedJSON}
}

func (app *application) generateQuizzes(taskID, presentationText, difficulty string, c chan TaskMessage) {
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
			genai.Text(generatePresentationPrompt("testme", presentationText, &difficulty, nil)),
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

	// For debugging
	log.Printf("Task %s: Generated JSON response: %s", taskID, jsonText[:min(200, len(jsonText))])

	var response PresentationResponse
	if err := json.Unmarshal([]byte(jsonText), &response); err != nil {
		log.Printf("Task %s: JSON parse error: %v\nData: %s", taskID, err, jsonText[:min(200, len(jsonText))])
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
