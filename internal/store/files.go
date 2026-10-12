package store

import (
	"context"

	"github.com/google/uuid"
)

// Records an uploaded lecture file against a module.
func (s *Store) CreateFile(ctx context.Context, userID, moduleID uuid.UUID, f File) (File, error) {
	// Checking the module through its subject keeps the ownership rule in the
	// same statement as the insert.
	const q = `
		INSERT INTO files (module_id, user_id, original_name, sha256, byte_size,
		                   content_type, disk_path, extracted_text)
		SELECT m.id, $2, $3, $4, $5, $6, $7, $8
		FROM modules m
		JOIN subjects s ON s.id = m.subject_id
		WHERE m.id = $1 AND s.user_id = $2
		RETURNING id, module_id, user_id, original_name, sha256, byte_size,
		          content_type, disk_path, uploaded_at`

	var out File
	err := s.pool.QueryRow(ctx, q, moduleID, userID, f.OriginalName, f.SHA256,
		f.ByteSize, f.ContentType, f.DiskPath, f.ExtractedText).Scan(
		&out.ID, &out.ModuleID, &out.UserID, &out.OriginalName, &out.SHA256,
		&out.ByteSize, &out.ContentType, &out.DiskPath, &out.UploadedAt,
	)
	return out, wrap(err)
}

// Lists a module's files along with which kinds of study material each has
// already produced, so the UI can show what is new without a second query.
func (s *Store) ListFiles(ctx context.Context, userID, moduleID uuid.UUID) ([]File, error) {
	const q = `
		SELECT f.id, f.module_id, f.user_id, f.original_name, f.sha256, f.byte_size,
		       f.content_type, f.disk_path, f.uploaded_at,
		       coalesce(array_agg(DISTINCT g.kind) FILTER (WHERE g.status = 'complete'), '{}') AS analysed
		FROM files f
		JOIN modules m  ON m.id = f.module_id
		JOIN subjects s ON s.id = m.subject_id
		LEFT JOIN generations g ON g.file_id = f.id
		WHERE f.module_id = $1 AND s.user_id = $2
		GROUP BY f.id
		ORDER BY f.original_name`

	rows, err := s.pool.Query(ctx, q, moduleID, userID)
	if err != nil {
		return nil, wrap(err)
	}
	defer rows.Close()

	files := []File{}
	for rows.Next() {
		var f File
		if err := rows.Scan(&f.ID, &f.ModuleID, &f.UserID, &f.OriginalName, &f.SHA256,
			&f.ByteSize, &f.ContentType, &f.DiskPath, &f.UploadedAt, &f.Analysed); err != nil {
			return nil, wrap(err)
		}
		files = append(files, f)
	}
	return files, wrap(rows.Err())
}

// Reads one file's details, including the text pulled out of it.
func (s *Store) File(ctx context.Context, userID, id uuid.UUID) (File, error) {
	const q = `
		SELECT f.id, f.module_id, f.user_id, f.original_name, f.sha256, f.byte_size,
		       f.content_type, f.disk_path, f.extracted_text, f.uploaded_at
		FROM files f
		WHERE f.id = $1 AND f.user_id = $2`

	var f File
	err := s.pool.QueryRow(ctx, q, id, userID).Scan(
		&f.ID, &f.ModuleID, &f.UserID, &f.OriginalName, &f.SHA256, &f.ByteSize,
		&f.ContentType, &f.DiskPath, &f.ExtractedText, &f.UploadedAt,
	)
	return f, wrap(err)
}

// Finds a file somebody has already uploaded with the same contents, so the
// same bytes are only ever written to disk once per person.
func (s *Store) FileBySHA(ctx context.Context, userID uuid.UUID, sha string) (File, error) {
	const q = `
		SELECT f.id, f.module_id, f.user_id, f.original_name, f.sha256, f.byte_size,
		       f.content_type, f.disk_path, f.extracted_text, f.uploaded_at
		FROM files f
		WHERE f.user_id = $1 AND f.sha256 = $2
		LIMIT 1`

	var f File
	err := s.pool.QueryRow(ctx, q, userID, sha).Scan(
		&f.ID, &f.ModuleID, &f.UserID, &f.OriginalName, &f.SHA256, &f.ByteSize,
		&f.ContentType, &f.DiskPath, &f.ExtractedText, &f.UploadedAt,
	)
	return f, wrap(err)
}

// Removes a file and everything generated from it, saying whether its bytes on
// disk are now unused.
func (s *Store) DeleteFile(ctx context.Context, userID, id uuid.UUID) (diskPath string, removeFromDisk bool, err error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", false, wrap(err)
	}
	defer tx.Rollback(ctx)

	var sha string
	if err := tx.QueryRow(ctx,
		`SELECT sha256, disk_path FROM files WHERE id = $1 AND user_id = $2`,
		id, userID).Scan(&sha, &diskPath); err != nil {
		return "", false, wrap(err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM files WHERE id = $1 AND user_id = $2`, id, userID); err != nil {
		return "", false, wrap(err)
	}

	var refs int
	if err := tx.QueryRow(ctx,
		`SELECT count(*)::int FROM files WHERE sha256 = $1 AND user_id = $2`,
		sha, userID).Scan(&refs); err != nil {
		return "", false, wrap(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", false, wrap(err)
	}
	return diskPath, refs == 0, nil
}

// Adds up how much disk space somebody is using, counting shared bytes once.
func (s *Store) UserStorageUsed(ctx context.Context, userID uuid.UUID) (int64, error) {
	const q = `
		SELECT coalesce(sum(byte_size), 0)::bigint
		FROM (
			SELECT DISTINCT ON (sha256) sha256, byte_size
			FROM files WHERE user_id = $1
		) distinct_files`

	var total int64
	err := s.pool.QueryRow(ctx, q, userID).Scan(&total)
	return total, wrap(err)
}

// Counts how many files a module is holding.
func (s *Store) CountFilesInModule(ctx context.Context, moduleID uuid.UUID) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx,
		`SELECT count(*)::int FROM files WHERE module_id = $1`, moduleID).Scan(&n)
	return n, wrap(err)
}

// Saves the text Tika pulled out of a file, so it never has to be read again.
func (s *Store) SetExtractedText(ctx context.Context, id uuid.UUID, text string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE files SET extracted_text = $2 WHERE id = $1`, id, text)
	return wrap(err)
}
