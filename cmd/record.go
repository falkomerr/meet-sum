package cmd

import (
	"bufio"
	"fmt"
	"os"

	"github.com/falkomer/meet-summarize/internal/config"
	"github.com/falkomer/meet-summarize/internal/recorder"
	"github.com/spf13/cobra"
)

var recordCmd = &cobra.Command{
	Use:   "record",
	Short: "Start audio recording",
	Long:  "Record system audio and microphone to a WAV file in the configured video directory.",
	RunE:  runRecord,
}

func init() {
	rootCmd.AddCommand(recordCmd)
}

func runRecord(cmd *cobra.Command, args []string) error {
	// Step 1: Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Step 2: Create recorder
	rec := recorder.NewRecorder(cfg.VideoDir)

	// Step 3: Start recording
	filePath, err := rec.Start()
	if err != nil {
		return fmt.Errorf("start recording: %w", err)
	}

	// Step 4: Wait for Enter
	fmt.Println("Recording... Press Enter to stop.")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()

	// Step 5: Stop recording
	if err := rec.Stop(); err != nil {
		return fmt.Errorf("stop recording: %w", err)
	}

	// Step 6: Report result
	fmt.Printf("Recording saved to: %s\n", filePath)
	return nil
}
