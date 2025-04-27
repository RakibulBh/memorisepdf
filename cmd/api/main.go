package main

import (
	"log"
	"time"

	"github.com/RakibulBh/relatewme/internal/env"
	"github.com/RakibulBh/relatewme/internal/llm"
	"github.com/google/go-tika/tika"
	"github.com/joho/godotenv"
)

func main() {
	startTime := time.Now()
	log.Printf("Application starting at %s", startTime.Format(time.RFC3339))

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found or could not be loaded: %v", err)
	}

	environment := env.GetString("ENV", "development")
	if environment != "development" {
		log.Printf("Running in %s environment", environment)
	} else {
		log.Printf("Running in development environment")
	}

	// Load configuration
	cfg := config{
		addr: ":" + env.GetString("PORT", "8080"),
		env:  environment,
		geimini: geiminiConfig{
			model:           env.GetString("GEIMINI_MODEL", "gemini-1.5-flash"),
			apiKey:          env.GetString("GEIMINI_API_KEY", ""),
			maxOutputTokens: env.GetInt("MAX_OUTPUT_TOKENS", 4096),
		},
		tika: tikaConfig{
			url: env.GetString("TIKA_URL", "https://apache-tike.fly.dev"),
		},
	}

	log.Printf("Configuration loaded: LLM model=%s, max_tokens=%d",
		cfg.geimini.model, cfg.geimini.maxOutputTokens)

	// Initialize LLM client with warmup
	log.Printf("Initializing LLM client...")
	llmStartTime := time.Now()
	llmClient := llm.New(cfg.geimini.model, cfg.geimini.apiKey)

	// Verify LLM client was initialized properly
	if llmClient != nil {
		log.Printf("LLM client initialized successfully in %v", time.Since(llmStartTime))
	} else {
		log.Printf("WARNING: LLM client may not have initialized properly")
	}

	log.Printf("Initializing Tika client...")
	tikaClient := tika.NewClient(nil, cfg.tika.url)
	if tikaClient != nil {
		log.Printf("Tika client initialized successfully")
	} else {
		log.Printf("WARNING: Tika client may not have initialized properly")
	}

	app := &application{
		config: cfg,
		llm:    llmClient,
		tika:   tikaClient,
	}

	// Prepare server
	log.Printf("Setting up HTTP server on %s", cfg.addr)
	mux := app.serve()

	// Record startup metrics
	log.Printf("Application ready in %v", time.Since(startTime))

	// Start listening for requests
	log.Printf("Starting HTTP server, listening on %s", cfg.addr)
	log.Fatal(app.run(mux))
}
