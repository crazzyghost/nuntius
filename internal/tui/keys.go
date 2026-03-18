package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keyboard shortcuts for the TUI.
type KeyMap struct {
	// Generate triggers AI commit message generation.
	Generate key.Binding
	// Commit runs git commit with the generated message.
	Commit key.Binding
	// Push runs git push.
	Push key.Binding
	// Quit exits the application.
	Quit key.Binding
	// Up scrolls the viewport up one line.
	Up key.Binding
	// Down scrolls the viewport down one line.
	Down key.Binding
	// PageUp scrolls the viewport up one page.
	PageUp key.Binding
	// PageDown scrolls the viewport down one page.
	PageDown key.Binding
	// Help toggles the help overlay.
	Help key.Binding
	// Tab cycles focus between action bar buttons.
	Tab key.Binding
	// Enter activates the focused button.
	Enter key.Binding
}

// DefaultKeyMap returns the default key bindings for Nuntius.
var DefaultKeyMap = KeyMap{
	Generate: key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("g", "generate message"),
	),
	Commit: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "commit"),
	),
	Push: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "push"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "scroll up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "scroll down"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("pgup"),
		key.WithHelp("pgup", "page up"),
	),
	PageDown: key.NewBinding(
		key.WithKeys("pgdown"),
		key.WithHelp("pgdn", "page down"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next button"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter", " "),
		key.WithHelp("enter", "activate"),
	),
}

// ShortHelp returns key bindings for the compact help view.
// Implements the help.KeyMap interface.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Generate, k.Commit, k.Push, k.Help, k.Quit}
}

// FullHelp returns key bindings for the expanded help view.
// Implements the help.KeyMap interface.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Generate, k.Commit, k.Push},
		{k.Up, k.Down, k.PageUp, k.PageDown},
		{k.Tab, k.Enter, k.Help, k.Quit},
	}
}

// SetEnabled enables or disables a key binding.
func SetEnabled(b *key.Binding, enabled bool) {
	b.SetEnabled(enabled)
}
