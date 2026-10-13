package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/RakibulBh/unmemorize/internal/auth"
)

// Sends somebody off to Google to sign in.
func (s *Server) handleGoogleStart(w http.ResponseWriter, r *http.Request) {
	redirectTo := s.safeRedirect(r.URL.Query().Get("redirect"))

	url, err := s.auth.AuthCodeURL(r.Context(), redirectTo)
	if err != nil {
		s.log.Error("start google sign-in", "err", err)
		s.writeError(w, http.StatusInternalServerError, "could not start sign-in")
		return
	}

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// Receives the reply from Google, creates the session, and sends the browser
// back to the app.
func (s *Server) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// Somebody who declines the Google consent screen arrives here with an
	// error rather than a code, which is not a fault worth logging.
	if reason := q.Get("error"); reason != "" {
		http.Redirect(w, r, s.cfg.WebURL+"/login?error="+reason, http.StatusTemporaryRedirect)
		return
	}

	state, code := q.Get("state"), q.Get("code")
	if state == "" || code == "" {
		s.writeError(w, http.StatusBadRequest, "missing state or code")
		return
	}

	user, token, redirectTo, err := s.auth.Complete(r.Context(), state, code,
		r.UserAgent(), r.RemoteAddr)
	if err != nil {
		if errors.Is(err, auth.ErrBadState) {
			http.Redirect(w, r, s.cfg.WebURL+"/login?error=expired", http.StatusTemporaryRedirect)
			return
		}
		s.log.Error("complete google sign-in", "err", err)
		http.Redirect(w, r, s.cfg.WebURL+"/login?error=failed", http.StatusTemporaryRedirect)
		return
	}

	s.auth.SetCookie(w, token)
	s.log.Info("signed in", "userId", user.ID, "email", user.Email)

	target := s.cfg.WebURL + "/dashboard"
	if redirectTo != "" {
		target = s.cfg.WebURL + redirectTo
	}
	http.Redirect(w, r, target, http.StatusTemporaryRedirect)
}

// Signs somebody out and forgets their session.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if token := s.auth.TokenFromRequest(r); token != "" {
		if err := s.auth.Revoke(r.Context(), token); err != nil {
			s.log.Error("revoke session", "err", err)
		}
	}

	s.auth.ClearCookie(w)
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "signed out"})
}

// Reports who is signed in.
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, auth.MustUser(r.Context()))
}

// Keeps a sign-in redirect inside this app, so the parameter cannot be used to
// bounce somebody to another site.
func (s *Server) safeRedirect(path string) string {
	if path == "" {
		return ""
	}
	// A protocol-relative path like //evil.com is a full URL to the browser,
	// so only a plain single-slash path is accepted.
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		return ""
	}
	return path
}
