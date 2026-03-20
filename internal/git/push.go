package git

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// PushOptions configures the behavior of a git push operation.
type PushOptions struct {
	// ForceWithLease uses --force-with-lease instead of a normal push.
	// This is safer than bare --force as it prevents overwriting others' work.
	ForceWithLease bool
	// SetUpstream runs `git push --set-upstream origin <branch>` for new branches
	// that have no remote tracking branch configured.
	SetUpstream bool
}

// PushResult holds the outcome of a successful git push.
type PushResult struct {
	// Remote is the remote name that was pushed to (e.g. "origin").
	Remote string
	// Branch is the branch that was pushed.
	Branch string
	// SetUpstream reports whether --set-upstream was used.
	SetUpstream bool
}

// CurrentBranch returns the name of the currently checked-out branch.
// Returns an empty string on error.
func CurrentBranch() string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// HasUpstream returns true if the current branch has a configured remote
// tracking branch. Returns (false, nil) when no upstream is set.
func HasUpstream() (bool, error) {
	branch := CurrentBranch()
	if branch == "" {
		return false, nil
	}
	cmd := exec.Command("git", "config", fmt.Sprintf("branch.%s.remote", branch))
	out, err := cmd.Output()
	if err != nil {
		// Exit code 1 means the config key doesn't exist — not a fatal error.
		return false, nil
	}
	return strings.TrimSpace(string(out)) != "", nil
}

// Push executes `git push` and returns the result.
// When opts.ForceWithLease is true, uses `--force-with-lease` (never bare `--force`).
// When opts.SetUpstream is true, uses `--set-upstream origin <branch>`.
// Returns an error with the remote's rejection message if push fails.
func Push(opts PushOptions) (PushResult, error) {
	args := BuildPushArgs(opts)

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
		Remote:      remote,
		Branch:      branch,
		SetUpstream: opts.SetUpstream,
	}, nil
}

// parseCurrentRemoteBranch returns the current remote and branch name
// by inspecting the symbolic ref and remote tracking branch.
func parseCurrentRemoteBranch() (remote, branch string) {
	branch = CurrentBranch()
	if branch == "" {
		branch = "unknown"
	}

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
	if opts.SetUpstream {
		branch := CurrentBranch()
		if branch == "" {
			branch = "HEAD"
		}
		args = append(args, "--set-upstream", "origin", branch)
	}
	return args
}

// UnpushedCount returns the number of commits on the current branch
// that haven't been pushed to its upstream tracking branch.
// Returns 0 if there are no unpushed commits or on error.
func UnpushedCount() int {
	remote, branch := parseCurrentRemoteBranch()
	upstream := remote + "/" + branch

	// Check if upstream ref exists.
	checkCmd := exec.Command("git", "rev-parse", "--verify", upstream)
	if err := checkCmd.Run(); err != nil {
		// No upstream — count all commits on this branch.
		logCmd := exec.Command("git", "rev-list", "--count", "HEAD")
		out, err := logCmd.Output()
		if err != nil {
			return 0
		}
		n, _ := strconv.Atoi(strings.TrimSpace(string(out)))
		return n
	}

	cmd := exec.Command("git", "rev-list", "--count", upstream+"..HEAD")
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return n
}
