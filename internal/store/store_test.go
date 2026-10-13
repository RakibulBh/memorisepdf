package store

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Integration tests against a real Postgres. They are skipped unless
// TEST_DATABASE_URL is set, so `go test ./...` still works without Docker.
// Start one with `make dev-up`, then:
//
//	TEST_DATABASE_URL=postgres://unmemorize:unmemorize@localhost:5432/unmemorize?sslmode=disable go test ./internal/store

// Opens a store against the test database, or skips the test if there isn't one.
func testStore(t *testing.T) *Store {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run store integration tests")
	}

	s, err := Open(context.Background(), url)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(s.Close)

	// The schema has to already be there; the api applies it on boot.
	var exists bool
	if err := s.pool.QueryRow(context.Background(),
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables
		                WHERE table_name = 'generations')`).Scan(&exists); err != nil {
		t.Fatalf("check schema: %v", err)
	}
	if !exists {
		t.Fatal("schema is missing; run the api once against this database first")
	}

	return s
}

// Creates a throwaway user whose rows are removed when the test ends.
func testUser(t *testing.T, s *Store) User {
	t.Helper()
	ctx := context.Background()

	sub := "test-" + uuid.NewString()
	user, err := s.UpsertUserByGoogle(ctx, sub, sub+"@example.test", "Test", "")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Everything else cascades from the user row.
	t.Cleanup(func() {
		if _, err := s.pool.Exec(context.Background(),
			`DELETE FROM users WHERE id = $1`, user.ID); err != nil {
			t.Errorf("clean up user: %v", err)
		}
	})

	return user
}

// Builds a module with the given lecture files already uploaded.
func testModule(t *testing.T, s *Store, user User, fileNames ...string) (uuid.UUID, []uuid.UUID) {
	t.Helper()
	ctx := context.Background()

	subject, err := s.CreateSubject(ctx, user.ID, "Subject "+uuid.NewString()[:8], "")
	if err != nil {
		t.Fatalf("create subject: %v", err)
	}
	module, err := s.CreateModule(ctx, user.ID, subject.ID, "Module")
	if err != nil {
		t.Fatalf("create module: %v", err)
	}

	var fileIDs []uuid.UUID
	for i, name := range fileNames {
		f, err := s.CreateFile(ctx, user.ID, module.ID, File{
			OriginalName:  name,
			SHA256:        uuid.NewString(),
			ByteSize:      int64(100 + i),
			ContentType:   "application/pdf",
			DiskPath:      "/tmp/" + name,
			ExtractedText: "text of " + name,
		})
		if err != nil {
			t.Fatalf("create file %s: %v", name, err)
		}
		fileIDs = append(fileIDs, f.ID)
	}

	return module.ID, fileIDs
}

// The rule the whole feature rests on: a file that has already produced this
// kind of material at this setting is not sent to the model again.
func TestFilesNeedingGenerationSkipsCompletedWork(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	user := testUser(t, s)

	moduleID, fileIDs := testModule(t, s, user, "lec01.pdf", "lec02.pdf", "lec03.pdf")

	pending, err := s.FilesNeedingGeneration(ctx, user.ID, moduleID, KindQuiz, "easy")
	if err != nil {
		t.Fatalf("FilesNeedingGeneration: %v", err)
	}
	if len(pending) != 3 {
		t.Fatalf("with nothing analysed, pending = %d, want 3", len(pending))
	}

	// Analysing two of them leaves exactly one to pay for.
	for _, id := range fileIDs[:2] {
		if err := s.SaveQuiz(ctx, id, "easy", "test-model", []QuizItem{{
			Question: "What is normalisation?",
			Answers: []QuizAnswerItem{
				{Text: "right", Correct: true, Explanation: "because"},
				{Text: "wrong", Correct: false, Explanation: "because not"},
			},
		}}); err != nil {
			t.Fatalf("SaveQuiz: %v", err)
		}
	}

	pending, err = s.FilesNeedingGeneration(ctx, user.ID, moduleID, KindQuiz, "easy")
	if err != nil {
		t.Fatalf("FilesNeedingGeneration: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("pending = %d, want 1", len(pending))
	}
	if pending[0].ID != fileIDs[2] {
		t.Errorf("pending file = %s, want the unanalysed lec03 (%s)", pending[0].ID, fileIDs[2])
	}

	// The two already-analysed files still contribute their questions.
	questions, err := s.ModuleQuiz(ctx, user.ID, moduleID, "easy")
	if err != nil {
		t.Fatalf("ModuleQuiz: %v", err)
	}
	if len(questions) != 2 {
		t.Fatalf("pool = %d questions, want 2", len(questions))
	}
	if len(questions[0].Answers) != 2 {
		t.Errorf("question has %d answers, want 2", len(questions[0].Answers))
	}
	if questions[0].FileName != "lec01.pdf" {
		t.Errorf("FileName = %q, want lec01.pdf", questions[0].FileName)
	}
}

// Asking for a harder quiz is different work, and must not disturb the easy one.
func TestVariantsAreTrackedSeparately(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	user := testUser(t, s)

	moduleID, fileIDs := testModule(t, s, user, "lec01.pdf")

	if err := s.SaveQuiz(ctx, fileIDs[0], "easy", "m", []QuizItem{
		{Question: "easy one", Answers: []QuizAnswerItem{{Text: "a", Correct: true}}},
	}); err != nil {
		t.Fatalf("SaveQuiz easy: %v", err)
	}

	hard, err := s.FilesNeedingGeneration(ctx, user.ID, moduleID, KindQuiz, "hard")
	if err != nil {
		t.Fatalf("FilesNeedingGeneration hard: %v", err)
	}
	if len(hard) != 1 {
		t.Errorf("hard pending = %d, want 1: a new difficulty is new work", len(hard))
	}

	// Flashcards are a different kind entirely, so still outstanding.
	cards, err := s.FilesNeedingGeneration(ctx, user.ID, moduleID, KindFlashcard, "")
	if err != nil {
		t.Fatalf("FilesNeedingGeneration flashcard: %v", err)
	}
	if len(cards) != 1 {
		t.Errorf("flashcard pending = %d, want 1", len(cards))
	}

	easyPool, err := s.ModuleQuiz(ctx, user.ID, moduleID, "easy")
	if err != nil {
		t.Fatalf("ModuleQuiz easy: %v", err)
	}
	if len(easyPool) != 1 {
		t.Errorf("easy pool = %d, want 1: it must survive a hard request", len(easyPool))
	}

	hardPool, err := s.ModuleQuiz(ctx, user.ID, moduleID, "hard")
	if err != nil {
		t.Fatalf("ModuleQuiz hard: %v", err)
	}
	if len(hardPool) != 0 {
		t.Errorf("hard pool = %d, want 0", len(hardPool))
	}
}

// A failure must be retried next time, not treated as finished.
func TestFailedGenerationIsRetried(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	user := testUser(t, s)

	moduleID, fileIDs := testModule(t, s, user, "lec01.pdf")

	if err := s.MarkGenerationFailed(ctx, fileIDs[0], KindQuiz, "easy", "m", "model timed out"); err != nil {
		t.Fatalf("MarkGenerationFailed: %v", err)
	}

	pending, err := s.FilesNeedingGeneration(ctx, user.ID, moduleID, KindQuiz, "easy")
	if err != nil {
		t.Fatalf("FilesNeedingGeneration: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("pending = %d, want 1: a failed file must be retried", len(pending))
	}

	// A later success replaces the failure rather than adding alongside it.
	if err := s.SaveQuiz(ctx, fileIDs[0], "easy", "m", []QuizItem{
		{Question: "q", Answers: []QuizAnswerItem{{Text: "a", Correct: true}}},
	}); err != nil {
		t.Fatalf("SaveQuiz: %v", err)
	}

	pending, err = s.FilesNeedingGeneration(ctx, user.ID, moduleID, KindQuiz, "easy")
	if err != nil {
		t.Fatalf("FilesNeedingGeneration: %v", err)
	}
	if len(pending) != 0 {
		t.Errorf("pending = %d after success, want 0", len(pending))
	}
}

// Re-running a generation must replace the previous items, not pile more on.
func TestSaveQuizReplacesEarlierItems(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	user := testUser(t, s)

	moduleID, fileIDs := testModule(t, s, user, "lec01.pdf")

	for _, count := range []int{3, 2} {
		items := make([]QuizItem, count)
		for i := range items {
			items[i] = QuizItem{
				Question: "question",
				Answers:  []QuizAnswerItem{{Text: "a", Correct: true}},
			}
		}
		if err := s.SaveQuiz(ctx, fileIDs[0], "easy", "m", items); err != nil {
			t.Fatalf("SaveQuiz: %v", err)
		}
	}

	questions, err := s.ModuleQuiz(ctx, user.ID, moduleID, "easy")
	if err != nil {
		t.Fatalf("ModuleQuiz: %v", err)
	}
	if len(questions) != 2 {
		t.Errorf("pool = %d, want 2: the second save must replace the first", len(questions))
	}
}

// One person must never see or touch another person's work.
func TestOwnershipIsEnforced(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	owner := testUser(t, s)
	stranger := testUser(t, s)

	moduleID, fileIDs := testModule(t, s, owner, "lec01.pdf")
	if err := s.SaveQuiz(ctx, fileIDs[0], "easy", "m", []QuizItem{
		{Question: "q", Answers: []QuizAnswerItem{{Text: "a", Correct: true}}},
	}); err != nil {
		t.Fatalf("SaveQuiz: %v", err)
	}

	if _, err := s.Module(ctx, stranger.ID, moduleID); !errors.Is(err, ErrNotFound) {
		t.Errorf("Module for a stranger = %v, want ErrNotFound", err)
	}
	if _, err := s.File(ctx, stranger.ID, fileIDs[0]); !errors.Is(err, ErrNotFound) {
		t.Errorf("File for a stranger = %v, want ErrNotFound", err)
	}

	files, err := s.ListFiles(ctx, stranger.ID, moduleID)
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("stranger sees %d files, want 0", len(files))
	}

	questions, err := s.ModuleQuiz(ctx, stranger.ID, moduleID, "easy")
	if err != nil {
		t.Fatalf("ModuleQuiz: %v", err)
	}
	if len(questions) != 0 {
		t.Errorf("stranger sees %d questions, want 0", len(questions))
	}

	pending, err := s.FilesNeedingGeneration(ctx, stranger.ID, moduleID, KindQuiz, "easy")
	if err != nil {
		t.Fatalf("FilesNeedingGeneration: %v", err)
	}
	if len(pending) != 0 {
		t.Errorf("stranger sees %d pending files, want 0", len(pending))
	}

	if _, err := s.DeleteModule(ctx, stranger.ID, moduleID); !errors.Is(err, ErrNotFound) {
		t.Errorf("DeleteModule by a stranger = %v, want ErrNotFound", err)
	}
}

// Names are unique per owner, case-insensitively, but two people may both have
// a subject called "Databases".
func TestSubjectNamesAreUniquePerUser(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	a := testUser(t, s)
	b := testUser(t, s)

	if _, err := s.CreateSubject(ctx, a.ID, "Databases", ""); err != nil {
		t.Fatalf("CreateSubject: %v", err)
	}
	if _, err := s.CreateSubject(ctx, a.ID, "  databases ", ""); !errors.Is(err, ErrDuplicate) {
		t.Errorf("same name again = %v, want ErrDuplicate", err)
	}
	if _, err := s.CreateSubject(ctx, b.ID, "Databases", ""); err != nil {
		t.Errorf("another user's identical name = %v, want nil", err)
	}
}

// Deleting a subject must take its whole tree with it.
func TestDeleteSubjectCascades(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	user := testUser(t, s)

	moduleID, fileIDs := testModule(t, s, user, "lec01.pdf", "lec02.pdf")
	for _, id := range fileIDs {
		if err := s.SaveFlashcards(ctx, id, "", "m",
			[]FlashcardItem{{Term: "t", Definition: "d"}}); err != nil {
			t.Fatalf("SaveFlashcards: %v", err)
		}
	}

	module, err := s.Module(ctx, user.ID, moduleID)
	if err != nil {
		t.Fatalf("Module: %v", err)
	}

	paths, err := s.DeleteSubject(ctx, user.ID, module.SubjectID)
	if err != nil {
		t.Fatalf("DeleteSubject: %v", err)
	}
	if len(paths) != 2 {
		t.Errorf("got %d disk paths to clean up, want 2", len(paths))
	}

	if _, err := s.Module(ctx, user.ID, moduleID); !errors.Is(err, ErrNotFound) {
		t.Errorf("module still readable after its subject was deleted: %v", err)
	}

	var generations int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*)::int FROM generations WHERE file_id = ANY($1)`,
		fileIDs).Scan(&generations); err != nil {
		t.Fatalf("count generations: %v", err)
	}
	if generations != 0 {
		t.Errorf("%d generation rows survived, want 0", generations)
	}
}

