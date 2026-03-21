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
// CLI flags (handled externally via MergeFlags) → env vars → repo .nuntius.toml → global config → defaults
func Load() (Config, error) {
	migrateConfigFiles()
	cfg := DefaultConfig()

	// Try repo-level config first
	repoConfig := ".nuntius.toml"
	globalConfig := globalConfigPath()

	loaded := false

	if fileExists(repoConfig) {
		if err := loadTOML(repoConfig, &cfg); err != nil {
			return Config{}, fmt.Errorf("loading repo config %s: %w", repoConfig, err)
		}
		loaded = true
	}

	if !loaded && globalConfig != "" && fileExists(globalConfig) {
		if err := loadTOML(globalConfig, &cfg); err != nil {
			return Config{}, fmt.Errorf("loading global config %s: %w", globalConfig, err)
		}
	}

	// Environment variables override file values
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

// migrateConfigFiles migrates legacy config and cache files to ~/.nuntius/.
// Migration is a best-effort copy — originals are preserved.
func migrateConfigFiles() {
	dir := NuntiusDir()
	if dir == "" {
		return
	}

	// Migrate global config: ~/.config/nuntius/config.toml → ~/.nuntius/config.toml
	newConfig := filepath.Join(dir, "config.toml")
	if !fileExists(newConfig) {
		if oldConfigDir, err := os.UserConfigDir(); err == nil {
			oldConfig := filepath.Join(oldConfigDir, "nuntius", "config.toml")
			if fileExists(oldConfig) {
				if err := copyFile(oldConfig, newConfig); err == nil {
					fmt.Fprintf(os.Stderr, "nuntius: migrated config from %s to %s\n", oldConfig, newConfig)
				}
			}
		}
	}

	// Migrate version cache: ~/.cache/nuntius/version-check.json → ~/.nuntius/version-check.json
	newCache := filepath.Join(dir, "version-check.json")
	if !fileExists(newCache) {
		if oldCacheDir, err := os.UserCacheDir(); err == nil {
			oldCache := filepath.Join(oldCacheDir, "nuntius", "version-check.json")
			if fileExists(oldCache) {
				_ = copyFile(oldCache, newCache)
			}
		}
	}
}

// copyFile copies src to dst, creating parent directories as needed.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading %s: %w", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", dst, err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", dst, err)
	}
	return nil
}

// applyEnvOverrides reads NUNTIUS_* environment variables and overrides
// the corresponding config fields. Env var naming convention:
// NUNTIUS_<SECTION>_<KEY> in SCREAMING_SNAKE_CASE.
func applyEnvOverrides(cfg *Config) {
	// AI section
	if v := os.Getenv("NUNTIUS_AI_PROVIDER"); v != "" {
		cfg.AI.Provider = v
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
	Model      string
	AutoCommit *bool
	AutoPush   *bool
}

// MergeFlags merges CLI flag overrides into the config.
func MergeFlags(cfg *Config, flags FlagOverrides) {
	if flags.Provider != "" {
		cfg.AI.Provider = flags.Provider
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
