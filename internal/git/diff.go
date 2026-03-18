package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// DefaultMaxDiffBytes is the default maximum size for diff output (32KB).
// This keeps the diff within token limits for AI models like Haiku/Flash.
const DefaultMaxDiffBytes = 32768

// truncationMarker is appended when the diff is truncated.
const truncationMarker = "\n... (truncated)"

// StagedDiff returns the unified diff for all staged changes.
// If nothing is staged, it returns an empty string (not an error).
// The diff is truncated to maxBytes if it exceeds that size.
// Pass 0 or a negative value for maxBytes to use DefaultMaxDiffBytes.
func StagedDiff(maxBytes int) (string, error) {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxDiffBytes
	}

	cmd := exec.Command("git", "diff", "--cached", "--unified=3")
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git diff failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("git diff failed: %w", err)
	}

	diff := string(out)

	// Nothing staged → empty diff is not an error
	if strings.TrimSpace(diff) == "" {
		return "", nil
	}

	return truncateDiff(diff, maxBytes), nil
}

// truncateDiff truncates the diff string to maxBytes, appending a
// truncation marker if it was shortened.
func truncateDiff(diff string, maxBytes int) string {
	if len(diff) <= maxBytes {
		return diff
	}

	// Keep the first maxBytes minus the marker length so the result
	// fits within the budget after appending the marker.
	cutoff := maxBytes - len(truncationMarker)
	if cutoff < 0 {
		cutoff = 0
	}

	return diff[:cutoff] + truncationMarker
}
