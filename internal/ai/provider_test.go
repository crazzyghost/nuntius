package ai

import (
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
		{"copilot", "copilot", "GITHUB_COPILOT_TOKEN", "test-token", "copilot", ModeAPI},
		{"ollama", "ollama", "", "", "ollama", ModeAPI},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envKey != "" {
				t.Setenv(tt.envKey, tt.envVal)
			}

			cfg := config.AIConfig{
				Provider:  tt.provider,
				APIKeyEnv: tt.envKey,
			}
			p, err := NewProvider(cfg)
			if err != nil {
				t.Fatalf("NewProvider(%q) error: %v", tt.provider, err)
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

func TestNewProvider_UnknownProvider(t *testing.T) {
	cfg := config.AIConfig{Provider: "nonexistent"}
	_, err := NewProvider(cfg)
	if err == nil {
		t.Fatal("expected error for unknown provider, got nil")
	}
	if want := `unknown AI provider: "nonexistent"`; err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestNewProvider_CustomRequiresCLICommand(t *testing.T) {
	cfg := config.AIConfig{Provider: "custom", CLICommand: ""}
	_, err := NewProvider(cfg)
	if err == nil {
		t.Fatal("expected error for custom provider without cli_command")
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
