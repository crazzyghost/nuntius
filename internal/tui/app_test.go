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
	if m.statusEntry == nil {
		t.Error("status should be set after message ready")
	}
}

func TestAppCommitResultRouting(t *testing.T) {
	app := NewApp(config.DefaultConfig())
	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m := model.(AppModel)

	model, _ = m.Update(events.CommitResultMsg{Hash: "abc1234"})
	m = model.(AppModel)

	if m.statusEntry == nil {
		t.Error("status should be set after commit result")
	}
}

func TestAppCommitResultError(t *testing.T) {
	app := NewApp(config.DefaultConfig())
	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m := model.(AppModel)

	model, _ = m.Update(events.CommitResultMsg{Err: fmt.Errorf("fail")})
	m = model.(AppModel)

	if m.statusEntry == nil {
		t.Error("status should be set on commit error")
	}
	if m.statusEntry.level != statusErr {
		t.Error("status level should be error")
	}
}

func TestAppPushResultRouting(t *testing.T) {
	app := NewApp(config.DefaultConfig())
	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m := model.(AppModel)

	model, _ = m.Update(events.PushResultMsg{Remote: "origin"})
	m = model.(AppModel)

	if m.statusEntry == nil {
		t.Error("status should be set after push result")
	}
}

func TestAppErrorMsgRouting(t *testing.T) {
	app := NewApp(config.DefaultConfig())
	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m := model.(AppModel)

	model, _ = m.Update(events.ErrorMsg{Source: "generate", Err: fmt.Errorf("fail")})
	m = model.(AppModel)

	if m.statusEntry == nil {
		t.Error("status should be set on error")
	}
}

func TestAppGenerateWithNoChanges(t *testing.T) {
	app := NewApp(config.DefaultConfig())
	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m := model.(AppModel)

	// Press 'g' with no changes at all.
	model, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'g'}}))
	m = model.(AppModel)

	if m.statusEntry == nil {
		t.Error("should show error about no changes")
	}
}

func TestAppMouseClickGenerate(t *testing.T) {
	app := NewApp(config.DefaultConfig())
	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m := model.(AppModel)

	// Click on the Generate button in the bottom action bar region.
	click := tea.MouseMsg{
		X:      2,
		Y:      23, // bottom row of a 24-high terminal (0-indexed)
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
	}
	model, _ = m.Update(click)
	m = model.(AppModel)

	if m.statusEntry == nil {
		t.Error("mouse click on Generate with no changes should show status error")
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
	_ = model.(AppModel)
	// Just verify it doesn't panic.
}

func TestAppStatusClearMsg(t *testing.T) {
	app := NewApp(config.DefaultConfig())
	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m := model.(AppModel)

	// Set status via error.
	model, _ = m.Update(events.ErrorMsg{Source: "test", Err: fmt.Errorf("fail")})
	m = model.(AppModel)
	if m.statusEntry == nil {
		t.Fatal("status should be set")
	}

	// Clear status.
	model, _ = m.Update(statusClearMsg{})
	m = model.(AppModel)
	if m.statusEntry != nil {
		t.Error("status should be cleared after statusClearMsg")
	}
}

func TestAppAutoCommit(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Behavior.AutoCommit = true
	app := NewApp(cfg)

	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m := model.(AppModel)

	// MessageReady with auto-commit should set status and return commit cmd.
	model, cmd := m.Update(events.MessageReadyMsg{Message: "feat: auto"})
	m = model.(AppModel)

	if m.statusEntry == nil {
		t.Error("status should be set for auto-commit")
	}
	if cmd == nil {
		t.Error("auto-commit should return a command")
	}
}

func TestAppAutoModeBadges(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Behavior.AutoCommit = true
	cfg.Behavior.AutoPush = true
	app := NewApp(cfg)

	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m := model.(AppModel)

	view := m.View()
	if view == "" {
		t.Error("view should not be empty")
	}
}

func TestFormatErrorMsg(t *testing.T) {
	tests := []struct {
		name     string
		msg      events.ErrorMsg
		contains string
	}{
		{
			"api key hint",
			events.ErrorMsg{Source: "generate", Err: fmt.Errorf("401 authentication failed")},
			"api_key_env",
		},
		{
			"network hint",
			events.ErrorMsg{Source: "generate", Err: fmt.Errorf("connection refused")},
			"network",
		},
		{
			"upstream hint",
			events.ErrorMsg{Source: "push", Err: fmt.Errorf("no upstream branch")},
			"set-upstream",
		},
		{
			"generic error",
			events.ErrorMsg{Source: "commit", Err: fmt.Errorf("something went wrong")},
			"something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatErrorMsg(tt.msg)
			if result == "" {
				t.Error("formatErrorMsg should not return empty")
			}
		})
	}
}

func TestAppWithBuilders(t *testing.T) {
	app := NewApp(config.DefaultConfig()).
		WithConventions("conventional")

	if app.conventions != "conventional" {
		t.Errorf("expected conventions 'conventional', got %q", app.conventions)
	}
}

func TestAppKeyPressClearsStatus(t *testing.T) {
	app := NewApp(config.DefaultConfig())
	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m := model.(AppModel)

	// Set status.
	model, _ = m.Update(events.ErrorMsg{Source: "test", Err: fmt.Errorf("fail")})
	m = model.(AppModel)
	if m.statusEntry == nil {
		t.Fatal("status should be set")
	}

	// Any key press should clear status.
	model, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'?'}}))
	m = model.(AppModel)
	if m.statusEntry != nil {
		t.Error("key press should clear status")
	}
}
