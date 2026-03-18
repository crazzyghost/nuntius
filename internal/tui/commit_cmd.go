package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/crazzyghost/nuntius/internal/events"
	"github.com/crazzyghost/nuntius/internal/git"
)

// commitCmd creates a tea.Cmd that stages all changes and then commits.
func commitCmd(message string) tea.Cmd {
	return func() tea.Msg {
		// Stage all changes before committing.
		if err := git.StageAll(); err != nil {
			return events.CommitResultMsg{Err: err}
		}

		result, err := git.Commit(message)
		if err != nil {
			return events.CommitResultMsg{Err: err}
		}
		return events.CommitResultMsg{Hash: result.Hash}
	}
}

// refreshStatusCmd creates a tea.Cmd that fetches fresh file status.
func refreshStatusCmd() tea.Cmd {
	return func() tea.Msg {
		files, err := git.Status()
		if err != nil {
			return events.FilesChangedMsg{Files: nil}
		}
		return events.FilesChangedMsg{Files: files}
	}
}
