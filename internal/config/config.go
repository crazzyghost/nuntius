// Package config handles configuration management including struct
// definitions, defaults, TOML loading, and convention discovery.
package config

// Config is the top-level configuration for Nuntius.
type Config struct {
	// AI configures the AI provider used for commit message generation.
	AI AIConfig `toml:"ai"`
	// Behavior controls automatic commit and push behavior.
	Behavior BehaviorConfig `toml:"behavior"`
	// Conventions configures commit message convention detection and style.
	Conventions ConventionsConfig `toml:"conventions"`
}

// AIConfig holds settings for the AI provider.
type AIConfig struct {
	// Provider selects the AI backend.
	// API providers: "claude", "codex", "copilot", "gemini", "ollama"
	// CLI providers: "copilot-cli", "gemini-cli", "claude-cli", "custom"
	Provider string `toml:"provider"`
	// Model overrides the provider's default model. Empty uses the provider default.
	Model string `toml:"model"`
	// APIKeyEnv is the name of the environment variable holding the API key.
	// Not required for ollama or CLI providers.
	APIKeyEnv string `toml:"api_key_env"`
	// OllamaURL is the Ollama API endpoint. Default: http://localhost:11434
	OllamaURL string `toml:"ollama_url"`
	// CLICommand is a custom CLI command for the "custom" provider.
	CLICommand string `toml:"cli_command"`
	// CLIArgs are extra arguments appended to the CLI command.
	CLIArgs []string `toml:"cli_args"`
	// TimeoutSeconds is the maximum number of seconds to wait for an AI response.
	// Defaults to 60 seconds when 0.
	TimeoutSeconds int `toml:"timeout_seconds"`
}

// BehaviorConfig controls automatic actions after message generation.
type BehaviorConfig struct {
	// AutoCommit commits automatically after generating a message.
	AutoCommit bool `toml:"auto_commit"`
	// AutoPush pushes automatically after a successful commit.
	AutoPush bool `toml:"auto_push"`
	// ForcePush uses --force-with-lease when pushing.
	ForcePush bool `toml:"force_push"`
}

// ConventionsConfig controls commit message convention detection and formatting.
type ConventionsConfig struct {
	// Style sets the commit convention. One of: "conventional", "gitmoji", "angular", "custom".
	Style string `toml:"style"`
	// CustomTemplate is a path to a custom prompt template file.
	CustomTemplate string `toml:"custom_template"`
}

// DefaultConfig returns a Config populated with safe, read-only defaults.
func DefaultConfig() Config {
	return Config{
		AI: AIConfig{
			Provider:  "claude",
			Model:     "",
			APIKeyEnv: "ANTHROPIC_API_KEY",
			OllamaURL: "http://localhost:11434",
		},
		Behavior: BehaviorConfig{
			AutoCommit: false,
			AutoPush:   false,
			ForcePush:  false,
		},
		Conventions: ConventionsConfig{
			Style:          "conventional",
			CustomTemplate: "",
		},
	}
}
