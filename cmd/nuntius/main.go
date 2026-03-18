package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/crazzyghost/nuntius/internal/ai"
	"github.com/crazzyghost/nuntius/internal/config"
	"github.com/crazzyghost/nuntius/internal/git"
	"github.com/crazzyghost/nuntius/internal/tui"
)

// version is injected at build time via -ldflags.
var version = "dev"

func main() {
	exitCode := run(os.Args[1:])
	os.Exit(exitCode)
}

// setupResult holds everything needed to launch the TUI.
type setupResult struct {
	cfg         config.Config
	app         tui.AppModel
	watcher     *git.Watcher
	cancel      context.CancelFunc
}

// setup parses flags, loads config, creates provider/watcher, and returns
// a setupResult ready to launch. Returns (result, exitCode, shouldLaunch).
func setup(args []string) (*setupResult, int, bool) {
	flags := flag.NewFlagSet("nuntius", flag.ContinueOnError)

	showVersion := flags.Bool("version", false, "Print version and exit")
	provider := flags.String("provider", "", "AI provider override")
	model := flags.String("model", "", "AI model override")
	autoCommit := flags.Bool("auto-commit", false, "Auto-commit after generation")
	autoCommitSet := false
	autoPush := flags.Bool("auto-push", false, "Auto-push after commit")
	autoPushSet := false

	flags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: nuntius [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Nuntius watches a Git repo for changes and generates AI-powered commit messages.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		return nil, 1, false
	}

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
		return nil, 0, false
	}

	// Load config (files + env vars)
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		return nil, 1, false
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
		return nil, 1, false
	}

	// Detect commit conventions.
	conventions := config.DetectConvention(cfg, 20)

	// Create AI provider.
	aiProvider, err := ai.NewProvider(cfg.AI)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: AI provider init failed: %v\n", err)
	}

	// Create file watcher.
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot determine working directory: %v\n", err)
		return nil, 1, false
	}

	watcher, err := git.NewWatcher(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot start file watcher: %v\n", err)
		return nil, 1, false
	}

	// Start watcher in background.
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		_ = watcher.Start(ctx)
	}()

	// Build the TUI model.
	app := tui.NewApp(cfg).
		WithWatcher(watcher).
		WithConventions(conventions)
	if aiProvider != nil {
		app = app.WithProvider(aiProvider)
	}

	return &setupResult{
		cfg:     cfg,
		app:     app,
		watcher: watcher,
		cancel:  cancel,
	}, 0, true
}

func run(args []string) int {
	result, exitCode, shouldLaunch := setup(args)
	if !shouldLaunch {
		return exitCode
	}
	defer result.cancel()
	defer result.watcher.Stop()

	// Launch Bubble Tea.
	p := tea.NewProgram(result.app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	return 0
}

// isGitRepo checks if the current directory is inside a Git repository.
func isGitRepo() bool {
	_, err := os.Stat(".git")
	return err == nil
}

