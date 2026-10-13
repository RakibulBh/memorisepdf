package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Finds the account behind a Google identity, creating it on first sign-in.
func (s *Store) UpsertUserByGoogle(ctx context.Context, googleSub, email, name, pictureURL string) (User, error) {
	const q = `
		INSERT INTO users (google_sub, email, name, picture_url)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (google_sub) DO UPDATE
		SET email         = EXCLUDED.email,
		    name          = EXCLUDED.name,
		    picture_url   = EXCLUDED.picture_url,
		    last_login_at = now()
		RETURNING id, google_sub, email, name, picture_url, created_at, last_login_at`

	var u User
	err := s.pool.QueryRow(ctx, q, googleSub, email, name, pictureURL).Scan(
		&u.ID, &u.GoogleSub, &u.Email, &u.Name, &u.PictureURL, &u.CreatedAt, &u.LastLoginAt,
	)
	return u, wrap(err)
}

// Looks up one account by its id.
func (s *Store) User(ctx context.Context, id uuid.UUID) (User, error) {
	const q = `
		SELECT id, google_sub, email, name, picture_url, created_at, last_login_at
		FROM users WHERE id = $1`

	var u User
	err := s.pool.QueryRow(ctx, q, id).Scan(
		&u.ID, &u.GoogleSub, &u.Email, &u.Name, &u.PictureURL, &u.CreatedAt, &u.LastLoginAt,
	)
	return u, wrap(err)
}

// Records a new signed-in session and the device it came from.
func (s *Store) CreateSession(ctx context.Context, token string, userID uuid.UUID, expiresAt time.Time, userAgent, ip string) error {
	const q = `
		INSERT INTO sessions (token, user_id, expires_at, user_agent, ip_address)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := s.pool.Exec(ctx, q, token, userID, expiresAt, userAgent, ip)
	return wrap(err)
}

// Finds the account for a session token, ignoring tokens that have run out.
func (s *Store) UserBySession(ctx context.Context, token string) (User, error) {
	const q = `
		SELECT u.id, u.google_sub, u.email, u.name, u.picture_url, u.created_at, u.last_login_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token = $1 AND s.expires_at > now()`

	var u User
	err := s.pool.QueryRow(ctx, q, token).Scan(
		&u.ID, &u.GoogleSub, &u.Email, &u.Name, &u.PictureURL, &u.CreatedAt, &u.LastLoginAt,
	)
	return u, wrap(err)
}

// Signs a session out for good.
func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token = $1`, token)
	return wrap(err)
}

// Clears out sessions and sign-in attempts that have expired.
func (s *Store) DeleteExpiredAuthRows(ctx context.Context) (int64, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at < now()`)
	if err != nil {
		return 0, wrap(err)
	}
	removed := tag.RowsAffected()

	tag, err = s.pool.Exec(ctx, `DELETE FROM oauth_states WHERE expires_at < now()`)
	if err != nil {
		return removed, wrap(err)
	}
	return removed + tag.RowsAffected(), nil
}

// Remembers a sign-in attempt so the reply from Google can be trusted.
func (s *Store) CreateOAuthState(ctx context.Context, state, verifier, redirectTo string, expiresAt time.Time) error {
	const q = `
		INSERT INTO oauth_states (state, code_verifier, redirect_to, expires_at)
		VALUES ($1, $2, $3, $4)`

	_, err := s.pool.Exec(ctx, q, state, verifier, redirectTo, expiresAt)
	return wrap(err)
}

// Reads back a sign-in attempt and removes it, so it can only be used once.
func (s *Store) ConsumeOAuthState(ctx context.Context, state string) (verifier, redirectTo string, err error) {
	const q = `
		DELETE FROM oauth_states
		WHERE state = $1 AND expires_at > now()
		RETURNING code_verifier, redirect_to`

	err = s.pool.QueryRow(ctx, q, state).Scan(&verifier, &redirectTo)
	return verifier, redirectTo, wrap(err)
}
