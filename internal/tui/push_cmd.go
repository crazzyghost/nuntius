package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/crazzyghost/nuntius/internal/events"
	"github.com/crazzyghost/nuntius/internal/git"
)

// pushCmd creates a tea.Cmd that runs git push.
func pushCmd(forceWithLease bool) tea.Cmd {
	return func() tea.Msg {
		result, err := git.Push(git.PushOptions{ForceWithLease: forceWithLease})
		if err != nil {
			return events.PushResultMsg{Err: err}
		}
		return events.PushResultMsg{Remote: result.Remote}
	}
}

// unpushedMsg reports whether there are unpushed commits.
type unpushedMsg struct {
	hasUnpushed bool
}

// checkUnpushedCmd checks if the current branch has unpushed commits.
func checkUnpushedCmd() tea.Cmd {
	return func() tea.Msg {
		return unpushedMsg{hasUnpushed: git.HasUnpushedCommits()}
	}
}
