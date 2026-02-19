//go:build windows

package recorder

import (
	"fmt"
	"os/exec"
	"strings"
)

// AudioDevices holds the DirectShow device identifiers for Windows.
type AudioDevices struct {
	SystemAudio string // e.g. "Stereo Mix (Realtek Audio)"
	Microphone  string // e.g. "Microphone (Realtek Audio)"
}

// getDevices detects DirectShow audio devices on Windows.
// System audio is captured via a WASAPI loopback device.
func getDevices() (*AudioDevices, error) {
	// ffmpeg writes device list to stderr; combined output captures it.
	out, _ := exec.Command(
		"ffmpeg",
		"-list_devices", "true",
		"-f", "dshow",
		"-i", "dummy",
	).CombinedOutput()

	output := string(out)

	loopback, mic, err := findDShowDevices(output)
	if err != nil {
		return nil, err
	}

	return &AudioDevices{
		SystemAudio: loopback,
		Microphone:  mic,
	}, nil
}

// findDShowDevices parses ffmpeg dshow device list output.
// It looks for a loopback/stereo-mix device for system audio and a
// microphone device for mic input.
//
// Example ffmpeg output line:
//
//	[dshow @ ...] "Stereo Mix (Realtek Audio)" (audio)
func findDShowDevices(output string) (loopback, mic string, err error) {
	loopbackKeywords := []string{"Stereo Mix", "loopback", "Loopback", "WASAPI", "What U Hear", "Wave Out Mix"}
	micKeywords := []string{"Microphone", "microphone", "mic", "Mic", "Input"}

	var firstAudio string

	for _, line := range strings.Split(output, "\n") {
		// Device lines contain (audio) and a quoted name.
		if !strings.Contains(line, "(audio)") {
			continue
		}

		name := extractQuotedName(line)
		if name == "" {
			continue
		}

		if firstAudio == "" {
			firstAudio = name
		}

		// Check for loopback device.
		if loopback == "" {
			for _, kw := range loopbackKeywords {
				if strings.Contains(name, kw) {
					loopback = name
					break
				}
			}
		}

		// Check for microphone device.
		if mic == "" {
			for _, kw := range micKeywords {
				if strings.Contains(name, kw) {
					mic = name
					break
				}
			}
		}
	}

	if loopback == "" {
		return "", "", fmt.Errorf(
			"no WASAPI loopback/Stereo Mix device found.\n" +
				"Enable 'Stereo Mix' in Windows Sound settings:\n" +
				"  Control Panel > Sound > Recording tab > right-click > Show Disabled Devices\n" +
				"  Then enable 'Stereo Mix'.\n" +
				"Run 'ffmpeg -list_devices true -f dshow -i dummy' to list available devices.",
		)
	}

	if mic == "" {
		// Fall back to the first audio device found if no mic keyword matched.
		if firstAudio != "" && firstAudio != loopback {
			mic = firstAudio
		} else {
			mic = loopback // worst-case: duplicate input is better than failing
		}
	}

	return loopback, mic, nil
}

// extractQuotedName returns the content of the first quoted string on a line.
func extractQuotedName(line string) string {
	start := strings.Index(line, `"`)
	if start < 0 {
		return ""
	}
	end := strings.Index(line[start+1:], `"`)
	if end < 0 {
		return ""
	}
	return line[start+1 : start+1+end]
}

// buildFFmpegArgs constructs the ffmpeg argument list for Windows.
// Captures system audio loopback and microphone via DirectShow, then merges them.
func buildFFmpegArgs(devices *AudioDevices, outputPath string) []string {
	return []string{
		// System audio input via DirectShow loopback.
		"-f", "dshow",
		"-i", fmt.Sprintf("audio=%s", devices.SystemAudio),
		// Microphone input via DirectShow.
		"-f", "dshow",
		"-i", fmt.Sprintf("audio=%s", devices.Microphone),
		// Merge both audio streams into a single stereo stream.
		"-filter_complex", "amerge=inputs=2",
		"-ac", "2",
		// WAV output.
		"-acodec", "pcm_s16le",
		outputPath,
	}
}
