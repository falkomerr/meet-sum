package cache

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// CachedTranscription holds the result of a cached transcription lookup.
type CachedTranscription struct {
	Transcription string
	Language      string
	Duration      float64
	CreatedAt     time.Time
}

// CacheEntry is a summary row returned by List().
type CacheEntry struct {
	FileHash     string
	WhisperModel string
	CreatedAt    time.Time
}

// Cache wraps an SQLite database used to store transcription results.
type Cache struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS transcriptions (
    file_hash         TEXT NOT NULL,
    whisper_model     TEXT NOT NULL,
    transcription     TEXT NOT NULL,
    language          TEXT,
    duration_seconds  REAL,
    created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (file_hash, whisper_model)
);`

// NewCache opens (or creates) the SQLite database at dbPath, applies the
// schema, and configures WAL mode for better concurrent performance.
func NewCache(dbPath string) (*Cache, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("cache: open db: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("cache: set WAL mode: %w", err)
	}

	if _, err := db.Exec("PRAGMA synchronous=NORMAL;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("cache: set synchronous=NORMAL: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("cache: create schema: %w", err)
	}

	return &Cache{db: db}, nil
}

// Close releases the database connection.
func (c *Cache) Close() error {
	return c.db.Close()
}

// GetFileHash returns the SHA-256 hex digest of the file at filePath.
func (c *Cache) GetFileHash(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("cache: open file for hashing: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("cache: hash file: %w", err)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// Get retrieves a cached transcription. Returns nil, nil when there is no
// matching entry.
func (c *Cache) Get(fileHash, whisperModel string) (*CachedTranscription, error) {
	const q = `
SELECT transcription, language, duration_seconds, created_at
FROM transcriptions
WHERE file_hash = ? AND whisper_model = ?
LIMIT 1;`

	row := c.db.QueryRow(q, fileHash, whisperModel)

	var t CachedTranscription
	var language sql.NullString
	var duration sql.NullFloat64
	var createdAt string

	err := row.Scan(&t.Transcription, &language, &duration, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cache: get: %w", err)
	}

	if language.Valid {
		t.Language = language.String
	}
	if duration.Valid {
		t.Duration = duration.Float64
	}

	t.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAt)
	if err != nil {
		// Fallback: try RFC3339
		t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	}

	return &t, nil
}

// Set inserts or replaces a transcription result in the cache.
func (c *Cache) Set(fileHash, whisperModel, transcription, language string, duration float64) error {
	const q = `
INSERT INTO transcriptions (file_hash, whisper_model, transcription, language, duration_seconds)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(file_hash, whisper_model) DO UPDATE SET
    transcription    = excluded.transcription,
    language         = excluded.language,
    duration_seconds = excluded.duration_seconds,
    created_at       = CURRENT_TIMESTAMP;`

	_, err := c.db.Exec(q, fileHash, whisperModel, transcription, language, duration)
	if err != nil {
		return fmt.Errorf("cache: set: %w", err)
	}
	return nil
}

// List returns all cached entries ordered by creation time descending.
func (c *Cache) List() ([]CacheEntry, error) {
	const q = `
SELECT file_hash, whisper_model, created_at
FROM transcriptions
ORDER BY created_at DESC;`

	rows, err := c.db.Query(q)
	if err != nil {
		return nil, fmt.Errorf("cache: list: %w", err)
	}
	defer rows.Close()

	var entries []CacheEntry
	for rows.Next() {
		var e CacheEntry
		var createdAt string
		if err := rows.Scan(&e.FileHash, &e.WhisperModel, &createdAt); err != nil {
			return nil, fmt.Errorf("cache: list scan: %w", err)
		}
		e.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAt)
		if err != nil {
			e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		}
		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cache: list rows: %w", err)
	}

	return entries, nil
}

// Clear deletes all cached transcription entries.
func (c *Cache) Clear() error {
	if _, err := c.db.Exec("DELETE FROM transcriptions;"); err != nil {
		return fmt.Errorf("cache: clear: %w", err)
	}
	return nil
}
