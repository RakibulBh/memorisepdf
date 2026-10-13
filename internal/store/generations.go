package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// PendingFile is a file in a module that still needs sending to the model,
// together with the text to send.
type PendingFile struct {
	ID            uuid.UUID
	OriginalName  string
	ExtractedText string
	DiskPath      string
	ContentType   string
}

// Finds the files in a module that have never produced this kind of material at
// this setting. These are the only files a regenerate needs to pay for.
func (s *Store) FilesNeedingGeneration(ctx context.Context, userID, moduleID uuid.UUID, kind, variant string) ([]PendingFile, error) {
	// A file is skipped when a completed generation already exists for it. A
	// failed or half-finished one is not a reason to skip, so a crash or a
	// model error is retried rather than silently leaving a gap.
	const q = `
		SELECT f.id, f.original_name, f.extracted_text, f.disk_path, f.content_type
		FROM files f
		JOIN modules m  ON m.id = f.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE f.module_id = $1
		  AND s.user_id = $2
		  AND NOT EXISTS (
		      SELECT 1 FROM generations g
		      WHERE g.file_id = f.id
		        AND g.kind = $3
		        AND g.variant = $4
		        AND g.status = 'complete'
		  )
		ORDER BY f.original_name`

	rows, err := s.pool.Query(ctx, q, moduleID, userID, kind, variant)
	if err != nil {
		return nil, wrap(err)
	}
	defer rows.Close()

	pending := []PendingFile{}
	for rows.Next() {
		var p PendingFile
		if err := rows.Scan(&p.ID, &p.OriginalName, &p.ExtractedText, &p.DiskPath, &p.ContentType); err != nil {
			return nil, wrap(err)
		}
		pending = append(pending, p)
	}
	return pending, wrap(rows.Err())
}

// Counts how many of a module's files have already produced this kind of
// material at this setting, so progress can be reported honestly.
func (s *Store) CountGenerated(ctx context.Context, userID, moduleID uuid.UUID, kind, variant string) (int, error) {
	const q = `
		SELECT count(*)::int
		FROM files f
		JOIN modules m  ON m.id = f.module_id
		JOIN subjects s ON s.id = m.subject_id
		JOIN generations g ON g.file_id = f.id
		WHERE f.module_id = $1 AND s.user_id = $2
		  AND g.kind = $3 AND g.variant = $4 AND g.status = 'complete'`

	var n int
	err := s.pool.QueryRow(ctx, q, moduleID, userID, kind, variant).Scan(&n)
	return n, wrap(err)
}

// QuizItem is one generated question on its way into the database.
type QuizItem struct {
	Question string
	Answers  []QuizAnswerItem
}

type QuizAnswerItem struct {
	Text        string
	Correct     bool
	Explanation string
}

// Saves a file's generated questions and marks the file as analysed, all in one
// go, so a crash part way through never leaves a file counted but empty.
func (s *Store) SaveQuiz(ctx context.Context, fileID uuid.UUID, variant, model string, items []QuizItem) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return wrap(err)
	}
	defer tx.Rollback(ctx)

	genID, err := upsertGeneration(ctx, tx, fileID, KindQuiz, variant, model, len(items))
	if err != nil {
		return err
	}

	for i, item := range items {
		var questionID uuid.UUID
		if err := tx.QueryRow(ctx, `
			INSERT INTO quiz_questions (generation_id, file_id, question, position)
			VALUES ($1, $2, $3, $4) RETURNING id`,
			genID, fileID, item.Question, i).Scan(&questionID); err != nil {
			return wrap(err)
		}

		for j, ans := range item.Answers {
			if _, err := tx.Exec(ctx, `
				INSERT INTO quiz_answers (question_id, text, is_correct, explanation, position)
				VALUES ($1, $2, $3, $4, $5)`,
				questionID, ans.Text, ans.Correct, ans.Explanation, j); err != nil {
				return wrap(err)
			}
		}
	}

	return wrap(tx.Commit(ctx))
}

