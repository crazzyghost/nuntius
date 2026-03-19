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
	geminiDefaultModel = "gemini-2.0-flash"
	geminiAPIURL       = "https://generativelanguage.googleapis.com/v1beta/models"
)

// Gemini implements the Provider interface using the Google Generative AI API.
type Gemini struct {
	apiKey string
	model  string
	client *http.Client
	apiURL string // overridable for testing
}

// NewGemini creates a Gemini provider from the given config.
func NewGemini(cfg config.AIConfig) (*Gemini, error) {
	envVar := cfg.APIKeyEnv
	if envVar == "" {
		envVar = "GEMINI_API_KEY"
	}
	apiKey := os.Getenv(envVar)
	if apiKey == "" {
		return nil, fmt.Errorf("gemini: API key not set — export %s", envVar)
	}

	model := cfg.Model
	if model == "" {
		model = geminiDefaultModel
	}

	return &Gemini{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 30 * time.Second},
		apiURL: geminiAPIURL,
	}, nil
}

// Name returns the provider name.
func (g *Gemini) Name() string { return "gemini" }

// Mode returns ModeAPI.
func (g *Gemini) Mode() ProviderMode { return ModeAPI }

// GenerateCommitMessage sends the prompt to the Gemini generateContent endpoint
// and returns the generated commit message.
func (g *Gemini) GenerateCommitMessage(ctx context.Context, req MessageRequest) (string, error) {
	prompt := BuildPrompt(req)

	body := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("gemini: marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", g.apiURL, g.model, g.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("gemini: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("gemini: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("gemini: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini: API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("gemini: parse response: %w", err)
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: empty response — no candidates")
	}

	return strings.TrimSpace(result.Candidates[0].Content.Parts[0].Text), nil
}
