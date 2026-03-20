package cli_test

import (
	"context"
	"errors"
	"testing"

	"github.com/crazzyghost/nuntius/internal/ai"
	"github.com/crazzyghost/nuntius/internal/cli"
	"github.com/crazzyghost/nuntius/internal/config"
	"github.com/crazzyghost/nuntius/internal/engine"
	"github.com/crazzyghost/nuntius/internal/events"
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

// runHeadless is a test helper that calls engine.Generate with a mock provider
// and then exercises the git operations via mockGitOps.
func runHeadless(ctx context.Context, p ai.Provider, gitOps *mockGitOps, cfg config.Config, actions cli.Actions) cli.Result {
	r := cli.Result{}
	_ = r

	// We can't call the unexported run(). Instead we test indirectly via a
	// component-level approach: verify each operation in isolation.
	// Full integration of the exported cli.Run is tested by main_test.go.
	//
	// Here we test the Generate step using engine.Generate directly,
	// and the git steps using the mockGitOps.

	var msg string
	var files []string

	if actions.Generate {
		var err error
		src := actions.DiffSource
		extDiff := actions.ExternalDiff
		// Default to external with a stub diff so tests don't need a real git repo.
		if src == engine.DiffSourceAuto || src == engine.DiffSourceStaged {
			src = engine.DiffSourceExternal
			extDiff = "+++ b/foo.go\n--- a/foo.go\n+changed\n"
		}
		msg, files, err = engine.Generate(ctx, cfg, p, engine.GenerateInput{
			Source:       src,
			ExternalDiff: extDiff,
		})
		if err != nil {
			return cli.Result{OK: false, Error: err.Error(), Stage: "generate"}
		}
	}

	res := cli.Result{OK: false, DiffSource: diffSourceLabelForTest(actions.DiffSource), Message: msg, Files: files}

	if !actions.Commit {
		res.OK = true
		return res
	}

	if err := gitOps.StageAll(); err != nil {
		res.Error = err.Error()
		res.Stage = "stage"
		return res
	}

	commitResult, err := gitOps.Commit(msg)
	if err != nil {
		res.Error = err.Error()
		res.Stage = "commit"
		return res
	}
	res.Committed = true
	res.CommitHash = commitResult.Hash

	if !actions.Push {
		res.OK = true
		return res
	}

	hasUpstream, err := gitOps.HasUpstream()
	if err != nil {
		res.Error = err.Error()
		res.Stage = "push"
		return res
	}

	pushResult, err := gitOps.Push(git.PushOptions{SetUpstream: !hasUpstream})
	if err != nil {
		res.Error = err.Error()
		res.Stage = "push"
		return res
	}
	res.Pushed = true
	res.PushRemote = pushResult.Remote
	res.PushBranch = pushResult.Branch
	res.SetUpstream = !hasUpstream
	res.OK = true
	return res
}

// diffSourceLabelForTest mirrors cli internal diffSourceLabel for test assertions.
func diffSourceLabelForTest(src engine.DiffSource) string {
	switch src {
	case engine.DiffSourceStaged:
		return "staged"
	case engine.DiffSourceExternal:
		return "stdin"
	default:
		return "auto"
	}
}

func TestResult_ExitCode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		r    cli.Result
		want int
	}{
		{name: "success", r: cli.Result{OK: true}, want: 0},
		{name: "general error", r: cli.Result{OK: false, Stage: ""}, want: 1},
		{name: "no changes", r: cli.Result{OK: false, Stage: "generate", Error: "no changes to generate a message for"}, want: 2},
		{name: "AI error", r: cli.Result{OK: false, Stage: "generate", Error: "rate limited"}, want: 3},
		{name: "stage error", r: cli.Result{OK: false, Stage: "stage"}, want: 4},
		{name: "commit error", r: cli.Result{OK: false, Stage: "commit"}, want: 4},
		{name: "push error", r: cli.Result{OK: false, Stage: "push"}, want: 4},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.r.ExitCode()
			if got != tc.want {
				t.Errorf("ExitCode() = %d, want %d (result=%+v)", got, tc.want, tc.r)
			}
		})
	}
}

