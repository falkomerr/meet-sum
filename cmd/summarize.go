package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/falkomer/meet-summarize/internal/cache"
	"github.com/falkomer/meet-summarize/internal/compressor"
	"github.com/falkomer/meet-summarize/internal/config"
	"github.com/falkomer/meet-summarize/internal/summarizer"
	"github.com/falkomer/meet-summarize/internal/transcriber"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var customPrompt string

var summarizeCmd = &cobra.Command{
	Use:   "summarize [file]",
	Short: "Transcribe and summarize a meeting recording",
	Long:  "Full pipeline: extract audio (if video), transcribe with Whisper, summarize with Ollama, save result as Markdown.\nIf no file is given, shows an interactive list of recordings to choose from.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runSummarize,
}

func init() {
	summarizeCmd.Flags().StringVar(&customPrompt, "prompt", "", "custom summarization prompt (overrides config default)")
	rootCmd.AddCommand(summarizeCmd)
}

func newSpinner(description string) *progressbar.ProgressBar {
	return progressbar.NewOptions(-1,
		progressbar.OptionSetDescription("  "+description),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionSetWidth(20),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[cyan]=[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)
}

func runSummarize(cmd *cobra.Command, args []string) error {
	cyan := color.New(color.FgCyan)
	green := color.New(color.FgGreen, color.Bold)

	// Step 1: Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Step 1.5: If no file given, show interactive selector
	var originalArg string
	if len(args) == 0 {
		selected, err := selectRecording(cfg.VideoDir)
		if err != nil {
			return err
		}
		originalArg = selected
	} else {
		originalArg = args[0]
	}

	inputFile := originalArg

	// Step 2: Resolve input file — try original, video_dir, and .zst variants.
	var decompressedTmp string
	if _, statErr := os.Stat(inputFile); os.IsNotExist(statErr) || compressor.IsCompressed(inputFile) {
		resolved, tmp, findErr := findInputFile(inputFile, cfg.VideoDir)
		if findErr != nil {
			return findErr
		}
		inputFile = resolved
		decompressedTmp = tmp
	}
	if decompressedTmp != "" {
		defer os.Remove(decompressedTmp)
	}

	// Derive basename from the original argument (strip .zst suffix if present).
	origBase := filepath.Base(originalArg)
	origBase = strings.TrimSuffix(origBase, ".zst")
	origExt := strings.ToLower(filepath.Ext(origBase))

	// Step 3: Determine file type
	audioFile := inputFile
	ext := strings.ToLower(filepath.Ext(inputFile))

	videoExts := map[string]bool{
		".mp4": true, ".mkv": true, ".webm": true, ".avi": true,
	}
	audioExts := map[string]bool{
		".wav": true, ".mp3": true, ".ogg": true, ".flac": true,
	}

	isVideo := videoExts[ext]
	isAudio := audioExts[ext]

	if !isVideo && !isAudio {
		return fmt.Errorf("unsupported file extension %q (expected video: .mp4/.mkv/.webm/.avi or audio: .wav/.mp3/.ogg/.flac)", ext)
	}

	// Step 4: Extract audio if video
	if isVideo {
		tmpDir := filepath.Join(os.TempDir(), "audio_from_video")
		if err := os.MkdirAll(tmpDir, 0o755); err != nil {
			return fmt.Errorf("create temp audio dir: %w", err)
		}
		basename := strings.TrimSuffix(filepath.Base(inputFile), ext)
		audioFile = filepath.Join(tmpDir, basename+".wav")

		cyan.Printf("● Extracting audio from video...\n")
		if err := transcriber.ExtractAudio(inputFile, audioFile); err != nil {
			return fmt.Errorf("extract audio: %w", err)
		}
		green.Printf("  ✓ Audio extracted\n")
	}

	// Step 5: Open cache
	cacheDBPath := filepath.Join(config.ConfigDir(), "cache.db")
	c, err := cache.NewCache(cacheDBPath)
	if err != nil {
		return fmt.Errorf("open cache: %w", err)
	}
	defer c.Close()

	// Step 6: Compute file hash
	fileHash, err := c.GetFileHash(audioFile)
	if err != nil {
		return fmt.Errorf("compute file hash: %w", err)
	}

	// Step 7 & 8: Check cache / transcribe
	var transcription string
	cached, err := c.Get(fileHash, cfg.WhisperModel)
	if err != nil {
		return fmt.Errorf("cache lookup: %w", err)
	}

	if cached != nil {
		green.Printf("  ✓ Using cached transcription\n")
		transcription = cached.Transcription
	} else {
		cyan.Printf("\n● Transcribing with Whisper (%s model)...\n", cfg.WhisperModel)

		bar := newSpinner("transcribing")
		scriptPath := findTranscribeScript()
		wt := transcriber.NewWhisperTranscriber(scriptPath, cfg.WhisperModel)

		type transcribeResult struct {
			result *transcriber.TranscriptionResult
			err    error
		}
		ch := make(chan transcribeResult, 1)
		go func() {
			r, e := wt.Transcribe(audioFile)
			ch <- transcribeResult{r, e}
		}()

	waitTranscribe:
		for {
			select {
			case res := <-ch:
				_ = bar.Finish()
				if res.err != nil {
					return fmt.Errorf("transcribe: %w", res.err)
				}
				transcription = res.result.Text
				green.Printf("  ✓ Transcribed (%s, %.0fs)\n", res.result.Language, res.result.Duration)
				if err := c.Set(fileHash, cfg.WhisperModel, res.result.Text, res.result.Language, res.result.Duration); err != nil {
					fmt.Fprintf(os.Stderr, "  warning: failed to cache transcription: %v\n", err)
				}
				break waitTranscribe
			default:
				_ = bar.Add(1)
				time.Sleep(80 * time.Millisecond)
			}
		}
	}

	// Step 9: Determine prompt
	prompt := cfg.DefaultPrompt
	if customPrompt != "" {
		prompt = customPrompt
	}

	// Step 10: Summarize
	var summaryText string
	if strings.TrimSpace(transcription) == "" {
		fmt.Println("  Transcription is empty, skipping summarization.")
		summaryText = "_Transcription is empty — no speech detected in recording._"
	} else if len(strings.Fields(transcription)) < 20 {
		fmt.Println("  Transcription is very short, skipping summarization.")
		summaryText = "**Transcription too short to analyze:**\n\n> " + strings.TrimSpace(transcription)
	} else {
		cyan.Printf("\n● Summarizing with Ollama (%s)...\n", cfg.APIModel)

		bar2 := newSpinner("generating summary")
		sum := summarizer.NewOllamaSummarizer(cfg.APIKey, cfg.APIBaseURL, cfg.APIModel)

		type sumResult struct {
			text string
			err  error
		}
		ch2 := make(chan sumResult, 1)
		go func() {
			t, e := sum.Summarize(transcription, prompt)
			ch2 <- sumResult{t, e}
		}()

	waitSummarize:
		for {
			select {
			case res := <-ch2:
				_ = bar2.Finish()
				if res.err != nil {
					return fmt.Errorf("summarize: %w", res.err)
				}
				summaryText = res.text
				green.Printf("  ✓ Summary generated\n")
				break waitSummarize
			default:
				_ = bar2.Add(1)
				time.Sleep(80 * time.Millisecond)
			}
		}
	}

	// Step 11: Save summary to transcribed_dir/<basename>.md
	basename := strings.TrimSuffix(origBase, origExt)
	summaryPath := filepath.Join(cfg.TranscribedDir, basename+".md")

	if err := os.WriteFile(summaryPath, []byte(summaryText), 0o644); err != nil {
		return fmt.Errorf("write summary: %w", err)
	}

	// Step 12: Compress original file
	if decompressedTmp == "" {
		cyan.Printf("\n● Compressing %s...\n", filepath.Base(inputFile))
		if _, err := compressor.Compress(inputFile); err != nil {
			fmt.Fprintf(os.Stderr, "  warning: failed to compress original file: %v\n", err)
		} else {
			green.Printf("  ✓ Compressed\n")
		}
	}

	// Step 13: Print result
	fmt.Printf("\n")
	green.Printf("✓ Summary saved to: %s\n", summaryPath)
	return nil
}

// selectRecording scans videoDir and shows an interactive list sorted newest → oldest.
func selectRecording(videoDir string) (string, error) {
	entries, err := os.ReadDir(videoDir)
	if err != nil {
		return "", fmt.Errorf("read video dir %q: %w", videoDir, err)
	}

	supportedExts := map[string]bool{
		".wav": true, ".mp3": true, ".ogg": true, ".flac": true,
		".mp4": true, ".mkv": true, ".webm": true, ".avi": true,
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		base := strings.TrimSuffix(name, ".zst")
		ext := strings.ToLower(filepath.Ext(base))
		if supportedExts[ext] {
			files = append(files, name)
		}
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no recording files found in %s", videoDir)
	}

	sort.Slice(files, func(i, j int) bool { return files[i] > files[j] })

	chosen, err := runInteractiveSelector(files)
	if err != nil {
		return "", err
	}
	return filepath.Join(videoDir, chosen), nil
}

// runInteractiveSelector shows an arrow-key driven menu in the terminal.
func runInteractiveSelector(items []string) (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return numberedSelector(items)
	}

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return numberedSelector(items)
	}
	defer term.Restore(fd, oldState)

	const maxVisible = 10
	cursor := 0
	var linesDrawn int

	redraw := func() {
		if linesDrawn > 0 {
			fmt.Printf("\033[%dA", linesDrawn)
		}
		fmt.Print("\033[J")

		fmt.Print("Select recording (\u2191\u2193 navigate, Enter select, q cancel):\r\n")
		linesDrawn = 1

		start := 0
		if cursor >= maxVisible {
			start = cursor - maxVisible + 1
		}
		end := start + maxVisible
		if end > len(items) {
			end = len(items)
		}

		for i := start; i < end; i++ {
			if i == cursor {
				fmt.Printf("  \033[32m\u25b6\033[0m \033[1m%s\033[0m\r\n", items[i])
			} else {
				fmt.Printf("    %s\r\n", items[i])
			}
			linesDrawn++
		}

		if len(items) > maxVisible {
			fmt.Printf("  \033[2m[%d / %d]\033[0m\r\n", cursor+1, len(items))
			linesDrawn++
		}
	}

	redraw()

	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return "", err
		}

		switch {
		case n == 1 && (buf[0] == 13 || buf[0] == 10): // Enter
			fmt.Print("\033[J\r\n")
			return items[cursor], nil

		case n == 1 && (buf[0] == 3 || buf[0] == 'q'): // Ctrl-C or q
			fmt.Print("\033[J\r\n")
			return "", fmt.Errorf("selection cancelled")

		case n == 3 && buf[0] == 27 && buf[1] == '[' && buf[2] == 'A': // arrow up
			if cursor > 0 {
				cursor--
			}

		case n == 3 && buf[0] == 27 && buf[1] == '[' && buf[2] == 'B': // arrow down
			if cursor < len(items)-1 {
				cursor++
			}
		}

		redraw()
	}
}

