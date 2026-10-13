package auth

import (
	"context"
	"errors"
	"net/http"

	"github.com/RakibulBh/memorisepdf/internal/store"
)

type contextKey struct{}

var userKey contextKey

// Turns away anybody without a valid session, and makes the signed-in account
// available to the handler behind it.
func (s *Service) RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := s.TokenFromRequest(r)
		if token == "" {
			unauthorised(w)
			return
		}

		user, err := s.UserByToken(r.Context(), token)
		if err != nil {
			// An unknown or expired token leaves a cookie the browser will keep
			// sending, so it is cleared on the way out.
			if errors.Is(err, store.ErrNotFound) {
				s.ClearCookie(w)
				unauthorised(w)
				return
			}
			http.Error(w, `{"error":"could not verify session"}`, http.StatusInternalServerError)
			return
		}

		next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), user)))
	})
}

// Attaches a signed-in account to a request's context.
func WithUser(ctx context.Context, user store.User) context.Context {
	return context.WithValue(ctx, userKey, user)
}

// Reads the signed-in account back out of a request's context.
func UserFrom(ctx context.Context) (store.User, bool) {
	user, ok := ctx.Value(userKey).(store.User)
	return user, ok
}

// Reads the signed-in account, trusting that RequireUser already ran.
func MustUser(ctx context.Context) store.User {
	user, ok := UserFrom(ctx)
	if !ok {
		panic("auth: no user in context; handler is not behind RequireUser")
	}
	return user
}

// Replies that the caller needs to sign in first.
func unauthorised(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":"not signed in"}`))
}
