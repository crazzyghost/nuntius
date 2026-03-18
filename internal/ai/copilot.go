package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/crazzyghost/nuntius/internal/config"
)

const (
	copilotDefaultModel = "gpt-4o-mini"
)

// Copilot implements the Provider interface using the GitHub Copilot
// chat completions API. Users who prefer the CLI approach should use
// provider = "copilot-cli" instead (handled by CLIAgent).
type Copilot struct {
	token  string
	model  string
	client *http.Client
	apiURL string // overridable for testing
}

// NewCopilot creates a Copilot API provider.
// It reads the token from the environment variable specified in config
// (default: GITHUB_COPILOT_TOKEN).
func NewCopilot(cfg config.AIConfig) (*Copilot, error) {
	envVar := cfg.APIKeyEnv
	if envVar == "" {
		envVar = "GITHUB_COPILOT_TOKEN"
	}
	token := os.Getenv(envVar)
	if token == "" {
		return nil, fmt.Errorf("copilot: token not set — export %s", envVar)
	}

	model := cfg.Model
	if model == "" {
		model = copilotDefaultModel
	}

	return &Copilot{
		token:  token,
		model:  model,
		client: &http.Client{Timeout: 30 * time.Second},
		apiURL: "https://api.githubcopilot.com/chat/completions",
	}, nil
}

func (c *Copilot) Name() string       { return "copilot" }
func (c *Copilot) Mode() ProviderMode { return ModeAPI }

// GenerateCommitMessage sends the prompt to the GitHub Copilot completions
// API (OpenAI-compatible format) and returns the generated commit message.
func (c *Copilot) GenerateCommitMessage(ctx context.Context, req MessageRequest) (string, error) {
	prompt := BuildPrompt(req)

	body := map[string]any{
		"model": c.model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a commit message generator. Write concise, high-quality commit messages."},
			{"role": "user", "content": prompt},
		},
		"max_tokens": 1024,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("copilot: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("copilot: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("copilot: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("copilot: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("copilot: API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("copilot: parse response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("copilot: empty response — no choices")
	}

	return strings.TrimSpace(result.Choices[0].Message.Content), nil
}
