package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// PushOptions configures the behavior of a git push operation.
type PushOptions struct {
	// ForceWithLease uses --force-with-lease instead of a normal push.
	// This is safer than bare --force as it prevents overwriting others' work.
	ForceWithLease bool
}

// PushResult holds the outcome of a successful git push.
type PushResult struct {
	// Remote is the remote name that was pushed to (e.g. "origin").
	Remote string
	// Branch is the branch that was pushed.
	Branch string
}

// Push executes `git push` and returns the result.
// When opts.ForceWithLease is true, uses `--force-with-lease` (never bare `--force`).
// Returns an error with the remote's rejection message if push fails.
func Push(opts PushOptions) (PushResult, error) {
	args := []string{"push"}
	if opts.ForceWithLease {
		args = append(args, "--force-with-lease")
	}

	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	output := string(out)

	if err != nil {
		// Detect common error patterns and provide helpful messages
		if strings.Contains(output, "no upstream branch") ||
			strings.Contains(output, "has no upstream branch") {
			return PushResult{}, fmt.Errorf("no upstream branch configured: use 'git push --set-upstream origin <branch>' to set one")
		}

		if strings.Contains(output, "rejected") {
			return PushResult{}, fmt.Errorf("push rejected by remote: %s", strings.TrimSpace(output))
		}

		return PushResult{}, fmt.Errorf("git push failed: %s", strings.TrimSpace(output))
	}

	remote, branch := parseCurrentRemoteBranch()

	return PushResult{
		Remote: remote,
		Branch: branch,
	}, nil
}

// parseCurrentRemoteBranch returns the current remote and branch name
// by inspecting the symbolic ref and remote tracking branch.
func parseCurrentRemoteBranch() (remote, branch string) {
	// Get current branch name
	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	branchOut, err := branchCmd.Output()
	if err != nil {
		return "origin", "unknown"
	}
	branch = strings.TrimSpace(string(branchOut))

	// Get remote for the current branch
	remoteCmd := exec.Command("git", "config", fmt.Sprintf("branch.%s.remote", branch))
	remoteOut, err := remoteCmd.Output()
	if err != nil {
		return "origin", branch
	}
	remote = strings.TrimSpace(string(remoteOut))

	if remote == "" {
		remote = "origin"
	}

	return remote, branch
}

// BuildPushArgs constructs the argument list for `git push` based on options.
// Exported for testing command construction.
func BuildPushArgs(opts PushOptions) []string {
	args := []string{"push"}
	if opts.ForceWithLease {
		args = append(args, "--force-with-lease")
	}
	return args
}

// HasUnpushedCommits returns true if the current branch has commits
// that haven't been pushed to its upstream tracking branch.
func HasUnpushedCommits() bool {
	remote, branch := parseCurrentRemoteBranch()
	upstream := remote + "/" + branch

	// Check if upstream ref exists.
	checkCmd := exec.Command("git", "rev-parse", "--verify", upstream)
	if err := checkCmd.Run(); err != nil {
		// No upstream — treat local-only branch as having unpushed commits
		// if it has any commits at all.
		logCmd := exec.Command("git", "rev-list", "--count", "HEAD")
		out, err := logCmd.Output()
		if err != nil {
			return false
		}
		return strings.TrimSpace(string(out)) != "0"
	}

	cmd := exec.Command("git", "rev-list", "--count", upstream+"..HEAD")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != "0"
}
