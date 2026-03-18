package main

import (
	"os"
	"path/filepath"
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
		t.Errorf("expected exit code 1 for --help (ContinueOnError), got %d", code)
	}
}

func TestRunNoGitRepo(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(dir)

	code := run([]string{})
	if code != 1 {
		t.Errorf("expected exit code 1 when not in a git repo, got %d", code)
	}
}

func TestRunInGitRepo(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(dir)

	os.Mkdir(filepath.Join(dir, ".git"), 0755)

	code := run([]string{})
	if code != 0 {
		t.Errorf("expected exit code 0 in a git repo, got %d", code)
	}
}

func TestRunFlagOverrides(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(dir)

	os.Mkdir(filepath.Join(dir, ".git"), 0755)

	code := run([]string{"--provider", "gemini", "--model", "flash", "--auto-commit"})
	if code != 0 {
		t.Errorf("expected exit code 0 with flag overrides, got %d", code)
	}
}

func TestRunInvalidFlag(t *testing.T) {
	code := run([]string{"--invalid-flag"})
	if code != 1 {
		t.Errorf("expected exit code 1 for invalid flag, got %d", code)
	}
}
