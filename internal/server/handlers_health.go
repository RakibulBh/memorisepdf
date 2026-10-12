package server

import (
	"context"
	"net/http"
	"sort"
	"time"

	"github.com/RakibulBh/unmemorize/internal/files"
)

// Answers that the server is running. Used by the deploy health check.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"env":    s.cfg.Env,
	})
}

// Answers whether the database and Tika are both reachable, so a deploy knows
// the container is actually able to serve rather than merely running.
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	checks := map[string]string{}
	ready := true

	if err := s.store.Ping(ctx); err != nil {
		checks["database"] = err.Error()
		ready = false
	} else {
		checks["database"] = "ok"
	}

	if err := s.tika.Ping(ctx); err != nil {
		checks["tika"] = err.Error()
		ready = false
	} else {
		checks["tika"] = "ok"
	}

	status := http.StatusOK
	if !ready {
		status = http.StatusServiceUnavailable
	}

	s.writeJSON(w, status, map[string]any{
		"ready":  ready,
		"checks": checks,
	})
}

// Lists the uploadable file types in a stable order, so the UI does not
// reshuffle them between requests.
func supportedExtensions() []string {
	exts := files.SupportedExtensions()
	sort.Strings(exts)
	return exts
}
