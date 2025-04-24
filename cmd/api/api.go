package main

import (
	"net/http"
	"time"

	"github.com/RakibulBh/relatewme/internal/env"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"google.golang.org/genai"
)

type geiminiConfig struct {
	model           string
	apiKey          string
	maxOutputTokens int
}

type application struct {
	config config
	llm    *genai.Client
}

type config struct {
	addr    string
	env     string
	geimini geiminiConfig
}

func (app *application) serve() http.Handler {

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{env.GetString("FRONTEND_URL", "http://localhost:3000")},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Healthcheck
	r.Get("/health", app.healthCheck)

	// retrieve the pdf content
	r.Post("/initiate", app.initiateProcessing)

	// initiate SSE
	r.Get("/sse", app.initiateSSE)

	return r
}

func (app *application) run(mux http.Handler) error {
	srv := http.Server{
		Addr:              app.config.addr,
		Handler:           mux,
		WriteTimeout:      60 * time.Second,
		ReadTimeout:       60 * time.Second,
		IdleTimeout:       time.Minute,
		ReadHeaderTimeout: 10 * time.Second,
	}

	return srv.ListenAndServe()
}
