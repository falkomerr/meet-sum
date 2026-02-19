//go:build linux

package recorder

import (
	"fmt"
	"os/exec"
	"strings"
)

// AudioDevices holds the PulseAudio device identifiers for Linux.
type AudioDevices struct {
	SystemAudio string // e.g. "alsa_output.pci-0000_00_1f.3.analog-stereo.monitor"
	Microphone  string // "default"
}

// getDevices detects PulseAudio sources on Linux.
// System audio is captured via a monitor source (loopback of the output sink).
func getDevices() (*AudioDevices, error) {
	out, err := exec.Command("pactl", "list", "short", "sources").Output()
	if err != nil {
		return nil, fmt.Errorf("run pactl: %w", err)
	}

	monitor, found := findMonitorSource(string(out))
	if !found {
		return nil, fmt.Errorf(
			"no PulseAudio monitor source found.\n" +
				"Ensure PulseAudio is running and a sink monitor is available.\n" +
				"Run 'pactl list short sources' to list available sources.",
		)
	}

	return &AudioDevices{
		SystemAudio: monitor,
		Microphone:  "default",
	}, nil
}

// findMonitorSource scans pactl output for a .monitor source.
// pactl short output columns: index, name, driver, sample_spec, state
func findMonitorSource(output string) (string, bool) {
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := fields[1]
		if strings.HasSuffix(name, ".monitor") {
			return name, true
		}
	}
	return "", false
}

// buildFFmpegArgs constructs the ffmpeg argument list for Linux.
// Captures system audio monitor and default microphone, then merges them.
func buildFFmpegArgs(devices *AudioDevices, outputPath string) []string {
	return []string{
		// System audio input via PulseAudio monitor.
		"-f", "pulse",
		"-i", devices.SystemAudio,
		// Microphone input via PulseAudio default source.
		"-f", "pulse",
		"-i", devices.Microphone,
		// Merge both audio streams into a single stereo stream.
		"-filter_complex", "amerge=inputs=2",
		"-ac", "2",
		// WAV output.
		"-acodec", "pcm_s16le",
		outputPath,
	}
}
