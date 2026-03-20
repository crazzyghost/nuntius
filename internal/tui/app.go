// Package tui implements the Bubble Tea TUI layer for Nuntius.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crazzyghost/nuntius/internal/ai"
	"github.com/crazzyghost/nuntius/internal/config"
	"github.com/crazzyghost/nuntius/internal/events"
	"github.com/crazzyghost/nuntius/internal/git"
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

	// Integration fields.
	watcher     *git.Watcher
	provider    ai.Provider
	conventions string

	// Version check fields.
	version       string
	buildDate     string
	noUpdateCheck bool
	updateNotice  string

	// Status line with auto-clear.
	statusEntry *statusEntry
}

// NewApp creates a new root TUI model with no external dependencies.
// Use WithWatcher, WithProvider, and WithConventions to wire in real services.
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

// WithWatcher sets the git file watcher on the app.
func (m AppModel) WithWatcher(w *git.Watcher) AppModel {
	m.watcher = w
	return m
}

// WithProvider sets the AI provider on the app.
func (m AppModel) WithProvider(p ai.Provider) AppModel {
	m.provider = p
	return m
}

// WithConventions sets the detected commit convention style.
func (m AppModel) WithConventions(c string) AppModel {
	m.conventions = c
	return m
}

// WithVersion sets the current build version and date for update checking.
func (m AppModel) WithVersion(v, buildDate string) AppModel {
	m.version = v
	m.buildDate = buildDate
	return m
}

// WithNoUpdateCheck disables the startup version check.
func (m AppModel) WithNoUpdateCheck() AppModel {
	m.noUpdateCheck = true
	return m
}

