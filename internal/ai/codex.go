package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/crazzyghost/nuntius/internal/config"
)

const (
	codexDefaultModel = "gemini-3.1-flash-lite-preview"
	codexAPIURL       = "https://api.openai.com/v1/chat/completions"
)

// Codex implements the Provider interface using the OpenAI Chat Completions API.
type Codex struct {
	apiKey string
	model  string
	client *http.Client
	apiURL string // overridable for testing
}

// NewCodex creates a Codex (OpenAI) provider from the given config.
func NewCodex(cfg config.AIConfig) (*Codex, error) {
	apiKey, err := ResolveAPIKey("codex")
	if err != nil {
		return nil, err
	}

	model := cfg.Model
	if model == "" {
		model = codexDefaultModel
	}

	return &Codex{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 30 * time.Second},
		apiURL: codexAPIURL,
	}, nil
}

func (c *Codex) Name() string       { return "codex" }
func (c *Codex) Mode() ProviderMode { return ModeAPI }

// GenerateCommitMessage sends the prompt to the OpenAI Chat Completions API
// and returns the generated commit message.
func (c *Codex) GenerateCommitMessage(ctx context.Context, req MessageRequest) (string, error) {
	prompt := BuildPrompt(req)

	// Split into system + user messages for the chat API
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
		return "", fmt.Errorf("codex: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("codex: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("codex: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("codex: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("codex: API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("codex: parse response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("codex: empty response — no choices")
	}

	return strings.TrimSpace(result.Choices[0].Message.Content), nil
}
