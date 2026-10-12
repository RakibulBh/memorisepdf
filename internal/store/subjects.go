package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Starts a new course for somebody.
func (s *Store) CreateSubject(ctx context.Context, userID uuid.UUID, name, colour string) (Subject, error) {
	const q = `
		INSERT INTO subjects (user_id, name, colour)
		VALUES ($1, btrim($2), $3)
		RETURNING id, user_id, name, colour, created_at, updated_at`

	var sub Subject
	err := s.pool.QueryRow(ctx, q, userID, name, colour).Scan(
		&sub.ID, &sub.UserID, &sub.Name, &sub.Colour, &sub.CreatedAt, &sub.UpdatedAt,
	)
	return sub, wrap(err)
}

// Lists somebody's courses, newest first, with how many modules each holds.
func (s *Store) ListSubjects(ctx context.Context, userID uuid.UUID) ([]Subject, error) {
	const q = `
		SELECT s.id, s.user_id, s.name, s.colour, s.created_at, s.updated_at,
		       count(m.id)::int AS module_count
		FROM subjects s
		LEFT JOIN modules m ON m.subject_id = s.id
		WHERE s.user_id = $1
		GROUP BY s.id
		ORDER BY s.created_at DESC`

	rows, err := s.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, wrap(err)
	}
	defer rows.Close()

	subjects := []Subject{}
	for rows.Next() {
		var sub Subject
		if err := rows.Scan(&sub.ID, &sub.UserID, &sub.Name, &sub.Colour,
			&sub.CreatedAt, &sub.UpdatedAt, &sub.ModuleCount); err != nil {
			return nil, wrap(err)
		}
		subjects = append(subjects, sub)
	}
	return subjects, wrap(rows.Err())
}

// Reads one course, but only if it belongs to the person asking.
func (s *Store) Subject(ctx context.Context, userID, id uuid.UUID) (Subject, error) {
	const q = `
		SELECT s.id, s.user_id, s.name, s.colour, s.created_at, s.updated_at,
		       (SELECT count(*)::int FROM modules m WHERE m.subject_id = s.id)
		FROM subjects s
		WHERE s.id = $1 AND s.user_id = $2`

	var sub Subject
	err := s.pool.QueryRow(ctx, q, id, userID).Scan(
		&sub.ID, &sub.UserID, &sub.Name, &sub.Colour, &sub.CreatedAt, &sub.UpdatedAt, &sub.ModuleCount,
	)
	return sub, wrap(err)
}

// Renames or recolours a course.
func (s *Store) UpdateSubject(ctx context.Context, userID, id uuid.UUID, name, colour string) (Subject, error) {
	const q = `
		UPDATE subjects
		SET name = btrim($3), colour = $4, updated_at = now()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, name, colour, created_at, updated_at`

	var sub Subject
	err := s.pool.QueryRow(ctx, q, id, userID, name, colour).Scan(
		&sub.ID, &sub.UserID, &sub.Name, &sub.Colour, &sub.CreatedAt, &sub.UpdatedAt,
	)
	return sub, wrap(err)
}

// Removes a course along with every module, file and question under it,
// returning where those files sit on disk so they can be deleted too.
func (s *Store) DeleteSubject(ctx context.Context, userID, id uuid.UUID) ([]string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	defer tx.Rollback(ctx)

	paths, err := orphanedPaths(ctx, tx, userID, `
		SELECT f.sha256, f.disk_path FROM files f
		JOIN modules m ON m.id = f.module_id
		WHERE m.subject_id = $1 AND f.user_id = $2`, id, userID)
	if err != nil {
		return nil, err
	}

	tag, err := tx.Exec(ctx, `DELETE FROM subjects WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return nil, wrap(err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, wrap(err)
	}
	return paths, nil
}

// Works out which files on disk nobody else is still pointing at, so deleting
// a subject never removes bytes another module shares.
func orphanedPaths(ctx context.Context, tx pgx.Tx, userID uuid.UUID, query string, args ...any) ([]string, error) {
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, wrap(err)
	}

	type candidate struct {
		sha  string
		path string
	}
	var candidates []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.sha, &c.path); err != nil {
			rows.Close()
			return nil, wrap(err)
		}
		candidates = append(candidates, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, wrap(err)
	}

	// The rows being deleted are still present inside this transaction, so a
	// reference count of one means this is the last holder of those bytes. The
	// count is per user because the path on disk is per user, so two people
	// with identical slides never block each other's cleanup.
	var paths []string
	seen := make(map[string]bool, len(candidates))
	for _, c := range candidates {
		if seen[c.sha] {
			continue
		}
		seen[c.sha] = true

		var refs int
		if err := tx.QueryRow(ctx,
			`SELECT count(*)::int FROM files WHERE sha256 = $1 AND user_id = $2`,
			c.sha, userID).Scan(&refs); err != nil {
			return nil, wrap(err)
		}
		if refs <= 1 {
			paths = append(paths, c.path)
		}
	}
	return paths, nil
}
