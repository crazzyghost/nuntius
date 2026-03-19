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
	ollamaDefaultModel = "llama3.2"
	ollamaDefaultURL   = "http://localhost:11434"
)

// Ollama implements the Provider interface using the Ollama local HTTP API.
// No API key is required — Ollama runs entirely on the local machine.
type Ollama struct {
	baseURL string
	model   string
	client  *http.Client
}

// NewOllama creates an Ollama provider. The base URL defaults to
// http://localhost:11434 and can be overridden via config or the
// NUNTIUS_AI_OLLAMA_URL environment variable.
func NewOllama(cfg config.AIConfig) (*Ollama, error) {
	baseURL := cfg.OllamaURL
	if baseURL == "" {
		baseURL = ollamaDefaultURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	model := cfg.Model
	if model == "" {
		model = ollamaDefaultModel
	}

	return &Ollama{
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{Timeout: 60 * time.Second},
	}, nil
}

func (o *Ollama) Name() string       { return "ollama" }
func (o *Ollama) Mode() ProviderMode { return ModeAPI }

// GenerateCommitMessage sends the prompt to the Ollama /api/generate endpoint
// using stream: false for a single-shot response.
func (o *Ollama) GenerateCommitMessage(ctx context.Context, req MessageRequest) (string, error) {
	// Validate that the requested model is available
	if err := o.validateModel(ctx); err != nil {
		return "", err
	}

	prompt := BuildPrompt(req)

	body := map[string]any{
		"model":  o.model,
		"prompt": prompt,
		"stream": false,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("ollama: marshal request: %w", err)
	}

	url := o.baseURL + "/api/generate"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("ollama: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("ollama: Ollama not running at %s — %w", o.baseURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ollama: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama: API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Response string `json:"response"`
		Done     bool   `json:"done"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("ollama: parse response: %w", err)
	}

	return strings.TrimSpace(result.Response), nil
}

// validateModel checks that the requested model is available in Ollama
// by calling GET /api/tags.
func (o *Ollama) validateModel(ctx context.Context) error {
	url := o.baseURL + "/api/tags"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("ollama: create tags request: %w", err)
	}

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("ollama: Ollama not running at %s — %w", o.baseURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ollama: read tags response: %w", err)
	}

	var tags struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(respBody, &tags); err != nil {
		return fmt.Errorf("ollama: parse tags response: %w", err)
	}

	for _, m := range tags.Models {
		// Ollama model names may include a tag suffix like ":latest"
		name := strings.Split(m.Name, ":")[0]
		if name == o.model || m.Name == o.model {
			return nil
		}
	}

	return fmt.Errorf("ollama: model %q not found — run: ollama pull %s", o.model, o.model)
}
