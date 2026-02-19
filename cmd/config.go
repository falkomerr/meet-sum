package cmd

import (
	"fmt"
	"strings"

	"github.com/falkomer/meet-summarize/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  "View and update meet-sum configuration.",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print current configuration",
	Long:  "Print the current configuration as YAML.",
	RunE:  runConfigShow,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Update a configuration value",
	Long: `Update a configuration value by key.

Supported keys:
  whisper-model      Whisper model name (tiny/base/small/medium/large/turbo)
  api-key            API key (not required for local Ollama)
  api-model          Ollama model name (e.g. qwen2.5)
  video-dir          Directory for video recordings
  transcribed-dir    Directory for transcription and summary output`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Mask API key for display.
	display := *cfg
	if len(display.APIKey) > 4 {
		display.APIKey = display.APIKey[:4] + strings.Repeat("*", len(display.APIKey)-4)
	}

	data, err := yaml.Marshal(&display)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	fmt.Printf("# Configuration file: %s\n", config.ConfigPath())
	fmt.Print(string(data))
	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	switch key {
	case "whisper-model":
		cfg.WhisperModel = value
	case "api-key":
		cfg.APIKey = value
	case "api-model":
		cfg.APIModel = value
	case "video-dir":
		cfg.VideoDir = value
	case "transcribed-dir":
		cfg.TranscribedDir = value
	default:
		return fmt.Errorf("unknown config key %q; supported keys: whisper-model, api-key, api-model, video-dir, transcribed-dir", key)
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Printf("Config updated: %s = %s\n", key, value)
	return nil
}
