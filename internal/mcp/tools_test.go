package mcp_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/crazzyghost/nuntius/internal/ai"
	"github.com/crazzyghost/nuntius/internal/config"
	"github.com/crazzyghost/nuntius/internal/events"
	"github.com/crazzyghost/nuntius/internal/git"
	nuntiusmcp "github.com/crazzyghost/nuntius/internal/mcp"
)

// mockGitOps is a test double for git.Ops.
type mockGitOps struct {
	statusFiles  []events.FileStatus
	statusErr    error
	stageErr     error
	commitResult git.CommitResult
	commitErr    error
	pushResult   git.PushResult
	pushErr      error
	hasUpstream  bool
	upstreamErr  error
}

func (m *mockGitOps) Status() ([]events.FileStatus, error) {
	return m.statusFiles, m.statusErr
}
func (m *mockGitOps) StageAll() error {
	return m.stageErr
}
func (m *mockGitOps) Commit(_ string) (git.CommitResult, error) {
	return m.commitResult, m.commitErr
}
func (m *mockGitOps) Push(_ git.PushOptions) (git.PushResult, error) {
	return m.pushResult, m.pushErr
}
func (m *mockGitOps) HasUpstream() (bool, error) {
	return m.hasUpstream, m.upstreamErr
}

// newTestServer creates a Server with mock dependencies for unit testing.
func newTestServer(provider *mockProvider, gitOps *mockGitOps) *nuntiusmcp.Server {
	cfg := config.DefaultConfig()
	var p ai.Provider
	if provider != nil {
		p = provider
	}
	return nuntiusmcp.NewWithGitOps(cfg, p, "test", gitOps)
}

func buildCallMsg(name string, args map[string]any) []byte {
	msg, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params":  map[string]any{"name": name, "arguments": args},
	})
	return msg
}

func parseToolResult(raw []byte) (text string, isError bool, err error) {
	var envelope struct {
		Result *struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return "", false, err
	}
	if envelope.Error != nil {
		return "", false, errors.New(envelope.Error.Message)
	}
	if envelope.Result == nil {
		return "", false, errors.New("no result in response")
	}
	if len(envelope.Result.Content) > 0 {
		return envelope.Result.Content[0].Text, envelope.Result.IsError, nil
	}
	return "", envelope.Result.IsError, nil
}

// --- generate_message tests ---

func TestHandleGenerateMessage_ProvidedDiff_Success(t *testing.T) {
	provider := &mockProvider{msg: "feat: add new feature"}
	gitOps := &mockGitOps{}
	srv := newTestServer(provider, gitOps)

	resp := srv.MCPServer().HandleMessage(context.Background(), buildCallMsg("generate_message", map[string]any{
		"diff_from": "provided",
		"diff":      "diff --git a/foo.go b/foo.go\n+++ b/foo.go\n+func hello() {}\n",
	}))
	raw, _ := json.Marshal(resp)

	text, isErr, err := parseToolResult(raw)
	if err != nil {
		t.Fatalf("unexpected RPC error: %v", err)
	}
	if isErr {
		t.Errorf("expected success, got tool error: %s", text)
	}
	if !strings.Contains(text, "feat: add new feature") {
		t.Errorf("expected message in result, got: %s", text)
	}
}

func TestHandleGenerateMessage_ProvidedDiff_MissingDiff(t *testing.T) {
	srv := newTestServer(nil, &mockGitOps{})

	resp := srv.MCPServer().HandleMessage(context.Background(), buildCallMsg("generate_message", map[string]any{
		"diff_from": "provided",
	}))
	raw, _ := json.Marshal(resp)

	text, isErr, err := parseToolResult(raw)
	if err != nil {
		t.Fatalf("unexpected RPC error: %v", err)
	}
	if !isErr {
		t.Errorf("expected tool error, got success: %s", text)
	}
	if !strings.Contains(text, "diff is required") {
		t.Errorf("expected 'diff is required' in error, got: %s", text)
	}
}

func TestHandleGenerateMessage_NoProvider(t *testing.T) {
	srv := newTestServer(nil, &mockGitOps{})

	resp := srv.MCPServer().HandleMessage(context.Background(), buildCallMsg("generate_message", map[string]any{
		"diff_from": "provided",
		"diff":      "diff --git a/foo.go b/foo.go\n--- a/foo.go\n+++ b/foo.go\n+func hello() {}\n",
	}))
	raw, _ := json.Marshal(resp)

	text, isErr, err := parseToolResult(raw)
	if err != nil {
		t.Fatalf("unexpected RPC error: %v", err)
	}
	if !isErr {
		t.Errorf("expected tool error for nil provider, got success: %s", text)
	}
}

