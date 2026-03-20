package cli

import (
	"context"
	"time"

	"github.com/crazzyghost/nuntius/internal/ai"
	"github.com/crazzyghost/nuntius/internal/config"
	"github.com/crazzyghost/nuntius/internal/engine"
	"github.com/crazzyghost/nuntius/internal/git"
)

// Actions describes which headless operations to perform.
type Actions struct {
	Generate bool
	Commit   bool
	Push     bool
	// DiffSource controls where the diff for generation comes from.
	// Defaults to engine.DiffSourceAuto (staged + unstaged).
	DiffSource engine.DiffSource
	// ExternalDiff holds a pre-read diff when DiffSource is DiffSourceExternal.
	// Populated by the caller (e.g. from stdin) before invoking Run.
	ExternalDiff string
}

// Run executes the headless CLI pipeline and returns a structured Result.
// It uses real git and AI provider operations.
func Run(ctx context.Context, cfg config.Config, provider ai.Provider, actions Actions) Result {
	return run(ctx, cfg, provider, actions, git.DefaultOps{})
}

// run is the internal implementation that accepts a git.Ops for testability.
func run(ctx context.Context, cfg config.Config, provider ai.Provider, actions Actions, gitOps git.Ops) Result {
	// Push-only mode: push existing unpushed commits without generating.
	if actions.Push && !actions.Generate && !actions.Commit {
		return doPush(ctx, gitOps, cfg.Behavior.ForcePush)
	}

	r := Result{DiffSource: diffSourceLabel(actions.DiffSource)}

	if !actions.Generate {
		// Invalid combination — caller should validate before calling Run.
		r.Error = "generate is required"
		r.Stage = "generate"
		return r
	}

	// Wrap the AI call with a configurable timeout.
	timeout := 60 * time.Second
	if cfg.AI.TimeoutSeconds > 0 {
		timeout = time.Duration(cfg.AI.TimeoutSeconds) * time.Second
	}
	genCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	msg, files, err := engine.Generate(genCtx, cfg, provider, engine.GenerateInput{
		Source:       actions.DiffSource,
		ExternalDiff: actions.ExternalDiff,
	})
	if err != nil {
		r.Error = err.Error()
		r.Stage = "generate"
		return r
	}
	r.Message = msg
	r.Files = files

	if !actions.Commit {
		r.OK = true
		return r // -g only
	}

	// Stage all changes.
	if err := gitOps.StageAll(); err != nil {
		r.Error = err.Error()
		r.Stage = "stage"
		return r
	}

	// Commit.
	commitResult, err := gitOps.Commit(msg)
	if err != nil {
		r.Error = err.Error()
		r.Stage = "commit"
		return r
	}
	r.Committed = true
	r.CommitHash = commitResult.Hash

	if !actions.Push {
		r.OK = true
		return r // -gc only
	}

	// Push.
	remote, branch, setUpstream, pushErr := executePush(ctx, gitOps, cfg.Behavior.ForcePush)
	if pushErr != nil {
		r.Error = pushErr.Error()
		r.Stage = "push"
		return r
	}
	r.Pushed = true
	r.PushRemote = remote
	r.PushBranch = branch
	r.SetUpstream = setUpstream
	r.OK = true
	return r
}

// doPush handles the push-only (-p without -g/-c) path.
func doPush(_ context.Context, gitOps git.Ops, forceWithLease bool) Result {
	r := Result{DiffSource: "auto"}
	remote, branch, setUpstream, err := executePush(context.Background(), gitOps, forceWithLease)
	if err != nil {
		r.Error = err.Error()
		r.Stage = "push"
		return r
	}
	r.Pushed = true
	r.PushRemote = remote
	r.PushBranch = branch
	r.SetUpstream = setUpstream
	r.OK = true
	return r
}

// executePush performs the actual push, auto-setting upstream when needed.
func executePush(_ context.Context, gitOps git.Ops, forceWithLease bool) (remote, branch string, setUpstream bool, err error) {
	hasUpstream, err := gitOps.HasUpstream()
	if err != nil {
		return "", "", false, err
	}

	opts := git.PushOptions{
		ForceWithLease: forceWithLease,
		SetUpstream:    !hasUpstream,
	}

	result, err := gitOps.Push(opts)
	if err != nil {
		return "", "", false, err
	}
	return result.Remote, result.Branch, !hasUpstream, nil
}

// diffSourceLabel converts an engine.DiffSource to the string written into
// Result.DiffSource, matching the CLI flag value (auto, staged, stdin).
func diffSourceLabel(src engine.DiffSource) string {
	switch src {
	case engine.DiffSourceStaged:
		return "staged"
	case engine.DiffSourceExternal:
		return "stdin"
	default:
		return "auto"
	}
}