// Init returns the initial commands for the application.
func (m AppModel) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.viewport.Init(),
		m.actionbar.Init(),
		// Fetch current git status immediately so existing changes are visible.
		refreshStatusCmd(),
		// Check for unpushed commits to enable Push button.
		checkUnpushedCmd(),
	}

	// Start listening for watcher events if a watcher is wired in.
	if m.watcher != nil {
		cmds = append(cmds, waitForFileChange(m.watcher))
	}

	// Non-blocking version check.
	if !m.noUpdateCheck && m.version != "" && m.version != "dev" {
		cmds = append(cmds, checkVersionCmd(m.version, m.buildDate))
	}

	return tea.Batch(cmds...)
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

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			// Recompute button zones so HitTest works.
			m.actionbar.View()
			idx := m.actionbar.HitTest(msg.X)
			// Accept clicks in the bottom region where the action bar lives.
			if idx >= 0 && msg.Y >= m.height-actionBarHeight {
				switch idx {
				case 0: // Generate
					if m.actionbar.GenerateEnabled() && m.viewport.HasChanges() {
						cmd := m.triggerGenerate()
						cmds = append(cmds, cmd...)
					}
					// Disabled button: silently ignore clicks.
				case 1: // Commit
					if m.actionbar.CommitEnabled() && m.viewport.HasMessage() {
						cmd := m.triggerCommit()
						cmds = append(cmds, cmd...)
					}
				case 2: // Push
					if m.actionbar.PushEnabled() {
						cmd := m.triggerPush()
						cmds = append(cmds, cmd...)
					}
				}
			}
		}
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		// Any key press clears the status line.
		m.statusEntry = nil

		// Global keys handled at root level.
		switch {
		case key.Matches(msg, m.keys.Quit):
			if m.watcher != nil {
				m.watcher.Stop()
			}
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			return m, nil

		case key.Matches(msg, m.keys.Generate):
			if m.actionbar.GenerateEnabled() && m.viewport.HasChanges() {
				cmd := m.triggerGenerate()
				cmds = append(cmds, cmd...)
			}
			// Disabled button: silently ignore key press.
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keys.Commit):
			if m.actionbar.CommitEnabled() && m.viewport.HasMessage() {
				cmd := m.triggerCommit()
				cmds = append(cmds, cmd...)
			}
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keys.Push):
			if m.actionbar.PushEnabled() {
				cmd := m.triggerPush()
				cmds = append(cmds, cmd...)
			}
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keys.Tab):
			m.actionbar.FocusNext()
			return m, nil

		case key.Matches(msg, m.keys.Enter):
			action := m.actionbar.FocusedAction()
			switch action {
			case "g":
				if m.actionbar.GenerateEnabled() && m.viewport.HasChanges() {
					cmd := m.triggerGenerate()
					cmds = append(cmds, cmd...)
				}
			case "c":
				if m.viewport.HasMessage() {
					cmd := m.triggerCommit()
					cmds = append(cmds, cmd...)
				}
			case "p":
				if m.actionbar.PushEnabled() {
					cmd := m.triggerPush()
					cmds = append(cmds, cmd...)
				}
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
		// Enable/disable Generate based on whether there are changes.
		m.actionbar.SetGenerateEnabled(len(msg.Files) > 0)
		// Re-check for unpushed commits (e.g. external commit).
		cmds = append(cmds, checkUnpushedCmd())
		// Keep listening for more watcher events.
		if m.watcher != nil {
			cmds = append(cmds, waitForFileChange(m.watcher))
		}

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
		m.setStatus("Message generated.", statusSuccess)
		cmds = append(cmds, scheduleStatusClear())

		// Auto-commit: chain into commit after generation.
		if m.config.Behavior.AutoCommit {
			m.setStatus("Auto-committing...", statusInfo)
			commitCmds := m.triggerCommit()
			cmds = append(cmds, commitCmds...)
		}

	case events.CommitResultMsg:
		var cmd tea.Cmd
		m.actionbar, cmd = m.actionbar.Update(msg)
		cmds = append(cmds, cmd)
		if msg.Err != nil {
			m.setStatus(fmt.Sprintf("Commit failed: %v", msg.Err), statusErr)
		} else {
			m.setStatus(fmt.Sprintf("Committed: %s", msg.Hash), statusSuccess)
			// Clear the consumed message and switch back to file list.
			m.viewport.ClearMessage()
			cmds = append(cmds, refreshStatusCmd())

			// Auto-push: chain into push after commit.
			if m.config.Behavior.AutoPush {
				m.setStatus("Auto-pushing...", statusInfo)
				pushCmds := m.triggerPush()
				cmds = append(cmds, pushCmds...)
			}
		}
		cmds = append(cmds, scheduleStatusClear())

	case events.PushResultMsg:
		var cmd tea.Cmd
		m.actionbar, cmd = m.actionbar.Update(msg)
		cmds = append(cmds, cmd)
		m.viewport.ClearLoading()
		if msg.Err != nil {
			m.setStatus(fmt.Sprintf("Push failed: %v", msg.Err), statusErr)
		} else {
			m.setStatus(fmt.Sprintf("Pushed to %s", msg.Remote), statusSuccess)
		}
		cmds = append(cmds, scheduleStatusClear())

	case events.ErrorMsg:
		var cmd tea.Cmd
		m.actionbar, cmd = m.actionbar.Update(msg)
		cmds = append(cmds, cmd)
		m.setStatus(formatErrorMsg(msg), statusErr)
		cmds = append(cmds, scheduleStatusClear())

	case statusClearMsg:
		m.statusEntry = nil

	case unpushedMsg:
		if msg.count > 0 {
			m.actionbar.EnablePush(msg.count)
		}

	case events.UpdateAvailableMsg:
		m.updateNotice = fmt.Sprintf("Update available: %s → %s", msg.Current, msg.Latest)

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

	// Status line.
	statusLine := ""
	if m.statusEntry != nil {
		statusLine = renderStatus(*m.statusEntry)
	}

	// Action bar with auto-mode badges and provider badge.
	abContent := m.actionbar.View()
	if m.config.Behavior.AutoCommit || m.config.Behavior.AutoPush {
		abContent += "  " + StatusMuted.Render(m.autoModeBadges())
	}
	// Provider/model badge: e.g. "claude · haiku"
	abContent += "  " + StatusMuted.Render(m.providerBadge())
	if m.version != "" {
		versionTag := StatusMuted.Render(m.version)
		abWidth := visibleWidth(abContent)
		vTagWidth := visibleWidth(versionTag)
		padding := m.width - abWidth - vTagWidth
		if padding > 0 {
			abContent += strings.Repeat(" ", padding) + versionTag
		}
	}

	// Update notice.
	updateLine := ""
	if m.updateNotice != "" {
		updateLine = StatusInfo.Render(m.updateNotice)
	}

	// Help overlay.
	helpView := ""
	if m.showHelp {
		helpView = "\n" + m.help.View(m.keys)
	}

	// Build layout parts — omit empty strings to avoid blank lines.
	parts := []string{vpPanel}
	if statusLine != "" {
		parts = append(parts, statusLine)
	}
	parts = append(parts, abContent)
	if updateLine != "" {
		parts = append(parts, updateLine)
	}
	if helpView != "" {
		parts = append(parts, helpView)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		parts...,
	)
}

