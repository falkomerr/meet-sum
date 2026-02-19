package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setHomeEnv redirects all home/config env vars to tmp so that
// os.UserConfigDir() and os.UserHomeDir() resolve inside the temp dir.
func setHomeEnv(t *testing.T, tmp string) {
	t.Helper()
	t.Setenv("HOME", tmp)
	// macOS: UserConfigDir uses $HOME/Library/Application Support
	// Linux: UserConfigDir uses $XDG_CONFIG_HOME, falling back to $HOME/.config
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))
	// Windows: UserConfigDir uses %AppData%
	t.Setenv("AppData", filepath.Join(tmp, "AppData", "Roaming"))
	// Also clear USERPROFILE (Windows fallback for UserHomeDir)
	t.Setenv("USERPROFILE", tmp)
}

// TestConfigDir verifies ConfigDir returns a non-empty path ending with "meet-sum".
func TestConfigDir(t *testing.T) {
	tmp := t.TempDir()
	setHomeEnv(t, tmp)

	dir := ConfigDir()

	if dir == "" {
		t.Fatal("ConfigDir() returned empty string")
	}
	if !strings.HasSuffix(dir, "meet-sum") {
		t.Errorf("ConfigDir() = %q, want path ending in \"meet-sum\"", dir)
	}
}

// TestConfigPath verifies ConfigPath is ConfigDir()+"/config.yaml".
func TestConfigPath(t *testing.T) {
	tmp := t.TempDir()
	setHomeEnv(t, tmp)

	dir := ConfigDir()
	path := ConfigPath()

	want := filepath.Join(dir, "config.yaml")
	if path != want {
		t.Errorf("ConfigPath() = %q, want %q", path, want)
	}
	if !strings.HasSuffix(path, "config.yaml") {
		t.Errorf("ConfigPath() = %q does not end with config.yaml", path)
	}
}

// TestEnsureDirs verifies that EnsureDirs creates VideoDir and TranscribedDir.
func TestEnsureDirs(t *testing.T) {
	tmp := t.TempDir()

	videoDir := filepath.Join(tmp, "videos")
	transcribedDir := filepath.Join(tmp, "transcribed")

	cfg := &Config{
		VideoDir:       videoDir,
		TranscribedDir: transcribedDir,
	}

	if err := EnsureDirs(cfg); err != nil {
		t.Fatalf("EnsureDirs() error = %v", err)
	}

	for _, dir := range []string{videoDir, transcribedDir} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("expected directory %q to exist, got error: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%q exists but is not a directory", dir)
		}
	}
}

// TestEnsureDirs_EmptyDirsSkipped verifies that empty dir fields are skipped without error.
func TestEnsureDirs_EmptyDirsSkipped(t *testing.T) {
	cfg := &Config{
		VideoDir:       "",
		TranscribedDir: "",
	}
	if err := EnsureDirs(cfg); err != nil {
		t.Fatalf("EnsureDirs() with empty dirs error = %v", err)
	}
}

// TestEnsureDirs_Idempotent verifies EnsureDirs can be called multiple times safely.
func TestEnsureDirs_Idempotent(t *testing.T) {
	tmp := t.TempDir()
	cfg := &Config{
		VideoDir:       filepath.Join(tmp, "vids"),
		TranscribedDir: filepath.Join(tmp, "trans"),
	}

	for i := 0; i < 3; i++ {
		if err := EnsureDirs(cfg); err != nil {
			t.Fatalf("EnsureDirs() call %d error = %v", i+1, err)
		}
	}
}

// TestLoad_DefaultsWhenNoFile verifies Load creates defaults when no config file exists.
func TestLoad_DefaultsWhenNoFile(t *testing.T) {
	tmp := t.TempDir()
	setHomeEnv(t, tmp)

	// Change cwd so relative VideoDir/TranscribedDir are created in a writable place.
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.WhisperModel != defaultWhisperModel {
		t.Errorf("WhisperModel = %q, want %q", cfg.WhisperModel, defaultWhisperModel)
	}
	if cfg.APIBaseURL != defaultAPIBaseURL {
		t.Errorf("APIBaseURL = %q, want %q", cfg.APIBaseURL, defaultAPIBaseURL)
	}
	if cfg.APIModel != defaultAPIModel {
		t.Errorf("APIModel = %q, want %q", cfg.APIModel, defaultAPIModel)
	}
	if cfg.VideoDir != defaultVideoDir {
		t.Errorf("VideoDir = %q, want %q", cfg.VideoDir, defaultVideoDir)
	}
	if cfg.TranscribedDir != defaultTranscribedDir {
		t.Errorf("TranscribedDir = %q, want %q", cfg.TranscribedDir, defaultTranscribedDir)
	}
	if cfg.DefaultPrompt == "" {
		t.Error("DefaultPrompt should not be empty")
	}
	// APIKey should be empty by default
	if cfg.APIKey != "" {
		t.Errorf("APIKey = %q, want empty string", cfg.APIKey)
	}
}

