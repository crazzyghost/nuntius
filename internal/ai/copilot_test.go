package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/crazzyghost/nuntius/internal/config"
)

func TestCopilot_GenerateCommitMessage_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("Authorization = %q, want %q", auth, "Bearer test-token")
		}

		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]string{
						"content": "docs: update README",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	t.Setenv("GITHUB_COPILOT_TOKEN", "test-token")

	cfg := config.AIConfig{Provider: "copilot", APIKeyEnv: "GITHUB_COPILOT_TOKEN"}
	c, err := NewCopilot(cfg)
	if err != nil {
		t.Fatalf("NewCopilot: %v", err)
	}
	c.apiURL = srv.URL

	msg, err := c.GenerateCommitMessage(context.Background(), MessageRequest{
		Diff:        "test diff",
		Conventions: "conventional",
	})
	if err != nil {
		t.Fatalf("GenerateCommitMessage: %v", err)
	}
	if msg != "docs: update README" {
		t.Errorf("msg = %q", msg)
	}
}

func TestCopilot_GenerateCommitMessage_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error": "forbidden"}`))
	}))
	defer srv.Close()

	t.Setenv("GITHUB_COPILOT_TOKEN", "test-token")

	cfg := config.AIConfig{Provider: "copilot", APIKeyEnv: "GITHUB_COPILOT_TOKEN"}
	c, err := NewCopilot(cfg)
	if err != nil {
		t.Fatalf("NewCopilot: %v", err)
	}
	c.apiURL = srv.URL

	_, err = c.GenerateCommitMessage(context.Background(), MessageRequest{Diff: "diff"})
	if err == nil {
		t.Fatal("expected error for 403")
	}
}

func TestCopilot_MissingToken(t *testing.T) {
	t.Setenv("GITHUB_COPILOT_TOKEN", "")

	cfg := config.AIConfig{Provider: "copilot", APIKeyEnv: "GITHUB_COPILOT_TOKEN"}
	_, err := NewCopilot(cfg)
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestCopilot_NameAndMode(t *testing.T) {
	t.Setenv("GITHUB_COPILOT_TOKEN", "test-token")

	cfg := config.AIConfig{Provider: "copilot", APIKeyEnv: "GITHUB_COPILOT_TOKEN"}
	c, err := NewCopilot(cfg)
	if err != nil {
		t.Fatalf("NewCopilot: %v", err)
	}
	if c.Name() != "copilot" {
		t.Errorf("Name() = %q", c.Name())
	}
	if c.Mode() != ModeAPI {
		t.Errorf("Mode() = %q", c.Mode())
	}
}
