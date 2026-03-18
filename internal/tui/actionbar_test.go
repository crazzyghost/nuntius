package tui

import (
	"fmt"
	"testing"

	"github.com/crazzyghost/nuntius/internal/events"
)

func TestNewActionBar(t *testing.T) {
	ab := NewActionBar()

	if !ab.GenerateEnabled() {
		t.Error("Generate should be enabled initially")
	}
	if ab.CommitEnabled() {
		t.Error("Commit should be disabled initially")
	}
	if ab.PushEnabled() {
		t.Error("Push should be disabled initially")
	}
	if ab.Committed() {
		t.Error("should not have committed initially")
	}
}

func TestActionBarGenerateFlow(t *testing.T) {
	ab := NewActionBar()

	// Request generation.
	ab, _ = ab.Update(events.GenerateRequestedMsg{})
	if ab.buttons[0].state != btnLoading {
		t.Error("Generate should be loading after GenerateRequestedMsg")
	}

	// Message ready.
	ab, _ = ab.Update(events.MessageReadyMsg{Message: "test"})
	if ab.buttons[0].state != btnNormal {
		t.Error("Generate should return to normal after MessageReadyMsg")
	}
	if !ab.CommitEnabled() {
		t.Error("Commit should be enabled after MessageReadyMsg")
	}
}

func TestActionBarCommitFlow(t *testing.T) {
	ab := NewActionBar()

	// Enable commit via message ready.
	ab, _ = ab.Update(events.MessageReadyMsg{Message: "test"})

	// Successful commit.
	ab, _ = ab.Update(events.CommitResultMsg{Hash: "abc1234"})
	if ab.buttons[1].state != btnSuccess {
		t.Error("Commit should show success after CommitResultMsg")
	}
	if !ab.Committed() {
		t.Error("should be committed after successful CommitResultMsg")
	}
	if !ab.PushEnabled() {
		t.Error("Push should be enabled after successful commit")
	}
}

func TestActionBarCommitError(t *testing.T) {
	ab := NewActionBar()
	ab, _ = ab.Update(events.MessageReadyMsg{Message: "test"})

	ab, _ = ab.Update(events.CommitResultMsg{Err: fmt.Errorf("nothing to commit")})
	if ab.buttons[1].state != btnError {
		t.Error("Commit should show error on failure")
	}
	if ab.Committed() {
		t.Error("should not be committed after failed commit")
	}
}

func TestActionBarPushFlow(t *testing.T) {
	ab := NewActionBar()
	ab, _ = ab.Update(events.MessageReadyMsg{Message: "test"})
	ab, _ = ab.Update(events.CommitResultMsg{Hash: "abc1234"})

	// Successful push.
	ab, _ = ab.Update(events.PushResultMsg{Remote: "origin"})
	if ab.buttons[2].state != btnSuccess {
		t.Error("Push should show success")
	}
}

func TestActionBarPushError(t *testing.T) {
	ab := NewActionBar()
	ab, _ = ab.Update(events.MessageReadyMsg{Message: "test"})
	ab, _ = ab.Update(events.CommitResultMsg{Hash: "abc1234"})

	ab, _ = ab.Update(events.PushResultMsg{Err: fmt.Errorf("rejected")})
	if ab.buttons[2].state != btnError {
		t.Error("Push should show error on failure")
	}
}

func TestActionBarFocusNext(t *testing.T) {
	ab := NewActionBar()

	// Only Generate is enabled; focus should stay on it.
	ab.FocusNext()
	if ab.focusIndex != 0 {
		t.Errorf("focus should stay on 0 when only Generate is enabled, got %d", ab.focusIndex)
	}

	// Enable Commit.
	ab, _ = ab.Update(events.MessageReadyMsg{Message: "test"})
	ab.FocusNext()
	if ab.focusIndex != 1 {
		t.Errorf("focus should move to 1, got %d", ab.focusIndex)
	}
}

func TestActionBarFocusedAction(t *testing.T) {
	ab := NewActionBar()

	if ab.FocusedAction() != "" {
		t.Error("no button should be focused initially")
	}

	ab.buttons[0].state = btnFocused
	if ab.FocusedAction() != "g" {
		t.Errorf("focused action should be 'g', got %q", ab.FocusedAction())
	}
}

func TestActionBarView(t *testing.T) {
	ab := NewActionBar()
	view := ab.View()
	if view == "" {
		t.Error("action bar view should not be empty")
	}
}

func TestButtonClearMsg(t *testing.T) {
	ab := NewActionBar()
	ab, _ = ab.Update(events.MessageReadyMsg{Message: "test"})
	ab, _ = ab.Update(events.CommitResultMsg{Hash: "abc"})

	// Clear the commit button.
	ab, _ = ab.Update(buttonClearMsg{index: 1})
	if ab.buttons[1].feedback != "" {
		t.Error("feedback should be cleared")
	}
}

func TestActionBarErrorMsg(t *testing.T) {
	ab := NewActionBar()

	ab, _ = ab.Update(events.ErrorMsg{Source: "generate", Err: fmt.Errorf("err")})
	if ab.buttons[0].state != btnError {
		t.Error("Generate should show error on ErrorMsg")
	}
}

func TestIsButtonEnabledBounds(t *testing.T) {
	ab := NewActionBar()
	if ab.IsButtonEnabled(-1) {
		t.Error("should return false for negative index")
	}
	if ab.IsButtonEnabled(3) {
		t.Error("should return false for out-of-range index")
	}
}

func TestVisibleWidth(t *testing.T) {
	if w := visibleWidth("hello"); w != 5 {
		t.Errorf("expected 5, got %d", w)
	}
	// ANSI escape should be stripped.
	if w := visibleWidth("\x1b[32mhi\x1b[0m"); w != 2 {
		t.Errorf("expected 2 for ANSI-colored 'hi', got %d", w)
	}
}

func TestHitTest(t *testing.T) {
	ab := NewActionBar()
	// Call View to populate zones.
	ab.View()

	// First button starts at x=0.
	idx := ab.HitTest(0)
	if idx != 0 {
		t.Errorf("expected button 0 at x=0, got %d", idx)
	}

	// Past all buttons should return -1.
	idx = ab.HitTest(999)
	if idx != -1 {
		t.Errorf("expected -1 for x=999, got %d", idx)
	}

	// Somewhere in the second button zone.
	z1 := ab.zones[1]
	if z1.endX > z1.startX {
		idx = ab.HitTest(z1.startX)
		if idx != 1 {
			t.Errorf("expected button 1 at x=%d, got %d", z1.startX, idx)
		}
	}
}
