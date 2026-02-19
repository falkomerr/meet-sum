//go:build darwin

package recorder

import (
	"fmt"
	"os/exec"
	"strings"
)

// AudioDevices holds the ffmpeg device identifiers for macOS.
type AudioDevices struct {
	SystemAudio string // e.g. "BlackHole 2ch"
	Microphone  string // e.g. "default"
}

// getDevices detects available audio devices on macOS.
// System audio is captured via BlackHole virtual audio driver.
// Returns an error with installation instructions if BlackHole is not found.
func getDevices() (*AudioDevices, error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, fmt.Errorf(
			"ffmpeg not found.\n" +
				"Install it with: brew install ffmpeg",
		)
	}

	// List avfoundation devices; ffmpeg writes device list to stderr.
	out, _ := exec.Command(
		"ffmpeg",
		"-f", "avfoundation",
		"-list_devices", "true",
		"-i", "",
	).CombinedOutput()

	output := string(out)

	blackhole, found := findBlackHole(output)
	if !found {
		return nil, fmt.Errorf(
			"BlackHole audio driver not found.\n" +
				"Install it with: brew install blackhole-2ch\n" +
				"Then set it as your system audio output in System Settings > Sound.",
		)
	}

	return &AudioDevices{
		SystemAudio: blackhole,
		Microphone:  "default",
	}, nil
}

// findBlackHole scans ffmpeg device list output for a BlackHole device name.
func findBlackHole(output string) (string, bool) {
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "BlackHole") {
			// Extract the device name between brackets: [AVFoundation audio device] [0] BlackHole 2ch
			idx := strings.Index(line, "BlackHole")
			if idx >= 0 {
				name := strings.TrimSpace(line[idx:])
				// Strip any trailing content after the device name (e.g. newline chars).
				if nl := strings.IndexAny(name, "\r\n"); nl >= 0 {
					name = name[:nl]
				}
				return name, true
			}
		}
	}
	return "", false
}

// buildFFmpegArgs constructs the ffmpeg argument list for macOS.
// It captures system audio (BlackHole) and microphone, then merges them.
func buildFFmpegArgs(devices *AudioDevices, outputPath string) []string {
	return []string{
		// System audio input (BlackHole loopback).
		"-f", "avfoundation",
		"-i", fmt.Sprintf(":%s", devices.SystemAudio),
		// Microphone input.
		"-f", "avfoundation",
		"-i", fmt.Sprintf(":%s", devices.Microphone),
		// Merge both audio streams into a single stereo stream.
		"-filter_complex", "amerge=inputs=2",
		"-ac", "2",
		// WAV output.
		"-acodec", "pcm_s16le",
		outputPath,
	}
}
