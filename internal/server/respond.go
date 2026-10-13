package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/RakibulBh/unmemorize/internal/store"
)

// Sends a JSON reply.
func (s *Server) writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		// The status line has already gone out, so there is nothing to do but
		// record it.
		s.log.Error("write response", "err", err)
	}
}

// Sends an error reply the frontend can show to somebody.
func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, map[string]string{"error": message})
}

// Turns a database error into the right reply, logging the ones that are our fault.
func (s *Server) writeStoreError(w http.ResponseWriter, err error, duplicateMessage string) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		s.writeError(w, http.StatusNotFound, "not found")
	case errors.Is(err, store.ErrDuplicate):
		s.writeError(w, http.StatusConflict, duplicateMessage)
	default:
		s.log.Error("database error", "err", err)
		s.writeError(w, http.StatusInternalServerError, "something went wrong")
	}
}

// Reads a JSON request body, refusing anything oversized or unexpected.
func (s *Server) readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return err
	}
	// A second JSON value in the body means the request was not what it claimed.
	if dec.More() {
		return errors.New("body must contain a single JSON object")
	}
	return nil
}

// Reads an id out of the request path, saying so plainly if it is not one.
func idParam(r *http.Request, name string) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, name))
}
