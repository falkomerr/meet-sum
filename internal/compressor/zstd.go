package compressor

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/klauspost/compress/zstd"
)

// Compress compresses srcPath to srcPath+".zst" using zstd at SpeedDefault level.
// On success, deletes the original file and returns the path to the .zst file.
func Compress(srcPath string) (string, error) {
	dstPath := srcPath + ".zst"

	src, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("compressor: open source file: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return "", fmt.Errorf("compressor: create destination file: %w", err)
	}

	enc, err := zstd.NewWriter(dst, zstd.WithEncoderLevel(zstd.SpeedDefault))
	if err != nil {
		dst.Close()
		os.Remove(dstPath)
		return "", fmt.Errorf("compressor: create zstd encoder: %w", err)
	}

	if _, err = io.Copy(enc, src); err != nil {
		enc.Close()
		dst.Close()
		os.Remove(dstPath)
		return "", fmt.Errorf("compressor: compress data: %w", err)
	}

	if err = enc.Close(); err != nil {
		dst.Close()
		os.Remove(dstPath)
		return "", fmt.Errorf("compressor: finalize zstd stream: %w", err)
	}

	if err = dst.Close(); err != nil {
		os.Remove(dstPath)
		return "", fmt.Errorf("compressor: close destination file: %w", err)
	}

	if err = os.Remove(srcPath); err != nil {
		return "", fmt.Errorf("compressor: delete original file: %w", err)
	}

	return dstPath, nil
}

// Decompress decompresses a .zst file and returns the path to the decompressed file.
// The output path is srcPath with the ".zst" suffix stripped.
func Decompress(srcPath string) (string, error) {
	if !strings.HasSuffix(srcPath, ".zst") {
		return "", fmt.Errorf("compressor: file does not have .zst extension: %s", srcPath)
	}

	dstPath := strings.TrimSuffix(srcPath, ".zst")

	src, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("compressor: open source file: %w", err)
	}
	defer src.Close()

	dec, err := zstd.NewReader(src)
	if err != nil {
		return "", fmt.Errorf("compressor: create zstd decoder: %w", err)
	}
	defer dec.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return "", fmt.Errorf("compressor: create destination file: %w", err)
	}

	if _, err = io.Copy(dst, dec); err != nil {
		dst.Close()
		os.Remove(dstPath)
		return "", fmt.Errorf("compressor: decompress data: %w", err)
	}

	if err = dst.Close(); err != nil {
		os.Remove(dstPath)
		return "", fmt.Errorf("compressor: close destination file: %w", err)
	}

	return dstPath, nil
}

// IsCompressed reports whether the given path has a .zst extension.
func IsCompressed(path string) bool {
	return strings.HasSuffix(path, ".zst")
}
