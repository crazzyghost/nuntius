package ai

import (
	"fmt"
	"strings"
	"testing"

	"github.com/crazzyghost/nuntius/internal/config"
)

func TestNewProvider_APIAdapters(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		envKey   string
		envVal   string
		wantName string
		wantMode ProviderMode
	}{
		{"claude", "claude", "ANTHROPIC_API_KEY", "test-key", "claude", ModeAPI},
		{"gemini", "gemini", "GEMINI_API_KEY", "test-key", "gemini", ModeAPI},
		{"codex", "codex", "OPENAI_API_KEY", "test-key", "codex", ModeAPI},
		{"ollama", "ollama", "", "", "ollama", ModeAPI},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envKey != "" {
				t.Setenv(tt.envKey, tt.envVal)
			}
			cfg := config.AIConfig{
				Provider:  tt.provider,
				Mode:      "api",
				APIKeyEnv: tt.envKey,
			}
			p, err := NewProvider(cfg)
			if err != nil {
				t.Fatalf("NewProvider(%q, api) error: %v", tt.provider, err)
			}
			if p.Name() != tt.wantName {
				t.Errorf("Name() = %q, want %q", p.Name(), tt.wantName)
			}
			if p.Mode() != tt.wantMode {
				t.Errorf("Mode() = %q, want %q", p.Mode(), tt.wantMode)
			}
		})
	}
}

func TestNewProvider_CopilotAPIMode_Error(t *testing.T) {
	cfg := config.AIConfig{Provider: "copilot", Mode: "api"}
	_, err := NewProvider(cfg)
	if err == nil {
		t.Fatal("expected error for copilot + api mode, got nil")
	}
	if !strings.Contains(err.Error(), "only supports cli mode") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "only supports cli mode")
	}
}

func TestNewProvider_CustomAPIMode_Error(t *testing.T) {
	cfg := config.AIConfig{Provider: "custom", Mode: "api"}
	_, err := NewProvider(cfg)
	if err == nil {
		t.Fatal("expected error for custom + api mode, got nil")
	}
	if !strings.Contains(err.Error(), "only supports cli mode") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "only supports cli mode")
	}
}

func TestNewProvider_MigrationHint_CLISuffix(t *testing.T) {
	tests := []struct {
		provider string
		wantBase string
	}{
		{"claude-cli", "claude"},
		{"gemini-cli", "gemini"},
		{"codex-cli", "codex"},
		{"copilot-cli", "copilot"},
	}
	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			cfg := config.AIConfig{Provider: tt.provider}
			_, err := NewProvider(cfg)
			if err == nil {
				t.Fatalf("expected error for %q, got nil", tt.provider)
			}
			wantHint := fmt.Sprintf(`(hint: use provider = %q with mode = "cli")`, tt.wantBase)
			if !strings.Contains(err.Error(), wantHint) {
				t.Errorf("error = %q, want hint %q", err.Error(), wantHint)
			}
		})
	}
}

func TestNewProvider_UnknownProvider(t *testing.T) {
	cfg := config.AIConfig{Provider: "nonexistent"}
	_, err := NewProvider(cfg)
	if err == nil {
		t.Fatal("expected error for unknown provider, got nil")
	}
	if !strings.Contains(err.Error(), "unknown AI provider") {
		t.Errorf("error = %q, want it to contain 'unknown AI provider'", err.Error())
	}
}

func TestNewProvider_UnknownProviderAPIMode(t *testing.T) {
	cfg := config.AIConfig{Provider: "nonexistent", Mode: "api"}
	_, err := NewProvider(cfg)
	if err == nil {
		t.Fatal("expected error for unknown provider in api mode, got nil")
	}
}

func TestNewProvider_CustomRequiresCLICommand(t *testing.T) {
	cfg := config.AIConfig{Provider: "custom", CLICommand: ""}
	_, err := NewProvider(cfg)
	if err == nil {
		t.Fatal("expected error for custom provider without cli_command")
	}
}

func TestNewProvider_DefaultModeCLI(t *testing.T) {
	// Without mode set, all known providers default to CLI.
	// CLI providers may fail with "not found" but should NOT return "unknown provider".
	knownProviders := []string{"claude", "codex", "gemini", "copilot", "ollama"}
	for _, provider := range knownProviders {
		t.Run(provider, func(t *testing.T) {
			cfg := config.AIConfig{Provider: provider}
			_, err := NewProvider(cfg)
			// May error with "not found", but never "unknown AI provider: ..."
			if err != nil && strings.Contains(err.Error(), "unknown AI provider") {
				t.Errorf("provider %q without mode should default to CLI, not unknown: %v", provider, err)
			}
		})
	}
}

func TestMessageRequest_Fields(t *testing.T) {
	req := MessageRequest{
		Diff:        "diff content",
		FileList:    []string{"a.go", "b.go"},
		Conventions: "conventional",
		Model:       "claude-3-haiku",
	}
	if req.Diff != "diff content" {
		t.Error("Diff field mismatch")
	}
	if len(req.FileList) != 2 {
		t.Error("FileList length mismatch")
	}
}

func TestProviderMode_Constants(t *testing.T) {
	if ModeAPI != "api" {
		t.Errorf("ModeAPI = %q, want %q", ModeAPI, "api")
	}
	if ModeCLI != "cli" {
		t.Errorf("ModeCLI = %q, want %q", ModeCLI, "cli")
	}
}

func TestResolveProvider_DefaultMode(t *testing.T) {
	tests := []struct {
		provider string
		want     ProviderMode
	}{
		{"claude", ModeCLI},
		{"codex", ModeCLI},
		{"copilot", ModeCLI},
		{"gemini", ModeCLI},
		{"ollama", ModeCLI},
		{"custom", ModeCLI},
	}
	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			cfg := config.AIConfig{Provider: tt.provider}
			_, got, _ := resolveProvider(cfg)
			if got != tt.want {
				t.Errorf("resolveProvider(%q) mode = %q, want %q", tt.provider, got, tt.want)
			}
		})
	}
}

func TestResolveProvider_ExplicitMode(t *testing.T) {
	cfg := config.AIConfig{Provider: "claude", Mode: "api"}
	_, mode, _ := resolveProvider(cfg)
	if mode != ModeAPI {
		t.Errorf("resolveProvider with explicit api mode = %q, want %q", mode, ModeAPI)
	}
}

func TestValidateModeSupport(t *testing.T) {
	tests := []struct {
		name    string
		mode    ProviderMode
		wantErr bool
	}{
		{"copilot", ModeCLI, false},
		{"copilot", ModeAPI, true},
		{"custom", ModeCLI, false},
		{"custom", ModeAPI, true},
		{"claude", ModeAPI, false},
		{"claude", ModeCLI, false},
		{"ollama", ModeAPI, false},
		{"ollama", ModeCLI, false},
	}
	for _, tt := range tests {
		t.Run(tt.name+"/"+string(tt.mode), func(t *testing.T) {
			err := validateModeSupport(tt.name, tt.mode)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateModeSupport(%q, %q) err = %v, wantErr = %v", tt.name, tt.mode, err, tt.wantErr)
			}
		})
	}
}
