package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/crazzyghost/nuntius/internal/events"
)

// buttonState represents the visual state of an action bar button.
type buttonState int

const (
	btnNormal buttonState = iota
	btnFocused
	btnDisabled
	btnLoading
	btnSuccess
	btnError
)

// feedbackDuration controls how many ticks success/error feedback is shown.
const feedbackDuration = 30

// buttonClearMsg clears the transient feedback state on a button.
type buttonClearMsg struct {
	index int
}

// ButtonModel represents a single action bar button.
type ButtonModel struct {
	label     string
	key       string
	state     buttonState
	spinner   spinner.Model
	feedback  string
	feedbackN int // remaining feedback ticks
}

// NewButton creates a new button with the given label and key hint.
func NewButton(label, key string) ButtonModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return ButtonModel{
		label: label,
		key:   key,
		state: btnNormal,
		spinner: s,
	}
}

// buttonZone tracks the horizontal range of a rendered button.
type buttonZone struct {
	startX int
	endX   int
}

// ActionBarModel is the bottom action bar containing the three buttons.
type ActionBarModel struct {
	buttons         [3]ButtonModel
	focusIndex      int
	committed       bool // tracks whether a commit has happened
	messageConsumed bool // true after commit consumes the generated message
	zones           [3]buttonZone
	unpushedCount   int
}

// NewActionBar creates a new action bar with Generate, Commit, and Push buttons.
func NewActionBar() ActionBarModel {
	ab := ActionBarModel{
		buttons: [3]ButtonModel{
			NewButton("Generate", "g"),
			NewButton("Commit", "c"),
			NewButton("Push", "p"),
		},
		focusIndex: 0,
	}
	// Commit and Push start disabled.
	ab.buttons[1].state = btnDisabled
	ab.buttons[2].state = btnDisabled
	return ab
}

// Init returns the initial command for the action bar.
func (m ActionBarModel) Init() tea.Cmd {
	return tea.Batch(
		m.buttons[0].spinner.Tick,
		m.buttons[1].spinner.Tick,
		m.buttons[2].spinner.Tick,
	)
}

