// Package logging builds the application logger: structured records written to
// both stdout and a file on the data volume, so failures survive a container
// restart and can be read without docker.
package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// maxBytes bounds the active file before it is rolled to <name>.1, keeping one
// previous generation. Docker caps its own json-file logs separately.
const maxBytes = 8 << 20

type rotatingFile struct {
	mu   sync.Mutex
	path string
	file *os.File
	size int64
}

// Opens the log file, ready to add to the end of it.
func openRotating(path string) (*rotatingFile, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	return &rotatingFile{path: path, file: f, size: info.Size()}, nil
}

// Adds to the log file, starting a new one once it gets too big.
func (r *rotatingFile) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.size+int64(len(p)) > maxBytes {
		if err := r.rotate(); err != nil {
			return 0, err
		}
	}

	n, err := r.file.Write(p)
	r.size += int64(n)
	return n, err
}

// rotate must be called with the lock held.
func (r *rotatingFile) rotate() error {
	if err := r.file.Close(); err != nil {
		return err
	}
	if err := os.Rename(r.path, r.path+".1"); err != nil && !os.IsNotExist(err) {
		return err
	}

	f, err := os.OpenFile(r.path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}

	r.file = f
	r.size = 0
	return nil
}

// Closes the log file.
func (r *rotatingFile) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.file.Close()
}

// Turns a setting like "debug" into a logging level.
func ParseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// New returns a logger writing to stdout and, when path is non-empty, to a
// rotating file. A file that cannot be opened is reported and skipped rather
// than preventing startup, so a permissions problem never blocks startup.
func New(path string, level slog.Level) (*slog.Logger, io.Closer) {
	opts := &slog.HandlerOptions{Level: level}

	if path == "" {
		return slog.New(slog.NewTextHandler(os.Stdout, opts)), io.NopCloser(nil)
	}

	rf, err := openRotating(path)
	if err != nil {
		log := slog.New(slog.NewTextHandler(os.Stdout, opts))
		log.Error("open log file, continuing with stdout only", "path", path, "err", err)
		return log, io.NopCloser(nil)
	}

	handler := slog.NewTextHandler(io.MultiWriter(os.Stdout, rf), opts)
	log := slog.New(handler)
	log.Info("logging to file", "path", absOrRaw(path), "maxBytes", maxBytes)

	return log, rf
}

// Gives the full path to a file where possible.
func absOrRaw(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

// Err formats an error for a log attribute, tolerating nil.
func Err(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprint(err)
}
