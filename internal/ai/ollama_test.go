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

func newOllamaTestServer(t *testing.T, model string, response string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			resp := map[string]any{
				"models": []map[string]string{
					{"name": model + ":latest"},
				},
			}
			json.NewEncoder(w).Encode(resp)

		case "/api/generate":
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if body["stream"] != false {
				t.Errorf("stream should be false")
			}

			resp := map[string]any{
				"response": response,
				"done":     true,
			}
			json.NewEncoder(w).Encode(resp)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestOllama_GenerateCommitMessage_Success(t *testing.T) {
	srv := newOllamaTestServer(t, "llama3.2", "chore: update dependencies")
	defer srv.Close()

	cfg := config.AIConfig{Provider: "ollama", OllamaURL: srv.URL}
	o, err := NewOllama(cfg)
	if err != nil {
		t.Fatalf("NewOllama: %v", err)
	}

	msg, err := o.GenerateCommitMessage(context.Background(), MessageRequest{
		Diff:        "test diff",
		Conventions: "conventional",
	})
	if err != nil {
		t.Fatalf("GenerateCommitMessage: %v", err)
	}
	if msg != "chore: update dependencies" {
		t.Errorf("msg = %q", msg)
	}
}

func TestOllama_ModelNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return empty model list
		resp := map[string]any{"models": []any{}}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	cfg := config.AIConfig{Provider: "ollama", OllamaURL: srv.URL, Model: "nonexistent"}
	o, err := NewOllama(cfg)
	if err != nil {
		t.Fatalf("NewOllama: %v", err)
	}

	_, err = o.GenerateCommitMessage(context.Background(), MessageRequest{Diff: "diff"})
	if err == nil {
		t.Fatal("expected error for missing model")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention model not found: %v", err)
	}
	if !strings.Contains(err.Error(), "ollama pull") {
		t.Errorf("error should suggest ollama pull: %v", err)
	}
}

func TestOllama_ConnectionRefused(t *testing.T) {
	cfg := config.AIConfig{Provider: "ollama", OllamaURL: "http://localhost:1"}
	o, err := NewOllama(cfg)
	if err != nil {
		t.Fatalf("NewOllama: %v", err)
	}

	_, err = o.GenerateCommitMessage(context.Background(), MessageRequest{Diff: "diff"})
	if err == nil {
		t.Fatal("expected connection error")
	}
	if !strings.Contains(err.Error(), "not running") {
		t.Errorf("error should mention Ollama not running: %v", err)
	}
}

func TestOllama_DefaultModel(t *testing.T) {
	cfg := config.AIConfig{Provider: "ollama"}
	o, err := NewOllama(cfg)
	if err != nil {
		t.Fatalf("NewOllama: %v", err)
	}
	if o.model != ollamaDefaultModel {
		t.Errorf("model = %q, want %q", o.model, ollamaDefaultModel)
	}
}

func TestOllama_DefaultURL(t *testing.T) {
	cfg := config.AIConfig{Provider: "ollama"}
	o, err := NewOllama(cfg)
	if err != nil {
		t.Fatalf("NewOllama: %v", err)
	}
	if o.baseURL != ollamaDefaultURL {
		t.Errorf("baseURL = %q, want %q", o.baseURL, ollamaDefaultURL)
	}
}

func TestOllama_CustomURL(t *testing.T) {
	cfg := config.AIConfig{Provider: "ollama", OllamaURL: "http://myhost:11434/"}
	o, err := NewOllama(cfg)
	if err != nil {
		t.Fatalf("NewOllama: %v", err)
	}
	// Trailing slash should be trimmed
	if o.baseURL != "http://myhost:11434" {
		t.Errorf("baseURL = %q, want %q", o.baseURL, "http://myhost:11434")
	}
}

func TestOllama_NameAndMode(t *testing.T) {
	cfg := config.AIConfig{Provider: "ollama"}
	o, err := NewOllama(cfg)
	if err != nil {
		t.Fatalf("NewOllama: %v", err)
	}
	if o.Name() != "ollama" {
		t.Errorf("Name() = %q", o.Name())
	}
	if o.Mode() != ModeAPI {
		t.Errorf("Mode() = %q", o.Mode())
	}
}

func TestOllama_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			resp := map[string]any{
				"models": []map[string]string{{"name": "llama3.2:latest"}},
			}
			json.NewEncoder(w).Encode(resp)
		case "/api/generate":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal error"))
		}
	}))
	defer srv.Close()

	cfg := config.AIConfig{Provider: "ollama", OllamaURL: srv.URL}
	o, err := NewOllama(cfg)
	if err != nil {
		t.Fatalf("NewOllama: %v", err)
	}

	_, err = o.GenerateCommitMessage(context.Background(), MessageRequest{Diff: "diff"})
	if err == nil {
		t.Fatal("expected error for 500")
	}
}
