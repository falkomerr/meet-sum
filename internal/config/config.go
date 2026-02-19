package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const (
	defaultWhisperModel  = "base"
	defaultAPIBaseURL    = "http://localhost:11434/v1"
	defaultAPIModel      = "mistral:7b"
	defaultVideoDir      = "./video_dist"
	defaultTranscribedDir = "./transcribed_dist"
)

var defaultPrompt = `You are a transcription summarizer. CRITICAL RULES:
1. Output ONLY in Russian language. NEVER use Chinese or any other language.
2. Write plain text only - no headers, no lists, no markdown.
3. Only include what was explicitly said in the transcript. Do not invent anything.
4. Be brief and accurate.
5. If names are mentioned, start with them.`

// Config holds all application configuration.
type Config struct {
	WhisperModel    string `mapstructure:"whisper_model"    yaml:"whisper_model"`
	APIKey          string `mapstructure:"api_key"          yaml:"api_key"`
	APIBaseURL      string `mapstructure:"api_base_url"     yaml:"api_base_url"`
	APIModel        string `mapstructure:"api_model"        yaml:"api_model"`
	VideoDir        string `mapstructure:"video_dir"        yaml:"video_dir"`
	TranscribedDir  string `mapstructure:"transcribed_dir"  yaml:"transcribed_dir"`
	DefaultPrompt   string `mapstructure:"default_prompt"   yaml:"default_prompt"`
}

// ConfigDir returns the directory where config is stored: ~/.config/meet-sum/
func ConfigDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		// fallback to ~/.config
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "meet-sum")
}

// ConfigPath returns the full path to the config file.
func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

// Load reads configuration from disk, creating defaults if the file does not exist.
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults.
	v.SetDefault("whisper_model", defaultWhisperModel)
	v.SetDefault("api_key", "")
	v.SetDefault("api_base_url", defaultAPIBaseURL)
	v.SetDefault("api_model", defaultAPIModel)
	v.SetDefault("video_dir", defaultVideoDir)
	v.SetDefault("transcribed_dir", defaultTranscribedDir)
	v.SetDefault("default_prompt", defaultPrompt)

	configDir := ConfigDir()
	configPath := ConfigPath()

	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// If the file doesn't exist yet, create it with defaults.
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0o755); err != nil {
			return nil, fmt.Errorf("create config dir: %w", err)
		}
		cfg := defaultConfig()
		if err := saveWithViper(v, cfg, configPath); err != nil {
			return nil, fmt.Errorf("write default config: %w", err)
		}
		if err := EnsureDirs(cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.DefaultPrompt == "" {
		cfg.DefaultPrompt = defaultPrompt
	}

	if err := EnsureDirs(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save writes cfg to the config file.
func Save(cfg *Config) error {
	v := viper.New()
	v.SetConfigFile(ConfigPath())
	v.SetConfigType("yaml")

	if err := os.MkdirAll(ConfigDir(), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	return saveWithViper(v, cfg, ConfigPath())
}

// EnsureDirs creates video_dir and transcribed_dir if they don't exist.
func EnsureDirs(cfg *Config) error {
	for _, dir := range []string{cfg.VideoDir, cfg.TranscribedDir} {
		if dir == "" {
			continue
		}
		expanded := expandHome(dir)
		if err := os.MkdirAll(expanded, 0o755); err != nil {
			return fmt.Errorf("create directory %q: %w", expanded, err)
		}
	}
	return nil
}

// defaultConfig returns a Config populated with defaults.
func defaultConfig() *Config {
	return &Config{
		WhisperModel:   defaultWhisperModel,
		APIKey:         "",
		APIBaseURL:     defaultAPIBaseURL,
		APIModel:       defaultAPIModel,
		VideoDir:       defaultVideoDir,
		TranscribedDir: defaultTranscribedDir,
		DefaultPrompt:  defaultPrompt,
	}
}

// saveWithViper populates viper with cfg values and writes to path.
func saveWithViper(v *viper.Viper, cfg *Config, path string) error {
	v.Set("whisper_model", cfg.WhisperModel)
	v.Set("api_key", cfg.APIKey)
	v.Set("api_base_url", cfg.APIBaseURL)
	v.Set("api_model", cfg.APIModel)
	v.Set("video_dir", cfg.VideoDir)
	v.Set("transcribed_dir", cfg.TranscribedDir)
	v.Set("default_prompt", cfg.DefaultPrompt)

	if err := v.WriteConfigAs(path); err != nil {
		return fmt.Errorf("write config to %s: %w", path, err)
	}
	return nil
}

// expandHome replaces a leading "~" with the user's home directory.
func expandHome(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[1:])
		}
	}
	return path
}
