package transcriber

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// WhisperTranscriber runs the Python transcribe.py script via subprocess.
type WhisperTranscriber struct {
	ScriptPath string // path to transcribe.py
	Model      string // whisper model name (e.g. "base", "small", "medium", "large")
}

// TranscriptionResult holds the structured output from Whisper.
type TranscriptionResult struct {
	Text     string  `json:"text"`
	Language string  `json:"language"`
	Duration float64 `json:"duration"`
}

// NewWhisperTranscriber creates a WhisperTranscriber with the given script path and model.
func NewWhisperTranscriber(scriptPath, model string) *WhisperTranscriber {
	return &WhisperTranscriber{
		ScriptPath: scriptPath,
		Model:      model,
	}
}

// Transcribe runs the Whisper Python script on the given audio file and returns the result.
func (w *WhisperTranscriber) Transcribe(audioPath string) (*TranscriptionResult, error) {
	// Create a temp file to receive JSON output.
	tmpFile, err := os.CreateTemp("", "whisper-result-*.json")
	if err != nil {
		return nil, fmt.Errorf("transcriber: create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// Build the command.
	cmd := exec.Command(
		"python3",
		w.ScriptPath,
		"--model", w.Model,
		"--input", audioPath,
		"--output", tmpPath,
	)

	// Stream stderr (progress messages) to our stderr so the user sees progress.
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Provide actionable error messages for common failure modes.
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("transcriber: python script exited with code %d", exitErr.ExitCode())
		}
		// exec.ErrNotFound or similar
		return nil, fmt.Errorf("transcriber: failed to run python3 (is it installed?): %w", err)
	}

	// Read the JSON output file.
	f, err := os.Open(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("transcriber: open output file: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("transcriber: read output file: %w", err)
	}

	var result TranscriptionResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("transcriber: parse JSON output: %w", err)
	}

	return &result, nil
}

// ExtractAudio uses ffmpeg to extract the audio track from a video file.
// The extracted audio is written to outputPath (e.g. a .wav file).
func ExtractAudio(videoPath, outputPath string) error {
	cmd := exec.Command(
		"ffmpeg",
		"-i", videoPath,
		"-q:a", "0",
		"-map", "a",
		"-y", outputPath,
	)
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("transcriber: ffmpeg exited with code %d", exitErr.ExitCode())
		}
		return fmt.Errorf("transcriber: failed to run ffmpeg (is it installed?): %w", err)
	}
	return nil
}
