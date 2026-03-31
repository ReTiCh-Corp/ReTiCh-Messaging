package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// FileStorage defines the interface for file storage backends.
type FileStorage interface {
	Save(filename string, reader io.Reader) (url string, err error)
	BaseURL() string
}

// LocalStorage stores files on the local filesystem.
type LocalStorage struct {
	uploadDir string
	baseURL   string
}

// NewLocalStorage creates a new local file storage.
// uploadDir is the directory where files are saved.
// baseURL is the public URL prefix for serving files (e.g. "http://localhost:8082/uploads").
func NewLocalStorage(uploadDir, baseURL string) (*LocalStorage, error) {
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload dir: %w", err)
	}
	return &LocalStorage{uploadDir: uploadDir, baseURL: baseURL}, nil
}

// Save writes the file to disk with a unique name and returns the public URL.
func (ls *LocalStorage) Save(filename string, reader io.Reader) (string, error) {
	ext := filepath.Ext(filename)
	uniqueName := uuid.New().String() + ext

	filePath := filepath.Join(ls.uploadDir, uniqueName)
	f, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, reader); err != nil {
		os.Remove(filePath)
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return ls.baseURL + "/" + uniqueName, nil
}

// BaseURL returns the public URL prefix.
func (ls *LocalStorage) BaseURL() string {
	return ls.baseURL
}