// Storage is counted once per distinct file, so the same deck in two modules
// does not count twice against the quota.
func TestUserStorageCountsSharedBytesOnce(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	user := testUser(t, s)

	subject, err := s.CreateSubject(ctx, user.ID, "Subject "+uuid.NewString()[:8], "")
	if err != nil {
		t.Fatalf("CreateSubject: %v", err)
	}

	sha := uuid.NewString()
	for _, name := range []string{"Module A", "Module B"} {
		module, err := s.CreateModule(ctx, user.ID, subject.ID, name)
		if err != nil {
			t.Fatalf("CreateModule: %v", err)
		}
		if _, err := s.CreateFile(ctx, user.ID, module.ID, File{
			OriginalName: "shared.pdf",
			SHA256:       sha,
			ByteSize:     500,
			DiskPath:     "/tmp/shared.pdf",
		}); err != nil {
			t.Fatalf("CreateFile: %v", err)
		}
	}

	used, err := s.UserStorageUsed(ctx, user.ID)
	if err != nil {
		t.Fatalf("UserStorageUsed: %v", err)
	}
	if used != 500 {
		t.Errorf("used = %d, want 500: the same bytes in two modules count once", used)
	}
}

// The same file cannot be uploaded to one module twice.
func TestDuplicateFileInModuleIsRejected(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	user := testUser(t, s)

	moduleID, _ := testModule(t, s, user)

	f := File{
		OriginalName: "lec01.pdf",
		SHA256:       uuid.NewString(),
		ByteSize:     100,
		DiskPath:     "/tmp/lec01.pdf",
	}
	if _, err := s.CreateFile(ctx, user.ID, moduleID, f); err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	if _, err := s.CreateFile(ctx, user.ID, moduleID, f); !errors.Is(err, ErrDuplicate) {
		t.Errorf("same file again = %v, want ErrDuplicate", err)
	}
}