// Update handles messages for the action bar.
func (m ActionBarModel) Update(msg tea.Msg) (ActionBarModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case events.GenerateRequestedMsg:
		m.buttons[0].state = btnLoading
		cmds = append(cmds, m.buttons[0].spinner.Tick)

	case events.MessageReadyMsg:
		m.buttons[0].state = btnNormal
		// Enable commit button — new message is ready.
		m.buttons[1].state = btnNormal
		m.messageConsumed = false

	case events.CommitResultMsg:
		if msg.Err != nil {
			m.buttons[1].state = btnError
			m.buttons[1].feedback = "✗ Failed"
			m.buttons[1].feedbackN = feedbackDuration
		} else {
			m.buttons[1].state = btnSuccess
			m.buttons[1].feedback = fmt.Sprintf("✓ %s", msg.Hash)
			m.buttons[1].feedbackN = feedbackDuration
			m.committed = true
			m.messageConsumed = true
			// Enable push button.
			m.buttons[2].state = btnNormal
		}
		cmds = append(cmds, clearButtonAfterDelay(1))

	case events.PushResultMsg:
		if msg.Err != nil {
			m.buttons[2].state = btnError
			m.buttons[2].feedback = "✗ Failed"
			m.buttons[2].feedbackN = feedbackDuration
		} else {
			m.buttons[2].state = btnSuccess
			m.buttons[2].feedback = "✓ Pushed"
			m.buttons[2].feedbackN = feedbackDuration
		}
		cmds = append(cmds, clearButtonAfterDelay(2))

	case events.ErrorMsg:
		switch msg.Source {
		case "generate":
			m.buttons[0].state = btnError
			m.buttons[0].feedback = "✗ Error"
			m.buttons[0].feedbackN = feedbackDuration
			cmds = append(cmds, clearButtonAfterDelay(0))
		case "commit":
			m.buttons[1].state = btnError
			m.buttons[1].feedback = "✗ Error"
			m.buttons[1].feedbackN = feedbackDuration
			cmds = append(cmds, clearButtonAfterDelay(1))
		case "push":
			m.buttons[2].state = btnError
			m.buttons[2].feedback = "✗ Error"
			m.buttons[2].feedbackN = feedbackDuration
			cmds = append(cmds, clearButtonAfterDelay(2))
		}

	case buttonClearMsg:
		if msg.index >= 0 && msg.index < 3 {
			btn := &m.buttons[msg.index]
			btn.feedback = ""
			btn.feedbackN = 0
			// Reset to appropriate resting state.
			switch msg.index {
			case 0:
				btn.state = btnNormal
			case 1:
				if !m.messageConsumed {
					btn.state = btnNormal
				} else {
					btn.state = btnDisabled
				}
			case 2:
				if m.committed {
					btn.state = btnNormal
				} else {
					btn.state = btnDisabled
				}
			}
		}

	case spinner.TickMsg:
		for i := range m.buttons {
			if m.buttons[i].state == btnLoading {
				var cmd tea.Cmd
				m.buttons[i].spinner, cmd = m.buttons[i].spinner.Update(msg)
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// clearButtonAfterDelay returns a command that sends a buttonClearMsg after a tick.
func clearButtonAfterDelay(index int) tea.Cmd {
	return tea.Tick(
		2*time.Second,
		func(_ time.Time) tea.Msg {
			return buttonClearMsg{index: index}
		},
	)
}

// View renders the action bar and records button hit zones.
func (m *ActionBarModel) View() string {
	sep := "  │  "
	var parts []string
	x := 0
	for i, btn := range m.buttons {
		if i > 0 {
			x += len(sep)
		}
		// Show unpushed count badge on the Push button.
		if i == 2 && m.unpushedCount > 0 && btn.state == btnNormal {
			btn.label = fmt.Sprintf("Push (%d↑)", m.unpushedCount)
		}
		rendered := renderButton(btn)
		w := visibleWidth(rendered)
		m.zones[i] = buttonZone{startX: x, endX: x + w}
		parts = append(parts, rendered)
		x += w
	}
	return strings.Join(parts, sep)
}

// visibleWidth returns the printed width of a string, stripping ANSI escapes.
func visibleWidth(s string) int {
	inEsc := false
	w := 0
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		w++
	}
	return w
}

// HitTest returns the button index (0-2) at the given X coordinate, or -1.
func (m ActionBarModel) HitTest(x int) int {
	for i, z := range m.zones {
		if x >= z.startX && x < z.endX {
			return i
		}
	}
	return -1
}

// renderButton renders a single button based on its state.
func renderButton(btn ButtonModel) string {
	label := fmt.Sprintf("[%s] %s", btn.key, btn.label)

	switch btn.state {
	case btnFocused:
		return ButtonFocused.Render(label)
	case btnDisabled:
		return ButtonDisabled.Render(label)
	case btnLoading:
		return ButtonLoading.Render(fmt.Sprintf("%s %s", btn.spinner.View(), btn.label))
	case btnSuccess:
		if btn.feedback != "" {
			return ButtonSuccess.Render(btn.feedback)
		}
		return ButtonSuccess.Render(label)
	case btnError:
		if btn.feedback != "" {
			return ButtonError.Render(btn.feedback)
		}
		return ButtonError.Render(label)
	default:
		return ButtonNormal.Render(label)
	}
}

// FocusNext moves focus to the next enabled button.
func (m *ActionBarModel) FocusNext() {
	m.clearFocus()
	for i := 1; i <= 3; i++ {
		next := (m.focusIndex + i) % 3
		if m.buttons[next].state != btnDisabled && m.buttons[next].state != btnLoading {
			m.focusIndex = next
			m.buttons[next].state = btnFocused
			return
		}
	}
}

// clearFocus removes focus from all buttons, restoring their base state.
func (m *ActionBarModel) clearFocus() {
	for i := range m.buttons {
		if m.buttons[i].state == btnFocused {
			m.buttons[i].state = btnNormal
		}
	}
}

// FocusedAction returns the key of the currently focused button, or "" if none is focused.
func (m ActionBarModel) FocusedAction() string {
	if m.buttons[m.focusIndex].state == btnFocused {
		return m.buttons[m.focusIndex].key
	}
	return ""
}

// IsButtonEnabled returns true if the button at the given index is actionable.
func (m ActionBarModel) IsButtonEnabled(index int) bool {
	if index < 0 || index >= 3 {
		return false
	}
	s := m.buttons[index].state
	return s != btnDisabled && s != btnLoading
}

// Committed returns whether a commit has been made.
func (m ActionBarModel) Committed() bool {
	return m.committed
}

// GenerateEnabled returns true if the Generate button is actionable.
func (m ActionBarModel) GenerateEnabled() bool {
	return m.IsButtonEnabled(0)
}

// CommitEnabled returns true if the Commit button is actionable.
func (m ActionBarModel) CommitEnabled() bool {
	return m.IsButtonEnabled(1)
}

// PushEnabled returns true if the Push button is actionable.
func (m ActionBarModel) PushEnabled() bool {
	return m.IsButtonEnabled(2)
}

// EnablePush enables the push button and sets the unpushed commit count.
func (m *ActionBarModel) EnablePush(count int) {
	m.unpushedCount = count
	if m.buttons[2].state == btnDisabled {
		m.buttons[2].state = btnNormal
		m.committed = true
	}
}
