package transcriber_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"testing"

	"github.com/falkomer/meet-summarize/internal/transcriber"
)

func TestNewWhisperTranscriber(t *testing.T) {
	w := transcriber.NewWhisperTranscriber("/path/to/script.py", "base")
	if w.ScriptPath != "/path/to/script.py" {
		t.Errorf("expected ScriptPath %q, got %q", "/path/to/script.py", w.ScriptPath)
	}
	if w.Model != "base" {
		t.Errorf("expected Model %q, got %q", "base", w.Model)
	}
}

func TestTranscriptionResult_JSONParsing(t *testing.T) {
	raw := `{"text":"hello","language":"en","duration":5.5}`
	var result transcriber.TranscriptionResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "hello" {
		t.Errorf("expected Text %q, got %q", "hello", result.Text)
	}
	if result.Language != "en" {
		t.Errorf("expected Language %q, got %q", "en", result.Language)
	}
	if result.Duration != 5.5 {
		t.Errorf("expected Duration %v, got %v", 5.5, result.Duration)
	}
}

func TestTranscribe_InvalidScript(t *testing.T) {
	w := transcriber.NewWhisperTranscriber("/nonexistent/path/to/script.py", "base")

	tmpAudio, err := os.CreateTemp(t.TempDir(), "audio-*.wav")
	if err != nil {
		t.Fatalf("failed to create temp audio file: %v", err)
	}
	tmpAudio.Close()

	_, err = w.Transcribe(tmpAudio.Name())
	if err == nil {
		t.Error("expected error when script does not exist, got nil")
	}
}

func TestTranscribe_ScriptWritesValidJSON(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}

	dir := t.TempDir()

	scriptContent := `import sys, json
args = sys.argv[1:]
for i, a in enumerate(args):
    if a == '--output':
        with open(args[i+1], 'w') as f:
            json.dump({'text': 'hello world', 'language': 'en', 'duration': 10.5}, f)
`
	scriptPath := dir + "/fake_whisper.py"
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("failed to write fake script: %v", err)
	}

	tmpAudio, err := os.CreateTemp(dir, "audio-*.wav")
	if err != nil {
		t.Fatalf("failed to create temp audio file: %v", err)
	}
	tmpAudio.Close()

	w := transcriber.NewWhisperTranscriber(scriptPath, "base")
	result, err := w.Transcribe(tmpAudio.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Text != "hello world" {
		t.Errorf("expected Text %q, got %q", "hello world", result.Text)
	}
	if result.Language != "en" {
		t.Errorf("expected Language %q, got %q", "en", result.Language)
	}
	if result.Duration != 10.5 {
		t.Errorf("expected Duration %v, got %v", 10.5, result.Duration)
	}
}

func TestExtractAudio_InvalidVideoPath(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	dir := t.TempDir()
	outputPath := dir + "/output.wav"

	err := transcriber.ExtractAudio("/nonexistent/video.mp4", outputPath)
	if err == nil {
		t.Error("expected error for invalid video path, got nil")
	}
}
