package onboarding

// StepID identifies a wizard step independently of its display index.
type StepID int

const (
	StepProvider StepID = iota
	StepModel
	StepMode
	StepAPIKey // conditional — only if API mode + key-requiring provider
	StepAutoCommit
	StepAutoPush
	StepUpdateCheck
)

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
		{Label: "Claude Haiku 4.5", Value: "claude-haiku-4-5"},
	},
	"codex": {
		{Label: "GPT-5.4-Mini", Value: "gpt-5.4-mini"},
		{Label: "GPT-5.1-Codex-Mini", Value: "gpt-5-codex-mini"},
		{Label: "GPT-5.3-Codex-Spark", Value: "gpt-5.3-codex-spark"},
	},
	"copilot": {
		{Label: "Claude Haiku 4.5", Value: "claude-haiku-4.5"},
		{Label: "GPT-5.4-Mini", Value: "gpt-5.4-mini"},
		{Label: "GPT-5.1-Codex-Mini", Value: "GPT-5.1-Codex-Mini"},
		{Label: "GPT-5 mini", Value: "gpt-5-mini"},
	},
	"gemini": {
		{Label: "Gemini 3.1 Flash Lite", Value: "gemini-3.1-flash-lite-preview"},
		{Label: "Gemini 3 Flash", Value: "gemini-3-flash-preview"},
		{Label: "Gemini 2.5 Flash", Value: "gemini-2.5-flash"},
		{Label: "Gemini 2.5 Flash Lite", Value: "gemini-2.5-flash-lite"},
	},
}

// ModeOptions lists connection modes. Default is "cli" (index 0).
var ModeOptions = []Option{
	{Label: "api — connects directly via API (requires API key)", Value: "api"},
	{Label: "cli — uses locally installed CLI tool (e.g. copilot, claude, gemini)", Value: "cli"},
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
	{Label: "off", Value: "false"},
	{Label: "on", Value: "true"},
}
