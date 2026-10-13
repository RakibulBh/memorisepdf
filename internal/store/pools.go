package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Collects every question stored for a module at one difficulty, old and new
// together, with each answer attached to its question.
func (s *Store) ModuleQuiz(ctx context.Context, userID, moduleID uuid.UUID, variant string) ([]QuizQuestion, error) {
	const q = `
		SELECT qq.id, qq.file_id, f.original_name, qq.question,
		       qa.id, qa.text, qa.is_correct, qa.explanation
		FROM quiz_questions qq
		JOIN generations g ON g.id = qq.generation_id
		JOIN files f       ON f.id = qq.file_id
		JOIN modules m     ON m.id = f.module_id
		JOIN subjects s    ON s.id = m.subject_id
		LEFT JOIN quiz_answers qa ON qa.question_id = qq.id
		WHERE f.module_id = $1 AND s.user_id = $2
		  AND g.kind = 'quiz' AND g.variant = $3 AND g.status = 'complete'
		ORDER BY f.original_name, qq.position, qa.position`

	rows, err := s.pool.Query(ctx, q, moduleID, userID, variant)
	if err != nil {
		return nil, wrap(err)
	}
	defer rows.Close()

	questions := []QuizQuestion{}
	index := map[uuid.UUID]int{}

	for rows.Next() {
		var (
			qID, fileID uuid.UUID
			fileName    string
			question    string
			ansID       *uuid.UUID
			ansText     *string
			ansCorrect  *bool
			ansExplain  *string
		)
		if err := rows.Scan(&qID, &fileID, &fileName, &question,
			&ansID, &ansText, &ansCorrect, &ansExplain); err != nil {
			return nil, wrap(err)
		}

		pos, seen := index[qID]
		if !seen {
			questions = append(questions, QuizQuestion{
				ID:       qID,
				FileID:   fileID,
				FileName: fileName,
				Question: question,
				Answers:  []QuizAnswer{},
			})
			pos = len(questions) - 1
			index[qID] = pos
		}

		// A question with no answer rows yields a single NULL row from the join.
		if ansID != nil {
			questions[pos].Answers = append(questions[pos].Answers, QuizAnswer{
				ID:          *ansID,
				Text:        derefString(ansText),
				Correct:     ansCorrect != nil && *ansCorrect,
				Explanation: derefString(ansExplain),
			})
		}
	}
	return questions, wrap(rows.Err())
}

// Collects every flashcard stored for a module.
func (s *Store) ModuleFlashcards(ctx context.Context, userID, moduleID uuid.UUID) ([]Flashcard, error) {
	const q = `
		SELECT fc.id, fc.file_id, f.original_name, fc.term, fc.definition
		FROM flashcards fc
		JOIN generations g ON g.id = fc.generation_id
		JOIN files f       ON f.id = fc.file_id
		JOIN modules m     ON m.id = f.module_id
		JOIN subjects s    ON s.id = m.subject_id
		WHERE f.module_id = $1 AND s.user_id = $2 AND g.status = 'complete'
		ORDER BY f.original_name, fc.position`

	rows, err := s.pool.Query(ctx, q, moduleID, userID)
	if err != nil {
		return nil, wrap(err)
	}
	defer rows.Close()

	cards := []Flashcard{}
	for rows.Next() {
		var c Flashcard
		if err := rows.Scan(&c.ID, &c.FileID, &c.FileName, &c.Term, &c.Definition); err != nil {
			return nil, wrap(err)
		}
		cards = append(cards, c)
	}
	return cards, wrap(rows.Err())
}

// Collects every teaching card stored for a module in one teaching style.
func (s *Store) ModuleTeachingCards(ctx context.Context, userID, moduleID uuid.UUID, variant string) ([]TeachingCard, error) {
	const q = `
		SELECT tc.id, tc.file_id, f.original_name, tc.subtopic, tc.teaching
		FROM teaching_cards tc
		JOIN generations g ON g.id = tc.generation_id
		JOIN files f       ON f.id = tc.file_id
		JOIN modules m     ON m.id = f.module_id
		JOIN subjects s    ON s.id = m.subject_id
		WHERE f.module_id = $1 AND s.user_id = $2
		  AND g.variant = $3 AND g.status = 'complete'
		ORDER BY f.original_name, tc.position`

	rows, err := s.pool.Query(ctx, q, moduleID, userID, variant)
	if err != nil {
		return nil, wrap(err)
	}
	defer rows.Close()

	cards := []TeachingCard{}
	for rows.Next() {
		var c TeachingCard
		if err := rows.Scan(&c.ID, &c.FileID, &c.FileName, &c.Subtopic, &c.Teaching); err != nil {
			return nil, wrap(err)
		}
		cards = append(cards, c)
	}
	return cards, wrap(rows.Err())
}

// Notes that somebody asked for a generation, used to hold them to an hourly limit.
func (s *Store) RecordGenerateEvent(ctx context.Context, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO generate_events (user_id) VALUES ($1)`, userID)
	return wrap(err)
}

// Counts how many generations somebody has asked for in the last hour.
func (s *Store) GenerateEventsSince(ctx context.Context, userID uuid.UUID, since time.Duration) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT count(*)::int FROM generate_events
		WHERE user_id = $1 AND created_at > now() - $2::interval`,
		userID, since.String()).Scan(&n)
	return n, wrap(err)
}

// Throws away rate-limit records that are too old to matter.
func (s *Store) DeleteOldGenerateEvents(ctx context.Context) (int64, error) {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM generate_events WHERE created_at < now() - interval '2 days'`)
	if err != nil {
		return 0, wrap(err)
	}
	return tag.RowsAffected(), nil
}

// Reads a text value that might be missing, giving back an empty string instead.
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
