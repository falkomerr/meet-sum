package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/falkomer/meet-summarize/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive setup wizard",
	Long:  "Run the interactive setup wizard to configure meet-sum.",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

// whisperOption is an entry in the model selector.
type whisperOption struct {
	Name string
	Size string
	Hint string
}

var whisperOptions = []whisperOption{
	{"tiny",   "39 MB",  "very fast   · Intel Mac, older hardware"},
	{"base",   "74 MB",  "fast        · Intel Mac 2018–2021"},
	{"small",  "244 MB", "medium      · M1/M2 base, good quality"},
	{"turbo",  "809 MB", "fast        · M1/M2/M3 — best balance ★"},
	{"medium", "769 MB", "slower      · M1 Pro/Max, M2 Pro/Max"},
	{"large",  "1.5 GB", "slow        · M2/M3 Max/Ultra — best quality"},
}

const defaultModelIdx = 3 // turbo

// printBanner displays the ASCII art banner when stdout is a TTY.
func printBanner() {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return
	}
	cyan := color.New(color.FgCyan, color.Bold)
	dim := color.New(color.FgWhite, color.Faint)

	cyan.Print("\n")
	cyan.Print("  ███╗   ███╗███████╗███████╗████████╗      ███████╗██╗   ██╗███╗   ███╗\n")
	cyan.Print("  ████╗ ████║██╔════╝██╔════╝╚══██╔══╝      ██╔════╝██║   ██║████╗ ████║\n")
	cyan.Print("  ██╔████╔██║█████╗  █████╗     ██║   █████╗███████╗██║   ██║██╔████╔██║\n")
	cyan.Print("  ██║╚██╔╝██║██╔══╝  ██╔══╝     ██║   ╚════╝╚════██║██║   ██║██║╚██╔╝██║\n")
	cyan.Print("  ██║ ╚═╝ ██║███████╗███████╗   ██║         ███████║╚██████╔╝██║ ╚═╝ ██║\n")
	cyan.Print("  ╚═╝     ╚═╝╚══════╝╚══════╝   ╚═╝         ╚══════╝ ╚═════╝ ╚═╝     ╚═╝\n")
	cyan.Print("\n")
	dim.Print("  Meeting Recorder & Summarizer — Local AI, No Cloud\n\n")
}

// selectWhisperModel shows an interactive arrow-key selector.
// Falls back to plain text input when stdin is not a TTY.
func selectWhisperModel() (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Whisper model [tiny/base/small/turbo/medium/large] (default: turbo): ")
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("read whisper model: %w", err)
		}
		m := strings.TrimSpace(line)
		if m == "" {
			return "turbo", nil
		}
		return m, nil
	}

	selected := defaultModelIdx
	totalLines := len(whisperOptions) + 2 // header line + blank line + options

	render := func(first bool) {
		if !first {
			fmt.Printf("\033[%dA", totalLines)
		}
		fmt.Printf("Select Whisper model (\u2191\u2193 navigate, Enter confirm):\r\n")
		fmt.Printf("\r\n")
		for i, opt := range whisperOptions {
			if i == selected {
				fmt.Printf("\033[1;36m\u25b6 %-6s  %-7s  %s\033[0m\r\n", opt.Name, opt.Size, opt.Hint)
			} else {
				fmt.Printf("  %-6s  %-7s  %s\r\n", opt.Name, opt.Size, opt.Hint)
			}
		}
	}

	render(true)

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return "", fmt.Errorf("enter raw mode: %w", err)
	}

	buf := make([]byte, 4)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			term.Restore(int(os.Stdin.Fd()), oldState) //nolint:errcheck
			return "", fmt.Errorf("read input: %w", err)
		}

		switch {
		case n == 1 && (buf[0] == '\r' || buf[0] == '\n'):
			term.Restore(int(os.Stdin.Fd()), oldState) //nolint:errcheck
			fmt.Printf("\033[%dA\033[J", totalLines)
			fmt.Printf("Whisper model: \033[1;32m%s\033[0m\n", whisperOptions[selected].Name)
			return whisperOptions[selected].Name, nil

		case n == 3 && buf[0] == 27 && buf[1] == '[':
			switch buf[2] {
			case 'A': // Up arrow
				if selected > 0 {
					selected--
				}
				render(false)
			case 'B': // Down arrow
				if selected < len(whisperOptions)-1 {
					selected++
				}
				render(false)
			}

		case n == 1 && buf[0] == 27: // ESC — cancel
			term.Restore(int(os.Stdin.Fd()), oldState) //nolint:errcheck
			return "", fmt.Errorf("cancelled by user")
		}
	}
}

func runInit(cmd *cobra.Command, args []string) error {
	printBanner()

	reader := bufio.NewReader(os.Stdin)

	// Step 1: Whisper model — arrow-key selector.
	whisperModel, err := selectWhisperModel()
	if err != nil {
		return fmt.Errorf("select whisper model: %w", err)
	}

	// Step 2: Video directory.
	fmt.Print("Video output directory (default: ./video_dist): ")
	videoDirLine, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read video dir: %w", err)
	}
	videoDir := strings.TrimSpace(videoDirLine)
	if videoDir == "" {
		videoDir = "./video_dist"
	}

	// Step 3: Transcribed directory.
	fmt.Print("Transcribed output directory (default: ./transcribed_dist): ")
	transcribedDirLine, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read transcribed dir: %w", err)
	}
	transcribedDir := strings.TrimSpace(transcribedDirLine)
	if transcribedDir == "" {
		transcribedDir = "./transcribed_dist"
	}

	// Step 4: Build and save config.
	cfg := &config.Config{
		APIKey:         "",
		WhisperModel:   whisperModel,
		VideoDir:       videoDir,
		TranscribedDir: transcribedDir,
		APIBaseURL:     "http://localhost:11434/v1",
		APIModel:       "qwen2.5",
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	// Step 5: Create directories.
	if err := config.EnsureDirs(cfg); err != nil {
		return fmt.Errorf("create directories: %w", err)
	}

	// Step 6: Success.
	green := color.New(color.FgGreen, color.Bold)
	green.Printf("\n✓ Configuration saved to: %s\n", config.ConfigPath())
	fmt.Printf("  Video directory:        %s\n", videoDir)
	fmt.Printf("  Transcribed directory:  %s\n", transcribedDir)
	green.Println("\nmeet-sum is ready! Run: meet-sum record")
	return nil
}
