// Command api serves the MemorisePDF backend.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/RakibulBh/memorisepdf/internal/auth"
	"github.com/RakibulBh/memorisepdf/internal/config"
	"github.com/RakibulBh/memorisepdf/internal/files"
	"github.com/RakibulBh/memorisepdf/internal/generate"
	"github.com/RakibulBh/memorisepdf/internal/llm"
	"github.com/RakibulBh/memorisepdf/internal/logging"
	"github.com/RakibulBh/memorisepdf/internal/migrations"
	"github.com/RakibulBh/memorisepdf/internal/server"
	"github.com/RakibulBh/memorisepdf/internal/store"
	"github.com/RakibulBh/memorisepdf/internal/tika"
)

// How long the server is given to finish in-flight requests when stopping. A
// generation stream can be mid-file, and cutting it off would lose that work.
const shutdownGrace = 30 * time.Second

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

// Starts everything up, serves until interrupted, then shuts down cleanly.
func run() error {
	// A missing .env is normal in a container, where the values come from the
	// environment directly.
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log, closer := logging.New(logPath(cfg), logging.ParseLevel(cfg.Log.Level))
	defer closer.Close()
	slog.SetDefault(log)

	log.Info("starting",
		"env", cfg.Env,
		"addr", cfg.Addr,
		"appUrl", cfg.AppURL,
		"webUrl", cfg.WebURL,
		"model", cfg.Gemini.Model,
	)

	// Interrupt and terminate both unwind through this context, so a container
	// stop is handled the same way as ctrl-c.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	log.Info("connected to postgres")

	if err := runMigrations(ctx, db, log); err != nil {
		return err
	}

	storage, err := files.NewStorage(cfg.DataDir, cfg.Limits.MaxFileSize)
	if err != nil {
		return err
	}

	tikaClient := tika.New(cfg.Tika.URL, cfg.Tika.Timeout)
	// Tika is checked but not required at boot: it may still be starting
	// alongside this container, and /ready reports the truth either way.
	if err := tikaClient.Ping(ctx); err != nil {
		log.Warn("tika is not reachable yet", "url", cfg.Tika.URL, "err", err)
	} else {
		log.Info("tika is reachable", "url", cfg.Tika.URL)
	}

	llmClient, err := llm.New(ctx, llm.Options{
		APIKey:          cfg.Gemini.APIKey,
		Model:           cfg.Gemini.Model,
		MaxOutputTokens: cfg.Gemini.MaxOutputTokens,
		MaxInputChars:   cfg.Limits.MaxExtractedChars,
		Timeout:         cfg.Gemini.Timeout,
		Logger:          log,
	})
	if err != nil {
		return err
	}

	authService := auth.New(auth.Options{
		Store:        db,
		ClientID:     cfg.Google.ClientID,
		ClientSecret: cfg.Google.ClientSecret,
		RedirectURL:  cfg.Google.RedirectURL,
		CookieName:   cfg.Session.CookieName,
		SessionTTL:   cfg.Session.TTL,
		Secure:       cfg.Session.Secure,
	})

	pipeline := generate.New(generate.Options{
		Store:          db,
		LLM:            llmClient,
		Tika:           tikaClient,
		Storage:        storage,
		Logger:         log,
		MaxCharsPerDoc: cfg.Limits.MaxExtractedChars,
	})

	api := server.New(server.Options{
		Config:   cfg,
		Store:    db,
		Auth:     authService,
		Storage:  storage,
		Pipeline: pipeline,
		Tika:     tikaClient,
		Logger:   log,
	})

	go cleanupLoop(ctx, authService, log)

	srv := api.Server(api.Routes())

	errs := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errs <- err
		}
	}()

	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
		log.Info("shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown did not finish cleanly", "err", err)
		return err
	}

	log.Info("stopped")
	return nil
}

// Brings the database schema up to date before the server accepts any requests.
func runMigrations(ctx context.Context, db *store.Store, log *slog.Logger) error {
	sqlDB := db.SQLDB()
	defer sqlDB.Close()

	if err := migrations.Up(ctx, sqlDB); err != nil {
		return err
	}

	version, err := migrations.Version(ctx, sqlDB)
	if err != nil {
		return err
	}

	log.Info("schema is up to date", "version", version)
	return nil
}

// Clears out expired sessions and old rate-limit records, now and then.
func cleanupLoop(ctx context.Context, authService *auth.Service, log *slog.Logger) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		if err := authService.Cleanup(ctx); err != nil && ctx.Err() == nil {
			log.Error("cleanup expired rows", "err", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Works out where to write the log file, keeping it on the data volume so it
// survives a container restart.
func logPath(cfg config.Config) string {
	if cfg.Log.File == "" {
		return ""
	}
	if filepathIsAbsolute(cfg.Log.File) {
		return cfg.Log.File
	}
	return cfg.DataDir + "/" + cfg.Log.File
}

// Says whether a path is already a full path from the root of the disk.
func filepathIsAbsolute(p string) bool {
	return len(p) > 0 && p[0] == '/'
}
