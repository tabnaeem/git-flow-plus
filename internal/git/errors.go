package git

import (
	"errors"
	"fmt"
	"strings"
)

// CommandError wraps a failed git invocation with enough detail (the exact
// arguments, git's stderr, and the process exit code) to surface a useful
// error to the end user without them needing --verbose.
type CommandError struct {
	Args     []string
	Stderr   string
	ExitCode int
	Err      error
}

func (e *CommandError) Error() string {
	if e.Stderr != "" {
		return fmt.Sprintf("git %s: %s", strings.Join(e.Args, " "), e.Stderr)
	}
	return fmt.Sprintf("git %s: %v", strings.Join(e.Args, " "), e.Err)
}

func (e *CommandError) Unwrap() error {
	return e.Err
}

// Sentinel errors returned by Client methods for conditions the caller is
// expected to branch on (e.g. deciding whether to create a branch or report
// a conflict), as opposed to raw command failures surfaced via CommandError.
var (
	// ErrNotRepository indicates the target directory is not inside a Git
	// working tree.
	ErrNotRepository = errors.New("not a git repository")
	// ErrBranchExists indicates a branch create was attempted for a branch
	// that already exists.
	ErrBranchExists = errors.New("branch already exists")
	// ErrBranchNotFound indicates an operation referenced a branch that does
	// not exist.
	ErrBranchNotFound = errors.New("branch not found")
	// ErrDirtyWorkingTree indicates an operation that requires a clean
	// working tree was attempted while uncommitted changes are present.
	ErrDirtyWorkingTree = errors.New("working tree has uncommitted changes")
)
