package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	flags, _, _, _, _, _, _ := newFlagSet(&stderr)
	flags.Usage()

	output := stderr.String()
	for _, want := range []string{"--version", "--provider", "--model", "--auto-commit", "--auto-push", "--no-update-check"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help output to contain %q, got %q", want, output)
		}
	}
	for _, unwanted := range []string{"  -version", "  -provider", "  -model", "  -auto-commit", "  -auto-push", "  -no-update-check"} {
		if strings.Contains(output, unwanted) {
			t.Fatalf("expected help output not to contain %q, got %q", unwanted, output)
		}
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

func TestSetupInGitRepo(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Create a minimal .git directory structure for the watcher.
	gitDir := filepath.Join(dir, ".git")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(gitDir, "refs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(gitDir, "refs", "heads"), 0o755); err != nil {
		t.Fatal(err)
	}

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

	// Clean up watcher.
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

	gitDir := filepath.Join(dir, ".git")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(gitDir, "refs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(gitDir, "refs", "heads"), 0o755); err != nil {
		t.Fatal(err)
	}

	result, exitCode, shouldLaunch := setup([]string{"--provider", "gemini", "--model", "flash", "--auto-commit"})
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

func TestRunInvalidFlag(t *testing.T) {
	code := run([]string{"--invalid-flag"})
	if code != 1 {
		t.Errorf("expected exit code 1 for invalid flag, got %d", code)
	}
}