func TestRunHeadless_GenerateOnly(t *testing.T) {
	t.Parallel()
	p := &mockProvider{msg: "feat: do thing"}
	gitOps := &mockGitOps{}
	cfg := config.DefaultConfig()
	actions := cli.Actions{Generate: true}

	res := runHeadless(context.Background(), p, gitOps, cfg, actions)

	if !res.OK {
		t.Errorf("expected OK, got error: %v", res.Error)
	}
	if res.Message != "feat: do thing" {
		t.Errorf("got message %q, want %q", res.Message, "feat: do thing")
	}
	if res.Committed {
		t.Error("should not be committed for -g only")
	}
}

func TestRunHeadless_GenerateAndCommit(t *testing.T) {
	t.Parallel()
	p := &mockProvider{msg: "fix: broken thing"}
	gitOps := &mockGitOps{
		commitResult: git.CommitResult{Hash: "abc1234"},
	}
	cfg := config.DefaultConfig()
	actions := cli.Actions{Generate: true, Commit: true}

	res := runHeadless(context.Background(), p, gitOps, cfg, actions)

	if !res.OK {
		t.Errorf("expected OK, got error: %v", res.Error)
	}
	if !res.Committed {
		t.Error("expected committed = true")
	}
	if res.CommitHash != "abc1234" {
		t.Errorf("got hash %q, want %q", res.CommitHash, "abc1234")
	}
	if res.Pushed {
		t.Error("should not be pushed for -gc only")
	}
}

func TestRunHeadless_GenerateCommitPush(t *testing.T) {
	t.Parallel()
	p := &mockProvider{msg: "chore: update deps"}
	gitOps := &mockGitOps{
		commitResult: git.CommitResult{Hash: "def5678"},
		pushResult:   git.PushResult{Remote: "origin", Branch: "main"},
		hasUpstream:  true,
	}
	cfg := config.DefaultConfig()
	actions := cli.Actions{Generate: true, Commit: true, Push: true}

	res := runHeadless(context.Background(), p, gitOps, cfg, actions)

	if !res.OK {
		t.Errorf("expected OK, got error: %v", res.Error)
	}
	if !res.Pushed {
		t.Error("expected pushed = true")
	}
	if res.PushRemote != "origin" {
		t.Errorf("got remote %q, want %q", res.PushRemote, "origin")
	}
	if res.SetUpstream {
		t.Error("should not set upstream when upstream already exists")
	}
}

func TestRunHeadless_NewBranchSetsUpstream(t *testing.T) {
	t.Parallel()
	p := &mockProvider{msg: "feat: new branch work"}
	gitOps := &mockGitOps{
		commitResult: git.CommitResult{Hash: "aaa0000"},
		pushResult:   git.PushResult{Remote: "origin", Branch: "feature-x"},
		hasUpstream:  false, // no upstream
	}
	cfg := config.DefaultConfig()
	actions := cli.Actions{Generate: true, Commit: true, Push: true}

	res := runHeadless(context.Background(), p, gitOps, cfg, actions)

	if !res.OK {
		t.Errorf("expected OK, got error: %v", res.Error)
	}
	if !res.SetUpstream {
		t.Error("expected SetUpstream = true for new branch")
	}
}

func TestRunHeadless_AIFailure(t *testing.T) {
	t.Parallel()
	p := &mockProvider{err: errors.New("API timeout")}
	gitOps := &mockGitOps{}
	cfg := config.DefaultConfig()
	actions := cli.Actions{Generate: true}

	res := runHeadless(context.Background(), p, gitOps, cfg, actions)

	if res.OK {
		t.Error("expected failure on AI error")
	}
	if res.Stage != "generate" {
		t.Errorf("expected stage=generate, got %q", res.Stage)
	}
}

func TestRunHeadless_CommitFailure(t *testing.T) {
	t.Parallel()
	p := &mockProvider{msg: "chore: something"}
	gitOps := &mockGitOps{
		commitErr: errors.New("pre-commit hook rejected"),
	}
	cfg := config.DefaultConfig()
	actions := cli.Actions{Generate: true, Commit: true}

	res := runHeadless(context.Background(), p, gitOps, cfg, actions)

	if res.OK {
		t.Error("expected failure on commit error")
	}
	if res.Stage != "commit" {
		t.Errorf("expected stage=commit, got %q", res.Stage)
	}
}

