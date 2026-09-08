package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LocalStorage writes files to disk and serves them from a static route. It is
// the default provider for development and single-node deployments.
type LocalStorage struct {
	basePath string // directory files are written to
	baseURL  string // public URL prefix, e.g. http://localhost:8080/uploads
}

func NewLocal(basePath, baseURL string) (*LocalStorage, error) {
	if err := os.MkdirAll(basePath, 0o755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &LocalStorage{basePath: basePath, baseURL: strings.TrimRight(baseURL, "/")}, nil
}

func (l *LocalStorage) Save(ctx context.Context, folder, filename string, r io.Reader, size int64, mimeType string) (*Object, error) {
	ext := filepath.Ext(filename)
	key := BuildKey(folder, filename, ext)
	full, err := l.resolve(key)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return nil, fmt.Errorf("create folder: %w", err)
	}
	dst, err := os.Create(full)
	if err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}
	defer dst.Close()

	written, err := io.Copy(dst, r)
	if err != nil {
		os.Remove(full) // do not leave a truncated file behind
		return nil, fmt.Errorf("write file: %w", err)
	}
	return &Object{Key: key, URL: l.URL(key), Size: written, MimeType: mimeType}, nil
}

func (l *LocalStorage) Delete(ctx context.Context, key string) error {
	full, err := l.resolve(key)
	if err != nil {
		return err
	}
	if err := os.Remove(full); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (l *LocalStorage) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	full, err := l.resolve(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(full)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	return f, err
}

func (l *LocalStorage) URL(key string) string {
	if key == "" {
		return ""
	}
	return l.baseURL + "/" + strings.TrimLeft(key, "/")
}

// resolve maps a key to an absolute path, rejecting traversal outside basePath.
func (l *LocalStorage) resolve(key string) (string, error) {
	clean := filepath.Clean("/" + strings.TrimLeft(key, "/"))
	full := filepath.Join(l.basePath, clean)
	base, err := filepath.Abs(l.basePath)
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(abs, base) {
		return "", errors.New("invalid storage key")
	}
	return abs, nil
}