// FlashcardItem is one generated flashcard on its way into the database.
type FlashcardItem struct {
	Term       string
	Definition string
}

// Saves a file's generated flashcards and marks the file as analysed.
func (s *Store) SaveFlashcards(ctx context.Context, fileID uuid.UUID, variant, model string, items []FlashcardItem) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return wrap(err)
	}
	defer tx.Rollback(ctx)

	genID, err := upsertGeneration(ctx, tx, fileID, KindFlashcard, variant, model, len(items))
	if err != nil {
		return err
	}

	for i, item := range items {
		if _, err := tx.Exec(ctx, `
			INSERT INTO flashcards (generation_id, file_id, term, definition, position)
			VALUES ($1, $2, $3, $4, $5)`,
			genID, fileID, item.Term, item.Definition, i); err != nil {
			return wrap(err)
		}
	}

	return wrap(tx.Commit(ctx))
}

// TeachingItem is one generated teaching card on its way into the database.
type TeachingItem struct {
	Subtopic string
	Teaching string
}

// Saves a file's generated teaching cards and marks the file as analysed.
func (s *Store) SaveTeachingCards(ctx context.Context, fileID uuid.UUID, variant, model string, items []TeachingItem) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return wrap(err)
	}
	defer tx.Rollback(ctx)

	genID, err := upsertGeneration(ctx, tx, fileID, KindTeaching, variant, model, len(items))
	if err != nil {
		return err
	}

	for i, item := range items {
		if _, err := tx.Exec(ctx, `
			INSERT INTO teaching_cards (generation_id, file_id, subtopic, teaching, position)
			VALUES ($1, $2, $3, $4, $5)`,
			genID, fileID, item.Subtopic, item.Teaching, i); err != nil {
			return wrap(err)
		}
	}

	return wrap(tx.Commit(ctx))
}

// Records that a generation was attempted and failed, so the reason survives
// and the next attempt knows to try again rather than skip the file.
func (s *Store) MarkGenerationFailed(ctx context.Context, fileID uuid.UUID, kind, variant, model, reason string) error {
	const q = `
		INSERT INTO generations (file_id, kind, variant, model, status, error)
		VALUES ($1, $2, $3, $4, 'failed', $5)
		ON CONFLICT (file_id, kind, variant) DO UPDATE
		SET status = 'failed', error = EXCLUDED.error, model = EXCLUDED.model, completed_at = NULL`

	if len(reason) > 500 {
		reason = reason[:500]
	}
	_, err := s.pool.Exec(ctx, q, fileID, kind, variant, model, reason)
	return wrap(err)
}

// Claims the ledger row for a file and clears out anything a previous failed
// attempt left behind, so a retry replaces rather than appends.
func upsertGeneration(ctx context.Context, tx pgx.Tx, fileID uuid.UUID, kind, variant, model string, count int) (uuid.UUID, error) {
	const q = `
		INSERT INTO generations (file_id, kind, variant, model, status, item_count, completed_at)
		VALUES ($1, $2, $3, $4, 'complete', $5, now())
		ON CONFLICT (file_id, kind, variant) DO UPDATE
		SET status       = 'complete',
		    model        = EXCLUDED.model,
		    item_count   = EXCLUDED.item_count,
		    error        = '',
		    completed_at = now()
		RETURNING id`

	var genID uuid.UUID
	if err := tx.QueryRow(ctx, q, fileID, kind, variant, model, count).Scan(&genID); err != nil {
		return uuid.Nil, wrap(err)
	}

	// Deleting the generation's children removes the leftovers of an earlier
	// attempt. The cascade on generation_id makes this one statement per table.
	for _, table := range []string{"quiz_questions", "flashcards", "teaching_cards"} {
		if _, err := tx.Exec(ctx,
			fmt.Sprintf(`DELETE FROM %s WHERE generation_id = $1`, table), genID); err != nil {
			return uuid.Nil, wrap(err)
		}
	}

	return genID, nil
}