func TestHandleGenerateMessage_AIError(t *testing.T) {
	provider := &mockProvider{err: errors.New("API timeout")}
	srv := newTestServer(provider, &mockGitOps{})

	resp := srv.MCPServer().HandleMessage(context.Background(), buildCallMsg("generate_message", map[string]any{
		"diff_from": "provided",
		"diff":      "diff --git a/foo.go b/foo.go\n--- a/foo.go\n+++ b/foo.go\n+func hello() {}\n",
	}))
	raw, _ := json.Marshal(resp)

	text, isErr, err := parseToolResult(raw)
	if err != nil {
		t.Fatalf("unexpected RPC error: %v", err)
	}
	if !isErr {
		t.Errorf("expected tool error, got success: %s", text)
	}
	if !strings.Contains(text, "generation failed") {
		t.Errorf("expected 'generation failed' in error, got: %s", text)
	}
}

// --- commit tests ---

func TestHandleCommit_Success(t *testing.T) {
	gitOps := &mockGitOps{
		commitResult: git.CommitResult{Hash: "abc1234"},
	}
	srv := newTestServer(nil, gitOps)

	resp := srv.MCPServer().HandleMessage(context.Background(), buildCallMsg("commit", map[string]any{
		"message": "feat: test commit",
	}))
	raw, _ := json.Marshal(resp)

	text, isErr, err := parseToolResult(raw)
	if err != nil {
		t.Fatalf("unexpected RPC error: %v", err)
	}
	if isErr {
		t.Errorf("expected success, got tool error: %s", text)
	}
	if !strings.Contains(text, "abc1234") {
		t.Errorf("expected hash in result, got: %s", text)
	}
}

func TestHandleCommit_EmptyMessage(t *testing.T) {
	srv := newTestServer(nil, &mockGitOps{})

	resp := srv.MCPServer().HandleMessage(context.Background(), buildCallMsg("commit", map[string]any{
		"message": "",
	}))
	raw, _ := json.Marshal(resp)

	text, isErr, err := parseToolResult(raw)
	if err != nil {
		t.Fatalf("unexpected RPC error: %v", err)
	}
	if !isErr {
		t.Errorf("expected tool error for empty message, got success: %s", text)
	}
	if !strings.Contains(text, "message is required") {
		t.Errorf("expected 'message is required' in error, got: %s", text)
	}
}

func TestHandleCommit_StageError(t *testing.T) {
	gitOps := &mockGitOps{stageErr: errors.New("cannot stage")}
	srv := newTestServer(nil, gitOps)

	resp := srv.MCPServer().HandleMessage(context.Background(), buildCallMsg("commit", map[string]any{
		"message": "feat: something",
	}))
	raw, _ := json.Marshal(resp)

	text, isErr, err := parseToolResult(raw)
	if err != nil {
		t.Fatalf("unexpected RPC error: %v", err)
	}
	if !isErr {
		t.Errorf("expected tool error, got success: %s", text)
	}
	if !strings.Contains(text, "staging failed") {
		t.Errorf("expected 'staging failed' in error, got: %s", text)
	}
}

func TestHandleCommit_CommitError(t *testing.T) {
	gitOps := &mockGitOps{commitErr: errors.New("nothing to commit")}
	srv := newTestServer(nil, gitOps)

	resp := srv.MCPServer().HandleMessage(context.Background(), buildCallMsg("commit", map[string]any{
		"message": "feat: something",
	}))
	raw, _ := json.Marshal(resp)

	text, isErr, err := parseToolResult(raw)
	if err != nil {
		t.Fatalf("unexpected RPC error: %v", err)
	}
	if !isErr {
		t.Errorf("expected tool error, got success: %s", text)
	}
	if !strings.Contains(text, "commit failed") {
		t.Errorf("expected 'commit failed' in error, got: %s", text)
	}
}

// --- push tests ---

func TestHandlePush_Success(t *testing.T) {
	gitOps := &mockGitOps{
		hasUpstream: true,
		pushResult:  git.PushResult{Remote: "origin", Branch: "main"},
	}
	srv := newTestServer(nil, gitOps)

	resp := srv.MCPServer().HandleMessage(context.Background(), buildCallMsg("push", map[string]any{}))
	raw, _ := json.Marshal(resp)

	text, isErr, err := parseToolResult(raw)
	if err != nil {
		t.Fatalf("unexpected RPC error: %v", err)
	}
	if isErr {
		t.Errorf("expected success, got tool error: %s", text)
	}
	if !strings.Contains(text, "origin") || !strings.Contains(text, "main") {
		t.Errorf("expected remote and branch in result, got: %s", text)
	}
}

