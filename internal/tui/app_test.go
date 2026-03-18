package tui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/crazzyghost/nuntius/internal/config"
	"github.com/crazzyghost/nuntius/internal/events"
)

func TestNewApp(t *testing.T) {
	app := NewApp(config.DefaultConfig())
	if app.Ready() {
		t.Error("app should not be ready before WindowSizeMsg")
	}
	if app.ShowHelp() {
		t.Error("help should be hidden initially")
	}
}

func TestAppInit(t *testing.T) {
	app := NewApp(config.DefaultConfig())
	cmd := app.Init()
	if cmd == nil {
		t.Error("Init should return a command")
	}
}

func TestAppWindowResize(t *testing.T) {
	app := NewApp(config.DefaultConfig())
	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m := model.(AppModel)

	if !m.Ready() {
		t.Error("app should be ready after WindowSizeMsg")
	}
	if m.Width() != 120 {
		t.Errorf("expected width 120, got %d", m.Width())
	}
	if m.Height() != 40 {
		t.Errorf("expected height 40, got %d", m.Height())
	}
}

func TestAppViewNotReady(t *testing.T) {
	app := NewApp(config.DefaultConfig())
	view := app.View()
	if view != "Starting Nuntius...\n" {
		t.Errorf("unexpected pre-ready view: %q", view)
	}
}

func TestAppViewReady(t *testing.T) {
	app := NewApp(config.DefaultConfig())
	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m := model.(AppModel)
	view := m.View()
	if view == "" {
		t.Error("view should not be empty when ready")
	}
}

func TestAppHelpToggle(t *testing.T) {
	app := NewApp(config.DefaultConfig())
	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m := model.(AppModel)

	// Toggle help on.
	model, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'?'}}))
	m = model.(AppModel)
	if !m.ShowHelp() {
		t.Error("help should be visible after '?' press")
	}

	// Toggle help off.
	model, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'?'}}))
	m = model.(AppModel)
	if m.ShowHelp() {
		t.Error("help should be hidden after second '?' press")
	}
}

func TestAppQuit(t *testing.T) {
	app := NewApp(config.DefaultConfig())
	model, cmd := app.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'q'}}))
	_ = model

	if cmd == nil {
		t.Error("quit should return a command")
	}
}

func TestAppFilesChangedRouting(t *testing.T) {
	app := NewApp(config.DefaultConfig())
	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m := model.(AppModel)

	files := []events.FileStatus{
		{Path: "test.go", Status: "modified", Staged: true},
	}
	model, _ = m.Update(events.FilesChangedMsg{Files: files})
	m = model.(AppModel)

	if len(m.viewport.Files()) != 1 {
		t.Error("files should be routed to viewport")
	}
}

func TestAppMessageReadyRouting(t *testing.T) {
	app := NewApp(config.DefaultConfig())
	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m := model.(AppModel)

	model, _ = m.Update(events.MessageReadyMsg{Message: "feat: test"})
	m = model.(AppModel)

	if !m.viewport.HasMessage() {
		t.Error("message should be routed to viewport")
	}
	if !m.actionbar.CommitEnabled() {
		t.Error("commit should be enabled after message ready")
	}
}

func TestAppCommitResultRouting(t *testing.T) {
	app := NewApp(config.DefaultConfig())
	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m := model.(AppModel)

	model, _ = m.Update(events.CommitResultMsg{Hash: "abc1234"})
	m = model.(AppModel)

	if m.status == "" {
		t.Error("status should be set after commit result")
	}
}

func TestAppCommitResultError(t *testing.T) {
	app := NewApp(config.DefaultConfig())
	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m := model.(AppModel)

	model, _ = m.Update(events.CommitResultMsg{Err: fmt.Errorf("fail")})
	m = model.(AppModel)

	if m.status == "" {
		t.Error("status should be set on commit error")
	}
}

func TestAppPushResultRouting(t *testing.T) {
	app := NewApp(config.DefaultConfig())
	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m := model.(AppModel)

	model, _ = m.Update(events.PushResultMsg{Remote: "origin"})
	m = model.(AppModel)

	if m.status == "" {
		t.Error("status should be set after push result")
	}
}

func TestAppErrorMsgRouting(t *testing.T) {
	app := NewApp(config.DefaultConfig())
	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m := model.(AppModel)

	model, _ = m.Update(events.ErrorMsg{Source: "generate", Err: fmt.Errorf("fail")})
	m = model.(AppModel)

	if m.status == "" {
		t.Error("status should be set on error")
	}
}

func TestAppGenerateWithNoStagedChanges(t *testing.T) {
	app := NewApp(config.DefaultConfig())
	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m := model.(AppModel)

	// Press 'g' with no staged changes.
	model, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'g'}}))
	m = model.(AppModel)

	if m.status == "" {
		t.Error("should show error about no staged changes")
	}
}

func TestAppTabCyclesFocus(t *testing.T) {
	app := NewApp(config.DefaultConfig())
	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m := model.(AppModel)

	// Enable commit by sending message ready.
	model, _ = m.Update(events.MessageReadyMsg{Message: "test"})
	m = model.(AppModel)

	// Tab should cycle focus.
	model, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyTab}))
	m = model.(AppModel)
	// After tab, focus should have moved.
	// Just verify it doesn't panic.
}
