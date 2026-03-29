package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/crazzyghost/nuntius/internal/config"
)

func TestClaude_GenerateCommitMessage_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("missing or wrong x-api-key header")
		}
		if r.Header.Get("anthropic-version") != claudeAPIVersion {
			t.Errorf("missing or wrong anthropic-version header")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("missing Content-Type header")
		}

		// Verify request body structure
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		if body["model"] != claudeDefaultModel {
			t.Errorf("model = %v, want %v", body["model"], claudeDefaultModel)
		}

		resp := map[string]any{
			"content": []map[string]string{
				{"text": "feat: add new feature"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	t.Setenv("NUNTIUS_AI_API_KEY", "test-key")

	cfg := config.AIConfig{Provider: "claude"}
	c, err := NewClaude(cfg)
	if err != nil {
		t.Fatalf("NewClaude: %v", err)
	}
	c.apiURL = srv.URL

	msg, err := c.GenerateCommitMessage(context.Background(), MessageRequest{
		Diff:        "test diff",
		Conventions: "conventional",
	})
	if err != nil {
		t.Fatalf("GenerateCommitMessage: %v", err)
	}
	if msg != "feat: add new feature" {
		t.Errorf("msg = %q, want %q", msg, "feat: add new feature")
	}
}

func TestClaude_GenerateCommitMessage_RateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("retry-after", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	t.Setenv("NUNTIUS_AI_API_KEY", "test-key")

	cfg := config.AIConfig{Provider: "claude"}
	c, err := NewClaude(cfg)
	if err != nil {
		t.Fatalf("NewClaude: %v", err)
	}
	c.apiURL = srv.URL

	_, err = c.GenerateCommitMessage(context.Background(), MessageRequest{Diff: "diff"})
	if err == nil {
		t.Fatal("expected rate limit error")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("error should mention 429: %v", err)
	}
}

func TestClaude_GenerateCommitMessage_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "internal"}`))
	}))
	defer srv.Close()

	t.Setenv("NUNTIUS_AI_API_KEY", "test-key")

	cfg := config.AIConfig{Provider: "claude"}
	c, err := NewClaude(cfg)
	if err != nil {
		t.Fatalf("NewClaude: %v", err)
	}
	c.apiURL = srv.URL

	_, err = c.GenerateCommitMessage(context.Background(), MessageRequest{Diff: "diff"})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should mention status code: %v", err)
	}
}

func TestClaude_MissingAPIKey(t *testing.T) {
	cfg := config.AIConfig{Provider: "claude"}
	_, err := NewClaude(cfg)
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
	if !strings.Contains(err.Error(), "NUNTIUS_AI_API_KEY") {
		t.Errorf("error should reference env var: %v", err)
	}
}

func TestClaude_DefaultModel(t *testing.T) {
	t.Setenv("NUNTIUS_AI_API_KEY", "test-key")

	cfg := config.AIConfig{Provider: "claude"}
	c, err := NewClaude(cfg)
	if err != nil {
		t.Fatalf("NewClaude: %v", err)
	}
	if c.model != claudeDefaultModel {
		t.Errorf("model = %q, want %q", c.model, claudeDefaultModel)
	}
}

func TestClaude_CustomModel(t *testing.T) {
	t.Setenv("NUNTIUS_AI_API_KEY", "test-key")

	cfg := config.AIConfig{Provider: "claude", Model: "claude-3-opus"}
	c, err := NewClaude(cfg)
	if err != nil {
		t.Fatalf("NewClaude: %v", err)
	}
	if c.model != "claude-3-opus" {
		t.Errorf("model = %q, want %q", c.model, "claude-3-opus")
	}
}

func TestClaude_NameAndMode(t *testing.T) {
	t.Setenv("NUNTIUS_AI_API_KEY", "test-key")

	cfg := config.AIConfig{Provider: "claude"}
	c, err := NewClaude(cfg)
	if err != nil {
		t.Fatalf("NewClaude: %v", err)
	}
	if c.Name() != "claude" {
		t.Errorf("Name() = %q, want %q", c.Name(), "claude")
	}
	if c.Mode() != ModeAPI {
		t.Errorf("Mode() = %q, want %q", c.Mode(), ModeAPI)
	}
}
