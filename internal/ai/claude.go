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
	claudeDefaultModel = "claude-3-haiku-20240307"
	claudeAPIURL       = "https://api.anthropic.com/v1/messages"
	claudeAPIVersion   = "2023-06-01"
)

// Claude implements the Provider interface using the Anthropic Messages API.
type Claude struct {
	apiKey string
	model  string
	client *http.Client
	apiURL string // overridable for testing
}

// NewClaude creates a Claude provider from the given config.
func NewClaude(cfg config.AIConfig) (*Claude, error) {
	envVar := cfg.APIKeyEnv
	if envVar == "" {
		envVar = "ANTHROPIC_API_KEY"
	}
	apiKey := os.Getenv(envVar)
	if apiKey == "" {
		return nil, fmt.Errorf("claude: API key not set — export %s", envVar)
	}

	model := cfg.Model
	if model == "" {
		model = claudeDefaultModel
	}

	return &Claude{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 30 * time.Second},
		apiURL: claudeAPIURL,
	}, nil
}

func (c *Claude) Name() string       { return "claude" }
func (c *Claude) Mode() ProviderMode { return ModeAPI }

// GenerateCommitMessage sends the prompt to the Claude Messages API and
// returns the generated commit message.
func (c *Claude) GenerateCommitMessage(ctx context.Context, req MessageRequest) (string, error) {
	prompt := BuildPrompt(req)

	body := map[string]any{
		"model":      c.model,
		"max_tokens": 1024,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("claude: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("claude: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", claudeAPIVersion)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("claude: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("claude: read response: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := resp.Header.Get("retry-after")
		return "", fmt.Errorf("claude: rate limited (429) — retry after %s", retryAfter)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("claude: API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("claude: parse response: %w", err)
	}
	if len(result.Content) == 0 {
		return "", fmt.Errorf("claude: empty response — no content blocks")
	}

	return strings.TrimSpace(result.Content[0].Text), nil
}
