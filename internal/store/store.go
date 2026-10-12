// Package store is the only place that talks to Postgres.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

// ErrNotFound is returned when a row that was asked for does not exist, or
// exists but belongs to somebody else — the two are deliberately
// indistinguishable to the caller so handlers cannot leak another user's ids.
var ErrNotFound = errors.New("not found")

// ErrDuplicate is returned when a uniqueness rule in the schema rejected a write.
var ErrDuplicate = errors.New("already exists")

type Store struct {
	pool *pgxpool.Pool
}

// Opens a pool of connections to Postgres and checks the database answers.
func Open(ctx context.Context, url string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute
	cfg.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	return &Store{pool: pool}, nil
}

// Hands back a standard library handle, which is what the migration runner needs.
func (s *Store) SQLDB() *sql.DB {
	return stdlib.OpenDBFromPool(s.pool)
}

// Checks the database is still reachable.
func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// Closes every connection in the pool.
func (s *Store) Close() {
	s.pool.Close()
}

// Turns a raw database error into one of this package's plain errors.
func wrap(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrDuplicate
	}
	return err
}
