// Package tui implements the Bubble Tea TUI layer for Nuntius.
package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crazzyghost/nuntius/internal/config"
	"github.com/crazzyghost/nuntius/internal/events"
)

const actionBarHeight = 3

// AppModel is the root Bubble Tea model that composes the viewport and
// action bar, manages global state, and routes messages between components.
type AppModel struct {
	viewport  ViewportModel
	actionbar ActionBarModel
	config    config.Config
	keys      KeyMap
	help      help.Model
	showHelp  bool
	width     int
	height    int
	ready     bool
	status    string // transient status message
}

// NewApp creates a new root TUI model.
func NewApp(cfg config.Config) AppModel {
	h := help.New()
	return AppModel{
		viewport:  NewViewport(),
		actionbar: NewActionBar(),
		config:    cfg,
		keys:      DefaultKeyMap,
		help:      h,
	}
}

// Init returns the initial commands for the application.
func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.viewport.Init(),
		m.actionbar.Init(),
	)
}

// Update handles all incoming messages.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.help.Width = msg.Width
		viewportHeight := m.height - actionBarHeight - 2 // borders
		if viewportHeight < 1 {
			viewportHeight = 1
		}
		m.viewport.SetSize(m.width-2, viewportHeight)
		return m, nil

	case tea.KeyMsg:
		// Global keys handled at root level.
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			return m, nil

		case key.Matches(msg, m.keys.Generate):
			if m.actionbar.GenerateEnabled() && m.viewport.HasStagedChanges() {
				m.status = ""
				cmd := m.dispatchGenerate()
				cmds = append(cmds, cmd)
			} else if !m.viewport.HasStagedChanges() {
				m.status = StatusError.Render("No staged changes to generate a message for.")
			}
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keys.Commit):
			if m.actionbar.CommitEnabled() && m.viewport.HasMessage() {
				m.status = ""
				cmd := m.dispatchCommit()
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keys.Push):
			if m.actionbar.PushEnabled() {
				m.status = ""
				cmd := m.dispatchPush()
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keys.Tab):
			m.actionbar.FocusNext()
			return m, nil

		case key.Matches(msg, m.keys.Enter):
			action := m.actionbar.FocusedAction()
			switch action {
			case "g":
				if m.viewport.HasStagedChanges() {
					cmd := m.dispatchGenerate()
					cmds = append(cmds, cmd)
				}
			case "c":
				if m.viewport.HasMessage() {
					cmd := m.dispatchCommit()
					cmds = append(cmds, cmd)
				}
			case "p":
				cmd := m.dispatchPush()
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

		// Scroll keys delegated to viewport.
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	// Route domain messages to the appropriate components.
	case events.FilesChangedMsg:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)

	case events.GenerateRequestedMsg:
		var vcmd, acmd tea.Cmd
		m.viewport, vcmd = m.viewport.Update(msg)
		m.actionbar, acmd = m.actionbar.Update(msg)
		cmds = append(cmds, vcmd, acmd)

	case events.MessageReadyMsg:
		var vcmd, acmd tea.Cmd
		m.viewport, vcmd = m.viewport.Update(msg)
		m.actionbar, acmd = m.actionbar.Update(msg)
		cmds = append(cmds, vcmd, acmd)

	case events.CommitResultMsg:
		var cmd tea.Cmd
		m.actionbar, cmd = m.actionbar.Update(msg)
		cmds = append(cmds, cmd)
		if msg.Err != nil {
			m.status = StatusError.Render(fmt.Sprintf("Commit failed: %v", msg.Err))
		} else {
			m.status = StatusOK.Render(fmt.Sprintf("Committed: %s", msg.Hash))
		}

	case events.PushResultMsg:
		var cmd tea.Cmd
		m.actionbar, cmd = m.actionbar.Update(msg)
		cmds = append(cmds, cmd)
		if msg.Err != nil {
			m.status = StatusError.Render(fmt.Sprintf("Push failed: %v", msg.Err))
		} else {
			m.status = StatusOK.Render(fmt.Sprintf("Pushed to %s", msg.Remote))
		}

	case events.ErrorMsg:
		var cmd tea.Cmd
		m.actionbar, cmd = m.actionbar.Update(msg)
		cmds = append(cmds, cmd)
		m.status = StatusError.Render(fmt.Sprintf("[%s] %v", msg.Source, msg.Err))

	default:
		// Forward spinner ticks and other messages.
		var vcmd, acmd tea.Cmd
		m.viewport, vcmd = m.viewport.Update(msg)
		m.actionbar, acmd = m.actionbar.Update(msg)
		cmds = append(cmds, vcmd, acmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the full TUI layout.
func (m AppModel) View() string {
	if !m.ready {
		return "Starting Nuntius...\n"
	}

	// Main viewport panel with border.
	vpContent := m.viewport.View()
	vpPanel := PanelBorder.
		Width(m.width - 2).
		Render(vpContent)

	// Action bar.
	abContent := m.actionbar.View()

	// Status line.
	statusLine := ""
	if m.status != "" {
		statusLine = m.status
	}

	// Help overlay.
	helpView := ""
	if m.showHelp {
		helpView = "\n" + m.help.View(m.keys)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		vpPanel,
		abContent,
		statusLine,
		helpView,
	)
}

// dispatchGenerate sends a GenerateRequestedMsg into the update loop.
func (m *AppModel) dispatchGenerate() tea.Cmd {
	return func() tea.Msg {
		return events.GenerateRequestedMsg{}
	}
}

// dispatchCommit sends a commit command placeholder.
// The actual git commit logic is wired in Phase 5 (integration).
func (m *AppModel) dispatchCommit() tea.Cmd {
	return func() tea.Msg {
		return events.GenerateRequestedMsg{} // placeholder — replaced in integration phase
	}
}

// dispatchPush sends a push command placeholder.
// The actual git push logic is wired in Phase 5 (integration).
func (m *AppModel) dispatchPush() tea.Cmd {
	return func() tea.Msg {
		return events.GenerateRequestedMsg{} // placeholder — replaced in integration phase
	}
}

// Width returns the current terminal width.
func (m AppModel) Width() int {
	return m.width
}

// Height returns the current terminal height.
func (m AppModel) Height() int {
	return m.height
}

// Ready returns whether the TUI has received its first window size message.
func (m AppModel) Ready() bool {
	return m.ready
}

// ShowHelp returns whether the help overlay is visible.
func (m AppModel) ShowHelp() bool {
	return m.showHelp
}
