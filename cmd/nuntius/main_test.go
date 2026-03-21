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

func TestRunJsonRequiresActionFlag(t *testing.T) {
	code := run([]string{"--json"})
	if code != 1 {
		t.Errorf("expected exit code 1 for --json without action flags, got %d", code)
	}
}

func TestRunJsonShortFlagRequiresActionFlag(t *testing.T) {
	code := run([]string{"-j"})
	if code != 1 {
		t.Errorf("expected exit code 1 for -j without action flags, got %d", code)
	}
}

func TestNewFlagSet_JsonFlag(t *testing.T) {
	var buf bytes.Buffer
	flags := newFlagSet(&buf)

	f := flags.Lookup("json")
	if f == nil {
		t.Fatal("expected --json flag to be registered")
	}
	if f.Shorthand != "j" {
		t.Errorf("expected --json shorthand to be 'j', got %q", f.Shorthand)
	}
}

func TestParseDiffFrom(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input   string
		wantSrc int // 0=auto, 1=staged, 2=external
		wantErr bool
	}{
		{"auto", 0, false},
		{"staged", 1, false},
		{"stdin", 2, false},
		{"invalid", 0, true},
		{"", 0, true},
		{"AUTO", 0, true},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			_, err := parseDiffFrom(tc.input)
			if tc.wantErr && err == nil {
				t.Errorf("parseDiffFrom(%q): expected error, got nil", tc.input)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("parseDiffFrom(%q): unexpected error: %v", tc.input, err)
			}
		})
	}
}

func TestReadStdinDiff_Empty(t *testing.T) {
	t.Parallel()
	_, err := readStdinDiff(strings.NewReader(""))
	if err == nil {
		t.Error("expected error for empty stdin")
	}
	if !strings.Contains(err.Error(), "no diff provided") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestReadStdinDiff_NonEmpty(t *testing.T) {
	t.Parallel()
	diff := "diff --git a/foo.go b/foo.go\n+changed\n"
	got, err := readStdinDiff(strings.NewReader(diff))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != diff {
		t.Errorf("got %q, want %q", got, diff)
	}
}

func TestReadStdinDiff_Truncated(t *testing.T) {
	t.Parallel()
	// Feed more bytes than DefaultMaxDiffBytes to trigger truncation.
	big := strings.Repeat("x", 33000)
	got, err := readStdinDiff(strings.NewReader(big))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) > 32768 {
		t.Errorf("expected truncated output <= 32768 bytes, got %d", len(got))
	}
	if !strings.HasSuffix(got, "\n... (truncated)") {
		t.Errorf("expected truncation marker, got suffix: %q", got[len(got)-20:])
	}
}

func TestDiffFromRequiresGenerate(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	mkGitDir(t, dir)

	code := run([]string{"--diff-from=staged"})
	if code != 1 {
		t.Errorf("expected exit 1 when --diff-from used without -g, got %d", code)
	}
}

func TestDiffFromInvalidValue(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	mkGitDir(t, dir)

	code := run([]string{"-g", "--diff-from=invalid"})
	if code != 1 {
		t.Errorf("expected exit 1 for invalid --diff-from value, got %d", code)
	}
}

func TestNewFlagSet_DiffFromFlag(t *testing.T) {
	var buf bytes.Buffer
	flags := newFlagSet(&buf)

	f := flags.Lookup("diff-from")
	if f == nil {
		t.Fatal("expected --diff-from flag to be registered")
	}
	if f.DefValue != "auto" {
		t.Errorf("expected default value 'auto', got %q", f.DefValue)
	}
}

func TestRunMCPVersion(t *testing.T) {
	code := run([]string{"mcp", "--version"})
	if code != 0 {
		t.Errorf("expected exit code 0 for 'nuntius mcp --version', got %d", code)
	}
}

func TestRunMCPHelp(t *testing.T) {
	code := run([]string{"mcp", "--help"})
	if code != 0 {
		t.Errorf("expected exit code 0 for 'nuntius mcp --help', got %d", code)
	}
}

func TestRunSubcommandRouting_DefaultPath(t *testing.T) {
	// Non-mcp first arg should NOT be treated as a subcommand.
	// It falls through to runDefault, which then tries to launch TUI in a non-git dir.
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// "invalidcmd" is not a recognised subcommand; runDefault runs and fails (not a git repo).
	code := run([]string{"invalidcmd"})
	if code != 1 {
		t.Errorf("expected exit code 1 for unrecognised arg in non-git dir, got %d", code)
	}
}

func TestRunMCP_Routing(t *testing.T) {
	// Verify that "mcp" prefix args are routed to runMCP (not runDefault).
	// --version returns 0 without requiring config or git.
	code := run([]string{"mcp", "--version"})
	if code != 0 {
		t.Errorf("expected exit 0 routing to runMCP --version, got %d", code)
	}
}

func TestSetupFlagIsRegistered(t *testing.T) {
	var buf bytes.Buffer
	flags := newFlagSet(&buf)
	f := flags.Lookup("setup")
	if f == nil {
		t.Fatal("expected --setup flag to be registered")
	}
}

func TestSetupMutualExclusionWithGenerate(t *testing.T) {
	code := run([]string{"--setup", "-g"})
	if code != 1 {
		t.Errorf("expected exit code 1 for --setup with -g, got %d", code)
	}
}

func TestSetupMutualExclusionWithAutoCommit(t *testing.T) {
	code := run([]string{"--setup", "-c"})
	if code != 1 {
		t.Errorf("expected exit code 1 for --setup with -c, got %d", code)
	}
}

func TestSetupMutualExclusionWithJson(t *testing.T) {
	// --setup + --json + -g → setup exclusion fires first (isHeadless=true, jsonMode=true).
	code := run([]string{"--setup", "--json", "-g"})
	if code != 1 {
		t.Errorf("expected exit code 1 for --setup with --json and -g, got %d", code)
	}
}

func TestSetupMutualExclusionWithMCP(t *testing.T) {
	code := run([]string{"mcp", "--setup"})
	if code != 1 {
		t.Errorf("expected exit code 1 for mcp + --setup, got %d", code)
	}
}

func TestSetupInHelpOutput(t *testing.T) {
	var buf bytes.Buffer
	flags := newFlagSet(&buf)
	flags.Usage()
	if !strings.Contains(buf.String(), "--setup") {
		t.Error("expected --setup to appear in help output")
	}
}
