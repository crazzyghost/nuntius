package tui

import (
	"fmt"
	"testing"

	"github.com/crazzyghost/nuntius/internal/events"
)

func TestNewViewport(t *testing.T) {
	v := NewViewport()
	if v.mode != fileListMode {
		t.Error("new viewport should start in fileListMode")
	}
	if v.loading {
		t.Error("new viewport should not be loading")
	}
	if v.HasMessage() {
		t.Error("new viewport should have no message")
	}
}

func TestViewportSetSize(t *testing.T) {
	v := NewViewport()
	v.SetSize(80, 24)

	if !v.ready {
		t.Error("viewport should be ready after SetSize")
	}
	if v.width != 80 {
		t.Errorf("expected width 80, got %d", v.width)
	}
	if v.height != 24 {
		t.Errorf("expected height 24, got %d", v.height)
	}
}

func TestViewportFilesChangedMsg(t *testing.T) {
	v := NewViewport()
	v.SetSize(80, 24)

	files := []events.FileStatus{
		{Path: "main.go", Status: "modified", Staged: true},
		{Path: "new.go", Status: "added", Staged: false},
	}

	v, _ = v.Update(events.FilesChangedMsg{Files: files})

	if len(v.Files()) != 2 {
		t.Errorf("expected 2 files, got %d", len(v.Files()))
	}
	if v.Mode() != fileListMode {
		t.Error("viewport should stay in fileListMode after FilesChangedMsg")
	}
}

func TestViewportMessageReadyMsg(t *testing.T) {
	v := NewViewport()
	v.SetSize(80, 24)

	v, _ = v.Update(events.MessageReadyMsg{Message: "feat: add new feature"})

	if v.Mode() != messageMode {
		t.Error("viewport should switch to messageMode after MessageReadyMsg")
	}
	if !v.HasMessage() {
		t.Error("viewport should have a message")
	}
	if v.Message() != "feat: add new feature" {
		t.Errorf("unexpected message: %s", v.Message())
	}
	if v.loading {
		t.Error("viewport should not be loading after message is ready")
	}
}

func TestViewportGenerateRequestedMsg(t *testing.T) {
	v := NewViewport()
	v.SetSize(80, 24)

	v, _ = v.Update(events.GenerateRequestedMsg{})

	if !v.loading {
		t.Error("viewport should be loading after GenerateRequestedMsg")
	}
}

func TestViewportHasStagedChanges(t *testing.T) {
	v := NewViewport()
	v.SetSize(80, 24)

	if v.HasStagedChanges() {
		t.Error("should have no staged changes initially")
	}

	files := []events.FileStatus{
		{Path: "main.go", Status: "modified", Staged: false},
	}
	v, _ = v.Update(events.FilesChangedMsg{Files: files})
	if v.HasStagedChanges() {
		t.Error("should have no staged changes with only unstaged files")
	}

	files = append(files, events.FileStatus{Path: "go.mod", Status: "modified", Staged: true})
	v, _ = v.Update(events.FilesChangedMsg{Files: files})
	if !v.HasStagedChanges() {
		t.Error("should have staged changes")
	}
}

func TestViewportSwitchModes(t *testing.T) {
	v := NewViewport()
	v.SetSize(80, 24)

	// Generate a message first.
	v, _ = v.Update(events.MessageReadyMsg{Message: "test message"})
	if v.Mode() != messageMode {
		t.Error("should be in messageMode")
	}

	v.SwitchToFileList()
	if v.Mode() != fileListMode {
		t.Error("should switch to fileListMode")
	}

	v.SwitchToMessage()
	if v.Mode() != messageMode {
		t.Error("should switch back to messageMode")
	}
}

func TestViewportSwitchToMessageWithoutMessage(t *testing.T) {
	v := NewViewport()
	v.SetSize(80, 24)

	v.SwitchToMessage()
	if v.Mode() != fileListMode {
		t.Error("should stay in fileListMode when no message exists")
	}
}

func TestViewportViewNotReady(t *testing.T) {
	v := NewViewport()
	view := v.View()
	if view != "Initializing..." {
		t.Errorf("expected Initializing... when not ready, got %q", view)
	}
}

func TestViewportViewReady(t *testing.T) {
	v := NewViewport()
	v.SetSize(80, 24)

	view := v.View()
	if view == "" {
		t.Error("view should not be empty when ready")
	}
}

func TestRenderFileListEmpty(t *testing.T) {
	v := NewViewport()
	v.SetSize(80, 24)

	view := v.View()
	if view == "" {
		t.Error("view should render even with no files")
	}
}

func TestRenderFileListWithFiles(t *testing.T) {
	v := NewViewport()
	v.SetSize(80, 24)

	files := []events.FileStatus{
		{Path: "staged.go", Status: "modified", Staged: true},
		{Path: "unstaged.go", Status: "modified", Staged: false},
		{Path: "new.txt", Status: "untracked", Staged: false},
	}
	v, _ = v.Update(events.FilesChangedMsg{Files: files})

	view := v.View()
	if view == "" {
		t.Error("view should not be empty with files")
	}
}

func TestViewportErrorMsgStopsLoading(t *testing.T) {
	v := NewViewport()
	v.SetSize(80, 24)

	// Start a generate request — spinner should be loading.
	v, _ = v.Update(events.GenerateRequestedMsg{})
	if !v.loading {
		t.Fatal("viewport should be loading after GenerateRequestedMsg")
	}

	// Simulate an API error — spinner should stop.
	v, _ = v.Update(events.ErrorMsg{Source: "generate", Err: fmt.Errorf("API error 401")})
	if v.loading {
		t.Error("viewport should stop loading after ErrorMsg")
	}
}

func TestViewportErrorMsgStopsPushLoading(t *testing.T) {
	v := NewViewport()
	v.SetSize(80, 24)

	// Start a push request — spinner should be loading.
	v, _ = v.Update(events.PushRequestedMsg{})
	if !v.loading {
		t.Fatal("viewport should be loading after PushRequestedMsg")
	}

	// Simulate push error — spinner should stop.
	v, _ = v.Update(events.ErrorMsg{Source: "push", Err: fmt.Errorf("push rejected")})
	if v.loading {
		t.Error("viewport should stop loading after push ErrorMsg")
	}
}
