// Package files keeps uploaded lecture files on disk, named after their
// contents so the same slides are never stored twice.
package files

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// ErrTooLarge means the upload was bigger than the configured limit.
var ErrTooLarge = errors.New("file too large")

// The document types Tika can usefully read for a lecture.
var allowedExtensions = map[string]string{
	".pdf":  "application/pdf",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	".ppt":  "application/vnd.ms-powerpoint",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".doc":  "application/msword",
	".txt":  "text/plain",
	".md":   "text/markdown",
}

type Storage struct {
	root    string
	maxSize int64
}

// Sets up the folder on disk where uploads are kept.
func NewStorage(dataDir string, maxSize int64) (*Storage, error) {
	root := filepath.Join(dataDir, "files")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create files directory: %w", err)
	}
	return &Storage{root: root, maxSize: maxSize}, nil
}

// Saved describes a file that has just been written to disk.
type Saved struct {
	SHA256      string
	ByteSize    int64
	DiskPath    string
	ContentType string
	AlreadyHeld bool
}

// Writes an upload to disk under a name taken from its contents, refusing
// anything over the size limit. Identical bytes already on disk are reused.
func (s *Storage) Save(userID uuid.UUID, originalName string, r io.Reader) (Saved, error) {
	contentType, ok := allowedExtensions[strings.ToLower(filepath.Ext(originalName))]
	if !ok {
		return Saved{}, fmt.Errorf("unsupported file type %q", filepath.Ext(originalName))
	}

	// Written to a temporary file first, because the final name is the hash of
	// the contents and so is not known until everything has been read.
	tmp, err := os.CreateTemp(s.root, ".upload-*")
	if err != nil {
		return Saved{}, fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	hasher := sha256.New()
	// One byte over the limit is enough to know the upload is too big.
	limited := io.LimitReader(r, s.maxSize+1)

	written, err := io.Copy(io.MultiWriter(tmp, hasher), limited)
	closeErr := tmp.Close()
	if err != nil {
		return Saved{}, fmt.Errorf("write upload: %w", err)
	}
	if closeErr != nil {
		return Saved{}, fmt.Errorf("close upload: %w", closeErr)
	}
	if written > s.maxSize {
		return Saved{}, ErrTooLarge
	}
	if written == 0 {
		return Saved{}, errors.New("file is empty")
	}

	sum := hex.EncodeToString(hasher.Sum(nil))
	dest := s.pathFor(userID, sum, filepath.Ext(originalName))

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return Saved{}, fmt.Errorf("create storage directory: %w", err)
	}

	// The same slides uploaded to a second module land on the same path, so
	// the bytes already there are correct and are kept.
	if _, err := os.Stat(dest); err == nil {
		return Saved{
			SHA256:      sum,
			ByteSize:    written,
			DiskPath:    dest,
			ContentType: contentType,
			AlreadyHeld: true,
		}, nil
	}

	if err := os.Rename(tmpName, dest); err != nil {
		return Saved{}, fmt.Errorf("store upload: %w", err)
	}
	if err := os.Chmod(dest, 0o600); err != nil {
		return Saved{}, fmt.Errorf("set permissions: %w", err)
	}

	return Saved{
		SHA256:      sum,
		ByteSize:    written,
		DiskPath:    dest,
		ContentType: contentType,
	}, nil
}

// Opens a stored file for reading.
func (s *Storage) Open(diskPath string) (*os.File, error) {
	if !strings.HasPrefix(filepath.Clean(diskPath), s.root) {
		return nil, fmt.Errorf("path %q is outside the storage directory", diskPath)
	}
	return os.Open(diskPath)
}

// Deletes a stored file, saying nothing if it has already gone.
func (s *Storage) Remove(diskPath string) error {
	if !strings.HasPrefix(filepath.Clean(diskPath), s.root) {
		return fmt.Errorf("refusing to delete %q, outside the storage directory", diskPath)
	}
	if err := os.Remove(diskPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// Deletes several stored files, reporting the first problem but trying them all.
func (s *Storage) RemoveAll(paths []string) error {
	var firstErr error
	for _, p := range paths {
		if err := s.Remove(p); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Works out where a file with these contents belongs, fanning out by the first
// two characters of the hash so no single folder grows unmanageably large.
func (s *Storage) pathFor(userID uuid.UUID, sum, ext string) string {
	return filepath.Join(s.root, userID.String(), sum[:2], sum+strings.ToLower(ext))
}

// Says whether a filename is a type that can be read.
func IsSupported(name string) bool {
	_, ok := allowedExtensions[strings.ToLower(filepath.Ext(name))]
	return ok
}

// Lists the file types that can be uploaded, for showing in the UI.
func SupportedExtensions() []string {
	exts := make([]string, 0, len(allowedExtensions))
	for ext := range allowedExtensions {
		exts = append(exts, ext)
	}
	return exts
}
