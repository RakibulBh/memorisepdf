package main

import (
	"fmt"
	"log"

	"github.com/RakibulBh/relatewme/internal/env"
	"github.com/RakibulBh/relatewme/internal/llm"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	if env.GetString("ENV", "development") != "development" {
		fmt.Print("Production environment")
	}

	cfg := config{
		addr: ":" + env.GetString("PORT", "8080"),
		env:  env.GetString("ENV", "development"),
		geimini: geiminiConfig{
			model:           env.GetString("GEIMINI_MODEL", "gemini-1.5-flash"),
			apiKey:          env.GetString("GEIMINI_API_KEY", ""),
			maxOutputTokens: env.GetInt("MAX_OUTPUT_TOKENS", 4096),
		},
	}

	// Init geimini client
	llmClient := llm.New(cfg.geimini.model, cfg.geimini.apiKey)

	app := &application{
		config: cfg,
		llm:    llmClient,
	}

	mux := app.serve()
	log.Fatal(app.run(mux))

}
