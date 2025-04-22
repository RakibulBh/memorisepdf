package main

import (
	"net/http"
	"time"

	"github.com/RakibulBh/relatewme/internal/env"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/redis/go-redis/v9"
)

type redisConfig struct {
	addr     string
	password string
	db       int
	protocol int
}

type application struct {
	config config
	redis  *redis.Client
	// store  store.Storage
}

type config struct {
	addr string
	env  string
	// db     dbConfig
	apiURL string
	redis  redisConfig
}

// type dbConfig struct {
// 	addr         string
// 	maxOpenConns int
// 	maxIdleConns int
// 	maxIdleTime  string
// }

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

	r.Post("/start", app.startChat)
	r.Get("/ws/{sessionKey}", app.handleChat)

	return r
}

func (app *application) run(mux http.Handler) error {
	srv := http.Server{
		Addr:              app.config.addr,
		Handler:           mux,
		WriteTimeout:      15 * time.Second,
		ReadTimeout:       15 * time.Second,
		IdleTimeout:       time.Minute,
		ReadHeaderTimeout: 10 * time.Second,
	}

	return srv.ListenAndServe()
}
