// Package engine provides the shared commit message generation pipeline
// used by the TUI, headless CLI, and MCP server.
package engine

import (
	"context"
	"fmt"

	"github.com/crazzyghost/nuntius/internal/ai"
	"github.com/crazzyghost/nuntius/internal/config"
	"github.com/crazzyghost/nuntius/internal/git"
)

// DiffSource controls where the diff input comes from.
type DiffSource int

const (
	// DiffSourceAuto collects both staged and unstaged changes (default TUI behavior).
	DiffSourceAuto DiffSource = iota
	// DiffSourceStaged collects only staged changes.
	DiffSourceStaged
	// DiffSourceExternal uses a pre-supplied diff string (stdin for CLI, provided for MCP).
	DiffSourceExternal
)

// GenerateInput holds parameters for Generate that vary by caller.
type GenerateInput struct {
	// Source controls where the diff comes from.
	Source DiffSource
	// ExternalDiff is the pre-supplied diff string. Only used when Source is DiffSourceExternal.
	ExternalDiff string
}

// Generate orchestrates commit message generation:
// diff acquisition → prompt building → AI provider call → message return.
//
// It returns the cleaned commit message and the list of affected file paths.
// Returns descriptive errors for common failure modes.
func Generate(ctx context.Context, cfg config.Config, provider ai.Provider, input GenerateInput) (string, []string, error) {
	if provider == nil {
		return "", nil, fmt.Errorf("no AI provider configured")
	}

	diff, files, err := collectDiff(input)
	if err != nil {
		return "", nil, err
	}

	if diff == "" {
		return "", nil, fmt.Errorf("no changes to generate a message for")
	}

	conventions := config.DetectConvention(cfg, 20)
	msg, err := provider.GenerateCommitMessage(ctx, ai.MessageRequest{
		Diff:        diff,
		FileList:    files,
		Conventions: conventions,
	})
	if err != nil {
		return "", nil, fmt.Errorf("AI generation failed: %w", err)
	}

	cleaned := ai.CleanMessage(msg)
	if cleaned == "" {
		return "", nil, fmt.Errorf("AI provider returned an empty message")
	}

	return cleaned, files, nil
}

// collectDiff gathers the diff and file list according to the source strategy.
func collectDiff(input GenerateInput) (string, []string, error) {
	switch input.Source {
	case DiffSourceExternal:
		if input.ExternalDiff == "" {
			return "", nil, fmt.Errorf("no diff provided")
		}
		return input.ExternalDiff, git.ParseDiffFileHeaders(input.ExternalDiff), nil

	case DiffSourceStaged:
		diff, err := git.StagedDiff(git.DefaultMaxDiffBytes)
		if err != nil {
			return "", nil, fmt.Errorf("getting staged diff: %w", err)
		}
		allFiles, _ := git.Status()
		var files []string
		for _, f := range allFiles {
			if f.Staged {
				files = append(files, f.Path)
			}
		}
		return diff, files, nil

	default: // DiffSourceAuto
		staged, err := git.StagedDiff(git.DefaultMaxDiffBytes)
		if err != nil {
			return "", nil, fmt.Errorf("getting staged diff: %w", err)
		}
		unstaged, err := git.Diff(git.DefaultMaxDiffBytes)
		if err != nil {
			return "", nil, fmt.Errorf("getting unstaged diff: %w", err)
		}
		untracked, err := git.UntrackedDiff(git.DefaultMaxDiffBytes)
		if err != nil {
			return "", nil, fmt.Errorf("getting untracked diff: %w", err)
		}

		diff := staged
		if unstaged != "" {
			if diff != "" {
				diff += "\n"
			}
			diff += unstaged
		}
		if untracked != "" {
			if diff != "" {
				diff += "\n"
			}
			diff += untracked
		}

		allFiles, _ := git.Status()
		var files []string
		for _, f := range allFiles {
			files = append(files, f.Path)
		}

		return diff, files, nil
	}
}
