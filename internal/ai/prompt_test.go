package ai

import (
	"strings"
	"testing"
)

func TestBuildPrompt_ConventionalCommits(t *testing.T) {
	req := MessageRequest{
		Diff:        "--- a/main.go\n+++ b/main.go\n@@ -1 +1 @@\n-old\n+new",
		FileList:    []string{"main.go"},
		Conventions: "conventional",
	}
	prompt := BuildPrompt(req)

	checks := []string{
		"commit message generator",
		"conventional",
		"`type(scope): subject`",
		"CHANGED FILES:",
		"main.go",
		"DIFF:",
		"-old",
		"+new",
		"Subject line max 72 characters",
		"imperative mood",
	}
	for _, want := range checks {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

func TestBuildPrompt_Gitmoji(t *testing.T) {
	req := MessageRequest{
		Diff:        "some diff",
		Conventions: "gitmoji",
	}
	prompt := BuildPrompt(req)
	if !strings.Contains(prompt, "gitmoji emoji") {
		t.Error("gitmoji rules not included")
	}
}

func TestBuildPrompt_Angular(t *testing.T) {
	req := MessageRequest{
		Diff:        "some diff",
		Conventions: "angular",
	}
	prompt := BuildPrompt(req)
	if !strings.Contains(prompt, "Angular commit format") {
		t.Error("angular rules not included")
	}
	if !strings.Contains(prompt, "Scope is mandatory") {
		t.Error("angular mandatory scope rule not included")
	}
}

func TestBuildPrompt_EmptyDiff(t *testing.T) {
	req := MessageRequest{
		Diff:        "",
		FileList:    []string{"README.md"},
		Conventions: "conventional",
	}
	prompt := BuildPrompt(req)
	if strings.Contains(prompt, "DIFF:") {
		t.Error("DIFF section should not appear for empty diff")
	}
	if !strings.Contains(prompt, "README.md") {
		t.Error("file list should still appear")
	}
}

func TestBuildPrompt_NoFiles(t *testing.T) {
	req := MessageRequest{
		Diff:        "some diff",
		Conventions: "conventional",
	}
	prompt := BuildPrompt(req)
	if strings.Contains(prompt, "CHANGED FILES:") {
		t.Error("CHANGED FILES section should not appear when file list is empty")
	}
}

func TestBuildPrompt_DiffTruncation(t *testing.T) {
	// Create a huge diff that exceeds maxPromptBytes
	largeDiff := strings.Repeat("x", maxPromptBytes+1000)
	req := MessageRequest{
		Diff:        largeDiff,
		Conventions: "conventional",
	}
	prompt := BuildPrompt(req)
	if !strings.Contains(prompt, "[diff truncated") {
		t.Error("large diff should be truncated with notice")
	}
	if len(prompt) > maxPromptBytes+500 {
		t.Errorf("prompt too large: %d bytes", len(prompt))
	}
}

func TestBuildPrompt_UnknownConvention(t *testing.T) {
	req := MessageRequest{
		Diff:        "diff",
		Conventions: "unknown",
	}
	prompt := BuildPrompt(req)
	// Should still produce a valid prompt, just without convention-specific rules
	if !strings.Contains(prompt, "commit message generator") {
		t.Error("base prompt missing")
	}
}

func TestConventionRules_AllTypes(t *testing.T) {
	tests := []struct {
		convention string
		contains   string
	}{
		{"conventional", "type(scope): subject"},
		{"gitmoji", "gitmoji emoji"},
		{"angular", "Angular commit format"},
		{"unknown", ""},
	}
	for _, tt := range tests {
		t.Run(tt.convention, func(t *testing.T) {
			rules := conventionRules(tt.convention)
			if tt.contains == "" {
				if rules != "" {
					t.Errorf("expected empty rules for %q, got %q", tt.convention, rules)
				}
			} else if !strings.Contains(rules, tt.contains) {
				t.Errorf("rules for %q missing %q", tt.convention, tt.contains)
			}
		})
	}
}
