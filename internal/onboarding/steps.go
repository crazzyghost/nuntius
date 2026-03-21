package onboarding

// Option represents a selectable item in a wizard step.
type Option struct {
	Label string
	Value string
}

// WizardResult holds the configuration choices made during onboarding.
type WizardResult struct {
	Provider        string
	Mode            string // "api" or "cli"
	Model           string
	AutoCommit      bool
	AutoPush        bool
	AutoUpdateCheck bool
}

// ProviderOptions lists all AI providers in alphabetical order.
// The default provider is "claude" (index 0).
var ProviderOptions = []Option{
	{Label: "claude", Value: "claude"},
	{Label: "codex", Value: "codex"},
	{Label: "copilot", Value: "copilot"},
	{Label: "gemini", Value: "gemini"},
	{Label: "ollama", Value: "ollama"},
}

// ModelOptions maps each provider to its available models in alphabetical order.
// Ollama is not listed here — it uses a free-text input.
var ModelOptions = map[string][]Option{
	"claude": {
		{Label: "claude-haiku-4.5", Value: "claude-haiku-4.5"},
		{Label: "claude-sonnet-4", Value: "claude-sonnet-4"},
	},
	"codex": {
		{Label: "gpt-4o-mini", Value: "gpt-4o-mini"},
		{Label: "o3-mini", Value: "o3-mini"},
	},
	"copilot": {
		{Label: "claude-haiku-4.5", Value: "claude-haiku-4.5"},
		{Label: "gpt-4o-mini", Value: "gpt-4o-mini"},
	},
	"gemini": {
		{Label: "gemini-2.5-flash", Value: "gemini-2.5-flash"},
		{Label: "gemini-2.5-pro", Value: "gemini-2.5-pro"},
	},
}

// ModeOptions lists connection modes. Default is "cli" (index 0).
var ModeOptions = []Option{
	{Label: "cli — uses locally installed CLI tool (e.g. gh, claude)", Value: "cli"},
	{Label: "api — connects directly via API (requires API key)", Value: "api"},
}

// defaultModeIndex returns the default cursor index for the mode selection step.
// All providers default to CLI mode (index 0).
func defaultModeIndex(_ string) int {
	return 0
}

// AutoCommitOptions lists auto-commit choices. Default is "off" (index 0).
var AutoCommitOptions = []Option{
	{Label: "off", Value: "false"},
	{Label: "on", Value: "true"},
}

// AutoPushOptions lists auto-push choices. Default is "off" (index 0).
var AutoPushOptions = []Option{
	{Label: "off", Value: "false"},
	{Label: "on", Value: "true"},
}

// UpdateCheckOptions lists update-check choices. Default is "on" (index 0).
var UpdateCheckOptions = []Option{
	{Label: "on", Value: "true"},
	{Label: "off", Value: "false"},
}
