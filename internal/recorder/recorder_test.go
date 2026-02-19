package recorder_test

import (
	"strings"
	"testing"

	"github.com/falkomer/meet-summarize/internal/recorder"
)

func TestNewRecorder(t *testing.T) {
	r := recorder.NewRecorder("/tmp/test-output")

	if r == nil {
		t.Fatal("NewRecorder returned nil")
	}
	if r.OutputDir != "/tmp/test-output" {
		t.Errorf("OutputDir = %q, want %q", r.OutputDir, "/tmp/test-output")
	}
}

func TestFilePath_Initial(t *testing.T) {
	r := recorder.NewRecorder("/tmp/test-output")

	if got := r.FilePath(); got != "" {
		t.Errorf("FilePath() before Start() = %q, want empty string", got)
	}
}

func TestStop_WhenNotRunning(t *testing.T) {
	r := recorder.NewRecorder("/tmp/test-output")

	err := r.Stop()
	if err == nil {
		t.Fatal("Stop() when not running expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not running") {
		t.Errorf("Stop() error = %q, want message containing \"not running\"", err.Error())
	}
}
