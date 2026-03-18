package tui

import (
	"testing"
)

func TestStatusIcon(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"modified", "M"},
		{"added", "A"},
		{"deleted", "D"},
		{"renamed", "R"},
		{"copied", "C"},
		{"untracked", "?"},
		{"unmerged", "U"},
		{"type-changed", "T"},
		{"unknown", " "},
		{"", " "},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := StatusIcon(tt.status)
			if got != tt.want {
				t.Errorf("StatusIcon(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestStylesAreDefined(t *testing.T) {
	// Verify all exported styles render without panicking.
	styles := []struct {
		name  string
		style func() string
	}{
		{"PanelBorder", func() string { return PanelBorder.Render("test") }},
		{"ActivePanelBorder", func() string { return ActivePanelBorder.Render("test") }},
		{"ButtonNormal", func() string { return ButtonNormal.Render("test") }},
		{"ButtonFocused", func() string { return ButtonFocused.Render("test") }},
		{"ButtonDisabled", func() string { return ButtonDisabled.Render("test") }},
		{"ButtonLoading", func() string { return ButtonLoading.Render("test") }},
		{"ButtonSuccess", func() string { return ButtonSuccess.Render("test") }},
		{"ButtonError", func() string { return ButtonError.Render("test") }},
		{"DiffAdd", func() string { return DiffAdd.Render("test") }},
		{"DiffRemove", func() string { return DiffRemove.Render("test") }},
		{"DiffContext", func() string { return DiffContext.Render("test") }},
		{"StatusOK", func() string { return StatusOK.Render("test") }},
		{"StatusError", func() string { return StatusError.Render("test") }},
		{"StatusInfo", func() string { return StatusInfo.Render("test") }},
		{"StatusMuted", func() string { return StatusMuted.Render("test") }},
		{"StagedFile", func() string { return StagedFile.Render("test") }},
		{"UnstagedFile", func() string { return UnstagedFile.Render("test") }},
		{"UntrackedFile", func() string { return UntrackedFile.Render("test") }},
		{"Title", func() string { return Title.Render("test") }},
		{"HelpStyle", func() string { return HelpStyle.Render("test") }},
	}

	for _, s := range styles {
		t.Run(s.name, func(t *testing.T) {
			result := s.style()
			if result == "" {
				t.Errorf("style %s rendered empty string", s.name)
			}
		})
	}
}
