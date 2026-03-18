package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// statusLevel indicates the severity of a status message.
type statusLevel int

const (
	statusInfo statusLevel = iota
	statusSuccess
	statusErr
)

// statusClearDelay is how long status messages persist before auto-clearing.
const statusClearDelay = 5 * time.Second

// statusClearMsg is sent to clear the status line after a timeout.
type statusClearMsg struct{}

// statusEntry holds a transient status message with its level.
type statusEntry struct {
	message string
	level   statusLevel
}

// renderStatus formats the status entry using the appropriate style.
func renderStatus(s statusEntry) string {
	switch s.level {
	case statusSuccess:
		return StatusOK.Render(s.message)
	case statusErr:
		return StatusError.Render(s.message)
	default:
		return StatusInfo.Render(s.message)
	}
}

// scheduleStatusClear returns a Cmd that sends a statusClearMsg after the delay.
func scheduleStatusClear() tea.Cmd {
	return tea.Tick(statusClearDelay, func(_ time.Time) tea.Msg {
		return statusClearMsg{}
	})
}
