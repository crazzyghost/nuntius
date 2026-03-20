// Package events defines the typed Bubble Tea message types shared
// across all TUI components. These are pure data types with no logic —
// they serve as the contract between components.
package events

// FileStatus represents the state of a single file in the working tree.
type FileStatus struct {
	// Path is the file path relative to the repo root.
	Path string
	// Status is the file's change type: "modified", "added", "deleted", "untracked", "renamed".
	Status string
	// Staged indicates whether the change is in the staging area.
	Staged bool
}

// FilesChangedMsg is sent by the Git watcher when the working tree changes.
// Source: Watcher → Consumed by: Viewport (Panel A)
type FilesChangedMsg struct {
	Files []FileStatus
}

// GenerateRequestedMsg is sent when the user requests AI commit message generation.
// Source: Action bar (Panel B) → Consumed by: AI gateway
type GenerateRequestedMsg struct{}

// PushRequestedMsg is sent when the user initiates a git push.
// It carries the loading text so both the viewport and action bar can show
// consistent in-progress feedback while restarting their spinner tick chains.
// Source: App → Consumed by: Viewport, ActionBar
type PushRequestedMsg struct {
	// LoadingText is the message to display while pushing.
	LoadingText string
}

// MessageReadyMsg is sent when the AI provider returns a generated commit message.
// Source: AI gateway → Consumed by: Viewport (Panel A)
type MessageReadyMsg struct {
	Message string
}

// CommitResultMsg is sent after a git commit operation completes.
// Source: Git commit → Consumed by: Action bar (Panel C)
type CommitResultMsg struct {
	// Hash is the commit SHA on success.
	Hash string
	// Err is non-nil if the commit failed.
	Err error
}

// PushResultMsg is sent after a git push operation completes.
// Source: Git push → Consumed by: Action bar (Panel D)
type PushResultMsg struct {
	// Remote is the remote name that was pushed to.
	Remote string
	// Err is non-nil if the push failed.
	Err error
}

// ErrorMsg is a generic error message that can originate from any component.
// Source: Any → Consumed by: Status bar
type ErrorMsg struct {
	// Source identifies which component produced the error.
	Source string
	// Err is the underlying error.
	Err error
}

// UpdateAvailableMsg is sent when a newer version of nuntius is available.
// Source: Version check → Consumed by: AppModel (displayed as a persistent notice)
type UpdateAvailableMsg struct {
	// Current is the running version.
	Current string
	// Latest is the newest version available.
	Latest string
}
