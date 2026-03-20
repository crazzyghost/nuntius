package mcp_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/crazzyghost/nuntius/internal/ai"
	"github.com/crazzyghost/nuntius/internal/config"
	"github.com/crazzyghost/nuntius/internal/events"
	"github.com/crazzyghost/nuntius/internal/git"
	nuntiusmcp "github.com/crazzyghost/nuntius/internal/mcp"
)

// newIntegrationServer creates a server with full mock dependencies.
func newIntegrationServer(provider *mockProvider, gitOps *mockGitOps) *nuntiusmcp.Server {
	cfg := config.DefaultConfig()
	var p ai.Provider
	if provider != nil {
		p = provider
	}
	return nuntiusmcp.NewWithGitOps(cfg, p, "integration-test", gitOps)
}

// initSession sends an MCP initialize message and returns the session context.
func initSession(t *testing.T, srv *nuntiusmcp.Server) context.Context {
	t.Helper()
	ctx := context.Background()
	srv.MCPServer().HandleMessage(ctx, []byte(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "initialize",
		"params": {
			"protocolVersion": "2024-11-05",
			"capabilities": {},
			"clientInfo": {"name": "integration-test", "version": "0.1"}
		}
	}`))
	return ctx
}

func TestIntegration_Initialize_ReturnsCapabilities(t *testing.T) {
	srv := newIntegrationServer(nil, &mockGitOps{})
	ctx := context.Background()

	resp := srv.MCPServer().HandleMessage(ctx, []byte(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "initialize",
		"params": {
			"protocolVersion": "2024-11-05",
			"capabilities": {},
			"clientInfo": {"name": "test", "version": "0.1"}
		}
	}`))

	raw, _ := json.Marshal(resp)
	respStr := string(raw)

	if !strings.Contains(respStr, "nuntius") {
		t.Errorf("expected server name in response, got: %s", respStr)
	}
	if !strings.Contains(respStr, "tools") {
		t.Errorf("expected tools capability in response, got: %s", respStr)
	}
}

func TestIntegration_ToolsList_AllFiveTools(t *testing.T) {
	srv := newIntegrationServer(nil, &mockGitOps{})
	ctx := initSession(t, srv)

	resp := srv.MCPServer().HandleMessage(ctx, []byte(`{
		"jsonrpc": "2.0",
		"id": 2,
		"method": "tools/list"
	}`))

	raw, _ := json.Marshal(resp)
	respStr := string(raw)

	for _, tool := range []string{"generate_message", "commit", "push", "generate_and_commit", "status"} {
		if !strings.Contains(respStr, tool) {
			t.Errorf("expected tool %q in tools/list response", tool)
		}
	}
}

func TestIntegration_GenerateMessage_ProvidedDiff(t *testing.T) {
	provider := &mockProvider{msg: "feat: add integration test"}
	srv := newIntegrationServer(provider, &mockGitOps{})
	ctx := initSession(t, srv)

	resp := srv.MCPServer().HandleMessage(ctx, buildCallMsg("generate_message", map[string]any{
		"diff_from": "provided",
		"diff":      "diff --git a/main_test.go b/main_test.go\n+++ b/main_test.go\n+func TestNew() {}\n",
	}))

	raw, _ := json.Marshal(resp)
	text, isErr, err := parseToolResult(raw)
	if err != nil {
		t.Fatalf("RPC error: %v", err)
	}
	if isErr {
		t.Fatalf("tool error: %s", text)
	}
	if !strings.Contains(text, "feat: add integration test") {
		t.Errorf("expected generated message in result, got: %s", text)
	}
	if !strings.Contains(text, "files") {
		t.Errorf("expected files key in result, got: %s", text)
	}
}

func TestIntegration_Commit_Success(t *testing.T) {
	gitOps := &mockGitOps{
		commitResult: git.CommitResult{Hash: "c0ffee"},
	}
	srv := newIntegrationServer(nil, gitOps)
	ctx := initSession(t, srv)

	resp := srv.MCPServer().HandleMessage(ctx, buildCallMsg("commit", map[string]any{
		"message": "feat: integration commit",
	}))

	raw, _ := json.Marshal(resp)
	text, isErr, err := parseToolResult(raw)
	if err != nil {
		t.Fatalf("RPC error: %v", err)
	}
	if isErr {
		t.Fatalf("tool error: %s", text)
	}
	if !strings.Contains(text, "c0ffee") {
		t.Errorf("expected commit hash in result, got: %s", text)
	}
}

