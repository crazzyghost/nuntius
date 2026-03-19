package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/crazzyghost/nuntius/internal/events"
)

// viewMode determines what the viewport is displaying.
type viewMode int

const (
	fileListMode viewMode = iota
	messageMode
)

// ViewportModel is the scrollable viewport panel (Panel A) that alternates
// between showing the changed file list and the generated commit message.
type ViewportModel struct {
	viewport viewport.Model
	mode     viewMode
	files    []events.FileStatus
	message  string
	loading  bool
	spinner  spinner.Model
	ready    bool
	width    int
	height   int
}

// NewViewport creates a new ViewportModel.
func NewViewport() ViewportModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return ViewportModel{
		mode:    fileListMode,
		spinner: s,
	}
}

// Init returns the initial command for the viewport (spinner tick).
func (m ViewportModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// SetSize updates the viewport dimensions.
func (m *ViewportModel) SetSize(width, height int) {
	m.width = width
	m.height = height

	if !m.ready {
		m.viewport = viewport.New(width, height)
		m.viewport.MouseWheelEnabled = true
		m.ready = true
	} else {
		m.viewport.Width = width
		m.viewport.Height = height
	}

	m.updateContent()
}

// Update handles messages for the viewport.
func (m ViewportModel) Update(msg tea.Msg) (ViewportModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case events.FilesChangedMsg:
		m.files = msg.Files
		if m.mode == fileListMode {
			m.updateContent()
		}

	case events.MessageReadyMsg:
		m.message = msg.Message
		m.loading = false
		m.mode = messageMode
		m.updateContent()

	case events.GenerateRequestedMsg:
		m.loading = true
		m.updateContent()

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
			m.updateContent()
		}
	}

	// Forward to the inner viewport for scrolling.
	if m.ready {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the viewport panel.
func (m ViewportModel) View() string {
	if !m.ready {
		return "Initializing..."
	}

	header := m.headerView()
	content := m.viewport.View()
	footer := m.footerView()

	return fmt.Sprintf("%s\n%s\n%s", header, content, footer)
}

// headerView renders the panel title.
func (m ViewportModel) headerView() string {
	var title string
	switch {
	case m.loading:
		title = Title.Render("📂 Changed Files")
	case m.mode == messageMode:
		title = Title.Render("📝 Commit Message")
	default:
		title = Title.Render("📂 Changed Files")
	}
	return title
}

// footerView shows scroll position.
func (m ViewportModel) footerView() string {
	percent := m.viewport.ScrollPercent() * 100
	return StatusMuted.Render(fmt.Sprintf(" %3.0f%%", percent))
}

// updateContent rebuilds the viewport content based on current mode.
func (m *ViewportModel) updateContent() {
	if !m.ready {
		return
	}

	var content string
	switch {
	case m.loading:
		content = m.renderLoading()
	case m.mode == messageMode:
		content = m.renderMessage()
	default:
		content = m.renderFileList()
	}

	m.viewport.SetContent(content)
}

// renderFileList renders the changed files with status indicators.
func (m *ViewportModel) renderFileList() string {
	if len(m.files) == 0 {
		return StatusMuted.Render("  No changes detected. Waiting for file changes...")
	}

	var b strings.Builder
	var staged, unstaged []events.FileStatus

	for _, f := range m.files {
		if f.Staged {
			staged = append(staged, f)
		} else {
			unstaged = append(unstaged, f)
		}
	}

	if len(staged) > 0 {
		b.WriteString(StagedFile.Render("Staged Changes"))
		b.WriteString("\n")
		for _, f := range staged {
			icon := StatusIcon(f.Status)
			line := fmt.Sprintf("  %s  %s", icon, f.Path)
			b.WriteString(StagedFile.Render(line))
			b.WriteString("\n")
		}
	}

	if len(unstaged) > 0 {
		if len(staged) > 0 {
			b.WriteString("\n")
		}
		b.WriteString(UnstagedFile.Render("Unstaged Changes"))
		b.WriteString("\n")
		for _, f := range unstaged {
			icon := StatusIcon(f.Status)
			line := fmt.Sprintf("  %s  %s", icon, f.Path)
			style := UnstagedFile
			if f.Status == "untracked" {
				style = UntrackedFile
			}
			b.WriteString(style.Render(line))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// renderMessage renders the generated commit message.
func (m *ViewportModel) renderMessage() string {
	if m.message == "" {
		return StatusMuted.Render("  No message generated yet.")
	}
	return m.message
}

// renderLoading renders the loading state.
func (m *ViewportModel) renderLoading() string {
	return fmt.Sprintf("\n\n  %s Generating commit message...", m.spinner.View())
}

// Mode returns the current display mode.
func (m ViewportModel) Mode() viewMode {
	return m.mode
}

// HasMessage returns true if a commit message has been generated.
func (m ViewportModel) HasMessage() bool {
	return m.message != ""
}

// Message returns the generated commit message.
func (m ViewportModel) Message() string {
	return m.message
}

// Files returns the current file list.
func (m ViewportModel) Files() []events.FileStatus {
	return m.files
}

// HasChanges returns true if there are any changes (staged or unstaged).
func (m ViewportModel) HasChanges() bool {
	return len(m.files) > 0
}

// HasStagedChanges returns true if any files are staged.
func (m ViewportModel) HasStagedChanges() bool {
	for _, f := range m.files {
		if f.Staged {
			return true
		}
	}
	return false
}

// SwitchToFileList switches the viewport to file list mode.
func (m *ViewportModel) SwitchToFileList() {
	m.mode = fileListMode
	m.updateContent()
}

// ClearMessage clears the generated message and switches to file list mode.
func (m *ViewportModel) ClearMessage() {
	m.message = ""
	m.mode = fileListMode
	m.updateContent()
}

// SwitchToMessage switches the viewport to message mode.
func (m *ViewportModel) SwitchToMessage() {
	if m.message != "" {
		m.mode = messageMode
		m.updateContent()
	}
}
