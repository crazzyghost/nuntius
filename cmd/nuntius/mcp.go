package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/crazzyghost/nuntius/internal/ai"
	"github.com/crazzyghost/nuntius/internal/config"
	"github.com/crazzyghost/nuntius/internal/git"
	nuntiusmcp "github.com/crazzyghost/nuntius/internal/mcp"
)

// runMCP handles the "nuntius mcp" subcommand.
func runMCP(args []string) int {
	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			mcpHelp()
			return 0
		case "--version":
			fmt.Printf("nuntius %s (commit=%s, built=%s)\n", version, commit, date)
			return 0
		}
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		return 1
	}

	// Provider may be nil — tool handlers return a tool error if AI is needed.
	var provider ai.Provider
	if p, err := ai.NewProvider(cfg.AI); err == nil {
		provider = p
	}

	// Prevent git credential prompts from blocking the JSON-RPC connection.
	git.SetNonInteractive(true)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	srv := nuntiusmcp.New(cfg, provider, version)
	if err := srv.ServeWithContext(ctx); err != nil && ctx.Err() == nil {
		fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
		return 1
	}
	return 0
}

// mcpHelp prints MCP subcommand help to stdout.
func mcpHelp() {
	fmt.Println("Usage: nuntius mcp [flags]")
	fmt.Println()
	fmt.Println("Start the nuntius MCP server over stdio.")
	fmt.Println("Connect using any MCP-compatible client (VS Code, Claude Desktop, etc.)")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --version   Print version and exit")
	fmt.Println("  --help      Show this help")
	fmt.Println()
	fmt.Println("Tools:")
	fmt.Println("  generate_message    Generate an AI-powered commit message")
	fmt.Println("  commit              Stage all changes and commit")
	fmt.Println("  push                Push commits to remote")
	fmt.Println("  generate_and_commit Generate a message and commit in one step")
	fmt.Println("  status              List changed files in the working tree")
}
