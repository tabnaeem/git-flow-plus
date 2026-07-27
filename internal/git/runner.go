// Package git wraps the git binary as a set of typed, mockable operations
// (branch, checkout, merge, tag, commit, push, pull, status). It shells out
// to the real "git" executable rather than reimplementing Git, so every
// operation behaves identically to plain Git and stays compatible with any
// hosting provider, credential helper, or GUI client already configured for
// the repository.
package git

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
)

// Runner executes a git subcommand with args inside dir and returns its
// trimmed stdout. It is the sole seam between this package's higher-level
// Client and the OS, so tests can substitute a fake Runner instead of
// invoking a real git binary.
type Runner interface {
	Run(ctx context.Context, dir string, args ...string) (string, error)
}

type execRunner struct{}

// NewExecRunner returns a Runner that invokes the git binary on PATH.
func NewExecRunner() Runner {
	return execRunner{}
}

func (execRunner) Run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		exitCode := -1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
		return "", &CommandError{
			Args:     args,
			Stderr:   strings.TrimSpace(stderr.String()),
			ExitCode: exitCode,
			Err:      err,
		}
	}

	return strings.TrimSpace(stdout.String()), nil
}

// Available reports whether a git binary can be found on PATH.
func Available() bool {
	_, err := exec.LookPath("git")
	return err == nil
}
