package onboarding

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const totalSteps = 6

var (
	wizardAccent  = lipgloss.AdaptiveColor{Light: "#7D56F4", Dark: "#7D56F4"}
	wizardMuted   = lipgloss.AdaptiveColor{Light: "#999999", Dark: "#626262"}
	wizardPrimary = lipgloss.AdaptiveColor{Light: "#242424", Dark: "#EEEEEE"}

	styleTitle    = lipgloss.NewStyle().Foreground(wizardAccent).Bold(true)
	styleCursor   = lipgloss.NewStyle().Foreground(wizardAccent).Bold(true)
	styleOption   = lipgloss.NewStyle().Foreground(wizardPrimary)
	styleProgress = lipgloss.NewStyle().Bold(true).Foreground(wizardPrimary)
	styleHint     = lipgloss.NewStyle().Foreground(wizardMuted)
)

// Wizard is the Bubble Tea model for the onboarding wizard.
// It is intended to run as a standalone tea.Program before the main TUI.
type Wizard struct {
	currentStep int
	cursors     [totalSteps]int
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
	// Step 3 (mode): default is cli = index 0.
	w.cursors[2] = defaultModeIndex("")

	w.cursors[5] = 1
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

// Result returns the wizard's configuration choices.
// Only valid when Done() is true.
func (w Wizard) Result() WizardResult {
	provider := ProviderOptions[w.cursors[0]].Value

	var model string
	if provider == "ollama" {
		model = strings.TrimSpace(w.ollamaInput.Value())
	} else {
		if models, ok := ModelOptions[provider]; ok && w.cursors[1] < len(models) {
			model = models[w.cursors[1]].Value
		}
	}

	return WizardResult{
		Provider:        provider,
		Mode:            ModeOptions[w.cursors[2]].Value,
		Model:           model,
		AutoCommit:      AutoCommitOptions[w.cursors[3]].Value == "true",
		AutoPush:        AutoPushOptions[w.cursors[4]].Value == "true",
		AutoUpdateCheck: UpdateCheckOptions[w.cursors[5]].Value == "true",
	}
}

// isOllamaModelStep returns true when on the model step with ollama selected.
func (w Wizard) isOllamaModelStep() bool {
	return w.currentStep == 1 && w.cursors[0] < len(ProviderOptions) &&
		ProviderOptions[w.cursors[0]].Value == "ollama"
}

// currentOptions returns the selectable options for the current step.
// Returns nil for the ollama model step (which uses a text input instead).
func (w Wizard) currentOptions() []Option {
	switch w.currentStep {
	case 0:
		return ProviderOptions
	case 1:
		if w.isOllamaModelStep() {
			return nil
		}
		provider := ProviderOptions[w.cursors[0]].Value
		return ModelOptions[provider]
	case 2:
		if ProviderOptions[w.cursors[0]].Value == "copilot" {
			return ModeOptions[1:2] // copilot only supports cli mode
		}
		return ModeOptions
	case 3:
		return AutoCommitOptions
	case 4:
		return AutoPushOptions
	case 5:
		return UpdateCheckOptions
	default:
		return nil
	}
}

// stepPrompt returns the display prompt for the current step.
func (w Wizard) stepPrompt() string {
	switch w.currentStep {
	case 0:
		return "Select your AI provider"
	case 1:
		if w.isOllamaModelStep() {
			return "Enter a model name for ollama"
		}
		provider := ProviderOptions[w.cursors[0]].Value
		return fmt.Sprintf("Select a model for %s", provider)
	case 2:
		return "Select connection mode"
	case 3:
		return "Auto-commit after generating a message?"
	case 4:
		return "Auto-push after committing?"
	case 5:
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

	return w.handleListKey(msg)
}

// handleOllamaModelKey processes keys on the ollama free-text input step.
func (w Wizard) handleOllamaModelKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		next, cmd := w.advanceStep()
		return next, cmd
	case "left":
		if w.ollamaInput.Position() == 0 && w.currentStep > 0 {
			w.currentStep--
			return w, nil
		}
	}

	var cmd tea.Cmd
	w.ollamaInput, cmd = w.ollamaInput.Update(msg)
	return w, cmd
}

// handleListKey processes keys on a list-selection step.
func (w Wizard) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if w.cursors[w.currentStep] > 0 {
			w.cursors[w.currentStep]--
		}
	case "down", "j":
		opts := w.currentOptions()
		if opts != nil && w.cursors[w.currentStep] < len(opts)-1 {
			w.cursors[w.currentStep]++
		}
	case "enter":
		next, cmd := w.advanceStep()
		return next, cmd
	case "backspace", "left":
		if w.currentStep > 0 {
			w.currentStep--
		}
	}
	return w, nil
}

// advanceStep moves to the next step or marks the wizard done on the last step.
func (w Wizard) advanceStep() (Wizard, tea.Cmd) {
	if w.currentStep == totalSteps-1 {
		w.done = true
		return w, tea.Quit
	}
	w.currentStep++
	if w.isOllamaModelStep() {
		cmd := w.ollamaInput.Focus()
		return w, cmd
	}
	return w, nil
}

// View renders the wizard UI.
func (w Wizard) View() string {
	var sb strings.Builder

	sb.WriteString(styleTitle.Render("Welcome to nuntius! Let's set up your preferences."))
	sb.WriteString("\n\n")

	progress := fmt.Sprintf("Step %d of %d — %s", w.currentStep+1, totalSteps, w.stepPrompt())
	sb.WriteString(styleProgress.Render(progress))
	sb.WriteString("\n\n")

	if w.isOllamaModelStep() {
		sb.WriteString("  Model name: ")
		sb.WriteString(w.ollamaInput.View())
		sb.WriteString("\n")
	} else {
		opts := w.currentOptions()
		cursor := w.cursors[w.currentStep]
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
