package git_test

import (
	"os/exec"
	"slices"
	"testing"

	"github.com/crazzyghost/nuntius/internal/git"
)

func TestSetNonInteractive_EnvVarsApplied(t *testing.T) {
	defer git.SetNonInteractive(false)

	git.SetNonInteractive(true)

	cmd := exec.Command("git", "version")
	git.ApplyEnvForTest(cmd)

	hasTerminalPrompt := slices.ContainsFunc(cmd.Env, func(s string) bool {
		return s == "GIT_TERMINAL_PROMPT=0"
	})
	hasSSHCommand := slices.ContainsFunc(cmd.Env, func(s string) bool {
		return s == "GIT_SSH_COMMAND=ssh -o BatchMode=yes"
	})

	if !hasTerminalPrompt {
		t.Error("expected GIT_TERMINAL_PROMPT=0 in env when non-interactive mode is enabled")
	}
	if !hasSSHCommand {
		t.Error("expected GIT_SSH_COMMAND=ssh -o BatchMode=yes in env when non-interactive mode is enabled")
	}
}

func TestSetNonInteractive_EnvVarsNotApplied(t *testing.T) {
	git.SetNonInteractive(false)

	cmd := exec.Command("git", "version")
	git.ApplyEnvForTest(cmd)

	if cmd.Env != nil {
		t.Errorf("expected nil cmd.Env when non-interactive mode is disabled, got %v", cmd.Env)
	}
}