func TestIntegration_Commit_EmptyMessage(t *testing.T) {
	srv := newIntegrationServer(nil, &mockGitOps{})
	ctx := initSession(t, srv)

	resp := srv.MCPServer().HandleMessage(ctx, buildCallMsg("commit", map[string]any{"message": ""}))

	raw, _ := json.Marshal(resp)
	text, isErr, err := parseToolResult(raw)
	if err != nil {
		t.Fatalf("RPC error: %v", err)
	}
	if !isErr {
		t.Errorf("expected tool error for empty message, got: %s", text)
	}
}

func TestIntegration_Push_Success(t *testing.T) {
	gitOps := &mockGitOps{
		hasUpstream: true,
		pushResult:  git.PushResult{Remote: "origin", Branch: "main"},
	}
	srv := newIntegrationServer(nil, gitOps)
	ctx := initSession(t, srv)

	resp := srv.MCPServer().HandleMessage(ctx, buildCallMsg("push", map[string]any{}))

	raw, _ := json.Marshal(resp)
	text, isErr, err := parseToolResult(raw)
	if err != nil {
		t.Fatalf("RPC error: %v", err)
	}
	if isErr {
		t.Fatalf("tool error: %s", text)
	}
	if !strings.Contains(text, "origin") {
		t.Errorf("expected remote in result, got: %s", text)
	}
}

func TestIntegration_Push_NothingToPush(t *testing.T) {
	gitOps := &mockGitOps{
		hasUpstream: true,
		pushErr:     errNothingToPush,
	}
	srv := newIntegrationServer(nil, gitOps)
	ctx := initSession(t, srv)

	resp := srv.MCPServer().HandleMessage(ctx, buildCallMsg("push", map[string]any{}))

	raw, _ := json.Marshal(resp)
	_, isErr, err := parseToolResult(raw)
	if err != nil {
		t.Fatalf("RPC error: %v", err)
	}
	if !isErr {
		t.Error("expected tool error when nothing to push")
	}
}

func TestIntegration_Status_WithFiles(t *testing.T) {
	gitOps := &mockGitOps{
		statusFiles: []events.FileStatus{
			{Path: "api.go", Status: "modified", Staged: true},
			{Path: "README.md", Status: "untracked", Staged: false},
		},
	}
	srv := newIntegrationServer(nil, gitOps)
	ctx := initSession(t, srv)

	resp := srv.MCPServer().HandleMessage(ctx, buildCallMsg("status", map[string]any{}))

	raw, _ := json.Marshal(resp)
	text, isErr, err := parseToolResult(raw)
	if err != nil {
		t.Fatalf("RPC error: %v", err)
	}
	if isErr {
		t.Fatalf("tool error: %s", text)
	}
	if !strings.Contains(text, "api.go") || !strings.Contains(text, "README.md") {
		t.Errorf("expected file paths in result, got: %s", text)
	}
}

func TestIntegration_GenerateAndCommit_Success(t *testing.T) {
	provider := &mockProvider{msg: "feat: integrated"}
	gitOps := &mockGitOps{
		commitResult: git.CommitResult{Hash: "beefdead"},
	}
	srv := newIntegrationServer(provider, gitOps)
	ctx := initSession(t, srv)

	resp := srv.MCPServer().HandleMessage(ctx, buildCallMsg("generate_and_commit", map[string]any{
		"diff_from": "provided",
		"diff":      "diff --git a/x.go b/x.go\n+++ b/x.go\n+package x\n",
	}))

	raw, _ := json.Marshal(resp)
	text, isErr, err := parseToolResult(raw)
	if err != nil {
		t.Fatalf("RPC error: %v", err)
	}
	if isErr {
		t.Fatalf("tool error: %s", text)
	}
	if !strings.Contains(text, "feat: integrated") || !strings.Contains(text, "beefdead") {
		t.Errorf("expected message and hash in result, got: %s", text)
	}
}

func TestIntegration_GenerateMessage_NoChanges(t *testing.T) {
	// nil provider → "no AI provider configured" error from engine.Generate
	srv := newIntegrationServer(nil, &mockGitOps{})
	ctx := initSession(t, srv)

	resp := srv.MCPServer().HandleMessage(ctx, buildCallMsg("generate_message", map[string]any{
		"diff_from": "provided",
		"diff":      "some diff",
	}))

	raw, _ := json.Marshal(resp)
	_, isErr, err := parseToolResult(raw)
	if err != nil {
		t.Fatalf("RPC error: %v", err)
	}
	if !isErr {
		t.Error("expected tool error when provider is nil")
	}
}

// errNothingToPush is a sentinel error for push tests.
var errNothingToPush = &nothingToPushError{}

type nothingToPushError struct{}

func (e *nothingToPushError) Error() string { return "nothing to push" }
