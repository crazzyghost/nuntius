package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// CommitResult holds the outcome of a successful git commit.
type CommitResult struct {
	// Hash is the short commit SHA.
	Hash string
	// Summary is the first line of the commit output.
	Summary string
}

// commitHashRegexp matches the short hash in git commit output, e.g.:
//
//	[main abc1234] commit message here
var commitHashRegexp = regexp.MustCompile(`\[.+\s+([0-9a-f]+)\]`)

// Commit creates a git commit with the provided message and returns
// the resulting commit hash. Multi-line messages are supported by
// piping through stdin via `git commit -F -`.
//
// Returns an error if there are no staged changes or the commit fails.
func Commit(message string) (CommitResult, error) {
	if strings.TrimSpace(message) == "" {
		return CommitResult{}, fmt.Errorf("commit message cannot be empty")
	}

	// Use `git commit -F -` to read the message from stdin.
	// This handles multi-line messages and avoids shell escaping issues.
	cmd := exec.Command("git", "commit", "-F", "-")
	applyEnv(cmd)
	cmd.Stdin = bytes.NewBufferString(message)

	out, err := cmd.CombinedOutput()
	output := string(out)

	if err != nil {
		// Check for "nothing to commit" scenario
		if strings.Contains(output, "nothing to commit") ||
			strings.Contains(output, "no changes added to commit") {
			return CommitResult{}, fmt.Errorf("no staged changes to commit")
		}
		return CommitResult{}, fmt.Errorf("git commit failed: %s", strings.TrimSpace(output))
	}

	hash := parseCommitHash(output)
	summary := firstLine(output)

	return CommitResult{
		Hash:    hash,
		Summary: summary,
	}, nil
}

// parseCommitHash extracts the short commit hash from git commit output.
// Example line: "[main abc1234] feat: add new feature"
func parseCommitHash(output string) string {
	matches := commitHashRegexp.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// firstLine returns the first non-empty line of the string.
func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}
