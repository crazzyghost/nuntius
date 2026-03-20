package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	mcpgo "github.com/mark3labs/mcp-go/mcp"

	"github.com/crazzyghost/nuntius/internal/engine"
	"github.com/crazzyghost/nuntius/internal/git"
)

func (s *Server) handleGenerateMessage(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	input, err := buildGenerateInput(req)
	if err != nil {
		return mcpgo.NewToolResultError(err.Error()), nil
	}

	msg, files, err := engine.Generate(ctx, s.cfg, s.provider, input)
	if err != nil {
		return mcpgo.NewToolResultError(fmt.Sprintf("generation failed: %v", err)), nil
	}

	out, _ := json.Marshal(map[string]any{
		"message": msg,
		"files":   files,
	})
	return mcpgo.NewToolResultText(string(out)), nil
}

func (s *Server) handleCommit(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	message := req.GetString("message", "")
	if message == "" {
		return mcpgo.NewToolResultError("message is required"), nil
	}

	if err := s.gitOps.StageAll(); err != nil {
		return mcpgo.NewToolResultError(fmt.Sprintf("staging failed: %v", err)), nil
	}

	result, err := s.gitOps.Commit(message)
	if err != nil {
		return mcpgo.NewToolResultError(fmt.Sprintf("commit failed: %v", err)), nil
	}

	out, _ := json.Marshal(map[string]any{
		"hash": result.Hash,
	})
	return mcpgo.NewToolResultText(string(out)), nil
}

func (s *Server) handlePush(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	force := req.GetBool("force", false)
	setUpstreamParam := req.GetBool("set_upstream", false)

	hasUpstream, err := s.gitOps.HasUpstream()
	if err != nil {
		return mcpgo.NewToolResultError(fmt.Sprintf("checking upstream: %v", err)), nil
	}

	opts := git.PushOptions{
		ForceWithLease: force,
		SetUpstream:    setUpstreamParam || !hasUpstream,
	}

	pushResult, err := s.gitOps.Push(opts)
	if err != nil {
		return mcpgo.NewToolResultError(fmt.Sprintf("push failed: %v", err)), nil
	}

	out, _ := json.Marshal(map[string]any{
		"remote": pushResult.Remote,
		"branch": pushResult.Branch,
	})
	return mcpgo.NewToolResultText(string(out)), nil
}

func (s *Server) handleGenerateAndCommit(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	input, err := buildGenerateInput(req)
	if err != nil {
		return mcpgo.NewToolResultError(err.Error()), nil
	}

	msg, files, err := engine.Generate(ctx, s.cfg, s.provider, input)
	if err != nil {
		return mcpgo.NewToolResultError(fmt.Sprintf("generation failed: %v", err)), nil
	}

	if err := s.gitOps.StageAll(); err != nil {
		return mcpgo.NewToolResultError(fmt.Sprintf("staging failed: %v", err)), nil
	}

	commitResult, err := s.gitOps.Commit(msg)
	if err != nil {
		return mcpgo.NewToolResultError(fmt.Sprintf("commit failed: %v", err)), nil
	}

	out, _ := json.Marshal(map[string]any{
		"message": msg,
		"hash":    commitResult.Hash,
		"files":   files,
	})
	return mcpgo.NewToolResultText(string(out)), nil
}

func (s *Server) handleStatus(_ context.Context, _ mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	files, err := s.gitOps.Status()
	if err != nil {
		return mcpgo.NewToolResultError(fmt.Sprintf("status failed: %v", err)), nil
	}

	type fileEntry struct {
		Path   string `json:"path"`
		Status string `json:"status"`
		Staged bool   `json:"staged"`
	}

	entries := make([]fileEntry, len(files))
	for i, f := range files {
		entries[i] = fileEntry{Path: f.Path, Status: f.Status, Staged: f.Staged}
	}

	out, _ := json.Marshal(map[string]any{
		"files": entries,
	})
	return mcpgo.NewToolResultText(string(out)), nil
}

// extractDiff returns the portion of s starting from the first line that looks
// like a unified diff header, stripping any leading non-diff content (e.g. error
// output prepended by an agent). Returns an empty string if no diff markers are found.
func extractDiff(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "diff --git ") ||
			strings.HasPrefix(line, "--- ") ||
			strings.HasPrefix(line, "+++ ") ||
			strings.HasPrefix(line, "@@ ") ||
			strings.HasPrefix(line, "Index: ") {
			return strings.Join(lines[i:], "\n")
		}
	}
	return ""
}

// buildGenerateInput maps the MCP request arguments to an engine.GenerateInput.
// Returns an error string if the arguments are invalid.
func buildGenerateInput(req mcpgo.CallToolRequest) (engine.GenerateInput, error) {
	diffFrom := req.GetString("diff_from", "auto")
	input := engine.GenerateInput{}

	switch diffFrom {
	case "provided":
		d := req.GetString("diff", "")
		if d == "" {
			return input, fmt.Errorf("diff is required when diff_from=provided")
		}
		d = extractDiff(d) // strip any leading non-diff content
		if d == "" {
			return input, fmt.Errorf("diff_from=provided but the supplied diff does not appear to be a valid unified diff")
		}
		if len(d) > git.DefaultMaxDiffBytes {
			cutoff := git.DefaultMaxDiffBytes - len(git.TruncationMarker)
			if cutoff < 0 {
				cutoff = 0
			}
			d = d[:cutoff] + git.TruncationMarker
		}
		input.Source = engine.DiffSourceExternal
		input.ExternalDiff = d
	case "staged":
		input.Source = engine.DiffSourceStaged
	default: // auto
		input.Source = engine.DiffSourceAuto
	}

	return input, nil
}
