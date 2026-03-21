package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestShouldOnboardWhenFileMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if !ShouldOnboard() {
		t.Error("ShouldOnboard() should return true when onboarding.json does not exist")
	}
}

func TestShouldOnboardAfterCompleted(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := MarkOnboardingCompleted(); err != nil {
		t.Fatalf("MarkOnboardingCompleted() error: %v", err)
	}
	if ShouldOnboard() {
		t.Error("ShouldOnboard() should return false after MarkOnboardingCompleted()")
	}
}

func TestShouldOnboardAfterSkipped(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := MarkOnboardingSkipped(); err != nil {
		t.Fatalf("MarkOnboardingSkipped() error: %v", err)
	}
	if ShouldOnboard() {
		t.Error("ShouldOnboard() should return false after MarkOnboardingSkipped()")
	}
}

func TestMarkOnboardingCompletedState(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := MarkOnboardingCompleted(); err != nil {
		t.Fatalf("MarkOnboardingCompleted() error: %v", err)
	}

	path := filepath.Join(home, ".nuntius", "onboarding.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading onboarding state: %v", err)
	}
	var state OnboardingState
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("unmarshaling onboarding state: %v", err)
	}
	if !state.Completed {
		t.Error("expected completed = true")
	}
	if state.Skipped {
		t.Error("expected skipped = false")
	}
	if state.CompletedAt.IsZero() {
		t.Error("expected non-zero completed_at")
	}
}

func TestMarkOnboardingSkippedState(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := MarkOnboardingSkipped(); err != nil {
		t.Fatalf("MarkOnboardingSkipped() error: %v", err)
	}

	path := filepath.Join(home, ".nuntius", "onboarding.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading onboarding state: %v", err)
	}
	var state OnboardingState
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("unmarshaling onboarding state: %v", err)
	}
	if state.Completed {
		t.Error("expected completed = false")
	}
	if !state.Skipped {
		t.Error("expected skipped = true")
	}
}
