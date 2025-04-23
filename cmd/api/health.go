package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"google.golang.org/genai"
)

func (app *application) healthCheck(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	requestID := generateRequestID()
	log.Printf("[%s] Health check requested from %s", requestID, getClientIP(r))

	// Basic response data
	data := map[string]interface{}{
		"status":           "ok",
		"version":          "1.0.0",
		"env":              app.config.env,
		"response_time_ms": 0, // Will be updated before sending
	}

	// Check if LLM client is available
	if app.llm == nil {
		data["llm_status"] = "not_initialized"
		log.Printf("[%s] WARNING: LLM client not initialized during health check", requestID)
	} else {
		// Attempt to warm up LLM with a minimal request
		// This helps avoid cold start issues with the first real request
		data["llm_status"] = "ready"

		// Only do the warmup if requested with ?warmup=true
		if r.URL.Query().Get("warmup") == "true" {
			log.Printf("[%s] Performing LLM warmup", requestID)
			data["llm_warmup"] = "requested"

			// Create a context with timeout for warmup
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Make a minimal LLM request to warm up connections
			warmupPrompt := "Hello. This is a warmup request. Please respond with 'ready'."
			warmupStartTime := time.Now()

			_, err := app.llm.Models.GenerateContent(ctx,
				app.config.geimini.model,
				genai.Text(warmupPrompt),
				&genai.GenerateContentConfig{
					MaxOutputTokens: 10,
					Temperature:     genai.Ptr[float32](0.1),
				},
			)

			warmupDuration := time.Since(warmupStartTime)
			log.Printf("[%s] LLM warmup completed in %v", requestID, warmupDuration)

			if err != nil {
				data["llm_warmup_result"] = "failed"
				data["llm_warmup_error"] = err.Error()
				log.Printf("[%s] LLM warmup failed: %v", requestID, err)
			} else {
				data["llm_warmup_result"] = "success"
				data["llm_warmup_duration_ms"] = warmupDuration.Milliseconds()
				log.Printf("[%s] LLM warmup successful", requestID)
			}
		}
	}

	// Calculate total response time
	responseTime := time.Since(startTime)
	data["response_time_ms"] = responseTime.Milliseconds()

	log.Printf("[%s] Health check completed in %v", requestID, responseTime)
	app.writeJSON(w, http.StatusOK, "health check", data)
}
