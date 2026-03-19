package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadNoFile(t *testing.T) {
	// Load from a directory with no config files — should return defaults
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	clearNuntiusEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := DefaultConfig()
	if cfg.AI.Provider != expected.AI.Provider {
		t.Errorf("expected provider %q, got %q", expected.AI.Provider, cfg.AI.Provider)
	}
	if cfg.Behavior.AutoCommit != expected.Behavior.AutoCommit {
		t.Errorf("expected auto_commit %v, got %v", expected.Behavior.AutoCommit, cfg.Behavior.AutoCommit)
	}
}

func TestLoadRepoFile(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	clearNuntiusEnv(t)

	tomlContent := `
[ai]
provider = "gemini"
model = "gemini-2.0-flash"

[behavior]
auto_commit = true

[conventions]
style = "gitmoji"
`
	if err := os.WriteFile(filepath.Join(dir, ".nuntius.toml"), []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.AI.Provider != "gemini" {
		t.Errorf("expected provider %q, got %q", "gemini", cfg.AI.Provider)
	}
	if cfg.AI.Model != "gemini-2.0-flash" {
		t.Errorf("expected model %q, got %q", "gemini-2.0-flash", cfg.AI.Model)
	}
	if !cfg.Behavior.AutoCommit {
		t.Error("expected auto_commit = true")
	}
	if cfg.Conventions.Style != "gitmoji" {
		t.Errorf("expected style %q, got %q", "gitmoji", cfg.Conventions.Style)
	}
	// Non-specified fields should retain defaults
	if cfg.AI.APIKeyEnv != "ANTHROPIC_API_KEY" {
		t.Errorf("expected default api_key_env, got %q", cfg.AI.APIKeyEnv)
	}
}

func TestLoadGlobalFile(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	clearNuntiusEnv(t)

	// Create a fake global config
	globalDir := filepath.Join(dir, "fakeconfig", "nuntius")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}
	globalFile := filepath.Join(globalDir, "config.toml")
	tomlContent := `
[ai]
provider = "ollama"
`
	if err := os.WriteFile(globalFile, []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}

	// We can't easily test global config path without mocking os.UserConfigDir,
	// so instead we test loadTOML directly
	cfg := DefaultConfig()
	err := loadTOML(globalFile, &cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AI.Provider != "ollama" {
		t.Errorf("expected provider %q, got %q", "ollama", cfg.AI.Provider)
	}
}

func TestLoadEnvOverride(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	clearNuntiusEnv(t)

	// Write a repo config
	tomlContent := `
[ai]
provider = "claude"
`
	if err := os.WriteFile(filepath.Join(dir, ".nuntius.toml"), []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Set env var to override
	t.Setenv("NUNTIUS_AI_PROVIDER", "codex")
	t.Setenv("NUNTIUS_BEHAVIOR_AUTO_COMMIT", "true")
	t.Setenv("NUNTIUS_CONVENTIONS_STYLE", "angular")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.AI.Provider != "codex" {
		t.Errorf("expected provider %q (env override), got %q", "codex", cfg.AI.Provider)
	}
	if !cfg.Behavior.AutoCommit {
		t.Error("expected auto_commit = true from env override")
	}
	if cfg.Conventions.Style != "angular" {
		t.Errorf("expected style %q from env override, got %q", "angular", cfg.Conventions.Style)
	}
}

func TestLoadInvalidTOML(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	clearNuntiusEnv(t)

	if err := os.WriteFile(filepath.Join(dir, ".nuntius.toml"), []byte("{{invalid toml content}}"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid TOML, got nil")
	}
}

func TestMergeFlags(t *testing.T) {
	cfg := DefaultConfig()

	autoCommit := true
	MergeFlags(&cfg, FlagOverrides{
		Provider:   "gemini",
		Model:      "gemini-pro",
		AutoCommit: &autoCommit,
	})

	if cfg.AI.Provider != "gemini" {
		t.Errorf("expected provider %q, got %q", "gemini", cfg.AI.Provider)
	}
	if cfg.AI.Model != "gemini-pro" {
		t.Errorf("expected model %q, got %q", "gemini-pro", cfg.AI.Model)
	}
	if !cfg.Behavior.AutoCommit {
		t.Error("expected auto_commit = true after flag merge")
	}
	// AutoPush should remain default
	if cfg.Behavior.AutoPush {
		t.Error("expected auto_push to remain false")
	}
}

// clearNuntiusEnv unsets all NUNTIUS_ environment variables to prevent test pollution.
func clearNuntiusEnv(t *testing.T) {
	t.Helper()
	envVars := []string{
		"NUNTIUS_AI_PROVIDER",
		"NUNTIUS_AI_MODEL",
		"NUNTIUS_AI_API_KEY_ENV",
		"NUNTIUS_AI_OLLAMA_URL",
		"NUNTIUS_AI_CLI_COMMAND",
		"NUNTIUS_AI_CLI_ARGS",
		"NUNTIUS_BEHAVIOR_AUTO_COMMIT",
		"NUNTIUS_BEHAVIOR_AUTO_PUSH",
		"NUNTIUS_BEHAVIOR_FORCE_PUSH",
		"NUNTIUS_CONVENTIONS_STYLE",
		"NUNTIUS_CONVENTIONS_CUSTOM_TEMPLATE",
	}
	for _, env := range envVars {
		t.Setenv(env, "")
		_ = os.Unsetenv(env)
	}
}
