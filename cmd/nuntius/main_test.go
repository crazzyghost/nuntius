package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/crazzyghost/nuntius/internal/cli"
)

func TestRunVersion(t *testing.T) {
	code := run([]string{"--version"})
	if code != 0 {
		t.Errorf("expected exit code 0 for --version, got %d", code)
	}
}

func TestRunHelp(t *testing.T) {
	code := run([]string{"--help"})
	if code != 1 {
		t.Errorf("expected exit code 1 for --help, got %d", code)
	}
}

func TestHelpOutputUsesDoubleDashFlags(t *testing.T) {
	var stderr bytes.Buffer
	flags := newFlagSet(&stderr)
	flags.Usage()

	output := stderr.String()
	for _, want := range []string{
		"--version", "--agent", "--model",
		"--generate", "--auto-commit", "--auto-push", "--no-update-check",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help output to contain %q, got:\n%s", want, output)
		}
	}
	// --provider is hidden; it must not appear in help.
	if strings.Contains(output, "--provider") {
		t.Fatalf("expected --provider to be hidden, but it appeared in help output")
	}
}

func TestHelpOutputContainsExamples(t *testing.T) {
	var stderr bytes.Buffer
	flags := newFlagSet(&stderr)
	flags.Usage()

	output := stderr.String()
	if !strings.Contains(output, "Examples:") {
		t.Error("expected help output to contain an Examples section")
	}
	if !strings.Contains(output, "nuntius -g") {
		t.Error("expected examples to include 'nuntius -g'")
	}
}

func TestRunNoGitRepo(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	code := run([]string{})
	if code != 1 {
		t.Errorf("expected exit code 1 when not in a git repo, got %d", code)
	}
}

func mkGitDir(t *testing.T, dir string) {
	t.Helper()
	gitDir := filepath.Join(dir, ".git")
	for _, sub := range []string{gitDir, filepath.Join(gitDir, "refs"), filepath.Join(gitDir, "refs", "heads")} {
		if err := os.Mkdir(sub, 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSetupInGitRepo(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	mkGitDir(t, dir)

	result, exitCode, shouldLaunch := setup([]string{})
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !shouldLaunch {
		t.Error("expected shouldLaunch to be true")
	}
	if result == nil {
		t.Fatal("expected non-nil setup result")
	}

	result.cancel()
	result.watcher.Stop()
}

func TestSetupFlagOverrides(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	mkGitDir(t, dir)

	result, exitCode, shouldLaunch := setup([]string{"--agent", "gemini", "--model", "flash", "--auto-commit"})
	if exitCode != 0 {
		t.Errorf("expected exit code 0 with flag overrides, got %d", exitCode)
	}
	if !shouldLaunch {
		t.Error("expected shouldLaunch to be true")
	}
	if result == nil {
		t.Fatal("expected non-nil setup result")
	}
	if result.cfg.AI.Provider != "gemini" {
		t.Errorf("expected provider 'gemini', got %q", result.cfg.AI.Provider)
	}
	if result.cfg.AI.Model != "flash" {
		t.Errorf("expected model 'flash', got %q", result.cfg.AI.Model)
	}
	if !result.cfg.Behavior.AutoCommit {
		t.Error("expected auto-commit to be true")
	}

	result.cancel()
	result.watcher.Stop()
}

func TestSetupDeprecatedProviderFlag(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	mkGitDir(t, dir)

	// --provider (deprecated) must still work.
	result, exitCode, shouldLaunch := setup([]string{"--provider", "gemini"})
	if exitCode != 0 {
		t.Errorf("expected exit code 0 with deprecated --provider flag, got %d", exitCode)
	}
	if !shouldLaunch {
		t.Error("expected shouldLaunch to be true")
	}
	if result == nil {
		t.Fatal("expected non-nil setup result")
	}
	if result.cfg.AI.Provider != "gemini" {
		t.Errorf("expected provider 'gemini' via deprecated --provider, got %q", result.cfg.AI.Provider)
	}

	result.cancel()
	result.watcher.Stop()
}

func TestSetupAgentShortFlag(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	mkGitDir(t, dir)

	result, exitCode, shouldLaunch := setup([]string{"-a", "claude"})
	if exitCode != 0 {
		t.Errorf("expected exit code 0 with -a flag, got %d", exitCode)
	}
	if !shouldLaunch {
		t.Error("expected shouldLaunch to be true")
	}
	if result == nil {
		t.Fatal("expected non-nil setup result")
	}
	if result.cfg.AI.Provider != "claude" {
		t.Errorf("expected provider 'claude' via -a, got %q", result.cfg.AI.Provider)
	}

	result.cancel()
	result.watcher.Stop()
}

func TestRunInvalidFlag(t *testing.T) {
	code := run([]string{"--invalid-flag"})
	if code != 1 {
		t.Errorf("expected exit code 1 for invalid flag, got %d", code)
	}
}

func TestValidateHeadlessCombination(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		generate bool
		commit   bool
		push     bool
		wantErr  bool
	}{
		{"-g only", true, false, false, false},
		{"-gc", true, true, false, false},
		{"-gcp", true, true, true, false},
		{"-p only", false, false, true, false},
		{"-c without -g: invalid", false, true, false, true},
		{"-gp without -c: invalid", true, false, true, true},
		{"-cp without -g: invalid", false, true, true, true},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateHeadlessCombination(cli.Actions{
				Generate: tc.generate,
				Commit:   tc.commit,
				Push:     tc.push,
			})
			if tc.wantErr && err == nil {
				t.Errorf("expected error for combo generate=%v commit=%v push=%v", tc.generate, tc.commit, tc.push)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error for combo generate=%v commit=%v push=%v: %v", tc.generate, tc.commit, tc.push, err)
			}
		})
	}
}

// TestHeadlessModeNotTriggeredByConfig verifies that config.toml auto_commit=true
// does NOT trigger headless mode (only explicit CLI flags do).
func TestHeadlessModeNotTriggeredByConfig(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	mkGitDir(t, dir)

	// Without any -g/-c/-p flags, setup() returns shouldLaunch=true (TUI path).
	result, exitCode, shouldLaunch := setup([]string{})
	if exitCode != 0 || !shouldLaunch || result == nil {
		t.Errorf("expected TUI path without headless flags, got exitCode=%d shouldLaunch=%v", exitCode, shouldLaunch)
		return
	}
	result.cancel()
	result.watcher.Stop()
}
