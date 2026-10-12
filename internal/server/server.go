// Package server exposes the JSON API the frontend talks to.
package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/RakibulBh/unmemorize/internal/auth"
	"github.com/RakibulBh/unmemorize/internal/config"
	"github.com/RakibulBh/unmemorize/internal/files"
	"github.com/RakibulBh/unmemorize/internal/generate"
	"github.com/RakibulBh/unmemorize/internal/store"
	"github.com/RakibulBh/unmemorize/internal/tika"
)

type Server struct {
	cfg      config.Config
	store    *store.Store
	auth     *auth.Service
	storage  *files.Storage
	pipeline *generate.Pipeline
	tika     *tika.Client
	log      *slog.Logger
}

type Options struct {
	Config   config.Config
	Store    *store.Store
	Auth     *auth.Service
	Storage  *files.Storage
	Pipeline *generate.Pipeline
	Tika     *tika.Client
	Logger   *slog.Logger
}

// Builds the API server.
func New(opts Options) *Server {
	return &Server{
		cfg:      opts.Config,
		store:    opts.Store,
		auth:     opts.Auth,
		storage:  opts.Storage,
		pipeline: opts.Pipeline,
		tika:     opts.Tika,
		log:      opts.Logger,
	}
}

// Lays out every route the API answers.
func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(s.requestLogger)

	// In production the frontend is served through the same proxy hostname, so
	// no cross-origin request is made at all. This matters in development,
	// where Next.js runs on a different port.
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{s.cfg.WebURL},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", s.handleHealth)
	r.Get("/ready", s.handleReady)

	r.Route("/auth", func(r chi.Router) {
		r.Get("/google", s.handleGoogleStart)
		r.Get("/google/callback", s.handleGoogleCallback)
		r.Post("/logout", s.handleLogout)
	})

	r.Route("/api", func(r chi.Router) {
		r.Use(s.auth.RequireUser)

		r.Get("/me", s.handleMe)
		r.Get("/meta", s.handleMeta)
		r.Get("/usage", s.handleUsage)

		r.Route("/subjects", func(r chi.Router) {
			r.Get("/", s.handleListSubjects)
			r.Post("/", s.handleCreateSubject)

			r.Route("/{subjectID}", func(r chi.Router) {
				r.Get("/", s.handleGetSubject)
				r.Patch("/", s.handleUpdateSubject)
				r.Delete("/", s.handleDeleteSubject)

				r.Get("/modules", s.handleListModules)
				r.Post("/modules", s.handleCreateModule)
			})
		})

		r.Route("/modules/{moduleID}", func(r chi.Router) {
			r.Get("/", s.handleGetModule)
			r.Patch("/", s.handleUpdateModule)
			r.Delete("/", s.handleDeleteModule)

			r.Get("/files", s.handleListFiles)
			r.Post("/files", s.handleUploadFiles)

			// Returns a progress stream rather than a plain response, so the
			// browser can show which file is being analysed as it happens.
			r.Post("/generate", s.handleGenerate)

			r.Get("/quiz", s.handleModuleQuiz)
			r.Get("/flashcards", s.handleModuleFlashcards)
			r.Get("/teaching", s.handleModuleTeaching)
		})

		r.Delete("/files/{fileID}", s.handleDeleteFile)
		r.Get("/files/{fileID}/download", s.handleDownloadFile)
	})

	return r
}

// Runs the HTTP server until it is told to stop.
func (s *Server) Server(handler http.Handler) *http.Server {
	return &http.Server{
		Addr:    s.cfg.Addr,
		Handler: handler,

		ReadHeaderTimeout: 20 * time.Second,
		// Uploads can be large and generation streams for minutes, so neither
		// the read nor the write side can carry a short deadline. Individual
		// handlers set their own timeouts instead.
		ReadTimeout:  0,
		WriteTimeout: 0,
		IdleTimeout:  2 * time.Minute,
	}
}

// Writes a line to the log for every request, with how long it took.
func (s *Server) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		level := slog.LevelInfo
		if ww.Status() >= 500 {
			level = slog.LevelError
		} else if ww.Status() >= 400 {
			level = slog.LevelWarn
		}

		s.log.Log(r.Context(), level, "request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"bytes", ww.BytesWritten(),
			"duration", time.Since(started).Round(time.Millisecond).String(),
			"requestId", middleware.GetReqID(r.Context()),
		)
	})
}
