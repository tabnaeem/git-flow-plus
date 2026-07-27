package logging

import (
	"context"
	"errors"
	"log/slog"
	"strings"
)

// AppError optionally augments an existing error with a human-facing
// cause and a suggested resolution, without changing how or where the
// underlying error was created — wrap it at the point an error is about
// to surface to a user (typically a CLI command handler), not deep in a
// service. LogError (below) renders these extra fields when present and
// silently omits them otherwise, so adopting AppError anywhere is
// additive: existing errors that never wrap with it still print a plain
// "[ERROR] <message>" line.
type AppError struct {
	err        error
	cause      string
	resolution string
}

// NewAppError wraps err with an optional cause and resolution. Either may
// be empty; LogError only prints the lines that have content.
func NewAppError(err error, cause, resolution string) *AppError {
	return &AppError{err: err, cause: cause, resolution: resolution}
}

// Error returns the wrapped error's message, unchanged.
func (e *AppError) Error() string { return e.err.Error() }

// Unwrap exposes the wrapped error to errors.Is/errors.As.
func (e *AppError) Unwrap() error { return e.err }

// Cause returns the human-facing explanation of what went wrong, or ""
// if none was given.
func (e *AppError) Cause() string { return e.cause }

// Resolution returns the suggested next step for the user, or "" if none
// was given.
func (e *AppError) Resolution() string { return e.resolution }

// causer and resolver are the optional interfaces LogError looks for
// while walking an error's Unwrap chain — satisfied by *AppError, but
// intentionally not tied to that concrete type, so any error type in the
// codebase can opt in without importing this package.
type causer interface{ Cause() string }
type resolver interface{ Resolution() string }

func findCause(err error) string {
	for err != nil {
		if c, ok := err.(causer); ok && c.Cause() != "" {
			return c.Cause()
		}
		err = errors.Unwrap(err)
	}
	return ""
}

func findResolution(err error) string {
	for err != nil {
		if r, ok := err.(resolver); ok && r.Resolution() != "" {
			return r.Resolution()
		}
		err = errors.Unwrap(err)
	}
	return ""
}

// LogError renders err through logger at LevelError as:
//
//	[ERROR] <message>
//	  Cause: <cause>
//	  Resolution: <resolution>
//
// The Cause/Resolution lines only appear if err (or anything in its
// Unwrap chain) provides them via AppError or an equivalent Cause()/
// Resolution() string method. No stack trace is ever included — Git Flow
// Plus's errors are plain wrapped errors (fmt.Errorf with %w), which
// never carry one, so this holds regardless of debug/verbose mode.
//
// This is meant for the CLI's single top-level error boundary (see
// cli.Execute) — most code should simply return the error and let this
// function format it once, rather than logging errors again on their way
// up the call stack.
func LogError(logger *slog.Logger, err error) {
	logger.Log(context.Background(), LevelError, formatError(err))
}

func formatError(err error) string {
	var b strings.Builder
	b.WriteString(err.Error())
	if c := findCause(err); c != "" {
		b.WriteString("\n  Cause: ")
		b.WriteString(c)
	}
	if r := findResolution(err); r != "" {
		b.WriteString("\n  Resolution: ")
		b.WriteString(r)
	}
	return b.String()
}
