package ai

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/crazzyghost/nuntius/internal/config"
)

// installHints maps base command names to install instructions.
var installHints = map[string]string{
	"gh":     "brew install gh — https://cli.github.com",
	"gemini": "pip install google-genai or npm install -g @anthropic-ai/gemini-cli",
	"claude": "pip install claude-cli or npm install -g @anthropic-ai/claude-code",
}

// CLIAgent implements the Provider interface by shelling out to a local
// CLI-based AI tool. It pipes the prompt to stdin and captures stdout.
type CLIAgent struct {
	name    string
	command string
	args    []string
	timeout time.Duration
}

// NewCLIAgent creates a CLI agent provider. The command is the base binary
// (e.g. "gh", "gemini", "claude") and args are additional arguments.
// For "custom" providers, cfg.CLICommand and cfg.CLIArgs are used.
func NewCLIAgent(cfg config.AIConfig, command string, args []string) (*CLIAgent, error) {
	// Determine the base binary name for LookPath
	baseBin := command
	if i := strings.IndexByte(command, ' '); i >= 0 {
		baseBin = command[:i]
	}

	if _, err := exec.LookPath(baseBin); err != nil {
		hint := installHints[baseBin]
		if hint == "" {
			hint = "ensure it is installed and in your $PATH"
		}
		return nil, fmt.Errorf("%s not found — %s", baseBin, hint)
	}

	name := cfg.Provider
	if name == "" {
		name = command
	}

	return &CLIAgent{
		name:    name,
		command: command,
		args:    args,
		timeout: 60 * time.Second,
	}, nil
}

func (c *CLIAgent) Name() string       { return c.name }
func (c *CLIAgent) Mode() ProviderMode { return ModeCLI }

// GenerateCommitMessage pipes the prompt to the CLI tool's stdin
// and returns the trimmed stdout as the commit message.
func (c *CLIAgent) GenerateCommitMessage(ctx context.Context, req MessageRequest) (string, error) {
	prompt := BuildPrompt(req)

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Build the command: split command into parts if it contains spaces
	parts := strings.Fields(c.command)
	cmdArgs := append(parts[1:], c.args...)

	cmd := exec.CommandContext(ctx, parts[0], cmdArgs...)
	cmd.Stdin = strings.NewReader(prompt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s failed: %s\n%s", c.name, err, strings.TrimSpace(stderr.String()))
	}

	result := strings.TrimSpace(stdout.String())
	if result == "" {
		return "", fmt.Errorf("%s: empty response", c.name)
	}

	return result, nil
}
