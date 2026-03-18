package git

import (
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
