package compressor_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/falkomer/meet-summarize/internal/compressor"
)

func TestIsCompressed(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"file.zst", true},
		{"/tmp/recording.wav.zst", true},
		{"file.wav", false},
		{"file.mp3", false},
		{"file.zst.bak", false},
		{"", false},
		{".zst", true},
	}

	for _, tc := range tests {
		got := compressor.IsCompressed(tc.path)
		if got != tc.want {
			t.Errorf("IsCompressed(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func writeTempFile(t *testing.T, dir, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("writeTempFile: %v", err)
	}
	return path
}

func TestCompress_CreatesZstFile(t *testing.T) {
	dir := t.TempDir()
	src := writeTempFile(t, dir, "data.txt", []byte("hello world test data"))

	zstPath, err := compressor.Compress(src)
	if err != nil {
		t.Fatalf("Compress returned error: %v", err)
	}

	if zstPath != src+".zst" {
		t.Errorf("returned path = %q, want %q", zstPath, src+".zst")
	}

	if _, err := os.Stat(zstPath); os.IsNotExist(err) {
		t.Errorf(".zst file does not exist at %q", zstPath)
	}
}

func TestCompress_DeletesOriginal(t *testing.T) {
	dir := t.TempDir()
	src := writeTempFile(t, dir, "data.txt", []byte("hello world test data"))

	_, err := compressor.Compress(src)
	if err != nil {
		t.Fatalf("Compress returned error: %v", err)
	}

	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("original file still exists at %q after Compress", src)
	}
}

func TestCompress_MissingSource(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "nonexistent.txt")

	_, err := compressor.Compress(missing)
	if err == nil {
		t.Error("Compress on missing source: expected error, got nil")
	}
}

func TestDecompress_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	original := []byte("hello world test data")
	src := writeTempFile(t, dir, "data.txt", original)

	zstPath, err := compressor.Compress(src)
	if err != nil {
		t.Fatalf("Compress returned error: %v", err)
	}

	// Original should be gone, .zst should exist.
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("original file still exists after Compress")
	}
	if _, err := os.Stat(zstPath); os.IsNotExist(err) {
		t.Errorf(".zst file does not exist after Compress")
	}

	outPath, err := compressor.Decompress(zstPath)
	if err != nil {
		t.Fatalf("Decompress returned error: %v", err)
	}

	if outPath != src {
		t.Errorf("Decompress returned path %q, want %q", outPath, src)
	}

	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile after Decompress: %v", err)
	}

	if string(got) != string(original) {
		t.Errorf("decompressed content = %q, want %q", got, original)
	}
}

func TestDecompress_NoZstExtension(t *testing.T) {
	dir := t.TempDir()
	src := writeTempFile(t, dir, "data.wav", []byte("not compressed"))

	_, err := compressor.Decompress(src)
	if err == nil {
		t.Error("Decompress on non-.zst file: expected error, got nil")
	}
}

func TestDecompress_MissingFile(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "nonexistent.zst")

	_, err := compressor.Decompress(missing)
	if err == nil {
		t.Error("Decompress on missing .zst file: expected error, got nil")
	}
}
