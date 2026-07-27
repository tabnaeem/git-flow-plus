package gitflow

import (
	"errors"
	"fmt"
)

// Errors returned by Service methods for conditions callers are expected to
// handle or explain to the end user, as distinct from underlying git command
// failures (which are wrapped and returned as-is).
var (
	// ErrEmptyName indicates a branch operation was called with a blank
	// (or whitespace-only) name.
	ErrEmptyName = errors.New("name must not be empty")
	// ErrNotInitialized indicates the repository is missing the main or
	// develop branch Git Flow Plus requires; run `git flow init` first.
	ErrNotInitialized = errors.New("repository is not initialized for Git Flow Plus; run 'git flow init' first")
	// ErrBranchAlreadyExists indicates a start operation targeted a branch
	// name that is already in use.
	ErrBranchAlreadyExists = errors.New("branch already exists")
	// ErrBranchMissing indicates a finish operation targeted a branch that
	// does not exist.
	ErrBranchMissing = errors.New("branch does not exist")
	// ErrDirtyWorkingTree indicates an operation that requires a clean
	// working tree was attempted with uncommitted changes present.
	ErrDirtyWorkingTree = errors.New("working tree has uncommitted changes; commit or stash them first")
)

// wrapf formats a message with args and wraps cause via %w when cause is
// non-nil, or returns a plain error built from the message otherwise.
func wrapf(cause error, format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	if cause == nil {
		return errors.New(msg)
	}
	return fmt.Errorf("%s: %w", msg, cause)
}
