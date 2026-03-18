package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func TestDefaultKeyMapBindings(t *testing.T) {
	km := DefaultKeyMap

	tests := []struct {
		name    string
		binding key.Binding
		keys    []string
	}{
		{"Generate", km.Generate, []string{"g"}},
		{"Commit", km.Commit, []string{"c"}},
		{"Push", km.Push, []string{"p"}},
		{"Quit", km.Quit, []string{"q", "ctrl+c"}},
		{"Up", km.Up, []string{"up", "k"}},
		{"Down", km.Down, []string{"down", "j"}},
		{"PageUp", km.PageUp, []string{"pgup"}},
		{"PageDown", km.PageDown, []string{"pgdown"}},
		{"Help", km.Help, []string{"?"}},
		{"Tab", km.Tab, []string{"tab"}},
		{"Enter", km.Enter, []string{"enter", " "}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			boundKeys := tt.binding.Keys()
			if len(boundKeys) != len(tt.keys) {
				t.Fatalf("expected %d keys, got %d", len(tt.keys), len(boundKeys))
			}
			for i, k := range tt.keys {
				if boundKeys[i] != k {
					t.Errorf("key %d: expected %q, got %q", i, k, boundKeys[i])
				}
			}
		})
	}
}

func TestKeyMapMatchesKey(t *testing.T) {
	km := DefaultKeyMap

	gKey := tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if !key.Matches(gKey, km.Generate) {
		t.Error("expected 'g' to match Generate")
	}

	cKey := tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if !key.Matches(cKey, km.Commit) {
		t.Error("expected 'c' to match Commit")
	}

	pKey := tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'p'}})
	if !key.Matches(pKey, km.Push) {
		t.Error("expected 'p' to match Push")
	}

	qKey := tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !key.Matches(qKey, km.Quit) {
		t.Error("expected 'q' to match Quit")
	}
}

func TestSetEnabled(t *testing.T) {
	b := key.NewBinding(key.WithKeys("x"))

	if !b.Enabled() {
		t.Fatal("binding should be enabled by default")
	}

	SetEnabled(&b, false)
	if b.Enabled() {
		t.Error("binding should be disabled after SetEnabled(false)")
	}

	SetEnabled(&b, true)
	if !b.Enabled() {
		t.Error("binding should be enabled after SetEnabled(true)")
	}
}

func TestShortHelp(t *testing.T) {
	km := DefaultKeyMap
	bindings := km.ShortHelp()
	if len(bindings) != 5 {
		t.Errorf("ShortHelp should return 5 bindings, got %d", len(bindings))
	}
}

func TestFullHelp(t *testing.T) {
	km := DefaultKeyMap
	groups := km.FullHelp()
	if len(groups) != 3 {
		t.Errorf("FullHelp should return 3 groups, got %d", len(groups))
	}
}
