// Package llm asks Gemini to turn lecture text into quizzes, flashcards and
// teaching cards, and insists on a reply shaped exactly as expected.
package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"google.golang.org/genai"
)

// The kinds of study material this package can produce. They match the values
// the generations ledger stores.
const (
	KindQuiz      = "quiz"
	KindFlashcard = "flashcard"
	KindTeaching  = "teaching"
)

// ErrEmptyResult means the model answered without producing any study material.
var ErrEmptyResult = errors.New("model returned no items")

type Client struct {
	genai           *genai.Client
	model           string
	maxOutputTokens int32
	maxInputChars   int
	timeout         time.Duration
	log             *slog.Logger
}

type Options struct {
	APIKey          string
	Model           string
	MaxOutputTokens int
	MaxInputChars   int
	Timeout         time.Duration
	Logger          *slog.Logger
}

// Connects to the Gemini API.
func New(ctx context.Context, opts Options) (*Client, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: opts.APIKey})
	if err != nil {
		return nil, fmt.Errorf("create gemini client: %w", err)
	}

	log := opts.Logger
	if log == nil {
		log = slog.Default()
	}

	return &Client{
		genai:           client,
		model:           opts.Model,
		maxOutputTokens: int32(opts.MaxOutputTokens),
		maxInputChars:   opts.MaxInputChars,
		timeout:         opts.Timeout,
		log:             log,
	}, nil
}

// Reports which model is being used, for recording against a generation.
func (c *Client) Model() string {
	return c.model
}

// Turns lecture text into quiz questions at the requested difficulty.
func (c *Client) Quiz(ctx context.Context, lectureText, difficulty string) ([]QuizItem, error) {
	var out QuizResponse
	if err := c.generate(ctx, KindQuiz, difficulty, lectureText, quizSchema(), &out); err != nil {
		return nil, err
	}

	items := make([]QuizItem, 0, len(out.Questions))
	for _, q := range out.Questions {
		if strings.TrimSpace(q.Question) == "" || len(q.Answers) == 0 {
			continue
		}

		// A question with no correct option, or several, cannot be marked, so
		// it is dropped rather than shown to somebody as unanswerable.
		correct := 0
		answers := make([]QuizAnswerItem, 0, len(q.Answers))
		for _, a := range q.Answers {
			if strings.TrimSpace(a.Text) == "" {
				continue
			}
			if a.Correct {
				correct++
			}
			answers = append(answers, QuizAnswerItem{
				Text:        strings.TrimSpace(a.Text),
				Correct:     a.Correct,
				Explanation: strings.TrimSpace(a.Explanation),
			})
		}
		if correct != 1 || len(answers) < 2 {
			c.log.Warn("dropping malformed question", "correctAnswers", correct, "options", len(answers))
			continue
		}

		items = append(items, QuizItem{Question: strings.TrimSpace(q.Question), Answers: answers})
	}

	if len(items) == 0 {
		return nil, ErrEmptyResult
	}
	return items, nil
}

// Turns lecture text into flashcards.
func (c *Client) Flashcards(ctx context.Context, lectureText string) ([]FlashcardItem, error) {
	var out FlashcardResponse
	if err := c.generate(ctx, KindFlashcard, "", lectureText, flashcardSchema(), &out); err != nil {
		return nil, err
	}

	items := make([]FlashcardItem, 0, len(out.Flashcards))
	for _, f := range out.Flashcards {
		term, def := strings.TrimSpace(f.Term), strings.TrimSpace(f.Definition)
		if term == "" || def == "" {
			continue
		}
		items = append(items, FlashcardItem{Term: term, Definition: def})
	}

	if len(items) == 0 {
		return nil, ErrEmptyResult
	}
	return items, nil
}

// Turns lecture text into teaching cards in the requested style.
func (c *Client) TeachingCards(ctx context.Context, lectureText, style string) ([]TeachingItem, error) {
	var out TeachingResponse
	if err := c.generate(ctx, KindTeaching, style, lectureText, teachingSchema(), &out); err != nil {
		return nil, err
	}

	items := make([]TeachingItem, 0, len(out.TeachingCards))
	for _, t := range out.TeachingCards {
		sub, teaching := strings.TrimSpace(t.Subtopic), strings.TrimSpace(t.Teaching)
		if sub == "" || teaching == "" {
			continue
		}
		items = append(items, TeachingItem{Subtopic: sub, Teaching: teaching})
	}

	if len(items) == 0 {
		return nil, ErrEmptyResult
	}
	return items, nil
}

// QuizItem and friends mirror what the store expects to be handed.
type QuizItem struct {
	Question string
	Answers  []QuizAnswerItem
}

type QuizAnswerItem struct {
	Text        string
	Correct     bool
	Explanation string
}

type FlashcardItem struct {
	Term       string
	Definition string
}

type TeachingItem struct {
	Subtopic string
	Teaching string
}

// Sends one request to the model and reads the reply into the given shape,
// retrying a couple of times because the API fails intermittently under load.
func (c *Client) generate(ctx context.Context, kind, variant, lectureText string, schema *genai.Schema, out any) error {
	if strings.TrimSpace(lectureText) == "" {
		return errors.New("lecture text is empty")
	}

	instruction, err := instructionFor(kind, variant)
	if err != nil {
		return err
	}
	prompt := buildPrompt(instruction, lectureText, c.maxInputChars)

	cfg := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemInstruction, genai.RoleUser),
		ResponseMIMEType:  "application/json",
		ResponseSchema:    schema,
		MaxOutputTokens:   c.maxOutputTokens,
		Temperature:       genai.Ptr[float32](0.4),
	}

	const attempts = 3
	var lastErr error

	for attempt := 1; attempt <= attempts; attempt++ {
		callCtx, cancel := context.WithTimeout(ctx, c.timeout)
		resp, err := c.genai.Models.GenerateContent(callCtx, c.model, genai.Text(prompt), cfg)
		cancel()

		if err != nil {
			lastErr = err
			c.log.Warn("gemini call failed", "kind", kind, "variant", variant,
				"attempt", attempt, "err", err)

			// A cancelled parent context means the caller gave up; retrying
			// would only waste quota.
			if ctx.Err() != nil {
				return fmt.Errorf("generate %s: %w", kind, ctx.Err())
			}
			if attempt < attempts {
				time.Sleep(time.Duration(attempt) * 2 * time.Second)
			}
			continue
		}

		text := resp.Text()
		if strings.TrimSpace(text) == "" {
			lastErr = errors.New("model returned an empty response")
			c.log.Warn("gemini returned empty response", "kind", kind, "attempt", attempt)
			if attempt < attempts {
				time.Sleep(time.Duration(attempt) * 2 * time.Second)
			}
			continue
		}

		if err := json.Unmarshal([]byte(text), out); err != nil {
			lastErr = fmt.Errorf("parse model response: %w", err)
			c.log.Warn("gemini returned unreadable json", "kind", kind,
				"attempt", attempt, "snippet", snippet(text))
			if attempt < attempts {
				time.Sleep(time.Duration(attempt) * 2 * time.Second)
			}
			continue
		}

		return nil
	}

	return fmt.Errorf("generate %s after %d attempts: %w", kind, attempts, lastErr)
}

// Shortens text so a failure can be logged without dumping the whole reply.
func snippet(s string) string {
	const limit = 300
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "…"
}
