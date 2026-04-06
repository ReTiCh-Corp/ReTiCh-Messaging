package storage

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestNewLocalStorage_CreatesUploadDir(t *testing.T) {
	dir := t.TempDir()
	uploadDir := filepath.Join(dir, "uploads")

	ls, err := NewLocalStorage(uploadDir, "http://localhost:8082/uploads")
	if err != nil {
		t.Fatalf("NewLocalStorage returned error: %v", err)
	}
	if ls == nil {
		t.Fatal("NewLocalStorage returned nil")
	}
}

func TestNewLocalStorage_ExistingDir(t *testing.T) {
	dir := t.TempDir()

	ls, err := NewLocalStorage(dir, "http://localhost:8082/uploads")
	if err != nil {
		t.Fatalf("NewLocalStorage with existing dir returned error: %v", err)
	}
	if ls == nil {
		t.Fatal("NewLocalStorage returned nil")
	}
}

func TestSave_WritesFileAndReturnsURL(t *testing.T) {
	dir := t.TempDir()
	baseURL := "http://localhost:8082/uploads"

	ls, err := NewLocalStorage(dir, baseURL)
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	content := "hello world"
	url, err := ls.Save("test.txt", strings.NewReader(content))
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	if url == "" {
		t.Fatal("Save returned empty URL")
	}
	if !strings.HasPrefix(url, baseURL+"/") {
		t.Errorf("URL %q does not start with baseURL %q", url, baseURL+"/")
	}
}

func TestSave_PreservesFileExtension(t *testing.T) {
	dir := t.TempDir()
	ls, err := NewLocalStorage(dir, "http://localhost:8082/uploads")
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	url, err := ls.Save("image.png", strings.NewReader("fake png data"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !strings.HasSuffix(url, ".png") {
		t.Errorf("expected URL to end in .png, got %q", url)
	}
}

func TestSave_UniqueNamesForSameFilename(t *testing.T) {
	dir := t.TempDir()
	ls, err := NewLocalStorage(dir, "http://localhost:8082/uploads")
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	url1, err := ls.Save("file.txt", strings.NewReader("data1"))
	if err != nil {
		t.Fatalf("Save 1: %v", err)
	}
	url2, err := ls.Save("file.txt", strings.NewReader("data2"))
	if err != nil {
		t.Fatalf("Save 2: %v", err)
	}
	if url1 == url2 {
		t.Error("expected two saves to produce different URLs")
	}
}

func TestBaseURL_ReturnsConfiguredValue(t *testing.T) {
	dir := t.TempDir()
	baseURL := "http://example.com/static"

	ls, err := NewLocalStorage(dir, baseURL)
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}
	if got := ls.BaseURL(); got != baseURL {
		t.Errorf("BaseURL() = %q, want %q", got, baseURL)
	}
}
