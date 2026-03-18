package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/crazzyghost/nuntius/internal/git"
)

// waitForFileChange bridges the git.Watcher channel into Bubble Tea's
// Cmd system. Each call blocks until a FilesChangedMsg arrives, then
// returns it as a tea.Msg.
func waitForFileChange(watcher *git.Watcher) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-watcher.Events
		if !ok {
			return nil
		}
		return msg
	}
}
