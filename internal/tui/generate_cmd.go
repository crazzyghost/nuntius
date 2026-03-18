package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/crazzyghost/nuntius/internal/ai"
	"github.com/crazzyghost/nuntius/internal/events"
	"github.com/crazzyghost/nuntius/internal/git"
)

// generateCmd creates a tea.Cmd that runs the full generate pipeline:
// get status → get diff (staged + unstaged) → build prompt → call AI provider.
func generateCmd(provider ai.Provider, conventions string) tea.Cmd {
	return func() tea.Msg {
		// Get file list.
		files, err := git.Status()
		if err != nil {
			return events.ErrorMsg{Source: "generate", Err: err}
		}

		// Collect diffs from both staged and unstaged changes.
		stagedDiff, err := git.StagedDiff(0)
		if err != nil {
			return events.ErrorMsg{Source: "generate", Err: err}
		}

		unstagedDiff, err := git.Diff(0)
		if err != nil {
			return events.ErrorMsg{Source: "generate", Err: err}
		}

		diff := stagedDiff
		if unstagedDiff != "" {
			if diff != "" {
				diff += "\n"
			}
			diff += unstagedDiff
		}

		if diff == "" {
			return events.ErrorMsg{Source: "generate", Err: fmt.Errorf("no changes to generate a message for")}
		}

		// Build file name list for prompt (all changed files).
		var fileNames []string
		for _, f := range files {
			fileNames = append(fileNames, f.Path)
		}

		// Build the prompt request.
		req := ai.MessageRequest{
			Diff:        diff,
			FileList:    fileNames,
			Conventions: conventions,
		}

		// Call the AI provider.
		msg, err := provider.GenerateCommitMessage(context.Background(), req)
		if err != nil {
			return events.ErrorMsg{Source: "generate", Err: err}
		}

		return events.MessageReadyMsg{Message: ai.CleanMessage(msg)}
	}
}
