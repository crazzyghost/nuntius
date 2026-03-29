package onboarding

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crazzyghost/nuntius/internal/ai"
)

// maxCursors is the size of the fixed cursor array (one per StepID).
const maxCursors = 7

var (
	wizardAccent  = lipgloss.AdaptiveColor{Light: "#7D56F4", Dark: "#7D56F4"}
	wizardMuted   = lipgloss.AdaptiveColor{Light: "#999999", Dark: "#626262"}
	wizardPrimary = lipgloss.AdaptiveColor{Light: "#242424", Dark: "#EEEEEE"}
	wizardSuccess = lipgloss.AdaptiveColor{Light: "#22C55E", Dark: "#22C55E"}
	wizardWarning = lipgloss.AdaptiveColor{Light: "#EAB308", Dark: "#EAB308"}

	styleTitle    = lipgloss.NewStyle().Foreground(wizardAccent).Bold(true)
	styleCursor   = lipgloss.NewStyle().Foreground(wizardAccent).Bold(true)
	styleOption   = lipgloss.NewStyle().Foreground(wizardPrimary)
	styleProgress = lipgloss.NewStyle().Bold(true).Foreground(wizardPrimary)
	styleHint     = lipgloss.NewStyle().Foreground(wizardMuted)
	styleSuccess  = lipgloss.NewStyle().Foreground(wizardSuccess).Bold(true)
	styleWarning  = lipgloss.NewStyle().Foreground(wizardWarning).Bold(true)
	styleCode     = lipgloss.NewStyle().Foreground(wizardMuted).Italic(true)
)

// Wizard is the Bubble Tea model for the onboarding wizard.
// It is intended to run as a standalone tea.Program before the main TUI.
type Wizard struct {
	stepIndex   int // index into activeSteps()
	cursors     [maxCursors]int
	ollamaInput textinput.Model
	done        bool
	skipped     bool
	width       int
	height      int
}

// NewWizard creates a new onboarding wizard with sensible defaults.
// Default selections: claude provider, cli mode, update check on.
func NewWizard() Wizard {
	ti := textinput.New()
	ti.Placeholder = "e.g. llama3.2"
	ti.CharLimit = 64

	w := Wizard{
		ollamaInput: ti,
	}
	// Mode cursor: default is api = index 0.
	w.cursors[StepMode] = defaultModeIndex("")
	// UpdateCheck cursor: default is on = index 1.
	w.cursors[StepUpdateCheck] = 1
	return w
}

// Init initializes the wizard. No initial commands needed.
func (w Wizard) Init() tea.Cmd {
	return nil
}

// Done returns true when the user completed all steps.
func (w Wizard) Done() bool { return w.done }

// Skipped returns true when the user skipped onboarding.
func (w Wizard) Skipped() bool { return w.skipped }

// activeSteps returns the ordered list of steps the user should navigate,
// given current selections. Called dynamically.
func (w Wizard) activeSteps() []StepID {
	steps := []StepID{StepProvider, StepModel, StepMode}

	provider := ProviderOptions[w.cursors[StepProvider]].Value
	mode := ModeOptions[w.cursors[StepMode]].Value
	if ai.RequiresAPIKey(provider, ai.ProviderMode(mode)) {
		steps = append(steps, StepAPIKey)
	}

	steps = append(steps, StepAutoCommit, StepAutoPush, StepUpdateCheck)
	return steps
}

// currentStepID returns the StepID for the current wizard position.
func (w Wizard) currentStepID() StepID {
	steps := w.activeSteps()
	if w.stepIndex >= len(steps) {
		return steps[len(steps)-1]
	}
	return steps[w.stepIndex]
}

// Result returns the wizard's configuration choices.
// Only valid when Done() is true.
func (w Wizard) Result() WizardResult {
	provider := ProviderOptions[w.cursors[StepProvider]].Value

	var model string
	if provider == "ollama" {
		model = strings.TrimSpace(w.ollamaInput.Value())
	} else {
		if models, ok := ModelOptions[provider]; ok && w.cursors[StepModel] < len(models) {
			model = models[w.cursors[StepModel]].Value
		}
	}

	return WizardResult{
		Provider:        provider,
		Mode:            ModeOptions[w.cursors[StepMode]].Value,
		Model:           model,
		AutoCommit:      AutoCommitOptions[w.cursors[StepAutoCommit]].Value == "true",
		AutoPush:        AutoPushOptions[w.cursors[StepAutoPush]].Value == "true",
		AutoUpdateCheck: UpdateCheckOptions[w.cursors[StepUpdateCheck]].Value == "true",
	}
}

// isOllamaModelStep returns true when on the model step with ollama selected.
func (w Wizard) isOllamaModelStep() bool {
	return w.currentStepID() == StepModel &&
		w.cursors[StepProvider] < len(ProviderOptions) &&
		ProviderOptions[w.cursors[StepProvider]].Value == "ollama"
}

// currentOptions returns the selectable options for the current step.
// Returns nil for the ollama model step (text input) and API key step (informational).
func (w Wizard) currentOptions() []Option {
	switch w.currentStepID() {
	case StepProvider:
		return ProviderOptions
	case StepModel:
		if w.isOllamaModelStep() {
			return nil
		}
		provider := ProviderOptions[w.cursors[StepProvider]].Value
		return ModelOptions[provider]
	case StepMode:
		if ProviderOptions[w.cursors[StepProvider]].Value == "copilot" {
			return ModeOptions[1:2] // copilot only supports cli mode
		}
		return ModeOptions
	case StepAPIKey:
		return nil // informational step
	case StepAutoCommit:
		return AutoCommitOptions
	case StepAutoPush:
		return AutoPushOptions
	case StepUpdateCheck:
		return UpdateCheckOptions
	default:
		return nil
	}
}

