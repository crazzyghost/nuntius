package git

import (
	"strings"
	"testing"
)

func TestBuildPushArgs_NormalPush(t *testing.T) {
	args := BuildPushArgs(PushOptions{ForceWithLease: false})
	expected := []string{"push"}

	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d", len(expected), len(args))
	}
	for i, arg := range args {
		if arg != expected[i] {
			t.Errorf("arg[%d] = %q, want %q", i, arg, expected[i])
		}
	}
}

func TestBuildPushArgs_ForceWithLease(t *testing.T) {
	args := BuildPushArgs(PushOptions{ForceWithLease: true})
	expected := []string{"push", "--force-with-lease"}

	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d", len(expected), len(args))
	}
	for i, arg := range args {
		if arg != expected[i] {
			t.Errorf("arg[%d] = %q, want %q", i, arg, expected[i])
		}
	}
}

func TestBuildPushArgs_NeverBareForce(t *testing.T) {
	// Verify that --force is never used, only --force-with-lease
	args := BuildPushArgs(PushOptions{ForceWithLease: true})
	for _, arg := range args {
		if arg == "--force" {
			t.Error("must use --force-with-lease, never bare --force")
		}
	}
}

func TestBuildPushArgs_SetUpstream(t *testing.T) {
	// SetUpstream must include --set-upstream, origin, and a branch name.
	args := BuildPushArgs(PushOptions{SetUpstream: true})

	// Minimum: ["push", "--set-upstream", "origin", "<branch>"]
	if len(args) < 4 {
		t.Fatalf("expected at least 4 args with SetUpstream, got %d: %v", len(args), args)
	}

	found := false
	for i, arg := range args {
		if arg == "--set-upstream" {
			found = true
			if i+2 >= len(args) {
				t.Fatal("expected 'origin' and branch after --set-upstream")
			}
			if args[i+1] != "origin" {
				t.Errorf("expected 'origin' after --set-upstream, got %q", args[i+1])
			}
			if args[i+2] == "" {
				t.Error("branch name after --set-upstream must not be empty")
			}
			break
		}
	}
	if !found {
		t.Errorf("expected --set-upstream in args, got %v", args)
	}
}

func TestBuildPushArgs_SetUpstreamAndForce(t *testing.T) {
	args := BuildPushArgs(PushOptions{ForceWithLease: true, SetUpstream: true})

	hasForce := false
	hasUpstream := false
	for _, arg := range args {
		if arg == "--force-with-lease" {
			hasForce = true
		}
		if arg == "--set-upstream" {
			hasUpstream = true
		}
	}
	if !hasForce {
		t.Error("expected --force-with-lease in args")
	}
	if !hasUpstream {
		t.Error("expected --set-upstream in args")
	}
}

func TestCurrentBranch_ReturnsNonEmpty(t *testing.T) {
	// This runs against the actual nuntius repo — should have a branch.
	branch := CurrentBranch()
	if branch == "" {
		t.Skip("not in a git repo with a valid HEAD")
	}
	if strings.Contains(branch, "\n") {
		t.Errorf("branch name contains newline: %q", branch)
	}
}

func TestHasUpstream_DoesNotError(t *testing.T) {
	// Just verify HasUpstream runs without returning an unexpected error.
	// The actual boolean depends on whether origin is configured in CI.
	_, err := HasUpstream()
	if err != nil {
		t.Errorf("HasUpstream returned unexpected error: %v", err)
	}
}
