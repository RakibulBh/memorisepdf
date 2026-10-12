package store

import (
	"time"

	"github.com/google/uuid"
)

// The three kinds of study material a file can be turned into.
const (
	KindQuiz      = "quiz"
	KindFlashcard = "flashcard"
	KindTeaching  = "teaching"
)

// How far along a generation got.
const (
	StatusPending  = "pending"
	StatusComplete = "complete"
	StatusFailed   = "failed"
)

type User struct {
	ID          uuid.UUID `json:"id"`
	GoogleSub   string    `json:"-"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	PictureURL  string    `json:"pictureUrl"`
	CreatedAt   time.Time `json:"createdAt"`
	LastLoginAt time.Time `json:"lastLoginAt"`
}

type Session struct {
	Token     string    `json:"-"`
	UserID    uuid.UUID `json:"userId"`
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
}

type Subject struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"-"`
	Name        string    `json:"name"`
	Colour      string    `json:"colour"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	ModuleCount int       `json:"moduleCount"`
}

type Module struct {
	ID          uuid.UUID `json:"id"`
	SubjectID   uuid.UUID `json:"subjectId"`
	SubjectName string    `json:"subjectName,omitempty"`
	Name        string    `json:"name"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	FileCount   int       `json:"fileCount"`
	TotalBytes  int64     `json:"totalBytes"`
}

type File struct {
	ID            uuid.UUID `json:"id"`
	ModuleID      uuid.UUID `json:"moduleId"`
	UserID        uuid.UUID `json:"-"`
	OriginalName  string    `json:"originalName"`
	SHA256        string    `json:"sha256"`
	ByteSize      int64     `json:"byteSize"`
	ContentType   string    `json:"contentType"`
	DiskPath      string    `json:"-"`
	ExtractedText string    `json:"-"`
	UploadedAt    time.Time `json:"uploadedAt"`

	// Which kinds of study material this file has already produced, filled in
	// by ListFiles so the UI can show it without a second query. Always sent,
	// as an empty list when nothing has been analysed, so a caller can read it
	// without checking whether the field is there.
	Analysed []string `json:"analysed"`
}

type Generation struct {
	ID          uuid.UUID  `json:"id"`
	FileID      uuid.UUID  `json:"fileId"`
	Kind        string     `json:"kind"`
	Variant     string     `json:"variant"`
	Model       string     `json:"model"`
	Status      string     `json:"status"`
	ItemCount   int        `json:"itemCount"`
	Error       string     `json:"error,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

type QuizQuestion struct {
	ID       uuid.UUID    `json:"id"`
	FileID   uuid.UUID    `json:"fileId"`
	FileName string       `json:"fileName,omitempty"`
	Question string       `json:"question"`
	Answers  []QuizAnswer `json:"answers"`
}

type QuizAnswer struct {
	ID          uuid.UUID `json:"id"`
	Text        string    `json:"text"`
	Correct     bool      `json:"correct"`
	Explanation string    `json:"explanation"`
}

type Flashcard struct {
	ID         uuid.UUID `json:"id"`
	FileID     uuid.UUID `json:"fileId"`
	FileName   string    `json:"fileName,omitempty"`
	Term       string    `json:"term"`
	Definition string    `json:"definition"`
}

type TeachingCard struct {
	ID       uuid.UUID `json:"id"`
	FileID   uuid.UUID `json:"fileId"`
	FileName string    `json:"fileName,omitempty"`
	Subtopic string    `json:"subtopic"`
	Teaching string    `json:"teaching"`
}
