package server

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/RakibulBh/memorisepdf/internal/auth"
	"github.com/RakibulBh/memorisepdf/internal/files"
	"github.com/RakibulBh/memorisepdf/internal/store"
)

// Lists a module's lecture files and which kinds of material each has produced.
func (s *Server) handleListFiles(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUser(r.Context())

	moduleID, err := idParam(r, "moduleID")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid module id")
		return
	}

	if _, err := s.store.Module(r.Context(), user.ID, moduleID); err != nil {
		s.writeStoreError(w, err, "")
		return
	}

	list, err := s.store.ListFiles(r.Context(), user.ID, moduleID)
	if err != nil {
		s.writeStoreError(w, err, "")
		return
	}
	s.writeJSON(w, http.StatusOK, list)
}

// uploadOutcome says what happened to one file in an upload.
type uploadOutcome struct {
	Name   string      `json:"name"`
	Status string      `json:"status"`
	Reason string      `json:"reason,omitempty"`
	File   *store.File `json:"file,omitempty"`
}

// Accepts one or more lecture files for a module, holding the uploader to
// their size and storage limits.
func (s *Server) handleUploadFiles(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUser(r.Context())

	moduleID, err := idParam(r, "moduleID")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid module id")
		return
	}

	module, err := s.store.Module(r.Context(), user.ID, moduleID)
	if err != nil {
		s.writeStoreError(w, err, "")
		return
	}

	// Streamed rather than read into memory, because a hundred-megabyte deck
	// would otherwise be held in RAM in full.
	reader, err := r.MultipartReader()
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "expected a multipart upload")
		return
	}

	used, err := s.store.UserStorageUsed(r.Context(), user.ID)
	if err != nil {
		s.writeStoreError(w, err, "")
		return
	}

	fileCount := module.FileCount
	outcomes := []uploadOutcome{}

	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			s.writeError(w, http.StatusBadRequest, "could not read upload: "+err.Error())
			return
		}

		name := part.FileName()
		if part.FormName() != "files" || name == "" {
			part.Close()
			continue
		}
		name = filepath.Base(name)

		if fileCount >= s.cfg.Limits.MaxFilesPerModule {
			part.Close()
			outcomes = append(outcomes, uploadOutcome{
				Name: name, Status: "rejected",
				Reason: fmt.Sprintf("this module already holds the maximum of %d files",
					s.cfg.Limits.MaxFilesPerModule),
			})
			continue
		}

		if !files.IsSupported(name) {
			part.Close()
			outcomes = append(outcomes, uploadOutcome{
				Name: name, Status: "rejected",
				Reason: "unsupported file type, expected one of pdf, pptx, ppt, docx, doc, txt or md",
			})
			continue
		}

		saved, err := s.storage.Save(user.ID, name, part)
		part.Close()
		if err != nil {
			reason := err.Error()
			if errors.Is(err, files.ErrTooLarge) {
				reason = fmt.Sprintf("larger than the %s limit",
					humanBytes(s.cfg.Limits.MaxFileSize))
			}
			outcomes = append(outcomes, uploadOutcome{Name: name, Status: "rejected", Reason: reason})
			continue
		}

		// Bytes already on disk cost nothing more, so they do not count again
		// towards the quota.
		if !saved.AlreadyHeld && used+saved.ByteSize > s.cfg.Limits.UserStorageQuota {
			if rmErr := s.storage.Remove(saved.DiskPath); rmErr != nil {
				s.log.Error("remove file rejected for quota", "err", rmErr)
			}
			outcomes = append(outcomes, uploadOutcome{
				Name: name, Status: "rejected",
				Reason: fmt.Sprintf("would exceed your %s storage limit",
					humanBytes(s.cfg.Limits.UserStorageQuota)),
			})
			continue
		}

		// Any text already extracted from identical bytes is carried over, so
		// the same deck is never sent to Tika twice.
		extracted := ""
		if saved.AlreadyHeld {
			if prior, err := s.store.FileBySHA(r.Context(), user.ID, saved.SHA256); err == nil {
				extracted = prior.ExtractedText
			}
		}

		record, err := s.store.CreateFile(r.Context(), user.ID, moduleID, store.File{
			OriginalName:  name,
			SHA256:        saved.SHA256,
			ByteSize:      saved.ByteSize,
			ContentType:   saved.ContentType,
			DiskPath:      saved.DiskPath,
			ExtractedText: extracted,
		})
		if err != nil {
			if errors.Is(err, store.ErrDuplicate) {
				outcomes = append(outcomes, uploadOutcome{
					Name: name, Status: "skipped",
					Reason: "these exact slides are already in this module",
				})
				continue
			}
			s.log.Error("record uploaded file", "err", err)
			outcomes = append(outcomes, uploadOutcome{
				Name: name, Status: "rejected", Reason: "could not save the file",
			})
			continue
		}

		fileCount++
		if !saved.AlreadyHeld {
			used += saved.ByteSize
		}
		outcomes = append(outcomes, uploadOutcome{Name: name, Status: "stored", File: &record})
	}

	if len(outcomes) == 0 {
		s.writeError(w, http.StatusBadRequest, "no files were sent")
		return
	}

	stored := 0
	for _, o := range outcomes {
		if o.Status == "stored" {
			stored++
		}
	}

	status := http.StatusCreated
	if stored == 0 {
		status = http.StatusUnprocessableEntity
	}

	s.writeJSON(w, status, map[string]any{
		"stored":       stored,
		"results":      outcomes,
		"storageUsed":  used,
		"storageQuota": s.cfg.Limits.UserStorageQuota,
	})
}