// triggerGenerate dispatches the generate flow.
func (m *AppModel) triggerGenerate() []tea.Cmd {
	// Send GenerateRequestedMsg to update spinners/states immediately.
	reqCmd := func() tea.Msg { return events.GenerateRequestedMsg{} }

	if m.provider == nil {
		errCmd := func() tea.Msg {
			return events.ErrorMsg{Source: "generate", Err: fmt.Errorf("no AI provider configured")}
		}
		return []tea.Cmd{reqCmd, errCmd}
	}

	return []tea.Cmd{reqCmd, generateCmd(m.provider, m.config)}
}

// triggerCommit dispatches the commit flow.
func (m *AppModel) triggerCommit() []tea.Cmd {
	message := m.viewport.Message()
	if message == "" {
		return nil
	}
	return []tea.Cmd{commitCmd(message)}
}

// triggerPush dispatches the push flow.
func (m *AppModel) triggerPush() []tea.Cmd {
	m.actionbar.SetPushLoading()
	count := m.actionbar.UnpushedCount()
	if count > 0 {
		m.viewport.SetLoading(fmt.Sprintf("Pushing %d commit(s)...", count))
	} else {
		m.viewport.SetLoading("Pushing...")
	}
	return []tea.Cmd{pushCmd(m.config.Behavior.ForcePush)}
}

// setStatus sets the transient status message.
func (m *AppModel) setStatus(message string, level statusLevel) {
	m.statusEntry = &statusEntry{message: message, level: level}
}

// autoModeBadges returns a compact string showing which auto modes are enabled.
func (m AppModel) autoModeBadges() string {
	var badges []string
	if m.config.Behavior.AutoCommit {
		badges = append(badges, "auto-commit")
	}
	if m.config.Behavior.AutoPush {
		badges = append(badges, "auto-push")
	}
	if len(badges) == 0 {
		return ""
	}
	return "[" + strings.Join(badges, " | ") + "]"
}

// providerBadge returns a display string like "claude · haiku" or "no provider".
func (m AppModel) providerBadge() string {
	if m.provider == nil {
		return "no provider"
	}
	name := m.provider.Name()
	model := m.config.AI.Model
	if model == "" {
		model = defaultModelLabel(name)
	}
	if model == "" {
		return name
	}
	return name + " · " + model
}

// defaultModelLabel maps a provider name to its default model display label.
func defaultModelLabel(providerName string) string {
	switch providerName {
	case "claude":
		return "haiku"
	case "gemini":
		return "flash"
	case "codex":
		return "gpt-4o-mini"
	case "ollama":
		return "llama3.2"
	case "copilot":
		return "copilot"
	default:
		if strings.HasSuffix(providerName, "-cli") || providerName == "custom" {
			return "cli"
		}
		return ""
	}
}

// formatErrorMsg creates a user-friendly error message with hints.
func formatErrorMsg(msg events.ErrorMsg) string {
	base := fmt.Sprintf("[%s] %v", msg.Source, msg.Err)

	errStr := msg.Err.Error()

	// Add hints for common errors.
	switch {
	case strings.Contains(errStr, "API key") || strings.Contains(errStr, "api_key") ||
		strings.Contains(errStr, "401") || strings.Contains(errStr, "authentication"):
		return base + " — hint: set the API key env var or configure api_key_env in .nuntius.toml"
	case strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "timeout"):
		return base + " — hint: check your network connection and retry"
	case strings.Contains(errStr, "no upstream"):
		return base + " — hint: run 'git push --set-upstream origin <branch>'"
	}

	return base
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

// UpdateNotice returns the version update notice, if any.
func (m AppModel) UpdateNotice() string {
	return m.updateNotice
}

// checkVersionCmd returns a Cmd that checks for a newer version in the background.
func checkVersionCmd(currentVersion, buildDate string) tea.Cmd {
	return func() tea.Msg {
		result := config.CheckForUpdate(currentVersion, buildDate)
		if result != nil && result.UpdateAvailable {
			return events.UpdateAvailableMsg{
				Current: result.Current,
				Latest:  result.LatestTag,
			}
		}
		return nil
	}
}
