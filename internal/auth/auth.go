// Package auth signs people in with their Google account and remembers them
// with a session token kept in the database.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/RakibulBh/unmemorize/internal/store"
)

// How long a half-finished sign-in stays valid before it has to be restarted.
const stateTTL = 10 * time.Minute

// Where Google publishes the details of the signed-in account.
const userInfoURL = "https://www.googleapis.com/oauth2/v3/userinfo"

type Service struct {
	store      *store.Store
	oauth      *oauth2.Config
	cookieName string
	sessionTTL time.Duration
	secure     bool
	http       *http.Client
}

type Options struct {
	Store        *store.Store
	ClientID     string
	ClientSecret string
	RedirectURL  string
	CookieName   string
	SessionTTL   time.Duration
	Secure       bool
}

// Sets up Google sign-in.
func New(opts Options) *Service {
	return &Service{
		store: opts.Store,
		oauth: &oauth2.Config{
			ClientID:     opts.ClientID,
			ClientSecret: opts.ClientSecret,
			RedirectURL:  opts.RedirectURL,
			Endpoint:     google.Endpoint,
			Scopes:       []string{"openid", "email", "profile"},
		},
		cookieName: opts.CookieName,
		sessionTTL: opts.SessionTTL,
		secure:     opts.Secure,
		http:       &http.Client{Timeout: 15 * time.Second},
	}
}

// Begins a sign-in, giving back the Google URL to send somebody to.
func (s *Service) AuthCodeURL(ctx context.Context, redirectTo string) (string, error) {
	state, err := randomToken()
	if err != nil {
		return "", err
	}
	verifier := oauth2.GenerateVerifier()

	if err := s.store.CreateOAuthState(ctx, state, verifier, redirectTo, time.Now().Add(stateTTL)); err != nil {
		return "", fmt.Errorf("record sign-in attempt: %w", err)
	}

	return s.oauth.AuthCodeURL(state,
		oauth2.AccessTypeOnline,
		oauth2.S256ChallengeOption(verifier),
	), nil
}

// ErrBadState means the reply from Google did not match a sign-in we started.
var ErrBadState = errors.New("sign-in request expired or was not recognised")

// Finishes a sign-in: checks the reply came from a sign-in we started, asks
// Google who the person is, and creates their session.
func (s *Service) Complete(ctx context.Context, state, code, userAgent, ip string) (user store.User, token, redirectTo string, err error) {
	verifier, redirectTo, err := s.store.ConsumeOAuthState(ctx, state)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return store.User{}, "", "", ErrBadState
		}
		return store.User{}, "", "", fmt.Errorf("read sign-in attempt: %w", err)
	}

	tok, err := s.oauth.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return store.User{}, "", "", fmt.Errorf("exchange code with google: %w", err)
	}

	profile, err := s.fetchProfile(ctx, tok)
	if err != nil {
		return store.User{}, "", "", err
	}
	if profile.Sub == "" || profile.Email == "" {
		return store.User{}, "", "", errors.New("google did not return an identity")
	}
	if !profile.EmailVerified {
		return store.User{}, "", "", errors.New("google account email is not verified")
	}

	user, err = s.store.UpsertUserByGoogle(ctx, profile.Sub, profile.Email, profile.Name, profile.Picture)
	if err != nil {
		return store.User{}, "", "", fmt.Errorf("save account: %w", err)
	}

	token, err = randomToken()
	if err != nil {
		return store.User{}, "", "", err
	}

	// Only the hash of the token is stored, so a leaked database row cannot be
	// replayed as a live session.
	if err := s.store.CreateSession(ctx, hashToken(token), user.ID,
		time.Now().Add(s.sessionTTL), userAgent, ip); err != nil {
		return store.User{}, "", "", fmt.Errorf("create session: %w", err)
	}

	return user, token, redirectTo, nil
}

type googleProfile struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

// Asks Google for the name, email and picture of the signed-in account.
func (s *Service) fetchProfile(ctx context.Context, tok *oauth2.Token) (googleProfile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userInfoURL, nil)
	if err != nil {
		return googleProfile{}, err
	}
	tok.SetAuthHeader(req)

	resp, err := s.http.Do(req)
	if err != nil {
		return googleProfile{}, fmt.Errorf("fetch google profile: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil {
		return googleProfile{}, fmt.Errorf("read google profile: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return googleProfile{}, fmt.Errorf("google profile request returned %s", resp.Status)
	}

	var profile googleProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return googleProfile{}, fmt.Errorf("parse google profile: %w", err)
	}
	return profile, nil
}

// Finds who a session token belongs to.
func (s *Service) UserByToken(ctx context.Context, token string) (store.User, error) {
	return s.store.UserBySession(ctx, hashToken(token))
}

// Signs a session out.
func (s *Service) Revoke(ctx context.Context, token string) error {
	return s.store.DeleteSession(ctx, hashToken(token))
}

// Puts the session cookie on a response.
func (s *Service) SetCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.cookieName,
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(s.sessionTTL),
		MaxAge:   int(s.sessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// Takes the session cookie off a response, signing the browser out.
func (s *Service) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// Reads the session token out of a request's cookies.
func (s *Service) TokenFromRequest(r *http.Request) string {
	c, err := r.Cookie(s.cookieName)
	if err != nil {
		return ""
	}
	return c.Value
}

// Removes sessions and sign-in attempts that have run out, and rate-limit
// records too old to matter.
func (s *Service) Cleanup(ctx context.Context) error {
	if _, err := s.store.DeleteExpiredAuthRows(ctx); err != nil {
		return err
	}
	_, err := s.store.DeleteOldGenerateEvents(ctx)
	return err
}

// Makes a long unguessable string for a session or sign-in attempt.
func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Scrambles a session token one way, so the database never holds the real thing.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