// stepPrompt returns the display prompt for the current step.
func (w Wizard) stepPrompt() string {
	switch w.currentStepID() {
	case StepProvider:
		return "Select your AI provider"
	case StepModel:
		if w.isOllamaModelStep() {
			return "Enter a model name for ollama"
		}
		provider := ProviderOptions[w.cursors[StepProvider]].Value
		return fmt.Sprintf("Select a model for %s", provider)
	case StepMode:
		return "Select connection mode"
	case StepAPIKey:
		return "API key check"
	case StepAutoCommit:
		return "Auto-commit after generating a message?"
	case StepAutoPush:
		return "Auto-push after committing?"
	case StepUpdateCheck:
		return "Check for updates on startup?"
	default:
		return ""
	}
}

// Update handles all incoming Bubble Tea messages.
func (w Wizard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w.width = msg.Width
		w.height = msg.Height
		return w, nil

	case tea.KeyMsg:
		return w.handleKey(msg)
	}

	// Forward other message types to the text input on the ollama model step.
	if w.isOllamaModelStep() {
		var cmd tea.Cmd
		w.ollamaInput, cmd = w.ollamaInput.Update(msg)
		return w, cmd
	}

	return w, nil
}

// handleKey processes keyboard input.
func (w Wizard) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		w.skipped = true
		return w, tea.Quit
	case "esc", "s":
		w.skipped = true
		return w, tea.Quit
	}

	if w.isOllamaModelStep() {
		return w.handleOllamaModelKey(msg)
	}

	if w.currentStepID() == StepAPIKey {
		return w.handleAPIKeyStepKey(msg)
	}

	return w.handleListKey(msg)
}

// handleOllamaModelKey processes keys on the ollama free-text input step.
func (w Wizard) handleOllamaModelKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		next, cmd := w.advanceStep()
		return next, cmd
	case "left":
		if w.ollamaInput.Position() == 0 && w.stepIndex > 0 {
			w.stepIndex--
			return w, nil
		}
	}

	var cmd tea.Cmd
	w.ollamaInput, cmd = w.ollamaInput.Update(msg)
	return w, cmd
}

// handleAPIKeyStepKey processes keys on the informational API key step.
func (w Wizard) handleAPIKeyStepKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		next, cmd := w.advanceStep()
		return next, cmd
	case "backspace", "left":
		if w.stepIndex > 0 {
			w.stepIndex--
		}
	}
	return w, nil
}

// handleListKey processes keys on a list-selection step.
func (w Wizard) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	sid := w.currentStepID()
	switch msg.String() {
	case "up", "k":
		if w.cursors[sid] > 0 {
			w.cursors[sid]--
		}
	case "down", "j":
		opts := w.currentOptions()
		if opts != nil && w.cursors[sid] < len(opts)-1 {
			w.cursors[sid]++
		}
	case "enter":
		next, cmd := w.advanceStep()
		return next, cmd
	case "backspace", "left":
		if w.stepIndex > 0 {
			w.stepIndex--
		}
	}
	return w, nil
}

// advanceStep moves to the next step or marks the wizard done on the last step.
func (w Wizard) advanceStep() (Wizard, tea.Cmd) {
	steps := w.activeSteps()
	if w.stepIndex >= len(steps)-1 {
		w.done = true
		return w, tea.Quit
	}
	w.stepIndex++
	if w.isOllamaModelStep() {
		cmd := w.ollamaInput.Focus()
		return w, cmd
	}
	return w, nil
}

// viewAPIKeyStep renders the informational API key step.
func (w Wizard) viewAPIKeyStep() string {
	key := os.Getenv(ai.APIKeyEnvVar)
	if key != "" {
		return styleSuccess.Render("✓ "+ai.APIKeyEnvVar+" detected") +
			"\n\n" +
			styleHint.Render("Press Enter to continue")
	}
	return styleWarning.Render("⚠ "+ai.APIKeyEnvVar+" is not set") +
		"\n\n" +
		"Run this in your terminal before using nuntius:\n\n" +
		styleCode.Render("  export "+ai.APIKeyEnvVar+"=<your-api-key>") +
		"\n\n" +
		styleHint.Render("nuntius never stores your API key.") +
		"\n" +
		styleHint.Render("Press Enter to continue")
}

// View renders the wizard UI.
func (w Wizard) View() string {
	var sb strings.Builder

	steps := w.activeSteps()
	totalSteps := len(steps)

	sb.WriteString(styleTitle.Render("Welcome to nuntius! Let's set up your preferences."))
	sb.WriteString("\n\n")

	progress := fmt.Sprintf("Step %d of %d — %s", w.stepIndex+1, totalSteps, w.stepPrompt())
	sb.WriteString(styleProgress.Render(progress))
	sb.WriteString("\n\n")

	if w.currentStepID() == StepAPIKey {
		sb.WriteString(w.viewAPIKeyStep())
	} else if w.isOllamaModelStep() {
		sb.WriteString("  Model name: ")
		sb.WriteString(w.ollamaInput.View())
		sb.WriteString("\n")
	} else {
		opts := w.currentOptions()
		sid := w.currentStepID()
		cursor := w.cursors[sid]
		for i, opt := range opts {
			if i == cursor {
				sb.WriteString(styleCursor.Render("  ❯ " + opt.Label))
			} else {
				sb.WriteString(styleOption.Render("    " + opt.Label))
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(styleHint.Render("  ↑/↓ navigate • Enter select • ← back • Esc skip"))

	return sb.String()
}
