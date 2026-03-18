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

func TestGemini_GenerateCommitMessage_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the URL contains the model and API key
		if !strings.Contains(r.URL.Path, geminiDefaultModel) {
			t.Errorf("URL should contain model name, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("key") != "test-key" {
			t.Errorf("URL should contain API key")
		}

		resp := map[string]any{
			"candidates": []map[string]any{
				{
					"content": map[string]any{
						"parts": []map[string]string{
							{"text": "fix: resolve null pointer"},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	t.Setenv("GEMINI_API_KEY", "test-key")

	cfg := config.AIConfig{Provider: "gemini", APIKeyEnv: "GEMINI_API_KEY"}
	g, err := NewGemini(cfg)
	if err != nil {
		t.Fatalf("NewGemini: %v", err)
	}
	g.apiURL = srv.URL

	msg, err := g.GenerateCommitMessage(context.Background(), MessageRequest{
		Diff:        "test diff",
		Conventions: "conventional",
	})
	if err != nil {
		t.Fatalf("GenerateCommitMessage: %v", err)
	}
	if msg != "fix: resolve null pointer" {
		t.Errorf("msg = %q, want %q", msg, "fix: resolve null pointer")
	}
}

func TestGemini_GenerateCommitMessage_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "bad request"}`))
	}))
	defer srv.Close()

	t.Setenv("GEMINI_API_KEY", "test-key")

	cfg := config.AIConfig{Provider: "gemini", APIKeyEnv: "GEMINI_API_KEY"}
	g, err := NewGemini(cfg)
	if err != nil {
		t.Fatalf("NewGemini: %v", err)
	}
	g.apiURL = srv.URL

	_, err = g.GenerateCommitMessage(context.Background(), MessageRequest{Diff: "diff"})
	if err == nil {
		t.Fatal("expected error for bad request")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("error should mention 400: %v", err)
	}
}

func TestGemini_MissingAPIKey(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")

	cfg := config.AIConfig{Provider: "gemini", APIKeyEnv: "GEMINI_API_KEY"}
	_, err := NewGemini(cfg)
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestGemini_DefaultModel(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "test-key")

	cfg := config.AIConfig{Provider: "gemini", APIKeyEnv: "GEMINI_API_KEY"}
	g, err := NewGemini(cfg)
	if err != nil {
		t.Fatalf("NewGemini: %v", err)
	}
	if g.model != geminiDefaultModel {
		t.Errorf("model = %q, want %q", g.model, geminiDefaultModel)
	}
}

func TestGemini_NameAndMode(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "test-key")

	cfg := config.AIConfig{Provider: "gemini", APIKeyEnv: "GEMINI_API_KEY"}
	g, err := NewGemini(cfg)
	if err != nil {
		t.Fatalf("NewGemini: %v", err)
	}
	if g.Name() != "gemini" {
		t.Errorf("Name() = %q", g.Name())
	}
	if g.Mode() != ModeAPI {
		t.Errorf("Mode() = %q", g.Mode())
	}
}

func TestGemini_EmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{"candidates": []any{}}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	t.Setenv("GEMINI_API_KEY", "test-key")

	cfg := config.AIConfig{Provider: "gemini", APIKeyEnv: "GEMINI_API_KEY"}
	g, err := NewGemini(cfg)
	if err != nil {
		t.Fatalf("NewGemini: %v", err)
	}
	g.apiURL = srv.URL

	_, err = g.GenerateCommitMessage(context.Background(), MessageRequest{Diff: "diff"})
	if err == nil {
		t.Fatal("expected error for empty candidates")
	}
	if !strings.Contains(err.Error(), "no candidates") {
		t.Errorf("error = %v", err)
	}
}
