package mcp

import (
	"strings"
	"testing"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
)

// TestExtractDiff verifies that extractDiff strips leading non-diff content and
// returns the diff starting at the first recognised diff header.
func TestExtractDiff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantEmpty  bool
		wantPrefix string
	}{
		{
			name:       "clean diff --git header returned as-is",
			input:      "diff --git a/foo.go b/foo.go\n--- a/foo.go\n+++ b/foo.go\n@@ -1 +1 @@\n-old\n+new",
			wantPrefix: "diff --git ",
		},
		{
			name:       "error preamble stripped before diff --git",
			input:      "✗ Permission denied\n\ndiff --git a/foo.go b/foo.go\n--- a/foo.go\n+++ b/foo.go\n@@ -1 +1 @@\n-old\n+new",
			wantPrefix: "diff --git ",
		},
		{
			name:       "error preamble stripped before --- header",
			input:      "shell: command not found\n--- a/x.go\n+++ b/x.go\n@@ -1 +1 @@\n-old\n+new",
			wantPrefix: "--- ",
		},
		{
			name:       "error preamble stripped before Index: header",
			input:      "some error\nIndex: foo.go\n===\n--- foo.go\n+++ foo.go",
			wantPrefix: "Index: ",
		},
		{
			name:      "plain error text with no diff markers returns empty",
			input:     "✗ Permission denied running git diff\nfatal: not a git repository",
			wantEmpty: true,
		},
		{
			name:      "empty string returns empty",
			input:     "",
			wantEmpty: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := extractDiff(tc.input)

			if tc.wantEmpty {
				if got != "" {
					t.Errorf("expected empty string, got: %q", got)
				}
				return
			}
			if !strings.HasPrefix(got, tc.wantPrefix) {
				t.Errorf("expected result to start with %q, got: %q", tc.wantPrefix, got)
			}
		})
	}
}

// TestBuildGenerateInput_ExternalDiffStripped calls buildGenerateInput directly
// and asserts that ExternalDiff starts with the diff header, not the error
// preamble that was prepended to the input.
func TestBuildGenerateInput_ExternalDiffStripped(t *testing.T) {
	t.Parallel()

	rawDiff := "✗ Permission denied\n\ndiff --git a/foo.go b/foo.go\n--- a/foo.go\n+++ b/foo.go\n@@ -1 +1 @@\n-old\n+new"

	req := mcpgo.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"diff_from": "provided",
		"diff":      rawDiff,
	}

	input, err := buildGenerateInput(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(input.ExternalDiff, "diff --git ") {
		t.Errorf("ExternalDiff should start with 'diff --git ', got: %q", input.ExternalDiff)
	}
	if strings.Contains(input.ExternalDiff, "Permission denied") {
		t.Errorf("ExternalDiff should not contain the error preamble, got: %q", input.ExternalDiff)
	}
}
