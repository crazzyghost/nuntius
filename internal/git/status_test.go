package git

import (
	"testing"

	"github.com/crazzyghost/nuntius/internal/events"
)

func TestParsePorcelainV2_OrdinaryModified(t *testing.T) {
	input := "1 .M N... 100644 100644 100644 abc123 def456 internal/config/config.go\n"
	files, err := parsePorcelainV2(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	assertFileStatus(t, files[0], "internal/config/config.go", "modified", false)
}

func TestParsePorcelainV2_StagedModified(t *testing.T) {
	input := "1 M. N... 100644 100644 100644 abc123 def456 internal/config/config.go\n"
	files, err := parsePorcelainV2(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	assertFileStatus(t, files[0], "internal/config/config.go", "modified", true)
}

func TestParsePorcelainV2_BothStagedAndUnstaged(t *testing.T) {
	input := "1 MM N... 100644 100644 100644 abc123 def456 main.go\n"
	files, err := parsePorcelainV2(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	assertFileStatus(t, files[0], "main.go", "modified", true)
	assertFileStatus(t, files[1], "main.go", "modified", false)
}

func TestParsePorcelainV2_Added(t *testing.T) {
	input := "1 A. N... 000000 100644 100644 0000000 abc123 new_file.go\n"
	files, err := parsePorcelainV2(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	assertFileStatus(t, files[0], "new_file.go", "added", true)
}

func TestParsePorcelainV2_Deleted(t *testing.T) {
	input := "1 D. N... 100644 000000 000000 abc123 0000000 old_file.go\n"
	files, err := parsePorcelainV2(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	assertFileStatus(t, files[0], "old_file.go", "deleted", true)
}

func TestParsePorcelainV2_Untracked(t *testing.T) {
	input := "? untracked-file.txt\n"
	files, err := parsePorcelainV2(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	assertFileStatus(t, files[0], "untracked-file.txt", "untracked", false)
}

func TestParsePorcelainV2_Renamed(t *testing.T) {
	input := "2 R. N... 100644 100644 100644 abc123 def456 R100 new_name.go\told_name.go\n"
	files, err := parsePorcelainV2(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	assertFileStatus(t, files[0], "new_name.go", "renamed", true)
}

func TestParsePorcelainV2_Unmerged(t *testing.T) {
	input := "u UU N... 100644 100644 100644 100644 abc123 def456 ghi789 conflict.go\n"
	files, err := parsePorcelainV2(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	assertFileStatus(t, files[0], "conflict.go", "unmerged", false)
}

func TestParsePorcelainV2_Ignored(t *testing.T) {
	input := "! ignored-file.log\n"
	files, err := parsePorcelainV2(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected 0 files for ignored entry, got %d", len(files))
	}
}

func TestParsePorcelainV2_EmptyOutput(t *testing.T) {
	files, err := parsePorcelainV2("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected 0 files for empty output, got %d", len(files))
	}
}

func TestParsePorcelainV2_MixedOutput(t *testing.T) {
	input := "1 M. N... 100644 100644 100644 abc123 def456 staged.go\n" +
		"1 .M N... 100644 100644 100644 abc123 def456 unstaged.go\n" +
		"? new_file.txt\n" +
		"1 A. N... 000000 100644 100644 0000000 abc123 added.go\n"

	files, err := parsePorcelainV2(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 4 {
		t.Fatalf("expected 4 files, got %d", len(files))
	}
	assertFileStatus(t, files[0], "staged.go", "modified", true)
	assertFileStatus(t, files[1], "unstaged.go", "modified", false)
	assertFileStatus(t, files[2], "new_file.txt", "untracked", false)
	assertFileStatus(t, files[3], "added.go", "added", true)
}

func TestStatusLabel(t *testing.T) {
	tests := []struct {
		code byte
		want string
	}{
		{'M', "modified"},
		{'A', "added"},
		{'D', "deleted"},
		{'R', "renamed"},
		{'C', "copied"},
		{'T', "type-changed"},
		{'U', "unmerged"},
		{'X', "unknown"},
	}
	for _, tt := range tests {
		got := statusLabel(tt.code)
		if got != tt.want {
			t.Errorf("statusLabel(%q) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func assertFileStatus(t *testing.T, got events.FileStatus, wantPath, wantStatus string, wantStaged bool) {
	t.Helper()
	if got.Path != wantPath {
		t.Errorf("Path = %q, want %q", got.Path, wantPath)
	}
	if got.Status != wantStatus {
		t.Errorf("Status = %q, want %q", got.Status, wantStatus)
	}
	if got.Staged != wantStaged {
		t.Errorf("Staged = %v, want %v", got.Staged, wantStaged)
	}
}
