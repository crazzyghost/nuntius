package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/crazzyghost/nuntius/internal/ai"
	"github.com/crazzyghost/nuntius/internal/config"
	"github.com/crazzyghost/nuntius/internal/engine"
	"github.com/crazzyghost/nuntius/internal/events"
)

// generateCmd creates a tea.Cmd that runs the full generate pipeline via the
// shared engine: diff acquisition → prompt building → AI provider call.
func generateCmd(provider ai.Provider, cfg config.Config) tea.Cmd {
	return func() tea.Msg {
		msg, _, err := engine.Generate(context.Background(), cfg, provider, engine.GenerateInput{
			Source: engine.DiffSourceAuto,
		})
		if err != nil {
			return events.ErrorMsg{Source: "generate", Err: err}
		}
		return events.MessageReadyMsg{Message: msg}
	}
}
