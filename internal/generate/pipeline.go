// Package generate turns a module's lecture files into study material, paying
// only for the files that have not been analysed before.
package generate

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/RakibulBh/unmemorize/internal/files"
	"github.com/RakibulBh/unmemorize/internal/llm"
	"github.com/RakibulBh/unmemorize/internal/store"
	"github.com/RakibulBh/unmemorize/internal/tika"
)

type Pipeline struct {
	store   *store.Store
	llm     *llm.Client
	tika    *tika.Client
	storage *files.Storage
	log     *slog.Logger

	maxChars int
}

type Options struct {
	Store          *store.Store
	LLM            *llm.Client
	Tika           *tika.Client
	Storage        *files.Storage
	Logger         *slog.Logger
	MaxCharsPerDoc int
}

// Wires up the generation pipeline.
func New(opts Options) *Pipeline {
	log := opts.Logger
	if log == nil {
		log = slog.Default()
	}
	return &Pipeline{
		store:    opts.Store,
		llm:      opts.LLM,
		tika:     opts.Tika,
		storage:  opts.Storage,
		log:      log,
		maxChars: opts.MaxCharsPerDoc,
	}
}

// Progress is one update sent to the browser while a module is being analysed.
type Progress struct {
	Stage     string `json:"stage"`
	Message   string `json:"message"`
	FileName  string `json:"fileName,omitempty"`
	Done      int    `json:"done"`
	Total     int    `json:"total"`
	Reused    int    `json:"reused"`
	Generated int    `json:"generated"`
	Failed    int    `json:"failed"`
}

// Result sums up what a run actually did.
type Result struct {
	Reused    int      `json:"reused"`
	Generated int      `json:"generated"`
	Failed    int      `json:"failed"`
	Errors    []string `json:"errors,omitempty"`
}

// ErrNoFiles means the module has nothing to analyse yet.
var ErrNoFiles = errors.New("module has no files")

// Analyses a module's files, skipping any that have already produced this kind
// of material at this setting, and reports progress as it goes.
func (p *Pipeline) Run(ctx context.Context, userID, moduleID uuid.UUID, kind, variant string, progress func(Progress)) (Result, error) {
	report := func(pr Progress) {
		if progress != nil {
			progress(pr)
		}
	}

	total, err := p.store.CountFilesInModule(ctx, moduleID)
	if err != nil {
		return Result{}, fmt.Errorf("count files: %w", err)
	}
	if total == 0 {
		return Result{}, ErrNoFiles
	}

	pending, err := p.store.FilesNeedingGeneration(ctx, userID, moduleID, kind, variant)
	if err != nil {
		return Result{}, fmt.Errorf("find files needing generation: %w", err)
	}

	result := Result{Reused: total - len(pending)}

	// Everything in the module has been analysed before, so there is nothing to
	// pay for and the stored pool can be served straight away.
	if len(pending) == 0 {
		report(Progress{
			Stage:   "finished",
			Message: fmt.Sprintf("Reusing %d already-analysed file(s), nothing new to generate", result.Reused),
			Done:    total, Total: total, Reused: result.Reused,
		})
		return result, nil
	}

	report(Progress{
		Stage: "started",
		Message: fmt.Sprintf("%d new file(s) to analyse, reusing %d",
			len(pending), result.Reused),
		Total: len(pending), Reused: result.Reused,
	})

	for i, file := range pending {
		if err := ctx.Err(); err != nil {
			return result, err
		}

		report(Progress{
			Stage:     "analysing",
			Message:   "Reading " + file.OriginalName,
			FileName:  file.OriginalName,
			Done:      i,
			Total:     len(pending),
			Reused:    result.Reused,
			Generated: result.Generated,
			Failed:    result.Failed,
		})

		if err := p.processFile(ctx, file, kind, variant); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, file.OriginalName+": "+err.Error())
			p.log.Error("generation failed for file",
				"file", file.OriginalName, "fileId", file.ID,
				"kind", kind, "variant", variant, "err", err)

			// The failure is written to the ledger so the next attempt retries
			// this file rather than treating it as finished.
			if markErr := p.store.MarkGenerationFailed(ctx, file.ID, kind, variant,
				p.llm.Model(), err.Error()); markErr != nil {
				p.log.Error("could not record generation failure", "err", markErr)
			}
			continue
		}

		result.Generated++
		report(Progress{
			Stage:     "file-done",
			Message:   "Finished " + file.OriginalName,
			FileName:  file.OriginalName,
			Done:      i + 1,
			Total:     len(pending),
			Reused:    result.Reused,
			Generated: result.Generated,
			Failed:    result.Failed,
		})
	}

	report(Progress{
		Stage: "finished",
		Message: fmt.Sprintf("Generated from %d file(s), reused %d, failed %d",
			result.Generated, result.Reused, result.Failed),
		Done:      len(pending),
		Total:     len(pending),
		Reused:    result.Reused,
		Generated: result.Generated,
		Failed:    result.Failed,
	})

	return result, nil
}

