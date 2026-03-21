// Package ai provides the AI provider abstraction for generating
// commit messages via API or CLI adapters.
package ai

import (
	"context"
	"fmt"
	"strings"

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
	// ModeAPI indicates an HTTP-based provider (Claude, Gemini, Codex, Ollama).
	ModeAPI ProviderMode = "api"
	// ModeCLI indicates a local CLI-based provider (copilot, gemini, claude, codex, ollama, custom).
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

// defaultModes maps each provider to its default mode (all CLI-first).
var defaultModes = map[string]ProviderMode{
	"claude":  ModeCLI,
	"codex":   ModeCLI,
	"copilot": ModeCLI,
	"gemini":  ModeCLI,
	"ollama":  ModeCLI,
	"custom":  ModeCLI,
}

// cliOnlyProviders lists providers that do not support API mode.
var cliOnlyProviders = map[string]bool{
	"copilot": true,
	"custom":  true,
}

// resolveProvider extracts the base provider name and effective mode from cfg.
// If Mode is empty, it falls back to the CLI default for the provider.
func resolveProvider(cfg config.AIConfig) (string, ProviderMode, error) {
	name := cfg.Provider
	mode := ProviderMode(cfg.Mode)

	if mode == "" {
		def, ok := defaultModes[name]
		if !ok {
			// Unknown provider — let the caller surface the error.
			return name, ModeCLI, nil
		}
		mode = def
	}
	return name, mode, nil
}

// validateModeSupport returns an error if the provider does not support the requested mode.
func validateModeSupport(name string, mode ProviderMode) error {
	if mode == ModeAPI && cliOnlyProviders[name] {
		return fmt.Errorf("%q provider only supports cli mode", name)
	}
	return nil
}

// newCLIProvider constructs the CLI adapter for the given base provider name.
func newCLIProvider(name string, cfg config.AIConfig) (Provider, error) {
	switch name {
	case "claude", "codex", "gemini", "copilot":
		return NewCLIAgent(cfg, name, nil)
	case "ollama":
		model := cfg.Model
		args := []string{"run"}
		if model != "" {
			args = append(args, model)
		}
		return NewCLIAgent(cfg, "ollama", args)
	case "custom":
		if cfg.CLICommand == "" {
			return nil, fmt.Errorf("provider %q requires cli_command to be set", name)
		}
		return NewCLIAgent(cfg, cfg.CLICommand, cfg.CLIArgs)
	default:
		return nil, fmt.Errorf("unknown AI provider: %q", name)
	}
}

// NewProvider creates the appropriate AI provider based on the config.
// Provider and mode are resolved independently:
//   - cfg.Provider selects the AI backend (e.g. "claude", "gemini", "ollama")
//   - cfg.Mode selects the connection mode ("api" or "cli"); defaults to "cli"
//
// Breaking changes from previous versions:
//   - "-cli" suffixed provider names (e.g. "claude-cli") are no longer accepted.
//     Use provider = "claude" with mode = "cli" instead.
//   - "copilot" is CLI-only; api mode is not supported.
//   - All providers now default to CLI mode.
func NewProvider(cfg config.AIConfig) (Provider, error) {
	// Migration hint for deprecated -cli suffix names.
	if strings.HasSuffix(cfg.Provider, "-cli") {
		base := strings.TrimSuffix(cfg.Provider, "-cli")
		return nil, fmt.Errorf("unknown AI provider: %q (hint: use provider = %q with mode = \"cli\")", cfg.Provider, base)
	}

	name, mode, err := resolveProvider(cfg)
	if err != nil {
		return nil, err
	}

	if err := validateModeSupport(name, mode); err != nil {
		return nil, err
	}

	if mode == ModeCLI {
		return newCLIProvider(name, cfg)
	}

	// API mode dispatch.
	switch name {
	case "claude":
		return NewClaude(cfg)
	case "codex":
		return NewCodex(cfg)
	case "gemini":
		return NewGemini(cfg)
	case "ollama":
		return NewOllama(cfg)
	default:
		return nil, fmt.Errorf("unknown AI provider: %q", name)
	}
}
