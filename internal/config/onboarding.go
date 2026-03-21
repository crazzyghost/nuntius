package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// OnboardingState records whether the user has completed or skipped onboarding.
type OnboardingState struct {
	Completed   bool      `json:"completed"`
	Skipped     bool      `json:"skipped"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}

func onboardingStatePath() string {
	dir := NuntiusDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "onboarding.json")
}

// ShouldOnboard returns true when no onboarding.json file exists yet.
func ShouldOnboard() bool {
	path := onboardingStatePath()
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}

// MarkOnboardingCompleted writes a completed onboarding state.
func MarkOnboardingCompleted() error {
	return writeOnboardingState(OnboardingState{
		Completed:   true,
		Skipped:     false,
		CompletedAt: time.Now().UTC(),
	})
}

// MarkOnboardingSkipped writes a skipped onboarding state.
func MarkOnboardingSkipped() error {
	return writeOnboardingState(OnboardingState{
		Completed: false,
		Skipped:   true,
	})
}

func writeOnboardingState(state OnboardingState) error {
	path := onboardingStatePath()
	if path == "" {
		return nil
	}
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshaling onboarding state: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing onboarding state to %s: %w", path, err)
	}
	return nil
}