// Reads one file's text if it has not been read before, asks the model for
// study material, and saves it.
func (p *Pipeline) processFile(ctx context.Context, file store.PendingFile, kind, variant string) error {
	text := file.ExtractedText

	// Tika only runs the first time a file is seen. Every later generation at a
	// different difficulty or style reuses the text saved on the file row.
	if text == "" {
		extracted, err := p.extract(ctx, file)
		if err != nil {
			return err
		}
		if err := p.store.SetExtractedText(ctx, file.ID, extracted); err != nil {
			return fmt.Errorf("save extracted text: %w", err)
		}
		text = extracted
	}

	switch kind {
	case store.KindQuiz:
		items, err := p.llm.Quiz(ctx, text, variant)
		if err != nil {
			return err
		}
		return p.store.SaveQuiz(ctx, file.ID, variant, p.llm.Model(), toQuizItems(items))

	case store.KindFlashcard:
		items, err := p.llm.Flashcards(ctx, text)
		if err != nil {
			return err
		}
		return p.store.SaveFlashcards(ctx, file.ID, variant, p.llm.Model(), toFlashcardItems(items))

	case store.KindTeaching:
		items, err := p.llm.TeachingCards(ctx, text, variant)
		if err != nil {
			return err
		}
		return p.store.SaveTeachingCards(ctx, file.ID, variant, p.llm.Model(), toTeachingItems(items))

	default:
		return fmt.Errorf("unknown kind %q", kind)
	}
}

// Pulls the readable text out of a stored file.
func (p *Pipeline) extract(ctx context.Context, file store.PendingFile) (string, error) {
	f, err := p.storage.Open(file.DiskPath)
	if err != nil {
		return "", fmt.Errorf("open stored file: %w", err)
	}
	defer f.Close()

	extractCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	text, err := p.tika.Extract(extractCtx, f, file.ContentType)
	if err != nil {
		return "", fmt.Errorf("extract text: %w", err)
	}
	if text == "" {
		return "", errors.New("no readable text found in the file")
	}
	if p.maxChars > 0 && len(text) > p.maxChars {
		text = text[:p.maxChars]
	}
	return text, nil
}

// Converts generated questions into the shape the database layer takes.
func toQuizItems(items []llm.QuizItem) []store.QuizItem {
	out := make([]store.QuizItem, 0, len(items))
	for _, q := range items {
		answers := make([]store.QuizAnswerItem, 0, len(q.Answers))
		for _, a := range q.Answers {
			answers = append(answers, store.QuizAnswerItem{
				Text:        a.Text,
				Correct:     a.Correct,
				Explanation: a.Explanation,
			})
		}
		out = append(out, store.QuizItem{Question: q.Question, Answers: answers})
	}
	return out
}

// Converts generated flashcards into the shape the database layer takes.
func toFlashcardItems(items []llm.FlashcardItem) []store.FlashcardItem {
	out := make([]store.FlashcardItem, 0, len(items))
	for _, f := range items {
		out = append(out, store.FlashcardItem{Term: f.Term, Definition: f.Definition})
	}
	return out
}

// Converts generated teaching cards into the shape the database layer takes.
func toTeachingItems(items []llm.TeachingItem) []store.TeachingItem {
	out := make([]store.TeachingItem, 0, len(items))
	for _, t := range items {
		out = append(out, store.TeachingItem{Subtopic: t.Subtopic, Teaching: t.Teaching})
	}
	return out
}
