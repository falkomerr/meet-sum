package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/falkomer/meet-summarize/internal/config"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List recorded and summarized files",
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

// videoExtensions are the extensions recognized as recordings.
var videoExtensions = map[string]bool{
	".wav":  true,
	".mp3":  true,
	".mp4":  true,
	".mkv":  true,
	".webm": true,
	".avi":  true,
	".ogg":  true,
	".flac": true,
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	videoDir := listExpandHome(cfg.VideoDir)
	transcribedDir := listExpandHome(cfg.TranscribedDir)

	fmt.Printf("=== Recordings (%s) ===\n", cfg.VideoDir)
	if err := printDirFiles(videoDir, isVideoFile); err != nil {
		return err
	}

	fmt.Printf("\n=== Summaries (%s) ===\n", cfg.TranscribedDir)
	if err := printDirFiles(transcribedDir, isSummaryFile); err != nil {
		return err
	}

	return nil
}

func printDirFiles(dir string, include func(name string) bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("  (no files)")
			return nil
		}
		return fmt.Errorf("read directory %q: %w", dir, err)
	}

	printed := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !include(name) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		compressed := strings.HasSuffix(strings.ToLower(name), ".zst")
		if compressed {
			fmt.Printf("  %-52s (%s compressed)\n", name, formatSize(info.Size()))
		} else {
			fmt.Printf("  %-52s (%s)\n", name, formatSize(info.Size()))
		}
		printed++
	}

	if printed == 0 {
		fmt.Println("  (no files)")
	}
	return nil
}

// isVideoFile returns true for audio/video files and their .zst compressed versions.
func isVideoFile(name string) bool {
	lower := strings.ToLower(name)
	// .zst compressed: strip the .zst suffix and check the inner extension
	if strings.HasSuffix(lower, ".zst") {
		inner := strings.TrimSuffix(lower, ".zst")
		return videoExtensions[filepath.Ext(inner)]
	}
	return videoExtensions[filepath.Ext(lower)]
}

// isSummaryFile returns true for .md files and .zst compressed files in the transcribed dir.
func isSummaryFile(name string) bool {
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".zst") {
		return true
	}
	return filepath.Ext(lower) == ".md"
}

// formatSize returns a human-readable file size string.
func formatSize(bytes int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// listExpandHome replaces a leading "~" with the user's home directory.
func listExpandHome(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[1:])
		}
	}
	return path
}