// Deletes a lecture file along with everything generated from it.
func (s *Server) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUser(r.Context())

	id, err := idParam(r, "fileID")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid file id")
		return
	}

	diskPath, removeFromDisk, err := s.store.DeleteFile(r.Context(), user.ID, id)
	if err != nil {
		s.writeStoreError(w, err, "")
		return
	}

	// The bytes stay if another module still points at them.
	if removeFromDisk {
		if err := s.storage.Remove(diskPath); err != nil {
			s.log.Error("remove deleted file from disk", "fileId", id, "err", err)
		}
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"status":        "deleted",
		"removedOnDisk": removeFromDisk,
	})
}

// Sends back the original lecture file.
func (s *Server) handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUser(r.Context())

	id, err := idParam(r, "fileID")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid file id")
		return
	}

	record, err := s.store.File(r.Context(), user.ID, id)
	if err != nil {
		s.writeStoreError(w, err, "")
		return
	}

	f, err := s.storage.Open(record.DiskPath)
	if err != nil {
		s.log.Error("open file for download", "fileId", id, "err", err)
		s.writeError(w, http.StatusInternalServerError, "could not read the file")
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", record.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(record.ByteSize, 10))
	// Quoted so a filename containing a space or comma cannot break the header.
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=%q", record.OriginalName))

	if _, err := io.Copy(w, f); err != nil {
		s.log.Error("stream file download", "fileId", id, "err", err)
	}
}

// Reports how much of their storage somebody has used.
func (s *Server) handleUsage(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUser(r.Context())

	used, err := s.store.UserStorageUsed(r.Context(), user.ID)
	if err != nil {
		s.writeStoreError(w, err, "")
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"storageUsed":  used,
		"storageQuota": s.cfg.Limits.UserStorageQuota,
		"maxFileSize":  s.cfg.Limits.MaxFileSize,
	})
}

// Turns a byte count into something readable like "100 MB".
func humanBytes(n int64) string {
	const unit = 1 << 10
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}

	value := float64(n)
	units := []string{"KB", "MB", "GB", "TB"}
	var suffix string
	for _, u := range units {
		value /= unit
		suffix = u
		if value < unit {
			break
		}
	}
	return strconv.FormatFloat(value, 'f', -1, 64) + " " + suffix
}
