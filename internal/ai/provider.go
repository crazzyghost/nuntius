// Package ai provides the AI provider abstraction for generating
// commit messages via API or CLI adapters.
package ai

import (
	"context"
	"fmt"

	"github.com/crazzyghost/nuntius/internal/config"
)

// Provider is the interface that all AI adapters must implement.
type Provider interface {
	// GenerateCommitMessage sends a diff and context to the AI provider
	// and returns a generated commit message.
	GenerateCommitMessage(ctx context.Context, req MessageRequest) (string, error)
	// Name returns the provider's display name.
	Name() string
	// Mode returns whether this provider uses an HTTP API or a local CLI.
	Mode() ProviderMode
}

// ProviderMode distinguishes between HTTP-based and CLI-based providers.
type ProviderMode string

const (
	// ModeAPI indicates an HTTP-based provider (Claude, Gemini, Codex, Copilot, Ollama).
	ModeAPI ProviderMode = "api"
	// ModeCLI indicates a local CLI-based provider (copilot-cli, gemini-cli, claude-cli, codex-cli).
	ModeCLI ProviderMode = "cli"
)

// MessageRequest holds the inputs needed by any provider to generate a commit message.
type MessageRequest struct {
	// Diff is the unified diff of staged changes.
	Diff string
	// FileList is the list of changed file paths.
	FileList []string
	// Conventions is the commit convention label (e.g. "conventional", "gitmoji").
	Conventions string
	// Model overrides the provider's default model. Empty uses the provider default.
	Model string
}

// NewProvider creates the appropriate AI provider based on the config.
func NewProvider(cfg config.AIConfig) (Provider, error) {
	switch cfg.Provider {
	// --- API adapters (HTTP) ---
	case "claude":
		return NewClaude(cfg)
	case "gemini":
		return NewGemini(cfg)
	case "codex":
		return NewCodex(cfg)
	case "copilot":
		return NewCopilot(cfg)
	case "ollama":
		return NewOllama(cfg)
	// --- CLI adapters (shell exec) ---
	case "copilot-cli":
		return NewCLIAgent(cfg, "copilot", nil)
	case "gemini-cli":
		return NewCLIAgent(cfg, "gemini", nil)
	case "claude-cli":
		return NewCLIAgent(cfg, "claude", nil)
	case "codex-cli":
		return NewCLIAgent(cfg, "codex", nil)
	case "custom":
		if cfg.CLICommand == "" {
			return nil, fmt.Errorf("provider %q requires cli_command to be set", cfg.Provider)
		}
		return NewCLIAgent(cfg, cfg.CLICommand, cfg.CLIArgs)
	default:
		return nil, fmt.Errorf("unknown AI provider: %q", cfg.Provider)
	}
}
