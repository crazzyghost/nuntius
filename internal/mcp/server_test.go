package mcp_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	mcpgo "github.com/mark3labs/mcp-go/mcp"

	"github.com/crazzyghost/nuntius/internal/ai"
	"github.com/crazzyghost/nuntius/internal/config"
	nuntiusmcp "github.com/crazzyghost/nuntius/internal/mcp"
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

func TestNew_CreatesServer(t *testing.T) {
	cfg := config.DefaultConfig()
	srv := nuntiusmcp.New(cfg, nil, "test")
	if srv == nil {
		t.Fatal("expected non-nil server")
	}
	if srv.MCPServer() == nil {
		t.Fatal("expected non-nil MCPServer")
	}
}

func TestNew_ToolsRegistered(t *testing.T) {
	cfg := config.DefaultConfig()
	srv := nuntiusmcp.New(cfg, nil, "1.0.0")

	// Initialize the session and list tools via HandleMessage.
	mcpSrv := srv.MCPServer()
	ctx := context.Background()

	// Send initialize.
	_ = mcpSrv.HandleMessage(ctx, []byte(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "initialize",
		"params": {
			"protocolVersion": "2024-11-05",
			"capabilities": {},
			"clientInfo": {"name": "test", "version": "0.1"}
		}
	}`))

	// List tools.
	resp := mcpSrv.HandleMessage(ctx, []byte(`{
		"jsonrpc": "2.0",
		"id": 2,
		"method": "tools/list"
	}`))

	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}

	var envelope struct {
		Result struct {
			Tools []mcpgo.Tool `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatalf("unmarshal tools/list response: %v", err)
	}

	want := []string{"generate_message", "commit", "push", "generate_and_commit", "status"}
	got := make(map[string]bool, len(envelope.Result.Tools))
	for _, tool := range envelope.Result.Tools {
		got[tool.Name] = true
	}

	for _, name := range want {
		if !got[name] {
			t.Errorf("expected tool %q to be registered, got: %v", name, toolNames(envelope.Result.Tools))
		}
	}
}

func TestNew_ServerInfo(t *testing.T) {
	cfg := config.DefaultConfig()
	srv := nuntiusmcp.New(cfg, nil, "2.3.4")
	mcpSrv := srv.MCPServer()
	ctx := context.Background()

	resp := mcpSrv.HandleMessage(ctx, []byte(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "initialize",
		"params": {
			"protocolVersion": "2024-11-05",
			"capabilities": {},
			"clientInfo": {"name": "test", "version": "0.1"}
		}
	}`))

	rpcResp, ok := resp.(mcpgo.JSONRPCResponse)
	if !ok {
		t.Fatalf("expected JSONRPCResponse, got %T", resp)
	}
	initResult, ok := rpcResp.Result.(mcpgo.InitializeResult)
	if !ok {
		t.Fatalf("expected InitializeResult, got %T", rpcResp.Result)
	}

	if initResult.ServerInfo.Name != "nuntius" {
		t.Errorf("expected server name %q, got %q", "nuntius", initResult.ServerInfo.Name)
	}
	if initResult.ServerInfo.Version != "2.3.4" {
		t.Errorf("expected version %q, got %q", "2.3.4", initResult.ServerInfo.Version)
	}
}

func TestNew_ToolCapabilitiesEnabled(t *testing.T) {
	cfg := config.DefaultConfig()
	srv := nuntiusmcp.New(cfg, nil, "1.0.0")
	mcpSrv := srv.MCPServer()
	ctx := context.Background()

	resp := mcpSrv.HandleMessage(ctx, []byte(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "initialize",
		"params": {
			"protocolVersion": "2024-11-05",
			"capabilities": {},
			"clientInfo": {"name": "test", "version": "0.1"}
		}
	}`))

	rpcResp, ok := resp.(mcpgo.JSONRPCResponse)
	if !ok {
		t.Fatalf("expected JSONRPCResponse, got %T", resp)
	}
	initResult, ok := rpcResp.Result.(mcpgo.InitializeResult)
	if !ok {
		t.Fatalf("expected InitializeResult, got %T", rpcResp.Result)
	}

	if initResult.Capabilities.Tools == nil {
		t.Error("expected tools capability to be non-nil")
	}
}

func TestNew_GenerateMessageToolSchema(t *testing.T) {
	cfg := config.DefaultConfig()
	srv := nuntiusmcp.New(cfg, nil, "1.0.0")
	mcpSrv := srv.MCPServer()
	ctx := context.Background()

	_ = mcpSrv.HandleMessage(ctx, []byte(`{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": {"protocolVersion": "2024-11-05", "capabilities": {}, "clientInfo": {"name": "test", "version": "0.1"}}
	}`))

	resp := mcpSrv.HandleMessage(ctx, []byte(`{"jsonrpc": "2.0", "id": 2, "method": "tools/list"}`))

	raw, _ := json.Marshal(resp)
	respStr := string(raw)

	// Verify expected schema properties exist in the response.
	for _, want := range []string{"diff_from", "diff", "auto", "staged", "provided"} {
		if !strings.Contains(respStr, want) {
			t.Errorf("expected tools/list response to contain %q", want)
		}
	}
}

func toolNames(tools []mcpgo.Tool) []string {
	names := make([]string, len(tools))
	for i, t := range tools {
		names[i] = t.Name
	}
	return names
}
