package files

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// Builds a storage rooted in a directory the test owns.
func newTestStorage(t *testing.T, maxSize int64) *Storage {
	t.Helper()

	s, err := NewStorage(t.TempDir(), maxSize)
	if err != nil {
		t.Fatalf("NewStorage: %v", err)
	}
	return s
}

func TestSaveNamesFileAfterItsContents(t *testing.T) {
	s := newTestStorage(t, 1<<20)
	user := uuid.New()
	body := "lecture one: normalisation"

	saved, err := s.Save(user, "lecture01.pdf", strings.NewReader(body))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	want := sha256.Sum256([]byte(body))
	if saved.SHA256 != hex.EncodeToString(want[:]) {
		t.Errorf("SHA256 = %q, want the hash of the body", saved.SHA256)
	}
	if saved.ByteSize != int64(len(body)) {
		t.Errorf("ByteSize = %d, want %d", saved.ByteSize, len(body))
	}
	if saved.AlreadyHeld {
		t.Error("AlreadyHeld = true for a file saved for the first time")
	}
	if saved.ContentType != "application/pdf" {
		t.Errorf("ContentType = %q, want application/pdf", saved.ContentType)
	}

	on, err := os.ReadFile(saved.DiskPath)
	if err != nil {
		t.Fatalf("read stored file: %v", err)
	}
	if string(on) != body {
		t.Errorf("stored contents = %q, want %q", on, body)
	}
}

// The same slides uploaded to a second module must not be written twice.
func TestSaveReusesIdenticalBytes(t *testing.T) {
	s := newTestStorage(t, 1<<20)
	user := uuid.New()
	body := "identical deck"

	first, err := s.Save(user, "week1.pdf", strings.NewReader(body))
	if err != nil {
		t.Fatalf("first Save: %v", err)
	}

	// A different filename, the same contents.
	second, err := s.Save(user, "copy-of-week1.pdf", strings.NewReader(body))
	if err != nil {
		t.Fatalf("second Save: %v", err)
	}

	if !second.AlreadyHeld {
		t.Error("AlreadyHeld = false, want true for bytes already on disk")
	}
	if second.DiskPath != first.DiskPath {
		t.Errorf("DiskPath = %q, want the same path as the first save (%q)",
			second.DiskPath, first.DiskPath)
	}
}

// Two people uploading the same slides must get separate paths, so one
// deleting their copy cannot remove the other's.
func TestSaveKeepsUsersApart(t *testing.T) {
	s := newTestStorage(t, 1<<20)
	body := "shared deck"

	a, err := s.Save(uuid.New(), "deck.pdf", strings.NewReader(body))
	if err != nil {
		t.Fatalf("Save for first user: %v", err)
	}
	b, err := s.Save(uuid.New(), "deck.pdf", strings.NewReader(body))
	if err != nil {
		t.Fatalf("Save for second user: %v", err)
	}

	if a.DiskPath == b.DiskPath {
		t.Errorf("both users stored at %q, want separate paths", a.DiskPath)
	}
	if b.AlreadyHeld {
		t.Error("AlreadyHeld = true across users, want false")
	}
}

func TestSaveRejectsOversizedFile(t *testing.T) {
	s := newTestStorage(t, 16)

	_, err := s.Save(uuid.New(), "big.pdf", strings.NewReader(strings.Repeat("x", 17)))
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err = %v, want ErrTooLarge", err)
	}
}

// A file exactly on the limit is allowed; the limit is inclusive.
func TestSaveAcceptsFileAtExactLimit(t *testing.T) {
	s := newTestStorage(t, 16)

	saved, err := s.Save(uuid.New(), "exact.pdf", strings.NewReader(strings.Repeat("x", 16)))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if saved.ByteSize != 16 {
		t.Errorf("ByteSize = %d, want 16", saved.ByteSize)
	}
}

func TestSaveRejectsEmptyAndUnsupported(t *testing.T) {
	s := newTestStorage(t, 1<<20)

	if _, err := s.Save(uuid.New(), "empty.pdf", strings.NewReader("")); err == nil {
		t.Error("Save accepted an empty file, want an error")
	}
	if _, err := s.Save(uuid.New(), "virus.exe", strings.NewReader("data")); err == nil {
		t.Error("Save accepted a .exe, want an error")
	}
}

// A rejected upload must not leave a stray temporary file behind.
func TestSaveCleansUpAfterRejection(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStorage(dir, 8)
	if err != nil {
		t.Fatalf("NewStorage: %v", err)
	}

	if _, err := s.Save(uuid.New(), "big.pdf", strings.NewReader(strings.Repeat("x", 99))); err == nil {
		t.Fatal("Save accepted an oversized file")
	}

	entries, err := os.ReadDir(filepath.Join(dir, "files"))
	if err != nil {
		t.Fatalf("read storage root: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".upload-") {
			t.Errorf("temporary file %q left behind after a rejected upload", e.Name())
		}
	}
}

// Nothing outside the storage directory may be read or deleted, however the
// path is dressed up.
func TestOpenAndRemoveRefusePathsOutsideStorage(t *testing.T) {
	s := newTestStorage(t, 1<<20)

	for _, path := range []string{
		"/etc/passwd",
		filepath.Join(s.root, "..", "..", "etc", "passwd"),
	} {
		if _, err := s.Open(path); err == nil {
			t.Errorf("Open(%q) succeeded, want refusal", path)
		}
		if err := s.Remove(path); err == nil {
			t.Errorf("Remove(%q) succeeded, want refusal", path)
		}
	}
}

// Removing a file that has already gone is not an error, so cleanup after a
// partial failure stays simple.
func TestRemoveIsForgivingAboutMissingFiles(t *testing.T) {
	s := newTestStorage(t, 1<<20)

	saved, err := s.Save(uuid.New(), "gone.pdf", strings.NewReader("bye"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := s.Remove(saved.DiskPath); err != nil {
		t.Fatalf("first Remove: %v", err)
	}
	if err := s.Remove(saved.DiskPath); err != nil {
		t.Errorf("second Remove: %v, want nil", err)
	}
}

func TestIsSupported(t *testing.T) {
	cases := map[string]bool{
		"slides.pdf":     true,
		"slides.PDF":     true,
		"deck.pptx":      true,
		"notes.md":       true,
		"archive.zip":    false,
		"noextension":    false,
		"sneaky.pdf.exe": false,
	}

	for name, want := range cases {
		if got := IsSupported(name); got != want {
			t.Errorf("IsSupported(%q) = %v, want %v", name, got, want)
		}
	}
}
