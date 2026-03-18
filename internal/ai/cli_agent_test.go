package ai

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/crazzyghost/nuntius/internal/config"
)

func TestCLIAgent_NameAndMode(t *testing.T) {
	// Use a command that exists on all platforms
	cmd := "echo"
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	cfg := config.AIConfig{Provider: "test-cli"}
	agent, err := NewCLIAgent(cfg, cmd, nil)
	if err != nil {
		t.Fatalf("NewCLIAgent: %v", err)
	}
	if agent.Name() != "test-cli" {
		t.Errorf("Name() = %q, want %q", agent.Name(), "test-cli")
	}
	if agent.Mode() != ModeCLI {
		t.Errorf("Mode() = %q, want %q", agent.Mode(), ModeCLI)
	}
}

func TestCLIAgent_BinaryNotFound(t *testing.T) {
	cfg := config.AIConfig{Provider: "missing-cli"}
	_, err := NewCLIAgent(cfg, "nonexistent_binary_xyz", nil)
	if err == nil {
		t.Fatal("expected error for missing binary")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestCLIAgent_GenerateCommitMessage_Success(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	// Create a mock script that echoes a commit message
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "mock-ai")
	script := "#!/bin/sh\ncat - > /dev/null\necho 'feat: add mock feature'\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	cfg := config.AIConfig{Provider: "mock-ai"}
	agent, err := NewCLIAgent(cfg, scriptPath, nil)
	if err != nil {
		t.Fatalf("NewCLIAgent: %v", err)
	}

	msg, err := agent.GenerateCommitMessage(context.Background(), MessageRequest{
		Diff:        "some diff",
		Conventions: "conventional",
	})
	if err != nil {
		t.Fatalf("GenerateCommitMessage: %v", err)
	}
	if msg != "feat: add mock feature" {
		t.Errorf("msg = %q, want %q", msg, "feat: add mock feature")
	}
}

func TestCLIAgent_GenerateCommitMessage_NonZeroExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "fail-ai")
	script := "#!/bin/sh\necho 'something went wrong' >&2\nexit 1\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	cfg := config.AIConfig{Provider: "fail-ai"}
	agent, err := NewCLIAgent(cfg, scriptPath, nil)
	if err != nil {
		t.Fatalf("NewCLIAgent: %v", err)
	}

	_, err = agent.GenerateCommitMessage(context.Background(), MessageRequest{Diff: "diff"})
	if err == nil {
		t.Fatal("expected error for non-zero exit")
	}
	if !strings.Contains(err.Error(), "failed") {
		t.Errorf("error should mention failure: %v", err)
	}
	if !strings.Contains(err.Error(), "something went wrong") {
		t.Errorf("error should include stderr: %v", err)
	}
}

func TestCLIAgent_GenerateCommitMessage_EmptyOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "empty-ai")
	script := "#!/bin/sh\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	cfg := config.AIConfig{Provider: "empty-ai"}
	agent, err := NewCLIAgent(cfg, scriptPath, nil)
	if err != nil {
		t.Fatalf("NewCLIAgent: %v", err)
	}

	_, err = agent.GenerateCommitMessage(context.Background(), MessageRequest{Diff: "diff"})
	if err == nil {
		t.Fatal("expected error for empty output")
	}
	if !strings.Contains(err.Error(), "empty response") {
		t.Errorf("error should mention empty: %v", err)
	}
}

func TestCLIAgent_WithArgs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "args-ai")
	// Script that echoes its arguments
	script := "#!/bin/sh\necho \"fix: resolved with args: $@\"\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	cfg := config.AIConfig{Provider: "args-ai"}
	agent, err := NewCLIAgent(cfg, scriptPath, []string{"--flag", "value"})
	if err != nil {
		t.Fatalf("NewCLIAgent: %v", err)
	}

	msg, err := agent.GenerateCommitMessage(context.Background(), MessageRequest{Diff: "diff"})
	if err != nil {
		t.Fatalf("GenerateCommitMessage: %v", err)
	}
	if !strings.Contains(msg, "--flag value") {
		t.Errorf("msg should include args: %q", msg)
	}
}