func TestHandlePush_Error(t *testing.T) {
	gitOps := &mockGitOps{
		hasUpstream: true,
		pushErr:     errors.New("nothing to push"),
	}
	srv := newTestServer(nil, gitOps)

	resp := srv.MCPServer().HandleMessage(context.Background(), buildCallMsg("push", map[string]any{}))
	raw, _ := json.Marshal(resp)

	text, isErr, err := parseToolResult(raw)
	if err != nil {
		t.Fatalf("unexpected RPC error: %v", err)
	}
	if !isErr {
		t.Errorf("expected tool error, got success: %s", text)
	}
	if !strings.Contains(text, "push failed") {
		t.Errorf("expected 'push failed' in error, got: %s", text)
	}
}

func TestHandlePush_SetsUpstreamWhenNone(t *testing.T) {
	var capturedOpts git.PushOptions
	gitOps := &mockGitOps{
		hasUpstream: false,
		pushResult:  git.PushResult{Remote: "origin", Branch: "feature"},
	}
	gitOps.pushResult.SetUpstream = true

	// Wrap Push to capture opts.
	capturingGitOps := &capturingPushGitOps{
		mockGitOps:   gitOps,
		capturedOpts: &capturedOpts,
	}
	cfg := config.DefaultConfig()
	srv := nuntiusmcp.NewWithGitOps(cfg, nil, "test", capturingGitOps)

	srv.MCPServer().HandleMessage(context.Background(), buildCallMsg("push", map[string]any{}))

	if !capturedOpts.SetUpstream {
		t.Error("expected SetUpstream=true when no upstream configured")
	}
}

// capturingPushGitOps wraps mockGitOps and captures PushOptions.
type capturingPushGitOps struct {
	*mockGitOps
	capturedOpts *git.PushOptions
}

func (c *capturingPushGitOps) Push(opts git.PushOptions) (git.PushResult, error) {
	*c.capturedOpts = opts
	return c.mockGitOps.Push(opts)
}

// --- generate_and_commit tests ---

func TestHandleGenerateAndCommit_Success(t *testing.T) {
	provider := &mockProvider{msg: "feat: generated"}
	gitOps := &mockGitOps{
		commitResult: git.CommitResult{Hash: "deadbeef"},
	}
	srv := newTestServer(provider, gitOps)

	resp := srv.MCPServer().HandleMessage(context.Background(), buildCallMsg("generate_and_commit", map[string]any{
		"diff_from": "provided",
		"diff":      "diff --git a/foo.go b/foo.go\n+++ b/foo.go\n+func hello() {}\n",
	}))
	raw, _ := json.Marshal(resp)

	text, isErr, err := parseToolResult(raw)
	if err != nil {
		t.Fatalf("unexpected RPC error: %v", err)
	}
	if isErr {
		t.Errorf("expected success, got tool error: %s", text)
	}
	if !strings.Contains(text, "feat: generated") {
		t.Errorf("expected message in result, got: %s", text)
	}
	if !strings.Contains(text, "deadbeef") {
		t.Errorf("expected hash in result, got: %s", text)
	}
}

func TestHandleGenerateAndCommit_GenerateFails(t *testing.T) {
	srv := newTestServer(nil, &mockGitOps{})

	resp := srv.MCPServer().HandleMessage(context.Background(), buildCallMsg("generate_and_commit", map[string]any{
		"diff_from": "provided",
		"diff":      "diff --git a/foo.go b/foo.go\n--- a/foo.go\n+++ b/foo.go\n+func hello() {}\n",
	}))
	raw, _ := json.Marshal(resp)

	_, isErr, err := parseToolResult(raw)
	if err != nil {
		t.Fatalf("unexpected RPC error: %v", err)
	}
	if !isErr {
		t.Error("expected tool error when provider is nil")
	}
}

// --- status tests ---

func TestHandleStatus_Success(t *testing.T) {
	gitOps := &mockGitOps{
		statusFiles: []events.FileStatus{
			{Path: "foo.go", Status: "modified", Staged: true},
			{Path: "bar.go", Status: "untracked", Staged: false},
		},
	}
	srv := newTestServer(nil, gitOps)

	resp := srv.MCPServer().HandleMessage(context.Background(), buildCallMsg("status", map[string]any{}))
	raw, _ := json.Marshal(resp)

	text, isErr, err := parseToolResult(raw)
	if err != nil {
		t.Fatalf("unexpected RPC error: %v", err)
	}
	if isErr {
		t.Errorf("expected success, got tool error: %s", text)
	}
	if !strings.Contains(text, "foo.go") || !strings.Contains(text, "bar.go") {
		t.Errorf("expected file paths in result, got: %s", text)
	}
}

