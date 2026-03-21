package onboarding

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/crazzyghost/nuntius/internal/config"
)

// WriteConfig writes the wizard result as TOML to ~/.nuntius/config.toml.
func WriteConfig(result WizardResult) error {
	dir := config.NuntiusDir()
	if dir == "" {
		return fmt.Errorf("cannot determine config directory: $HOME is not set")
	}
	path := filepath.Join(dir, "config.toml")
	return writeConfigToPath(path, result)
}

// writeConfigToPath writes the config to the given path. Separated for testability.
func writeConfigToPath(path string, result WizardResult) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating config file %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	_, err = fmt.Fprintf(f,
		"[ai]\nprovider = %q\nmodel = %q\n\n[behavior]\nauto_commit = %v\nauto_push = %v\nauto_update_check = %v\n",
		result.Provider,
		result.Model,
		result.AutoCommit,
		result.AutoPush,
		result.AutoUpdateCheck,
	)
	if err != nil {
		return fmt.Errorf("writing config file %s: %w", path, err)
	}
	return nil
}
