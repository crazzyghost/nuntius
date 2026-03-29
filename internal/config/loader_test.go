package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadNoFile(t *testing.T) {
	// Load from a directory with no config files — should return defaults.
	// Set HOME to a temp dir to avoid picking up the real global config.
	home := t.TempDir()
	t.Setenv("HOME", home)

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
	home := t.TempDir()
	t.Setenv("HOME", home)

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
	home := t.TempDir()
	t.Setenv("HOME", home)

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
	home := t.TempDir()
	t.Setenv("HOME", home)

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
		"NUNTIUS_AI_MODE",
		"NUNTIUS_AI_MODEL",
		"NUNTIUS_AI_OLLAMA_URL",
		"NUNTIUS_AI_CLI_COMMAND",
		"NUNTIUS_AI_CLI_ARGS",
		"NUNTIUS_BEHAVIOR_AUTO_COMMIT",
		"NUNTIUS_BEHAVIOR_AUTO_PUSH",
		"NUNTIUS_BEHAVIOR_FORCE_PUSH",
		"NUNTIUS_BEHAVIOR_AUTO_UPDATE_CHECK",
		"NUNTIUS_CONVENTIONS_STYLE",
		"NUNTIUS_CONVENTIONS_CUSTOM_TEMPLATE",
	}
	for _, env := range envVars {
		t.Setenv(env, "")
		_ = os.Unsetenv(env)
	}
}

func TestNuntiusDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	got := NuntiusDir()
	want := filepath.Join(dir, ".nuntius")
	if got != want {
		t.Errorf("NuntiusDir() = %q, want %q", got, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Errorf("NuntiusDir() should create the directory: %v", err)
	}
}

func TestNuntiusDirIdempotent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	want := filepath.Join(dir, ".nuntius")
	if got := NuntiusDir(); got != want {
		t.Errorf("first call: NuntiusDir() = %q, want %q", got, want)
	}
	if got := NuntiusDir(); got != want {
		t.Errorf("second call: NuntiusDir() = %q, want %q", got, want)
	}
}

func TestGlobalConfigPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got := globalConfigPath()
	want := filepath.Join(home, ".nuntius", "config.toml")
	if got != want {
		t.Errorf("globalConfigPath() = %q, want %q", got, want)
	}
}

func TestAutoUpdateCheckTOML(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	clearNuntiusEnv(t)

	content := "[behavior]\nauto_update_check = false\n"
	if err := os.WriteFile(filepath.Join(dir, ".nuntius.toml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Behavior.AutoUpdateCheck {
		t.Error("expected AutoUpdateCheck = false from TOML")
	}
}

func TestAutoUpdateCheckEnvVar(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	clearNuntiusEnv(t)
	t.Setenv("NUNTIUS_BEHAVIOR_AUTO_UPDATE_CHECK", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Behavior.AutoUpdateCheck {
		t.Error("expected AutoUpdateCheck = false from env var")
	}
}

// TestLoadLayeredConfig verifies that repo config overlays on top of global config.
func TestLoadLayeredConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	clearNuntiusEnv(t)

	// Write global config with provider, model, and auto_commit
	globalDir := filepath.Join(home, ".nuntius")
	if err := os.MkdirAll(globalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	globalContent := "[ai]\nprovider = \"claude\"\nmodel = \"claude-haiku-4.5\"\n\n[behavior]\nauto_commit = true\n"
	if err := os.WriteFile(filepath.Join(globalDir, "config.toml"), []byte(globalContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write repo config — only overrides provider
	repoContent := "[ai]\nprovider = \"ollama\"\n"
	if err := os.WriteFile(filepath.Join(repo, ".nuntius.toml"), []byte(repoContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Repo overrides provider
	if cfg.AI.Provider != "ollama" {
		t.Errorf("expected provider 'ollama' (from repo), got %q", cfg.AI.Provider)
	}
	// Global model is preserved (not mentioned in repo config)
	if cfg.AI.Model != "claude-haiku-4.5" {
		t.Errorf("expected model 'claude-haiku-4.5' (from global, preserved), got %q", cfg.AI.Model)
	}
	// Global auto_commit is preserved (not mentioned in repo config)
	if !cfg.Behavior.AutoCommit {
		t.Error("expected auto_commit=true (from global, preserved by repo overlay)")
	}
}

// TestLoadOnlyGlobalConfig verifies that global config is applied when no repo config exists.
func TestLoadOnlyGlobalConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	clearNuntiusEnv(t)

	globalDir := filepath.Join(home, ".nuntius")
	if err := os.MkdirAll(globalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	globalContent := "[ai]\nprovider = \"gemini\"\n\n[behavior]\nauto_push = true\n"
	if err := os.WriteFile(filepath.Join(globalDir, "config.toml"), []byte(globalContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.AI.Provider != "gemini" {
		t.Errorf("expected provider 'gemini' from global, got %q", cfg.AI.Provider)
	}
	if !cfg.Behavior.AutoPush {
		t.Error("expected auto_push=true from global config")
	}
}

// TestLoadNoMigrationFromOldPath verifies that having a config at the legacy
// ~/.config/nuntius/ path does NOT auto-migrate or get loaded.
func TestLoadNoMigrationFromOldPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	repo := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	clearNuntiusEnv(t)

	// Write config at the OLD XDG path only (not at ~/.nuntius/)
	oldConfigDir := filepath.Join(home, ".config", "nuntius")
	if err := os.MkdirAll(oldConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldConfigDir, "config.toml"), []byte("[ai]\nprovider = \"gemini\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defaults := DefaultConfig()
	// Old path should NOT be read — expect default provider
	if cfg.AI.Provider != defaults.AI.Provider {
		t.Errorf("expected default provider (no migration from old path), got %q", cfg.AI.Provider)
	}
	// New path should NOT have been created by migration
	newConfig := filepath.Join(home, ".nuntius", "config.toml")
	if fileExists(newConfig) {
		t.Error("Load() should not auto-migrate from the legacy config path")
	}
}

func TestLoadEnvOverrideMode(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	clearNuntiusEnv(t)
	t.Setenv("NUNTIUS_AI_MODE", "api")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AI.Mode != "api" {
		t.Errorf("expected mode %q from env, got %q", "api", cfg.AI.Mode)
	}
}

func TestMergeFlagsMode(t *testing.T) {
	cfg := DefaultConfig()
	MergeFlags(&cfg, FlagOverrides{Mode: "api"})
	if cfg.AI.Mode != "api" {
		t.Errorf("expected mode %q after flag merge, got %q", "api", cfg.AI.Mode)
	}
}
