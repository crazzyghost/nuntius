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

func TestCodex_GenerateCommitMessage_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-key" {
			t.Errorf("Authorization = %q, want %q", auth, "Bearer test-key")
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		if body["model"] != codexDefaultModel {
			t.Errorf("model = %v, want %v", body["model"], codexDefaultModel)
		}

		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]string{
						"content": "refactor: extract helper function",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	t.Setenv("NUNTIUS_AI_API_KEY", "test-key")

	cfg := config.AIConfig{Provider: "codex"}
	c, err := NewCodex(cfg)
	if err != nil {
		t.Fatalf("NewCodex: %v", err)
	}
	c.apiURL = srv.URL

	msg, err := c.GenerateCommitMessage(context.Background(), MessageRequest{
		Diff:        "test diff",
		Conventions: "conventional",
	})
	if err != nil {
		t.Fatalf("GenerateCommitMessage: %v", err)
	}
	if msg != "refactor: extract helper function" {
		t.Errorf("msg = %q", msg)
	}
}

func TestCodex_GenerateCommitMessage_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "invalid key"}`))
	}))
	defer srv.Close()

	t.Setenv("NUNTIUS_AI_API_KEY", "test-key")

	cfg := config.AIConfig{Provider: "codex"}
	c, err := NewCodex(cfg)
	if err != nil {
		t.Fatalf("NewCodex: %v", err)
	}
	c.apiURL = srv.URL

	_, err = c.GenerateCommitMessage(context.Background(), MessageRequest{Diff: "diff"})
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error should mention 401: %v", err)
	}
}

func TestCodex_MissingAPIKey(t *testing.T) {
	t.Setenv("NUNTIUS_AI_API_KEY", "")
	cfg := config.AIConfig{Provider: "codex"}
	_, err := NewCodex(cfg)
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestCodex_DefaultModel(t *testing.T) {
	t.Setenv("NUNTIUS_AI_API_KEY", "test-key")

	cfg := config.AIConfig{Provider: "codex"}
	c, err := NewCodex(cfg)
	if err != nil {
		t.Fatalf("NewCodex: %v", err)
	}
	if c.model != codexDefaultModel {
		t.Errorf("model = %q, want %q", c.model, codexDefaultModel)
	}
}

func TestCodex_NameAndMode(t *testing.T) {
	t.Setenv("NUNTIUS_AI_API_KEY", "test-key")

	cfg := config.AIConfig{Provider: "codex"}
	c, err := NewCodex(cfg)
	if err != nil {
		t.Fatalf("NewCodex: %v", err)
	}
	if c.Name() != "codex" {
		t.Errorf("Name() = %q", c.Name())
	}
	if c.Mode() != ModeAPI {
		t.Errorf("Mode() = %q", c.Mode())
	}
}

func TestCodex_EmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{"choices": []any{}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	t.Setenv("NUNTIUS_AI_API_KEY", "test-key")

	cfg := config.AIConfig{Provider: "codex"}
	c, err := NewCodex(cfg)
	if err != nil {
		t.Fatalf("NewCodex: %v", err)
	}
	c.apiURL = srv.URL

	_, err = c.GenerateCommitMessage(context.Background(), MessageRequest{Diff: "diff"})
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
}
