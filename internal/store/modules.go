package store

import (
	"context"

	"github.com/google/uuid"
)

// Adds a module to one of somebody's courses.
func (s *Store) CreateModule(ctx context.Context, userID, subjectID uuid.UUID, name string) (Module, error) {
	// The subject lookup is part of the insert, so a module can never be
	// attached to a course belonging to somebody else.
	const q = `
		INSERT INTO modules (subject_id, name)
		SELECT s.id, btrim($3)
		FROM subjects s
		WHERE s.id = $1 AND s.user_id = $2
		RETURNING id, subject_id, name, created_at, updated_at`

	var m Module
	err := s.pool.QueryRow(ctx, q, subjectID, userID, name).Scan(
		&m.ID, &m.SubjectID, &m.Name, &m.CreatedAt, &m.UpdatedAt,
	)
	return m, wrap(err)
}

// Lists the modules of one course, with how many files each holds.
func (s *Store) ListModules(ctx context.Context, userID, subjectID uuid.UUID) ([]Module, error) {
	const q = `
		SELECT m.id, m.subject_id, s.name, m.name, m.created_at, m.updated_at,
		       count(f.id)::int                  AS file_count,
		       coalesce(sum(f.byte_size), 0)::bigint AS total_bytes
		FROM modules m
		JOIN subjects s ON s.id = m.subject_id
		LEFT JOIN files f ON f.module_id = m.id
		WHERE m.subject_id = $1 AND s.user_id = $2
		GROUP BY m.id, s.name
		ORDER BY m.created_at DESC`

	rows, err := s.pool.Query(ctx, q, subjectID, userID)
	if err != nil {
		return nil, wrap(err)
	}
	defer rows.Close()

	modules := []Module{}
	for rows.Next() {
		var m Module
		if err := rows.Scan(&m.ID, &m.SubjectID, &m.SubjectName, &m.Name,
			&m.CreatedAt, &m.UpdatedAt, &m.FileCount, &m.TotalBytes); err != nil {
			return nil, wrap(err)
		}
		modules = append(modules, m)
	}
	return modules, wrap(rows.Err())
}

// Reads one module, but only if the person asking owns its course.
func (s *Store) Module(ctx context.Context, userID, id uuid.UUID) (Module, error) {
	const q = `
		SELECT m.id, m.subject_id, s.name, m.name, m.created_at, m.updated_at,
		       (SELECT count(*)::int FROM files f WHERE f.module_id = m.id),
		       (SELECT coalesce(sum(f.byte_size), 0)::bigint FROM files f WHERE f.module_id = m.id)
		FROM modules m
		JOIN subjects s ON s.id = m.subject_id
		WHERE m.id = $1 AND s.user_id = $2`

	var m Module
	err := s.pool.QueryRow(ctx, q, id, userID).Scan(
		&m.ID, &m.SubjectID, &m.SubjectName, &m.Name,
		&m.CreatedAt, &m.UpdatedAt, &m.FileCount, &m.TotalBytes,
	)
	return m, wrap(err)
}

// Renames a module.
func (s *Store) UpdateModule(ctx context.Context, userID, id uuid.UUID, name string) (Module, error) {
	const q = `
		UPDATE modules m
		SET name = btrim($3), updated_at = now()
		WHERE m.id = $1
		  AND EXISTS (SELECT 1 FROM subjects s WHERE s.id = m.subject_id AND s.user_id = $2)
		RETURNING m.id, m.subject_id, m.name, m.created_at, m.updated_at`

	var mod Module
	err := s.pool.QueryRow(ctx, q, id, userID, name).Scan(
		&mod.ID, &mod.SubjectID, &mod.Name, &mod.CreatedAt, &mod.UpdatedAt,
	)
	return mod, wrap(err)
}

// Removes a module and everything generated from its files, returning the disk
// paths that are now unused.
func (s *Store) DeleteModule(ctx context.Context, userID, id uuid.UUID) ([]string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	defer tx.Rollback(ctx)

	paths, err := orphanedPaths(ctx, tx, userID, `
		SELECT f.sha256, f.disk_path FROM files f
		WHERE f.module_id = $1 AND f.user_id = $2`, id, userID)
	if err != nil {
		return nil, err
	}

	tag, err := tx.Exec(ctx, `
		DELETE FROM modules m
		WHERE m.id = $1
		  AND EXISTS (SELECT 1 FROM subjects s WHERE s.id = m.subject_id AND s.user_id = $2)`,
		id, userID)
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