// numberedSelector is a fallback for non-interactive terminals.
func numberedSelector(items []string) (string, error) {
	fmt.Println("Available recordings:")
	for i, item := range items {
		fmt.Printf("  %2d. %s\n", i+1, item)
	}
	fmt.Print("Enter number: ")
	var n int
	if _, err := fmt.Scan(&n); err != nil || n < 1 || n > len(items) {
		return "", fmt.Errorf("invalid selection")
	}
	return items[n-1], nil
}

// findInputFile resolves the input file path, trying video_dir and .zst variants.
func findInputFile(inputFile, videoDir string) (resolved string, decompressedTmp string, err error) {
	candidates := []string{
		inputFile,
		filepath.Join(videoDir, filepath.Base(inputFile)),
		inputFile + ".zst",
		filepath.Join(videoDir, filepath.Base(inputFile)) + ".zst",
	}

	for _, c := range candidates {
		if _, statErr := os.Stat(c); statErr != nil {
			continue
		}
		if compressor.IsCompressed(c) {
			fmt.Printf("Found compressed file %s, decompressing...\n", c)
			tmp, decompErr := compressor.Decompress(c)
			if decompErr != nil {
				return "", "", fmt.Errorf("decompress %s: %w", c, decompErr)
			}
			return tmp, tmp, nil
		}
		return c, "", nil
	}

	return "", "", fmt.Errorf("input file does not exist: %s", inputFile)
}

// findTranscribeScript tries to locate transcribe.py relative to common locations.
func findTranscribeScript() string {
	exe, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "scripts", "transcribe.py")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	wd, err := os.Getwd()
	if err == nil {
		candidate := filepath.Join(wd, "scripts", "transcribe.py")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return "scripts/transcribe.py"
}
