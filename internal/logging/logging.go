// Package logging provides structured, leveled logging for Git Flow Plus.
//
// Two output formats are supported: a colorized, human-readable console
// format (FormatText, the default — "[INFO] message  key: value") for
// interactive use, and newline-delimited JSON (FormatJSON) for CI/CD and
// other machine consumers. Seven levels are supported — see levels.go —
// running from Trace (most verbose) through Fatal (unrecoverable).
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
)

// Format selects the output encoding for log records.
type Format int

const (
	// FormatText renders human-readable, optionally colorized console
	// lines (default, for TTY use).
	FormatText Format = iota
	// FormatJSON renders newline-delimited JSON log records (for machine
	// consumption, e.g. CI/CD log aggregation).
	FormatJSON
)

// Options configures logger construction.
type Options struct {
	// Writer is the destination for log output.
	Writer io.Writer
	// Verbose enables debug-level logging when true; otherwise info-level.
	// Superseded by Trace when both are set.
	Verbose bool
	// Trace enables trace-level logging (the most detailed level) when
	// true — implies Verbose.
	Trace bool
	// Level, when non-nil, is used verbatim as the minimum enabled level,
	// overriding Verbose/Trace. This is what lets callers honor an
	// arbitrary configured level (e.g. "warn" in a testing environment)
	// rather than only the three levels Verbose/Trace can express.
	Level *slog.Level
	// Format selects text or JSON output.
	Format Format
	// Color enables ANSI color codes in FormatText output. Callers are
	// responsible for deciding when color is appropriate (a real
	// terminal, no NO_COLOR, no --no-color flag, ...); this package
	// applies whatever it's told. Ignored when Format is FormatJSON.
	Color bool
}

// New builds a *slog.Logger according to opts.
func New(opts Options) *slog.Logger {
	level := LevelInfo
	switch {
	case opts.Trace:
		level = LevelTrace
	case opts.Verbose:
		level = LevelDebug
	}
	if opts.Level != nil {
		level = *opts.Level
	}

	var handler slog.Handler
	switch opts.Format {
	case FormatJSON:
		handler = slog.NewJSONHandler(opts.Writer, &slog.HandlerOptions{Level: level})
	default:
		handler = newConsoleHandler(opts.Writer, level, opts.Color)
	}

	return slog.New(handler)
}

// Trace logs msg at LevelTrace — finer-grained than Debug, for very
// verbose diagnostic detail (e.g. individual git invocations).
func Trace(logger *slog.Logger, msg string, args ...any) {
	logger.Log(context.Background(), LevelTrace, msg, args...)
}

// Success logs msg at LevelSuccess — a distinguished "this operation
// completed" signal, rendered as "[SUCCESS]" rather than "[INFO]".
func Success(logger *slog.Logger, msg string, args ...any) {
	logger.Log(context.Background(), LevelSuccess, msg, args...)
}

// Fatal logs msg at LevelFatal and then terminates the process with exit
// code 1. Reserved for unrecoverable failures outside the normal command
// error path (e.g. a startup precondition that makes it unsafe to
// continue at all) — everyday command failures should be returned as
// errors so Cobra's usual error handling in internal/cli can format and
// exit through the standard path instead.
func Fatal(logger *slog.Logger, msg string, args ...any) {
	logger.Log(context.Background(), LevelFatal, msg, args...)
	os.Exit(1)
}
