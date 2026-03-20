package main

import (
	"context"
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	pflag "github.com/spf13/pflag"

	"github.com/crazzyghost/nuntius/internal/ai"
	"github.com/crazzyghost/nuntius/internal/cli"
	"github.com/crazzyghost/nuntius/internal/config"
	"github.com/crazzyghost/nuntius/internal/git"
	"github.com/crazzyghost/nuntius/internal/tui"
)

// Build-time variables injected via -ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	exitCode := run(os.Args[1:])
	os.Exit(exitCode)
}

// setupResult holds everything needed to launch the TUI.
type setupResult struct {
	cfg     config.Config
	app     tui.AppModel
	watcher *git.Watcher
	cancel  context.CancelFunc
}

// setup parses flags, loads config, creates provider/watcher, and returns
// a setupResult ready to launch. Returns (result, exitCode, shouldLaunch).
func setup(args []string) (*setupResult, int, bool) {
	flags := newFlagSet(os.Stderr)
	autoCommitSet := false
	autoPushSet := false

	if err := flags.Parse(args); err != nil {
		return nil, 1, false
	}

	// Emit deprecation warning if --provider was used.
	flags.Visit(func(f *pflag.Flag) {
		switch f.Name {
		case "provider":
			fmt.Fprintln(os.Stderr, "Warning: --provider is deprecated, use --agent (-a) instead")
		case "auto-commit":
			autoCommitSet = true
		case "auto-push":
			autoPushSet = true
		}
	})

	showVersion, _ := flags.GetBool("version")
	if showVersion {
		fmt.Printf("nuntius %s (commit=%s, built=%s)\n", version, commit, date)
		return nil, 0, false
	}

	// Resolve the agent/provider flag (--agent/-a is primary, --provider is deprecated alias).
	agent := resolveAgentFlag(flags)
	model, _ := flags.GetString("model")
	autoCommit, _ := flags.GetBool("auto-commit")
	autoPush, _ := flags.GetBool("auto-push")
	noUpdateCheck, _ := flags.GetBool("no-update-check")

	// Load config (files + env vars)
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		return nil, 1, false
	}

	// Merge CLI flags (highest priority)
	overrides := config.FlagOverrides{
		Provider: agent,
		Model:    model,
	}
	if autoCommitSet {
		overrides.AutoCommit = &autoCommit
	}
	if autoPushSet {
		overrides.AutoPush = &autoPush
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
		WithConventions(conventions).
		WithVersion(version, date)
	if aiProvider != nil {
		app = app.WithProvider(aiProvider)
	}
	if noUpdateCheck {
		app = app.WithNoUpdateCheck()
	}

	return &setupResult{
		cfg:     cfg,
		app:     app,
		watcher: watcher,
		cancel:  cancel,
	}, 0, true
}

