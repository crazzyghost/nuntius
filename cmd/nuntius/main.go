package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/crazzyghost/nuntius/internal/config"
)

// version is injected at build time via -ldflags.
var version = "dev"

func main() {
	exitCode := run(os.Args[1:])
	os.Exit(exitCode)
}

func run(args []string) int {
	flags := flag.NewFlagSet("nuntius", flag.ContinueOnError)

	showVersion := flags.Bool("version", false, "Print version and exit")
	provider := flags.String("provider", "", "AI provider override")
	model := flags.String("model", "", "AI model override")
	autoCommit := flags.Bool("auto-commit", false, "Auto-commit after generation")
	autoCommitSet := false
	autoPush := flags.Bool("auto-push", false, "Auto-push after commit")
	autoPushSet := false

	// Custom usage
	flags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: nuntius [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Nuntius watches a Git repo for changes and generates AI-powered commit messages.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		return 1
	}

	// Track which boolean flags were explicitly set
	flags.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "auto-commit":
			autoCommitSet = true
		case "auto-push":
			autoPushSet = true
		}
	})

	if *showVersion {
		fmt.Printf("nuntius %s\n", version)
		return 0
	}

	// Load config (files + env vars)
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		return 1
	}

	// Merge CLI flags (highest priority)
	overrides := config.FlagOverrides{
		Provider: *provider,
		Model:    *model,
	}
	if autoCommitSet {
		overrides.AutoCommit = autoCommit
	}
	if autoPushSet {
		overrides.AutoPush = autoPush
	}
	config.MergeFlags(&cfg, overrides)

	// Validate current directory is a git repo
	if !isGitRepo() {
		fmt.Fprintf(os.Stderr, "Error: current directory is not a Git repository\n")
		return 1
	}

	// Placeholder — TUI will be connected in Phase 4
	_ = cfg
	fmt.Printf("nuntius %s — ready (provider: %s)\n", version, cfg.AI.Provider)
	return 0
}

// isGitRepo checks if the current directory is inside a Git repository.
func isGitRepo() bool {
	_, err := os.Stat(".git")
	return err == nil
}

