// Package migrations carries the database schema inside the binary, so a deploy
// can never start against a schema older than the code expects.
package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var files embed.FS

// Brings the database up to the newest schema the binary knows about.
func Up(ctx context.Context, db *sql.DB) error {
	goose.SetBaseFS(files)
	goose.SetLogger(goose.NopLogger())

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}
	if err := goose.UpContext(ctx, db, "."); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

// Reports which schema version the database is currently on.
func Version(ctx context.Context, db *sql.DB) (int64, error) {
	goose.SetBaseFS(files)
	if err := goose.SetDialect("postgres"); err != nil {
		return 0, err
	}
	return goose.GetDBVersionContext(ctx, db)
}
