package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

// NuntiusDir returns the path to the ~/.nuntius/ directory, creating it if absent.
// Returns empty string when $HOME cannot be determined (containers, CI).
func NuntiusDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	dir := filepath.Join(home, ".nuntius")
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

// Load reads configuration from TOML files and environment variables,
// merging them with the following priority (highest first):
//
// CLI flags (handled externally via MergeFlags) → env vars → repo .nuntius.toml → global ~/.nuntius/config.toml → defaults
//
// Config files are layered: global config is loaded first, then repo config overlays
// specific fields on top. This means a repo config that only sets [ai] provider
// inherits all other settings (auto_commit, model, etc.) from the global config.
func Load() (Config, error) {
	cfg := DefaultConfig()

	repoConfig := ".nuntius.toml"
	globalConfig := globalConfigPath()

	// Layer 1: Global config (lower precedence)
	if globalConfig != "" && fileExists(globalConfig) {
		if err := loadTOML(globalConfig, &cfg); err != nil {
			return Config{}, fmt.Errorf("loading global config %s: %w", globalConfig, err)
		}
	}

	// Layer 2: Repo config overlays on top (higher precedence)
	if fileExists(repoConfig) {
		if err := loadTOML(repoConfig, &cfg); err != nil {
			return Config{}, fmt.Errorf("loading repo config %s: %w", repoConfig, err)
		}
	}

	// Layer 3: Environment variables (highest file-level precedence)
	applyEnvOverrides(&cfg)

	return cfg, nil
}

// loadTOML decodes a TOML file into the config struct.
func loadTOML(path string, cfg *Config) error {
	_, err := toml.DecodeFile(path, cfg)
	if err != nil {
		return fmt.Errorf("invalid TOML in %s: %w", path, err)
	}
	return nil
}

// globalConfigPath returns the path to the global config file (~/.nuntius/config.toml),
// or an empty string if the home directory cannot be determined.
func globalConfigPath() string {
	dir := NuntiusDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "config.toml")
}

// applyEnvOverrides reads NUNTIUS_* environment variables and overrides
// the corresponding config fields. Env var naming convention:
// NUNTIUS_<SECTION>_<KEY> in SCREAMING_SNAKE_CASE.
func applyEnvOverrides(cfg *Config) {
	// AI section
	if v := os.Getenv("NUNTIUS_AI_PROVIDER"); v != "" {
		cfg.AI.Provider = v
	}
	if v := os.Getenv("NUNTIUS_AI_MODE"); v != "" {
		cfg.AI.Mode = v
	}
	if v := os.Getenv("NUNTIUS_AI_MODEL"); v != "" {
		cfg.AI.Model = v
	}
	if v := os.Getenv("NUNTIUS_AI_API_KEY_ENV"); v != "" {
		cfg.AI.APIKeyEnv = v
	}
	if v := os.Getenv("NUNTIUS_AI_OLLAMA_URL"); v != "" {
		cfg.AI.OllamaURL = v
	}
	if v := os.Getenv("NUNTIUS_AI_CLI_COMMAND"); v != "" {
		cfg.AI.CLICommand = v
	}
	if v := os.Getenv("NUNTIUS_AI_CLI_ARGS"); v != "" {
		cfg.AI.CLIArgs = strings.Split(v, ",")
	}

	// Behavior section
	if v := os.Getenv("NUNTIUS_BEHAVIOR_AUTO_COMMIT"); v != "" {
		cfg.Behavior.AutoCommit = parseBool(v)
	}
	if v := os.Getenv("NUNTIUS_BEHAVIOR_AUTO_PUSH"); v != "" {
		cfg.Behavior.AutoPush = parseBool(v)
	}
	if v := os.Getenv("NUNTIUS_BEHAVIOR_FORCE_PUSH"); v != "" {
		cfg.Behavior.ForcePush = parseBool(v)
	}
	if v := os.Getenv("NUNTIUS_BEHAVIOR_AUTO_UPDATE_CHECK"); v != "" {
		cfg.Behavior.AutoUpdateCheck = parseBool(v)
	}

	// Conventions section
	if v := os.Getenv("NUNTIUS_CONVENTIONS_STYLE"); v != "" {
		cfg.Conventions.Style = v
	}
	if v := os.Getenv("NUNTIUS_CONVENTIONS_CUSTOM_TEMPLATE"); v != "" {
		cfg.Conventions.CustomTemplate = v
	}
}

// parseBool parses a boolean string, returning false for unrecognized values.
func parseBool(s string) bool {
	b, err := strconv.ParseBool(s)
	if err != nil {
		return false
	}
	return b
}

// fileExists returns true if the given path exists and is a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// MergeFlags applies CLI flag overrides to the config.
// Only non-zero-value flags are merged.
type FlagOverrides struct {
	Provider   string
	Mode       string
	Model      string
	AutoCommit *bool
	AutoPush   *bool
}

// MergeFlags merges CLI flag overrides into the config.
func MergeFlags(cfg *Config, flags FlagOverrides) {
	if flags.Provider != "" {
		cfg.AI.Provider = flags.Provider
	}
	if flags.Mode != "" {
		cfg.AI.Mode = flags.Mode
	}
	if flags.Model != "" {
		cfg.AI.Model = flags.Model
	}
	if flags.AutoCommit != nil {
		cfg.Behavior.AutoCommit = *flags.AutoCommit
	}
	if flags.AutoPush != nil {
		cfg.Behavior.AutoPush = *flags.AutoPush
	}
}