func run(args []string) int {
	flags := newFlagSet(os.Stderr)

	if err := flags.Parse(args); err != nil {
		return 1
	}

	// Emit deprecation warning if --provider was used.
	flags.Visit(func(f *pflag.Flag) {
		if f.Name == "provider" {
			fmt.Fprintln(os.Stderr, "Warning: --provider is deprecated, use --agent (-a) instead")
		}
	})

	showVersion, _ := flags.GetBool("version")
	if showVersion {
		fmt.Printf("nuntius %s (commit=%s, built=%s)\n", version, commit, date)
		return 0
	}

	// Detect headless mode by checking whether any action flag was explicitly
	// set on the command line. This is intentionally NOT based on config values
	// so that config.toml auto_commit=true does not trigger headless mode.
	isHeadless := flags.Changed("generate") || flags.Changed("auto-commit") || flags.Changed("auto-push")

	if !isHeadless {
		// TUI path — delegate to the full setup function.
		return launchTUI(args)
	}

	// --- Headless path ---

	// Load config (files + env vars)
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		return 1
	}

	// Merge CLI flags.
	agent := resolveAgentFlag(flags)
	model, _ := flags.GetString("model")
	overrides := config.FlagOverrides{
		Provider: agent,
		Model:    model,
	}
	flags.Visit(func(f *pflag.Flag) {
		switch f.Name {
		case "auto-commit":
			v, _ := flags.GetBool("auto-commit")
			overrides.AutoCommit = &v
		case "auto-push":
			v, _ := flags.GetBool("auto-push")
			overrides.AutoPush = &v
		}
	})
	config.MergeFlags(&cfg, overrides)

	// Validate current directory is a git repo.
	if !isGitRepo() {
		fmt.Fprintf(os.Stderr, "Error: current directory is not a Git repository\n")
		return 1
	}

	generate, _ := flags.GetBool("generate")
	autoCommit, _ := flags.GetBool("auto-commit")
	autoPush, _ := flags.GetBool("auto-push")

	actions := cli.Actions{
		Generate: generate,
		Commit:   autoCommit,
		Push:     autoPush,
	}

	// Validate flag combinations before doing any work.
	if err := validateHeadlessCombination(actions); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	// Create AI provider (only needed for generate).
	var aiProvider ai.Provider
	if actions.Generate {
		var provErr error
		aiProvider, provErr = ai.NewProvider(cfg.AI)
		if provErr != nil {
			fmt.Fprintf(os.Stderr, "Error: AI provider init failed: %v\n", provErr)
			return 1
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	result := cli.Run(ctx, cfg, aiProvider, actions)

	// Write commit message to stdout (clean for piping).
	if result.Message != "" {
		if _, err := fmt.Fprint(os.Stdout, result.Message); err != nil {
			fmt.Fprintf(os.Stderr, "error writing message: %v\n", err)
		}
	}
	if result.Committed {
		fmt.Fprintf(os.Stderr, "committed %s\n", result.CommitHash)
	}
	if result.Pushed {
		remote := result.PushRemote
		if remote == "" {
			remote = "origin"
		}
		if result.PushBranch != "" {
			fmt.Fprintf(os.Stderr, "pushed to %s/%s\n", remote, result.PushBranch)
		} else {
			fmt.Fprintf(os.Stderr, "pushed to %s\n", remote)
		}
	}
	if !result.OK {
		fmt.Fprintf(os.Stderr, "Error: %s\n", result.Error)
	}
	return result.ExitCode()
}

// launchTUI runs the Bubble Tea TUI and returns an exit code.
func launchTUI(args []string) int {
	result, exitCode, shouldLaunch := setup(args)
	if !shouldLaunch {
		return exitCode
	}
	defer result.cancel()
	defer result.watcher.Stop()

	p := tea.NewProgram(result.app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}

// validateHeadlessCombination checks that the flag combination is valid.
func validateHeadlessCombination(a cli.Actions) error {
	if a.Commit && !a.Generate {
		return fmt.Errorf("--auto-commit (-c) requires --generate (-g)")
	}
	if a.Push && a.Generate && !a.Commit {
		return fmt.Errorf("--auto-push (-p) requires --auto-commit (-c) when used with --generate (-g)")
	}
	return nil
}

// resolveAgentFlag returns the AI provider value from flags, preferring
// --agent/-a over the deprecated --provider alias.
func resolveAgentFlag(flags *pflag.FlagSet) string {
	if flags.Changed("agent") {
		v, _ := flags.GetString("agent")
		return v
	}
	if flags.Changed("provider") {
		v, _ := flags.GetString("provider")
		return v
	}
	v, _ := flags.GetString("agent")
	return v
}

// newFlagSet builds and returns the pflag.FlagSet with all registered flags.
func newFlagSet(output io.Writer) *pflag.FlagSet {
	flags := pflag.NewFlagSet("nuntius", pflag.ContinueOnError)
	flags.SetOutput(output)
	flags.SortFlags = false

	flags.Bool("version", false, "Print version and exit")

	// AI provider flag — primary: --agent/-a; deprecated alias: --provider (hidden).
	flags.StringP("agent", "a", "", "AI provider (claude, gemini, codex, copilot, ollama)")
	flags.String("provider", "", "AI provider override (deprecated: use --agent/-a)")
	flags.Lookup("provider").Hidden = true

	flags.String("model", "", "AI model override")

	// Headless action flags — also control TUI auto-behavior when set via config.
	flags.BoolP("generate", "g", false, "Generate commit message and print to stdout")
	flags.BoolP("auto-commit", "c", false, "Stage all and commit with generated message")
	flags.BoolP("auto-push", "p", false, "Push after commit (sets upstream for new branches)")

	flags.Bool("force-push", false, "Use --force-with-lease when pushing (no short alias)")
	flags.Bool("no-update-check", false, "Disable startup version check")

	flags.Usage = func() {
		_, _ = fmt.Fprintf(flags.Output(), "Usage: nuntius [flags]\n\n")
		_, _ = fmt.Fprintf(flags.Output(), "Nuntius watches a Git repo for changes and generates AI-powered commit messages.\n\n")
		_, _ = fmt.Fprintf(flags.Output(), "Flags:\n")
		flags.PrintDefaults()
		_, _ = fmt.Fprintf(flags.Output(), "\nExamples:\n")
		_, _ = fmt.Fprintf(flags.Output(), "  nuntius              Launch interactive TUI\n")
		_, _ = fmt.Fprintf(flags.Output(), "  nuntius -g           Generate message and print to stdout\n")
		_, _ = fmt.Fprintf(flags.Output(), "  nuntius -gc          Generate and commit\n")
		_, _ = fmt.Fprintf(flags.Output(), "  nuntius -gcp         Generate, commit, and push\n")
		_, _ = fmt.Fprintf(flags.Output(), "  nuntius -p           Push existing unpushed commits\n")
		_, _ = fmt.Fprintf(flags.Output(), "  nuntius -g | git commit -F -   Pipe message to git commit\n")
	}

	return flags
}

// isGitRepo checks if the current directory is inside a Git repository.
func isGitRepo() bool {
	_, err := os.Stat(".git")
	return err == nil
}
