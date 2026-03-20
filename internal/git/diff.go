package git

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// DefaultMaxDiffBytes is the default maximum size for diff output (32KB).
// This keeps the diff within token limits for AI models like Haiku/Flash.
const DefaultMaxDiffBytes = 32768

// TruncationMarker is appended when the diff output is truncated to DefaultMaxDiffBytes.
const TruncationMarker = "\n... (truncated)"

// diffGitHeaderRE matches "diff --git a/<path> b/<dest>" lines, capturing the
// destination path (b/ side) — the post-rename name for renamed files.
var diffGitHeaderRE = regexp.MustCompile(`^diff --git a/.+ b/(.+)$`)

// diffPlusHeaderRE matches "+++ b/<path>" lines used as a fallback for plain
// unified diffs that do not contain "diff --git" headers.
var diffPlusHeaderRE = regexp.MustCompile(`^\+\+\+ b/(.+)$`)

// StagedDiff returns the unified diff for all staged changes.
// If nothing is staged, it returns an empty string (not an error).
// The diff is truncated to maxBytes if it exceeds that size.
// Pass 0 or a negative value for maxBytes to use DefaultMaxDiffBytes.
func StagedDiff(maxBytes int) (string, error) {
	return diffWith([]string{"--cached"}, maxBytes)
}

// Diff returns the unified diff for unstaged (working tree) changes.
// If there are no unstaged changes, it returns an empty string.
// The diff is truncated to maxBytes if it exceeds that size.
// Pass 0 or a negative value for maxBytes to use DefaultMaxDiffBytes.
func Diff(maxBytes int) (string, error) {
	return diffWith(nil, maxBytes)
}

// diffWith runs git diff with the given extra args and returns the truncated output.
func diffWith(extraArgs []string, maxBytes int) (string, error) {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxDiffBytes
	}

	args := append([]string{"diff", "--unified=3"}, extraArgs...)
	cmd := exec.Command("git", args...)
	applyEnv(cmd)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git diff failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("git diff failed: %w", err)
	}

	diff := string(out)

	if strings.TrimSpace(diff) == "" {
		return "", nil
	}

	return truncateDiff(diff, maxBytes), nil
}

// StageAll stages all changes (tracked and untracked) via git add -A.
func StageAll() error {
	cmd := exec.Command("git", "add", "-A")
	applyEnv(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git add failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// truncateDiff truncates the diff string to maxBytes, appending a
// truncation marker if it was shortened.
func truncateDiff(diff string, maxBytes int) string {
	if len(diff) <= maxBytes {
		return diff
	}

	// Keep the first maxBytes minus the marker length so the result
	// fits within the budget after appending the marker.
	cutoff := maxBytes - len(TruncationMarker)
	if cutoff < 0 {
		cutoff = 0
	}

	return diff[:cutoff] + TruncationMarker
}

// ParseDiffFileHeaders extracts unique file paths from unified diff output.
//
// It first scans for "diff --git a/<src> b/<dst>" headers, taking the
// destination path (b/ side). If no git-style headers are found it falls back
// to scanning "+++ b/<path>" lines so that plain unified diffs (e.g. from CI
// or custom tools) are also supported.
//
// The leading "a/" / "b/" prefixes are stripped. "/dev/null" is ignored.
func ParseDiffFileHeaders(diff string) []string {
	seen := make(map[string]struct{})
	var files []string

	for _, line := range strings.Split(diff, "\n") {
		if m := diffGitHeaderRE.FindStringSubmatch(line); len(m) == 2 {
			path := strings.TrimSpace(m[1])
			if path != "" && path != "/dev/null" {
				if _, ok := seen[path]; !ok {
					seen[path] = struct{}{}
					files = append(files, path)
				}
			}
		}
	}

	// Fallback: no "diff --git" headers found — try "+++ b/" lines.
	if len(files) == 0 {
		for _, line := range strings.Split(diff, "\n") {
			if m := diffPlusHeaderRE.FindStringSubmatch(line); len(m) == 2 {
				path := strings.TrimSpace(m[1])
				if path != "" && path != "/dev/null" {
					if _, ok := seen[path]; !ok {
						seen[path] = struct{}{}
						files = append(files, path)
					}
				}
			}
		}
	}

	return files
}
