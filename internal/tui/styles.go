package tui

import "github.com/charmbracelet/lipgloss"

// Color palette — adaptive for light and dark terminals.
var (
	subtle  = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	accent  = lipgloss.AdaptiveColor{Light: "#7D56F4", Dark: "#7D56F4"}
	primary = lipgloss.AdaptiveColor{Light: "#242424", Dark: "#EEEEEE"}
	muted   = lipgloss.AdaptiveColor{Light: "#999999", Dark: "#626262"}
	green   = lipgloss.AdaptiveColor{Light: "#00A600", Dark: "#00D787"}
	red     = lipgloss.AdaptiveColor{Light: "#CC0000", Dark: "#FF5F87"}
	yellow  = lipgloss.AdaptiveColor{Light: "#B58900", Dark: "#FFD700"}
	cyan    = lipgloss.AdaptiveColor{Light: "#2AA198", Dark: "#00D7D7"}
)

// Panel styles.
var (
	// PanelBorder is the default style for bordered panels.
	PanelBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subtle)

	// ActivePanelBorder highlights the currently focused panel.
	ActivePanelBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(accent)
)

// Button styles.
var (
	// ButtonNormal is the default button appearance.
	ButtonNormal = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(primary)

	// ButtonFocused highlights the currently selected button.
	ButtonFocused = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(accent).
			Bold(true)

	// ButtonDisabled dims unavailable buttons.
	ButtonDisabled = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(muted)

	// ButtonLoading indicates an in-progress operation.
	ButtonLoading = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(yellow)

	// ButtonSuccess indicates a completed operation.
	ButtonSuccess = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(green).
			Bold(true)

	// ButtonError indicates a failed operation.
	ButtonError = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(red).
			Bold(true)
)

// Diff styles for syntax highlighting.
var (
	// DiffAdd styles added lines in diffs.
	DiffAdd = lipgloss.NewStyle().Foreground(green)

	// DiffRemove styles removed lines in diffs.
	DiffRemove = lipgloss.NewStyle().Foreground(red)

	// DiffContext styles unchanged context lines in diffs.
	DiffContext = lipgloss.NewStyle().Foreground(muted)
)

// Status indicator styles.
var (
	// StatusOK styles positive status messages.
	StatusOK = lipgloss.NewStyle().Foreground(green)

	// StatusError styles error messages.
	StatusError = lipgloss.NewStyle().Foreground(red)

	// StatusInfo styles informational messages.
	StatusInfo = lipgloss.NewStyle().Foreground(cyan)

	// StatusMuted styles secondary/muted text.
	StatusMuted = lipgloss.NewStyle().Foreground(muted)
)

// File status indicator styles.
var (
	// StagedFile styles staged file entries.
	StagedFile = lipgloss.NewStyle().Foreground(green)

	// UnstagedFile styles unstaged file entries.
	UnstagedFile = lipgloss.NewStyle().Foreground(yellow)

	// UntrackedFile styles untracked file entries.
	UntrackedFile = lipgloss.NewStyle().Foreground(muted)
)

// Title and header styles.
var (
	// Title is the style for section titles.
	// PaddingLeft(2) aligns the icon/title with the indented content below it.
	Title = lipgloss.NewStyle().
		Foreground(accent).
		Bold(true).
		PaddingLeft(2)

	// HelpStyle styles the help text at the bottom.
	HelpStyle = lipgloss.NewStyle().
			Foreground(muted)
)

// StatusIcon returns a short status indicator for a file change type.
func StatusIcon(status string) string {
	switch status {
	case "modified":
		return "M"
	case "added":
		return "A"
	case "deleted":
		return "D"
	case "renamed":
		return "R"
	case "copied":
		return "C"
	case "untracked":
		return "?"
	case "unmerged":
		return "U"
	case "type-changed":
		return "T"
	default:
		return " "
	}
}
