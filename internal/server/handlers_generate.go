package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/RakibulBh/unmemorize/internal/auth"
	"github.com/RakibulBh/unmemorize/internal/generate"
	"github.com/RakibulBh/unmemorize/internal/llm"
	"github.com/RakibulBh/unmemorize/internal/store"
)

type generateRequest struct {
	Kind    string `json:"kind"`
	Variant string `json:"variant"`
}

// Analyses a module's new files and streams progress back as it happens.
//
// This answers a POST with an event stream rather than using a separate
// subscribe endpoint: the work and the progress for it then share one request,
// so a dropped connection cancels the work it belonged to and no half-finished
// job can be left behind.
func (s *Server) handleGenerate(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUser(r.Context())

	moduleID, err := idParam(r, "moduleID")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid module id")
		return
	}

	var req generateRequest
	if err := s.readJSON(w, r, &req); err != nil {
		s.writeError(w, http.StatusBadRequest, "could not read request: "+err.Error())
		return
	}

	kind, variant, ok := normaliseGenerateRequest(req)
	if !ok {
		s.writeError(w, http.StatusBadRequest,
			"kind must be quiz, flashcard or teaching, with a matching difficulty or style")
		return
	}

	if _, err := s.store.Module(r.Context(), user.ID, moduleID); err != nil {
		s.writeStoreError(w, err, "")
		return
	}

	// Checked before the stream starts, so a refusal is an ordinary JSON error
	// the frontend can show plainly.
	count, err := s.store.GenerateEventsSince(r.Context(), user.ID, time.Hour)
	if err != nil {
		s.writeStoreError(w, err, "")
		return
	}
	if count >= s.cfg.Limits.GeneratePerHour {
		s.writeError(w, http.StatusTooManyRequests,
			fmt.Sprintf("you have reached the limit of %d generations per hour",
				s.cfg.Limits.GeneratePerHour))
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeError(w, http.StatusInternalServerError, "streaming is not supported")
		return
	}

	if err := s.store.RecordGenerateEvent(r.Context(), user.ID); err != nil {
		s.log.Error("record generate event", "err", err)
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// Stops a reverse proxy holding the stream back until it completes.
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	send := func(event string, payload any) {
		body, err := json.Marshal(payload)
		if err != nil {
			s.log.Error("encode progress event", "err", err)
			return
		}
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, body)
		flusher.Flush()
	}

	result, err := s.pipeline.Run(r.Context(), user.ID, moduleID, kind, variant,
		func(p generate.Progress) { send("progress", p) })

	if err != nil {
		switch {
		case errors.Is(err, generate.ErrNoFiles):
			send("error", map[string]string{"error": "upload some lecture files first"})
		case errors.Is(err, r.Context().Err()):
			// The browser went away; there is nobody left to tell.
			s.log.Info("generation cancelled by client",
				"moduleId", moduleID, "kind", kind)
		default:
			s.log.Error("generation failed", "moduleId", moduleID, "kind", kind, "err", err)
			send("error", map[string]string{"error": "generation failed"})
		}
		return
	}

	send("done", result)
}

// Checks a generation request and fills in the default setting for its kind.
func normaliseGenerateRequest(req generateRequest) (kind, variant string, ok bool) {
	switch req.Kind {
	case store.KindQuiz:
		variant = req.Variant
		if variant == "" {
			variant = "medium"
		}
		return store.KindQuiz, variant, llm.ValidDifficulty(variant)

	case store.KindFlashcard:
		// Flashcards have no setting, so the variant is always empty. Anything
		// else would create a second ledger row for the same work.
		return store.KindFlashcard, "", true

	case store.KindTeaching:
		variant = req.Variant
		if variant == "" {
			variant = "simple-language"
		}
		return store.KindTeaching, variant, llm.ValidTeachingStyle(variant)

	default:
		return "", "", false
	}
}

// Hands back every question stored for a module at one difficulty.
func (s *Server) handleModuleQuiz(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUser(r.Context())

	moduleID, err := idParam(r, "moduleID")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid module id")
		return
	}

	difficulty := r.URL.Query().Get("difficulty")
	if difficulty == "" {
		difficulty = "medium"
	}
	if !llm.ValidDifficulty(difficulty) {
		s.writeError(w, http.StatusBadRequest, "difficulty must be easy, medium or hard")
		return
	}

	if _, err := s.store.Module(r.Context(), user.ID, moduleID); err != nil {
		s.writeStoreError(w, err, "")
		return
	}

	questions, err := s.store.ModuleQuiz(r.Context(), user.ID, moduleID, difficulty)
	if err != nil {
		s.writeStoreError(w, err, "")
		return
	}

	pending, err := s.store.FilesNeedingGeneration(r.Context(), user.ID, moduleID, store.KindQuiz, difficulty)
	if err != nil {
		s.writeStoreError(w, err, "")
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"difficulty":   difficulty,
		"questions":    questions,
		"filesPending": len(pending),
	})
}

// Hands back every flashcard stored for a module.
func (s *Server) handleModuleFlashcards(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUser(r.Context())

	moduleID, err := idParam(r, "moduleID")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid module id")
		return
	}

	if _, err := s.store.Module(r.Context(), user.ID, moduleID); err != nil {
		s.writeStoreError(w, err, "")
		return
	}

	cards, err := s.store.ModuleFlashcards(r.Context(), user.ID, moduleID)
	if err != nil {
		s.writeStoreError(w, err, "")
		return
	}

	pending, err := s.store.FilesNeedingGeneration(r.Context(), user.ID, moduleID, store.KindFlashcard, "")
	if err != nil {
		s.writeStoreError(w, err, "")
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"flashcards":   cards,
		"filesPending": len(pending),
	})
}

// Hands back every teaching card stored for a module in one style.
func (s *Server) handleModuleTeaching(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUser(r.Context())

	moduleID, err := idParam(r, "moduleID")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid module id")
		return
	}

	style := r.URL.Query().Get("style")
	if style == "" {
		style = "simple-language"
	}
	if !llm.ValidTeachingStyle(style) {
		s.writeError(w, http.StatusBadRequest,
			"style must be simple-language, analogy-driven or scaffolded-learning")
		return
	}

	if _, err := s.store.Module(r.Context(), user.ID, moduleID); err != nil {
		s.writeStoreError(w, err, "")
		return
	}

	cards, err := s.store.ModuleTeachingCards(r.Context(), user.ID, moduleID, style)
	if err != nil {
		s.writeStoreError(w, err, "")
		return
	}

	pending, err := s.store.FilesNeedingGeneration(r.Context(), user.ID, moduleID, store.KindTeaching, style)
	if err != nil {
		s.writeStoreError(w, err, "")
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"style":         style,
		"teachingCards": cards,
		"filesPending":  len(pending),
	})
}

// Tells the frontend which difficulties, styles and file types exist, so the
// options are never hardcoded in two places.
func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]any{
		"difficulties":        llm.Difficulties,
		"teachingStyles":      llm.TeachingStyles,
		"supportedExtensions": supportedExtensions(),
		"maxFileSize":         s.cfg.Limits.MaxFileSize,
		"maxFilesPerModule":   s.cfg.Limits.MaxFilesPerModule,
		"storageQuota":        s.cfg.Limits.UserStorageQuota,
	})
}
