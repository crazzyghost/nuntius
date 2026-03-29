package ai

import (
	"fmt"
	"strings"
	"testing"
)

func TestResolveAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		provider string
		wantKey  string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "key present",
			envValue: "sk-test-123",
			provider: "claude",
			wantKey:  "sk-test-123",
		},
		{
			name:     "key absent",
			envValue: "",
			provider: "claude",
			wantErr:  true,
			errMsg:   "NUNTIUS_AI_API_KEY",
		},
		{
			name:     "key absent includes provider",
			envValue: "",
			provider: "gemini",
			wantErr:  true,
			errMsg:   "gemini",
		},
		{
			name:     "key present for codex",
			envValue: "sk-codex-456",
			provider: "codex",
			wantKey:  "sk-codex-456",
		},
		{
			name:     "error format includes export hint",
			envValue: "",
			provider: "codex",
			wantErr:  true,
			errMsg:   "export NUNTIUS_AI_API_KEY",
		},
		{
			name:     "whitespace-only key treated as empty",
			envValue: "   ",
			provider: "claude",
			wantKey:  "   ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv("NUNTIUS_AI_API_KEY", tt.envValue)
			}
			key, err := ResolveAPIKey(tt.provider)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error %q should contain %q", err, tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if key != tt.wantKey {
				t.Errorf("got key %q, want %q", key, tt.wantKey)
			}
		})
	}
}

func TestRequiresAPIKey(t *testing.T) {
	tests := []struct {
		provider string
		mode     ProviderMode
		want     bool
	}{
		{"claude", ModeAPI, true},
		{"codex", ModeAPI, true},
		{"gemini", ModeAPI, true},
		{"ollama", ModeAPI, false},
		{"claude", ModeCLI, false},
		{"copilot", ModeCLI, false},
	}
	for _, tt := range tests {
		name := fmt.Sprintf("%s_%s", tt.provider, tt.mode)
		t.Run(name, func(t *testing.T) {
			got := RequiresAPIKey(tt.provider, tt.mode)
			if got != tt.want {
				t.Errorf("RequiresAPIKey(%q, %q) = %v, want %v",
					tt.provider, tt.mode, got, tt.want)
			}
		})
	}
}

func TestAPIKeyEnvVar(t *testing.T) {
	if APIKeyEnvVar != "NUNTIUS_AI_API_KEY" {
		t.Errorf("APIKeyEnvVar = %q, want %q", APIKeyEnvVar, "NUNTIUS_AI_API_KEY")
	}
}
