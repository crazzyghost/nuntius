package config

import "testing"

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// AI defaults
	if cfg.AI.Provider != "" {
		t.Errorf("expected AI.Provider = %q, got %q", "", cfg.AI.Provider)
	}
	if cfg.AI.Model != "" {
		t.Errorf("expected AI.Model = %q, got %q", "", cfg.AI.Model)
	}
	if cfg.AI.APIKeyEnv != "" {
		t.Errorf("expected AI.APIKeyEnv = %q, got %q", "", cfg.AI.APIKeyEnv)
	}
	if cfg.AI.OllamaURL != "http://localhost:11434" {
		t.Errorf("expected AI.OllamaURL = %q, got %q", "http://localhost:11434", cfg.AI.OllamaURL)
	}
	if cfg.AI.CLICommand != "" {
		t.Errorf("expected AI.CLICommand = %q, got %q", "", cfg.AI.CLICommand)
	}
	if len(cfg.AI.CLIArgs) != 0 {
		t.Errorf("expected AI.CLIArgs to be empty, got %v", cfg.AI.CLIArgs)
	}

	// Behavior defaults — all false (read-only by default)
	if cfg.Behavior.AutoCommit {
		t.Error("expected Behavior.AutoCommit = false")
	}
	if cfg.Behavior.AutoPush {
		t.Error("expected Behavior.AutoPush = false")
	}
	if cfg.Behavior.ForcePush {
		t.Error("expected Behavior.ForcePush = false")
	}

	// Conventions defaults
	if cfg.Conventions.Style != "conventional" {
		t.Errorf("expected Conventions.Style = %q, got %q", "conventional", cfg.Conventions.Style)
	}
	if cfg.Conventions.CustomTemplate != "" {
		t.Errorf("expected Conventions.CustomTemplate = %q, got %q", "", cfg.Conventions.CustomTemplate)
	}
}

func TestDefaultConfigAutoUpdateCheck(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Behavior.AutoUpdateCheck {
		t.Error("expected Behavior.AutoUpdateCheck = true (default)")
	}
}
