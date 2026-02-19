package recorder

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Recorder manages an ffmpeg subprocess that captures system audio and microphone.
type Recorder struct {
	OutputDir    string
	cmd          *exec.Cmd
	cancel       context.CancelFunc
	filePath     string
	stdin        io.WriteCloser
	restoreAudio func() // restores system audio output after recording
}

// NewRecorder creates a Recorder that will write recordings to outputDir.
func NewRecorder(outputDir string) *Recorder {
	return &Recorder{OutputDir: outputDir}
}

// Start begins recording and returns the path of the output file.
func (r *Recorder) Start() (string, error) {
	devices, err := getDevices()
	if err != nil {
		return "", fmt.Errorf("get audio devices: %w", err)
	}

	restore, err := routeSystemAudioToCapture()
	if err != nil {
		return "", fmt.Errorf("route system audio: %w", err)
	}
	r.restoreAudio = restore

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	r.filePath = filepath.Join(r.OutputDir, fmt.Sprintf("recording_%s.wav", timestamp))

	if err := os.MkdirAll(r.OutputDir, 0o755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}

	args := buildFFmpegArgs(devices, r.filePath)

	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel

	r.cmd = exec.CommandContext(ctx, "ffmpeg", args...)

	stdin, err := r.cmd.StdinPipe()
	if err != nil {
		cancel()
		return "", fmt.Errorf("create stdin pipe: %w", err)
	}
	r.stdin = stdin

	if err := r.cmd.Start(); err != nil {
		cancel()
		r.restoreAudio()
		return "", fmt.Errorf("start ffmpeg: %w", err)
	}

	return r.filePath, nil
}

// Stop sends "q" to ffmpeg's stdin for a graceful shutdown.
// Falls back to context cancellation if that fails.
// Restores the original system audio output device.
func (r *Recorder) Stop() error {
	if r.cmd == nil || r.cmd.Process == nil {
		return fmt.Errorf("recorder is not running")
	}

	if r.restoreAudio != nil {
		r.restoreAudio()
		r.restoreAudio = nil
	}

	// Try graceful stop first: ffmpeg quits cleanly on "q\n".
	_, writeErr := fmt.Fprintln(r.stdin, "q")
	_ = r.stdin.Close()

	done := make(chan error, 1)
	go func() {
		done <- r.cmd.Wait()
	}()

	select {
	case err := <-done:
		if writeErr != nil {
			// stdin write failed but process may have exited fine via context.
			return writeErr
		}
		return err
	case <-time.After(5 * time.Second):
		// Graceful stop timed out — force via context cancellation.
		if r.cancel != nil {
			r.cancel()
		}
		return <-done
	}
}

// FilePath returns the path of the file being recorded.
func (r *Recorder) FilePath() string {
	return r.filePath
}
