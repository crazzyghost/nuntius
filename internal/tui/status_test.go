package tui

import (
	"testing"
)

func TestRenderStatus(t *testing.T) {
	tests := []struct {
		name  string
		entry statusEntry
	}{
		{"info", statusEntry{message: "Loading...", level: statusInfo}},
		{"success", statusEntry{message: "Committed: abc1234", level: statusSuccess}},
		{"error", statusEntry{message: "Push failed", level: statusErr}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderStatus(tt.entry)
			if result == "" {
				t.Error("renderStatus should not return empty string")
			}
		})
	}
}

func TestScheduleStatusClear(t *testing.T) {
	cmd := scheduleStatusClear()
	if cmd == nil {
		t.Error("scheduleStatusClear should return a non-nil Cmd")
	}
}
