// Package mcp implements a Model Context Protocol (MCP) server that exposes
// nuntius functionality — commit message generation, committing, pushing, and
// status — to any MCP-compatible AI agent or IDE extension.
package mcp

import (
	"context"
	"os"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/crazzyghost/nuntius/internal/ai"
	"github.com/crazzyghost/nuntius/internal/config"
	"github.com/crazzyghost/nuntius/internal/git"
)

// Server is the nuntius MCP server.
type Server struct {
	cfg      config.Config
	provider ai.Provider
	gitOps   git.Ops
	srv      *mcpserver.MCPServer
}

// New creates a new MCP server with the given config, AI provider, and version string.
func New(cfg config.Config, provider ai.Provider, version string) *Server {
	return NewWithGitOps(cfg, provider, version, git.DefaultOps{})
}

// NewWithGitOps creates a Server with an injectable git.Ops for testing.
func NewWithGitOps(cfg config.Config, provider ai.Provider, version string, gitOps git.Ops) *Server {
	s := &Server{
		cfg:      cfg,
		provider: provider,
		gitOps:   gitOps,
	}
	s.srv = mcpserver.NewMCPServer("nuntius", version, mcpserver.WithToolCapabilities(true))
	s.registerTools()
	return s
}

// MCPServer returns the underlying MCPServer for in-process testing.
func (s *Server) MCPServer() *mcpserver.MCPServer {
	return s.srv
}

// registerTools registers all five nuntius tools on the MCP server.
func (s *Server) registerTools() {
	s.srv.AddTool(
		mcpgo.NewTool("generate_message",
			mcpgo.WithDescription("Generate an AI-powered commit message from git changes"),
			mcpgo.WithString("diff_from",
				mcpgo.Description("Diff source: auto (staged+unstaged), staged, or provided"),
				mcpgo.Enum("auto", "staged", "provided"),
			),
			mcpgo.WithString("diff",
				mcpgo.Description("Unified diff content (required when diff_from=provided)"),
			),
		),
		s.handleGenerateMessage,
	)

	s.srv.AddTool(
		mcpgo.NewTool("commit",
			mcpgo.WithDescription("Stage all changes and commit with the given message"),
			mcpgo.WithString("message",
				mcpgo.Required(),
				mcpgo.Description("The commit message"),
			),
		),
		s.handleCommit,
	)

	s.srv.AddTool(
		mcpgo.NewTool("push",
			mcpgo.WithDescription("Push commits to remote"),
			mcpgo.WithBoolean("force",
				mcpgo.Description("Use --force-with-lease"),
			),
			mcpgo.WithBoolean("set_upstream",
				mcpgo.Description("Set upstream tracking for new branches"),
			),
		),
		s.handlePush,
	)

	s.srv.AddTool(
		mcpgo.NewTool("generate_and_commit",
			mcpgo.WithDescription("Generate a commit message, then stage and commit"),
			mcpgo.WithString("diff_from",
				mcpgo.Description("Diff source: auto, staged, or provided"),
				mcpgo.Enum("auto", "staged", "provided"),
			),
			mcpgo.WithString("diff",
				mcpgo.Description("Unified diff content (required when diff_from=provided)"),
			),
		),
		s.handleGenerateAndCommit,
	)

	s.srv.AddTool(
		mcpgo.NewTool("status",
			mcpgo.WithDescription("List changed files in the working tree"),
		),
		s.handleStatus,
	)
}

// Serve starts the MCP server over stdio. It handles SIGINT/SIGTERM internally.
func (s *Server) Serve() error {
	return mcpserver.ServeStdio(s.srv)
}

// ServeWithContext starts the MCP server over stdio and respects context cancellation.
// Use this when you need external control over server shutdown (e.g. signal.NotifyContext).
func (s *Server) ServeWithContext(ctx context.Context) error {
	stdio := mcpserver.NewStdioServer(s.srv)
	return stdio.Listen(ctx, os.Stdin, os.Stdout)
}
