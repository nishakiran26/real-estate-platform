// Package storage provides a provider-agnostic file store. Handlers and services
// depend only on the Storage interface, so swapping local disk for S3,
// Cloudinary or any other object store is a wiring change in main.go.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrFileTooLarge   = errors.New("file exceeds the maximum allowed size")
	ErrTypeNotAllowed = errors.New("file type is not allowed")
	ErrNotFound       = errors.New("file not found")
)

// Object describes a stored file.
type Object struct {
	Key      string `json:"key"`
	URL      string `json:"url"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
}

// Storage is the contract every provider implements.
type Storage interface {
	// Save persists r under a generated key inside folder and returns its metadata.
	Save(ctx context.Context, folder, filename string, r io.Reader, size int64, mimeType string) (*Object, error)
	// Delete removes a previously stored object. Deleting a missing key is not an error.
	Delete(ctx context.Context, key string) error
	// URL resolves a storage key to a publicly reachable URL.
	URL(key string) string
	// Open returns a reader for a stored object; callers must close it.
	Open(ctx context.Context, key string) (io.ReadCloser, error)
}

// AllowedImageTypes and AllowedDocTypes gate uploads by MIME type.
var (
	AllowedImageTypes = map[string]string{
		"image/jpeg": ".jpg",
		"image/jpg":  ".jpg",
		"image/png":  ".png",
		"image/webp": ".webp",
	}
	AllowedDocTypes = map[string]string{
		"application/pdf": ".pdf",
	}
)

// MaxImageSize and MaxDocSize are the upload ceilings in bytes.
const (
	MaxImageSize = 5 << 20  // 5 MB
	MaxDocSize   = 20 << 20 // 20 MB
)

// ValidateUpload checks size and MIME type against the given allow-list and
// returns the canonical extension for the type.
func ValidateUpload(mimeType string, size, maxSize int64, allowed map[string]string) (string, error) {
	if size <= 0 {
		return "", errors.New("file is empty")
	}
	if size > maxSize {
		return "", fmt.Errorf("%w: %d bytes (limit %d)", ErrFileTooLarge, size, maxSize)
	}
	// Strip any "; charset=..." parameters before matching.
	if parsed, _, err := mime.ParseMediaType(mimeType); err == nil {
		mimeType = parsed
	}
	ext, ok := allowed[strings.ToLower(mimeType)]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrTypeNotAllowed, mimeType)
	}
	return ext, nil
}

// BuildKey produces a collision-free storage key that preserves the original
// extension: e.g. "properties/12/9f8e...-brochure.pdf".
func BuildKey(folder, filename, ext string) string {
	base := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	base = sanitize(base)
	if base == "" {
		base = "file"
	}
	if len(base) > 60 {
		base = base[:60]
	}
	folder = strings.Trim(folder, "/")
	return fmt.Sprintf("%s/%s-%s%s", folder, uuid.NewString(), base, ext)
}

func sanitize(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_', r == ' ':
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
