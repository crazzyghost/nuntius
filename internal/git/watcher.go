package git

import (
	"context"
	"path/filepath"
	"sync"
	"time"

	"github.com/crazzyghost/nuntius/internal/events"
	"github.com/fsnotify/fsnotify"
)

// defaultDebounce is the debounce window for file system events.
const defaultDebounce = 100 * time.Millisecond

// Watcher monitors the .git directory for changes and emits debounced
// FilesChangedMsg events. After each debounce window, it calls Status()
// and sends the result on the Events channel.
type Watcher struct {
	// Events receives FilesChangedMsg after each debounced change detection.
	Events chan events.FilesChangedMsg

	repoPath string
	watcher  *fsnotify.Watcher
	debounce time.Duration
	done     chan struct{}
	once     sync.Once
}

// NewWatcher creates a new file system watcher for the given repository path.
// The watcher monitors the .git directory for changes to index, HEAD, and refs.
func NewWatcher(repoPath string) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// Watch the .git directory for changes
	gitDir := filepath.Join(repoPath, ".git")
	if err := fsw.Add(gitDir); err != nil {
		fsw.Close()
		return nil, err
	}

	// Also watch refs directory for branch/tag changes
	refsDir := filepath.Join(gitDir, "refs")
	_ = fsw.Add(refsDir)

	// Watch refs/heads for branch updates
	headsDir := filepath.Join(refsDir, "heads")
	_ = fsw.Add(headsDir)

	return &Watcher{
		Events:   make(chan events.FilesChangedMsg, 1),
		repoPath: repoPath,
		watcher:  fsw,
		debounce: defaultDebounce,
		done:     make(chan struct{}),
	}, nil
}

// Start begins watching for file system events. It blocks until the
// context is cancelled or Stop is called. Should be run in a goroutine.
func (w *Watcher) Start(ctx context.Context) error {
	defer w.watcher.Close()

	var timer *time.Timer
	var timerC <-chan time.Time

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-w.done:
			return nil

		case event, ok := <-w.watcher.Events:
			if !ok {
				return nil
			}

			// Only react to meaningful git events
			if !isRelevantGitEvent(event.Name) {
				continue
			}

			// Reset the debounce timer on each event
			if timer != nil {
				timer.Stop()
			}
			timer = time.NewTimer(w.debounce)
			timerC = timer.C

		case <-timerC:
			// Debounce timer fired — fetch fresh status
			timerC = nil
			w.emitStatus()

		case _, ok := <-w.watcher.Errors:
			if !ok {
				return nil
			}
			// Log or handle watcher errors; continue watching
			continue
		}
	}
}

// Stop cleanly shuts down the watcher.
func (w *Watcher) Stop() {
	w.once.Do(func() {
		close(w.done)
	})
}

// emitStatus calls Status() and sends the result on the Events channel.
// Non-blocking: if the channel is full, the old value is drained first.
func (w *Watcher) emitStatus() {
	files, err := Status()
	if err != nil {
		// On error, emit empty file list rather than blocking
		files = nil
	}

	msg := events.FilesChangedMsg{Files: files}

	// Non-blocking send: drain the channel if full
	select {
	case w.Events <- msg:
	default:
		// Channel full — drain and resend
		select {
		case <-w.Events:
		default:
		}
		w.Events <- msg
	}
}

// isRelevantGitEvent determines if a filesystem event path corresponds
// to a git state change worth reacting to. We care about:
// - index (staging area changes)
// - HEAD (checkout, commit) — but not logs/HEAD
// - refs/ (branch updates, tags)
// - COMMIT_EDITMSG (commit in progress)
func isRelevantGitEvent(path string) bool {
	// Exclude changes under logs/ — these are reflogs, not state changes
	if containsDir(path, "logs") {
		return false
	}

	base := filepath.Base(path)
	switch base {
	case "index", "HEAD", "COMMIT_EDITMSG", "MERGE_HEAD", "FETCH_HEAD":
		return true
	}

	// Any change under refs/ is relevant
	if containsDir(path, "refs") {
		return true
	}

	return false
}

// containsDir checks if any component of the path matches the given directory name.
func containsDir(path string, dir string) bool {
	for p := path; p != "." && p != "/"; p = filepath.Dir(p) {
		if filepath.Base(p) == dir {
			return true
		}
	}
	return false
}
