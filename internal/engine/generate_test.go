package engine_test

import (
	"context"
	"errors"
	"testing"

	"github.com/crazzyghost/nuntius/internal/ai"
	"github.com/crazzyghost/nuntius/internal/config"
	"github.com/crazzyghost/nuntius/internal/engine"
	"github.com/crazzyghost/nuntius/internal/git"
)

// mockProvider is a test double for ai.Provider.
type mockProvider struct {
	msg string
	err error
}

func (m *mockProvider) GenerateCommitMessage(_ context.Context, _ ai.MessageRequest) (string, error) {
	return m.msg, m.err
}

func (m *mockProvider) Name() string          { return "mock" }
func (m *mockProvider) Mode() ai.ProviderMode { return ai.ModeAPI }

func TestGenerate_ExternalDiffSource(t *testing.T) {
	t.Parallel()
	p := &mockProvider{msg: "feat: add thing"}
	cfg := config.DefaultConfig()
	msg, files, err := engine.Generate(context.Background(), cfg, p, engine.GenerateInput{
		Source:       engine.DiffSourceExternal,
		ExternalDiff: "+++ b/main.go\n--- a/main.go\n+added line\n",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg != "feat: add thing" {
		t.Errorf("got %q, want %q", msg, "feat: add thing")
	}
	if len(files) == 0 {
		t.Error("expected files to be parsed from diff headers")
	}
}

func TestGenerate_ExternalDiffSource_NoDiff(t *testing.T) {
	t.Parallel()
	p := &mockProvider{msg: "feat: thing"}
	cfg := config.DefaultConfig()
	_, _, err := engine.Generate(context.Background(), cfg, p, engine.GenerateInput{
		Source:       engine.DiffSourceExternal,
		ExternalDiff: "",
	})
	if err == nil {
		t.Fatal("expected error for empty external diff")
	}
}

func TestGenerate_NoProvider(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	_, _, err := engine.Generate(context.Background(), cfg, nil, engine.GenerateInput{
		Source: engine.DiffSourceAuto,
	})
	if err == nil {
		t.Fatal("expected error for nil provider")
	}
	if err.Error() != "no AI provider configured" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGenerate_AIFailure(t *testing.T) {
	t.Parallel()
	p := &mockProvider{err: errors.New("rate limited")}
	cfg := config.DefaultConfig()
	// Use External so we don't call real git commands.
	_, _, err := engine.Generate(context.Background(), cfg, p, engine.GenerateInput{
		Source:       engine.DiffSourceExternal,
		ExternalDiff: "+++ b/foo.go\n--- a/foo.go\n+line\n",
	})
	if err == nil {
		t.Fatal("expected error on AI failure")
	}
	if !errors.Is(err, errors.New("rate limited")) && !containsStr(err.Error(), "rate limited") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGenerate_EmptyMessageFromAI(t *testing.T) {
	t.Parallel()
	p := &mockProvider{msg: "   "}
	cfg := config.DefaultConfig()
	_, _, err := engine.Generate(context.Background(), cfg, p, engine.GenerateInput{
		Source:       engine.DiffSourceExternal,
		ExternalDiff: "+++ b/foo.go\n--- a/foo.go\n+line\n",
	})
	if err == nil {
		t.Fatal("expected error for empty AI response")
	}
}

func TestParseDiffFileHeaders(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		diff string
		want []string
	}{
		{
			name: "single file",
			diff: "--- a/foo.go\n+++ b/foo.go\n+added\n",
			want: []string{"foo.go"},
		},
		{
			name: "multiple files",
			diff: "--- a/a.go\n+++ b/a.go\n--- a/b.go\n+++ b/b.go\n",
			want: []string{"a.go", "b.go"},
		},
		{
			name: "deduplicated",
			diff: "--- a/a.go\n+++ b/a.go\n--- a/a.go\n+++ b/a.go\n",
			want: []string{"a.go"},
		},
		{
			name: "new file",
			diff: "--- /dev/null\n+++ b/new.go\n+line\n",
			want: []string{"new.go"},
		},
		{
			name: "empty diff",
			diff: "",
			want: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := git.ParseDiffFileHeaders(tc.diff)
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i, f := range got {
				if f != tc.want[i] {
					t.Errorf("files[%d] = %q, want %q", i, f, tc.want[i])
				}
			}
		})
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
