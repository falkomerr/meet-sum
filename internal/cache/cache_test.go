package cache_test

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/falkomer/meet-summarize/internal/cache"
)

func newTestCache(t *testing.T) *cache.Cache {
	t.Helper()
	dir := t.TempDir()
	c, err := cache.NewCache(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

func TestNewCache(t *testing.T) {
	dir := t.TempDir()
	c, err := cache.NewCache(filepath.Join(dir, "cache.db"))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestGetFileHash(t *testing.T) {
	c := newTestCache(t)

	content := []byte("hello, meet-summarize")
	dir := t.TempDir()
	fpath := filepath.Join(dir, "audio.txt")
	if err := os.WriteFile(fpath, content, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	sum := sha256.Sum256(content)
	expected := fmt.Sprintf("%x", sum)

	got, err := c.GetFileHash(fpath)
	if err != nil {
		t.Fatalf("GetFileHash: %v", err)
	}
	if got != expected {
		t.Errorf("hash mismatch: got %q, want %q", got, expected)
	}
}

func TestGetCacheMiss(t *testing.T) {
	c := newTestCache(t)

	result, err := c.Get("nonexistent-hash", "base")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil on cache miss, got %+v", result)
	}
}

func TestSetAndGet(t *testing.T) {
	c := newTestCache(t)

	fileHash := "abc123"
	model := "base"
	transcription := "This is a test transcription."
	language := "en"
	duration := 42.5

	if err := c.Set(fileHash, model, transcription, language, duration); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := c.Get(fileHash, model)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("expected a result, got nil")
	}
	if got.Transcription != transcription {
		t.Errorf("Transcription: got %q, want %q", got.Transcription, transcription)
	}
	if got.Language != language {
		t.Errorf("Language: got %q, want %q", got.Language, language)
	}
	if got.Duration != duration {
		t.Errorf("Duration: got %v, want %v", got.Duration, duration)
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestSetOverwrite(t *testing.T) {
	c := newTestCache(t)

	fileHash := "deadbeef"
	model := "small"

	if err := c.Set(fileHash, model, "original text", "en", 10.0); err != nil {
		t.Fatalf("Set (first): %v", err)
	}

	updated := "updated transcription"
	if err := c.Set(fileHash, model, updated, "fr", 20.0); err != nil {
		t.Fatalf("Set (second): %v", err)
	}

	got, err := c.Get(fileHash, model)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("expected a result after overwrite, got nil")
	}
	if got.Transcription != updated {
		t.Errorf("expected updated transcription %q, got %q", updated, got.Transcription)
	}
	if got.Language != "fr" {
		t.Errorf("expected updated language %q, got %q", "fr", got.Language)
	}
	if got.Duration != 20.0 {
		t.Errorf("expected updated duration 20.0, got %v", got.Duration)
	}
}

func TestList(t *testing.T) {
	c := newTestCache(t)

	entries := []struct {
		hash  string
		model string
	}{
		{"hash1", "base"},
		{"hash2", "small"},
		{"hash3", "medium"},
	}

	for _, e := range entries {
		if err := c.Set(e.hash, e.model, "text", "en", 1.0); err != nil {
			t.Fatalf("Set(%s, %s): %v", e.hash, e.model, err)
		}
	}

	list, err := c.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) == 0 {
		t.Error("expected non-empty list, got empty")
	}
	if len(list) != len(entries) {
		t.Errorf("expected %d entries, got %d", len(entries), len(list))
	}
	for _, item := range list {
		if item.FileHash == "" {
			t.Error("FileHash should not be empty")
		}
		if item.WhisperModel == "" {
			t.Error("WhisperModel should not be empty")
		}
		if item.CreatedAt.IsZero() {
			t.Error("CreatedAt should not be zero")
		}
	}
}

func TestClear(t *testing.T) {
	c := newTestCache(t)

	if err := c.Set("h1", "base", "text", "en", 1.0); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := c.Set("h2", "small", "text", "en", 2.0); err != nil {
		t.Fatalf("Set: %v", err)
	}

	if err := c.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	list, err := c.List()
	if err != nil {
		t.Fatalf("List after Clear: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list after Clear, got %d entries", len(list))
	}
}

func TestGetFileHash_MissingFile(t *testing.T) {
	c := newTestCache(t)

	_, err := c.GetFileHash("/nonexistent/path/to/file.mp3")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}