// TestLoad_CreatesConfigFileWhenMissing verifies Load writes a config file to disk.
func TestLoad_CreatesConfigFileWhenMissing(t *testing.T) {
	tmp := t.TempDir()
	setHomeEnv(t, tmp)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	_, err = Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	configPath := ConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("expected config file to be created at %q, but it does not exist", configPath)
	}
}

// TestSaveLoad_Roundtrip verifies that Save followed by Load returns the same config.
func TestSaveLoad_Roundtrip(t *testing.T) {
	tmp := t.TempDir()
	setHomeEnv(t, tmp)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	videoDir := filepath.Join(tmp, "my_videos")
	transcribedDir := filepath.Join(tmp, "my_transcribed")

	original := &Config{
		WhisperModel:   "large",
		APIKey:         "sk-test-12345",
		APIBaseURL:     "https://custom.api.example.com/v1/",
		APIModel:       "gpt-4o",
		VideoDir:       videoDir,
		TranscribedDir: transcribedDir,
		DefaultPrompt:  "Summarize this meeting briefly.",
	}

	// Create dirs so EnsureDirs (called by Load) succeeds
	if err := os.MkdirAll(videoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(transcribedDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := Save(original); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() after Save() error = %v", err)
	}

	if loaded.WhisperModel != original.WhisperModel {
		t.Errorf("WhisperModel: got %q, want %q", loaded.WhisperModel, original.WhisperModel)
	}
	if loaded.APIKey != original.APIKey {
		t.Errorf("APIKey: got %q, want %q", loaded.APIKey, original.APIKey)
	}
	if loaded.APIBaseURL != original.APIBaseURL {
		t.Errorf("APIBaseURL: got %q, want %q", loaded.APIBaseURL, original.APIBaseURL)
	}
	if loaded.APIModel != original.APIModel {
		t.Errorf("APIModel: got %q, want %q", loaded.APIModel, original.APIModel)
	}
	if loaded.VideoDir != original.VideoDir {
		t.Errorf("VideoDir: got %q, want %q", loaded.VideoDir, original.VideoDir)
	}
	if loaded.TranscribedDir != original.TranscribedDir {
		t.Errorf("TranscribedDir: got %q, want %q", loaded.TranscribedDir, original.TranscribedDir)
	}
	if loaded.DefaultPrompt != original.DefaultPrompt {
		t.Errorf("DefaultPrompt: got %q, want %q", loaded.DefaultPrompt, original.DefaultPrompt)
	}
}

// TestSave_CreatesConfigDir verifies Save creates the config directory if it does not exist.
func TestSave_CreatesConfigDir(t *testing.T) {
	tmp := t.TempDir()
	setHomeEnv(t, tmp)

	cfg := &Config{
		WhisperModel:   "small",
		APIKey:         "key",
		APIBaseURL:     "https://api.example.com/",
		APIModel:       "gpt-3.5",
		VideoDir:       filepath.Join(tmp, "v"),
		TranscribedDir: filepath.Join(tmp, "t"),
		DefaultPrompt:  "prompt",
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	configDir := ConfigDir()
	info, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("expected config dir %q to exist: %v", configDir, err)
	}
	if !info.IsDir() {
		t.Errorf("%q is not a directory", configDir)
	}

	configPath := ConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("expected config file %q to exist after Save()", configPath)
	}
}

// TestLoad_ReadsExistingFile verifies Load reads values from an existing config file.
func TestLoad_ReadsExistingFile(t *testing.T) {
	tmp := t.TempDir()
	setHomeEnv(t, tmp)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	// Write a config file manually before calling Load.
	configDir := ConfigDir()
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}

	videoDir := filepath.Join(tmp, "existing_videos")
	transcribedDir := filepath.Join(tmp, "existing_transcribed")
	if err := os.MkdirAll(videoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(transcribedDir, 0o755); err != nil {
		t.Fatal(err)
	}

	yamlContent := "whisper_model: medium\napi_key: mykey\napi_base_url: https://test.api/\napi_model: mymodel\nvideo_dir: " +
		videoDir + "\ntranscribed_dir: " + transcribedDir + "\ndefault_prompt: test prompt\n"

	configPath := ConfigPath()
	if err := os.WriteFile(configPath, []byte(yamlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.WhisperModel != "medium" {
		t.Errorf("WhisperModel = %q, want \"medium\"", cfg.WhisperModel)
	}
	if cfg.APIKey != "mykey" {
		t.Errorf("APIKey = %q, want \"mykey\"", cfg.APIKey)
	}
	if cfg.APIBaseURL != "https://test.api/" {
		t.Errorf("APIBaseURL = %q, want \"https://test.api/\"", cfg.APIBaseURL)
	}
	if cfg.APIModel != "mymodel" {
		t.Errorf("APIModel = %q, want \"mymodel\"", cfg.APIModel)
	}
	if cfg.DefaultPrompt != "test prompt" {
		t.Errorf("DefaultPrompt = %q, want \"test prompt\"", cfg.DefaultPrompt)
	}
}