// A session token must only resolve while it is valid, and not after sign-out.
func TestSessionLifecycle(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	user := testUser(t, s)

	token := "hashed-" + uuid.NewString()
	if err := s.CreateSession(ctx, token, user.ID, timeFromNow(1), "test-agent", "127.0.0.1"); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	got, err := s.UserBySession(ctx, token)
	if err != nil {
		t.Fatalf("UserBySession: %v", err)
	}
	if got.ID != user.ID {
		t.Errorf("session resolved to %s, want %s", got.ID, user.ID)
	}

	if err := s.DeleteSession(ctx, token); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	if _, err := s.UserBySession(ctx, token); !errors.Is(err, ErrNotFound) {
		t.Errorf("revoked session = %v, want ErrNotFound", err)
	}

	// An expired token must not resolve either.
	expired := "hashed-" + uuid.NewString()
	if err := s.CreateSession(ctx, expired, user.ID, timeFromNow(-1), "", ""); err != nil {
		t.Fatalf("CreateSession expired: %v", err)
	}
	if _, err := s.UserBySession(ctx, expired); !errors.Is(err, ErrNotFound) {
		t.Errorf("expired session = %v, want ErrNotFound", err)
	}
}

// A sign-in attempt may only be exchanged once.
func TestOAuthStateIsSingleUse(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	state := "state-" + uuid.NewString()
	if err := s.CreateOAuthState(ctx, state, "verifier", "/dashboard", timeFromNow(1)); err != nil {
		t.Fatalf("CreateOAuthState: %v", err)
	}

	verifier, redirect, err := s.ConsumeOAuthState(ctx, state)
	if err != nil {
		t.Fatalf("ConsumeOAuthState: %v", err)
	}
	if verifier != "verifier" || redirect != "/dashboard" {
		t.Errorf("got (%q, %q), want (verifier, /dashboard)", verifier, redirect)
	}

	if _, _, err := s.ConsumeOAuthState(ctx, state); !errors.Is(err, ErrNotFound) {
		t.Errorf("second exchange = %v, want ErrNotFound", err)
	}
}

// Gives a time a number of hours from now, for session expiry in tests.
func timeFromNow(hours int) time.Time {
	return time.Now().Add(time.Duration(hours) * time.Hour)
}