func TestHandleStatus_EmptyTree(t *testing.T) {
	gitOps := &mockGitOps{statusFiles: []events.FileStatus{}}
	srv := newTestServer(nil, gitOps)

	resp := srv.MCPServer().HandleMessage(context.Background(), buildCallMsg("status", map[string]any{}))
	raw, _ := json.Marshal(resp)

	text, isErr, err := parseToolResult(raw)
	if err != nil {
		t.Fatalf("unexpected RPC error: %v", err)
	}
	if isErr {
		t.Errorf("expected success, got tool error: %s", text)
	}
	if !strings.Contains(text, `"files":[]`) && !strings.Contains(text, `"files": []`) {
		t.Errorf("expected empty files list in result, got: %s", text)
	}
}

func TestHandleStatus_Error(t *testing.T) {
	gitOps := &mockGitOps{statusErr: errors.New("not a git repo")}
	srv := newTestServer(nil, gitOps)

	resp := srv.MCPServer().HandleMessage(context.Background(), buildCallMsg("status", map[string]any{}))
	raw, _ := json.Marshal(resp)

	text, isErr, err := parseToolResult(raw)
	if err != nil {
		t.Fatalf("unexpected RPC error: %v", err)
	}
	if !isErr {
		t.Errorf("expected tool error, got success: %s", text)
	}
	if !strings.Contains(text, "status failed") {
		t.Errorf("expected 'status failed' in error, got: %s", text)
	}
}

// --- buildGenerateInput diff validation tests ---

func TestBuildGenerateInput_ProvidedDiffValidation(t *testing.T) {
	t.Parallel()

	validDiffPlusPlus := "diff --git a/foo.go b/foo.go\n--- a/foo.go\n+++ b/foo.go\n@@ -1,3 +1,4 @@\n+func hello() {}\n"
	validGitHeader := "diff --git a/bar.go b/bar.go\n"
	validHunkOnly := "@@ -1,4 +1,5 @@\n context line\n"
	validOldFile := "--- a/x.go\n+++ b/x.go\n"

	tests := []struct {
		name        string
		args        map[string]any
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty diff returns error",
			args:        map[string]any{"diff_from": "provided"},
			wantErr:     true,
			errContains: "diff is required",
		},
		{
			name:    "valid unified diff succeeds",
			args:    map[string]any{"diff_from": "provided", "diff": validDiffPlusPlus},
			wantErr: false,
		},
		{
			name:        "plain error text rejected",
			args:        map[string]any{"diff_from": "provided", "diff": "✗ Permission denied running git diff\nfatal: not a git repository"},
			wantErr:     true,
			errContains: "does not appear to be a valid unified diff",
		},
		{
			name: "error text prepended to valid diff",
			args: map[string]any{
				"diff_from": "provided",
				"diff":      "✗ Permission denied\n\ndiff --git a/foo.go b/foo.go\n--- a/foo.go\n+++ b/foo.go\n@@ -1 +1 @@\n-old\n+new",
			},
			wantErr: false,
		},
		{
			name:    "git diff header accepted",
			args:    map[string]any{"diff_from": "provided", "diff": validGitHeader},
			wantErr: false,
		},
		{
			name:    "hunk header only accepted",
			args:    map[string]any{"diff_from": "provided", "diff": validHunkOnly},
			wantErr: false,
		},
		{
			name:    "old file header accepted",
			args:    map[string]any{"diff_from": "provided", "diff": validOldFile},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			provider := &mockProvider{msg: "feat: ok"}
			srv := newTestServer(provider, &mockGitOps{})

			resp := srv.MCPServer().HandleMessage(context.Background(), buildCallMsg("generate_message", tc.args))
			raw, _ := json.Marshal(resp)

			text, isErr, err := parseToolResult(raw)
			if err != nil {
				t.Fatalf("unexpected RPC error: %v", err)
			}

			if tc.wantErr {
				if !isErr {
					t.Errorf("expected tool error, got success: %s", text)
					return
				}
				if tc.errContains != "" && !strings.Contains(text, tc.errContains) {
					t.Errorf("expected error to contain %q, got: %s", tc.errContains, text)
				}
			} else {
				if isErr {
					t.Errorf("expected success, got tool error: %s", text)
				}
			}
		})
	}
}
