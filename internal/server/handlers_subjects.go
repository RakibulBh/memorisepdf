package server

import (
	"net/http"
	"strings"
	"unicode"

	"github.com/RakibulBh/memorisepdf/internal/auth"
)

// The longest a subject or module name may be.
const maxNameLength = 120

type subjectRequest struct {
	Name   string `json:"name"`
	Colour string `json:"colour"`
}

// Lists the courses somebody has created.
func (s *Server) handleListSubjects(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUser(r.Context())

	subjects, err := s.store.ListSubjects(r.Context(), user.ID)
	if err != nil {
		s.writeStoreError(w, err, "")
		return
	}
	s.writeJSON(w, http.StatusOK, subjects)
}

// Creates a course.
func (s *Server) handleCreateSubject(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUser(r.Context())

	var req subjectRequest
	if err := s.readJSON(w, r, &req); err != nil {
		s.writeError(w, http.StatusBadRequest, "could not read request: "+err.Error())
		return
	}

	name, ok := cleanName(req.Name)
	if !ok {
		s.writeError(w, http.StatusBadRequest, "name must be between 1 and 120 characters")
		return
	}

	subject, err := s.store.CreateSubject(r.Context(), user.ID, name, strings.TrimSpace(req.Colour))
	if err != nil {
		s.writeStoreError(w, err, "you already have a subject with that name")
		return
	}
	s.writeJSON(w, http.StatusCreated, subject)
}

// Reads one course.
func (s *Server) handleGetSubject(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUser(r.Context())

	id, err := idParam(r, "subjectID")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid subject id")
		return
	}

	subject, err := s.store.Subject(r.Context(), user.ID, id)
	if err != nil {
		s.writeStoreError(w, err, "")
		return
	}
	s.writeJSON(w, http.StatusOK, subject)
}

// Renames or recolours a course.
func (s *Server) handleUpdateSubject(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUser(r.Context())

	id, err := idParam(r, "subjectID")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid subject id")
		return
	}

	var req subjectRequest
	if err := s.readJSON(w, r, &req); err != nil {
		s.writeError(w, http.StatusBadRequest, "could not read request: "+err.Error())
		return
	}

	name, ok := cleanName(req.Name)
	if !ok {
		s.writeError(w, http.StatusBadRequest, "name must be between 1 and 120 characters")
		return
	}

	subject, err := s.store.UpdateSubject(r.Context(), user.ID, id, name, strings.TrimSpace(req.Colour))
	if err != nil {
		s.writeStoreError(w, err, "you already have a subject with that name")
		return
	}
	s.writeJSON(w, http.StatusOK, subject)
}

// Deletes a course, everything inside it, and its files on disk.
func (s *Server) handleDeleteSubject(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUser(r.Context())

	id, err := idParam(r, "subjectID")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid subject id")
		return
	}

	paths, err := s.store.DeleteSubject(r.Context(), user.ID, id)
	if err != nil {
		s.writeStoreError(w, err, "")
		return
	}

	// The database rows are gone either way, so a file that will not delete is
	// recorded rather than failing the request.
	if err := s.storage.RemoveAll(paths); err != nil {
		s.log.Error("remove files for deleted subject", "subjectId", id, "err", err)
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"status":       "deleted",
		"filesRemoved": len(paths),
	})
}

type moduleRequest struct {
	Name string `json:"name"`
}

// Lists the modules of one course.
func (s *Server) handleListModules(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUser(r.Context())

	subjectID, err := idParam(r, "subjectID")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid subject id")
		return
	}

	// Confirms the course exists and belongs to the caller, so an unknown id
	// gives a clear 404 rather than a confusing empty list.
	if _, err := s.store.Subject(r.Context(), user.ID, subjectID); err != nil {
		s.writeStoreError(w, err, "")
		return
	}

	modules, err := s.store.ListModules(r.Context(), user.ID, subjectID)
	if err != nil {
		s.writeStoreError(w, err, "")
		return
	}
	s.writeJSON(w, http.StatusOK, modules)
}

// Adds a module to a course.
func (s *Server) handleCreateModule(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUser(r.Context())

	subjectID, err := idParam(r, "subjectID")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid subject id")
		return
	}

	var req moduleRequest
	if err := s.readJSON(w, r, &req); err != nil {
		s.writeError(w, http.StatusBadRequest, "could not read request: "+err.Error())
		return
	}

	name, ok := cleanName(req.Name)
	if !ok {
		s.writeError(w, http.StatusBadRequest, "name must be between 1 and 120 characters")
		return
	}

	module, err := s.store.CreateModule(r.Context(), user.ID, subjectID, name)
	if err != nil {
		s.writeStoreError(w, err, "that subject already has a module with that name")
		return
	}
	s.writeJSON(w, http.StatusCreated, module)
}

// Reads one module.
func (s *Server) handleGetModule(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUser(r.Context())

	id, err := idParam(r, "moduleID")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid module id")
		return
	}

	module, err := s.store.Module(r.Context(), user.ID, id)
	if err != nil {
		s.writeStoreError(w, err, "")
		return
	}
	s.writeJSON(w, http.StatusOK, module)
}

// Renames a module.
func (s *Server) handleUpdateModule(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUser(r.Context())

	id, err := idParam(r, "moduleID")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid module id")
		return
	}

	var req moduleRequest
	if err := s.readJSON(w, r, &req); err != nil {
		s.writeError(w, http.StatusBadRequest, "could not read request: "+err.Error())
		return
	}

	name, ok := cleanName(req.Name)
	if !ok {
		s.writeError(w, http.StatusBadRequest, "name must be between 1 and 120 characters")
		return
	}

	module, err := s.store.UpdateModule(r.Context(), user.ID, id, name)
	if err != nil {
		s.writeStoreError(w, err, "that subject already has a module with that name")
		return
	}
	s.writeJSON(w, http.StatusOK, module)
}

// Deletes a module, everything generated from its files, and the files themselves.
func (s *Server) handleDeleteModule(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUser(r.Context())

	id, err := idParam(r, "moduleID")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid module id")
		return
	}

	paths, err := s.store.DeleteModule(r.Context(), user.ID, id)
	if err != nil {
		s.writeStoreError(w, err, "")
		return
	}

	if err := s.storage.RemoveAll(paths); err != nil {
		s.log.Error("remove files for deleted module", "moduleId", id, "err", err)
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"status":       "deleted",
		"filesRemoved": len(paths),
	})
}

// Tidies up a name and says whether it is usable.
func cleanName(name string) (string, bool) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" || len([]rune(trimmed)) > maxNameLength {
		return "", false
	}

	// Control characters are invisible but still display somewhere, so a name
	// containing them would be impossible to tell apart from another one.
	for _, r := range trimmed {
		if r == 0 || unicode.IsControl(r) {
			return "", false
		}
	}

	return trimmed, true
}
