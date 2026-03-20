// Package git provides Git abstraction for repository operations
// including watching, diffing, committing, and pushing.
package git

import (
	"os"
	"os/exec"
)

// nonInteractive is a package-level toggle for non-interactive mode.
// When true, git subprocesses are prevented from prompting for TTY input.
var nonInteractive bool

// SetNonInteractive enables or disables non-interactive mode for all git commands.
// When enabled, git subprocesses are prevented from prompting for TTY input.
// This should be enabled in headless/agent mode.
func SetNonInteractive(enabled bool) {
	nonInteractive = enabled
}

// applyEnv configures the exec.Cmd with non-interactive env vars if enabled.
// It appends to the current process environment, preserving existing variables.
func applyEnv(cmd *exec.Cmd) {
	if !nonInteractive {
		return
	}
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_SSH_COMMAND=ssh -o BatchMode=yes",
	)
}