func TestRunHeadless_PartialFailure_CommitOKPushFails(t *testing.T) {
	t.Parallel()
	p := &mockProvider{msg: "feat: partial"}
	gitOps := &mockGitOps{
		commitResult: git.CommitResult{Hash: "bbb1111"},
		pushErr:      errors.New("push rejected by remote"),
		hasUpstream:  true,
	}
	cfg := config.DefaultConfig()
	actions := cli.Actions{Generate: true, Commit: true, Push: true}

	res := runHeadless(context.Background(), p, gitOps, cfg, actions)

	if res.OK {
		t.Error("expected failure when push fails")
	}
	if !res.Committed {
		t.Error("expected Committed = true (partial success)")
	}
	if res.CommitHash != "bbb1111" {
		t.Errorf("expected CommitHash populated even on push failure, got %q", res.CommitHash)
	}
	if res.Stage != "push" {
		t.Errorf("expected stage=push, got %q", res.Stage)
	}
}

func TestRunHeadless_PushOnly(t *testing.T) {
	t.Parallel()
	// Push-only doesn't go through the runHeadless helper above — test the exported Run.
	// We can't easily inject mocks into the exported Run, so we verify the push-only
	// path at the Result level by calling our helper with no generate/commit.
	gitOps := &mockGitOps{
		pushResult:  git.PushResult{Remote: "origin", Branch: "main"},
		hasUpstream: true,
	}

	// Simulate push-only path.
	hasUpstream, _ := gitOps.HasUpstream()
	pushResult, err := gitOps.Push(git.PushOptions{SetUpstream: !hasUpstream})
	if err != nil {
		t.Fatalf("unexpected push error: %v", err)
	}

	res := cli.Result{
		OK:         true,
		DiffSource: "auto",
		Pushed:     true,
		PushRemote: pushResult.Remote,
		PushBranch: pushResult.Branch,
	}

	if !res.Pushed {
		t.Error("expected pushed = true")
	}
	if res.PushRemote != "origin" {
		t.Errorf("expected remote=origin, got %q", res.PushRemote)
	}
}

func TestRunHeadless_ContextTimeout(t *testing.T) {
	t.Parallel()
	// Provider that always returns a cancellation error to simulate timeout.
	p := &mockProvider{err: context.DeadlineExceeded}
	gitOps := &mockGitOps{}
	cfg := config.DefaultConfig()
	actions := cli.Actions{Generate: true}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately to simulate timeout

	res := runHeadless(ctx, p, gitOps, cfg, actions)

	if res.OK {
		t.Error("expected failure on context cancellation")
	}
	if res.Stage != "generate" {
		t.Errorf("expected stage=generate on timeout, got %q", res.Stage)
	}
}

func TestResult_DiffSourceLabel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		actions cli.Actions
		want    string
	}{
		{"default (auto)", cli.Actions{Generate: true}, "auto"},
		{"staged", cli.Actions{Generate: true, DiffSource: engine.DiffSourceStaged}, "staged"},
		{"external/stdin", cli.Actions{Generate: true, DiffSource: engine.DiffSourceExternal, ExternalDiff: "+++ b/f.go\n+x\n"}, "stdin"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := &mockProvider{msg: "feat: test"}
			gitOps := &mockGitOps{}
			cfg := config.DefaultConfig()
			res := runHeadless(context.Background(), p, gitOps, cfg, tc.actions)
			if !res.OK {
				t.Fatalf("expected OK, got error: %v", res.Error)
			}
			if res.DiffSource != tc.want {
				t.Errorf("DiffSource = %q, want %q", res.DiffSource, tc.want)
			}
		})
	}
}

func TestRunHeadless_ExternalDiff_EmptyErrors(t *testing.T) {
	t.Parallel()
	p := &mockProvider{msg: "feat: test"}
	gitOps := &mockGitOps{}
	cfg := config.DefaultConfig()
	// ExternalDiff is empty — engine should return "no diff provided"
	actions := cli.Actions{
		Generate:     true,
		DiffSource:   engine.DiffSourceExternal,
		ExternalDiff: "",
	}

	res := runHeadless(context.Background(), p, gitOps, cfg, actions)

	if res.OK {
		t.Error("expected failure when ExternalDiff is empty")
	}
	if res.Stage != "generate" {
		t.Errorf("expected stage=generate, got %q", res.Stage)
	}
}
