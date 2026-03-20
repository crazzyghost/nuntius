package git

import "github.com/crazzyghost/nuntius/internal/events"

// Ops abstracts git operations behind an interface for testability.
// Any type that implements all five methods satisfies this interface.
type Ops interface {
	Status() ([]events.FileStatus, error)
	StageAll() error
	Commit(message string) (CommitResult, error)
	Push(opts PushOptions) (PushResult, error)
	HasUpstream() (bool, error)
}

// DefaultOps is the production implementation of Ops that delegates directly
// to the package-level git functions.
type DefaultOps struct{}

func (DefaultOps) Status() ([]events.FileStatus, error) { return Status() }
func (DefaultOps) StageAll() error                      { return StageAll() }
func (DefaultOps) Commit(msg string) (CommitResult, error) {
	return Commit(msg)
}
func (DefaultOps) Push(opts PushOptions) (PushResult, error) { return Push(opts) }
func (DefaultOps) HasUpstream() (bool, error)                { return HasUpstream() }
