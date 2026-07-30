package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
)

// ANSI color codes, used only when a consoleHandler (or another colorized
// renderer, e.g. `git flow doctor` in internal/cli) is constructed with
// color enabled. Detection of whether the destination actually supports
// color (a real terminal, no NO_COLOR, etc.) is the caller's
// responsibility — see internal/cli/root.go — so this package stays free
// of any terminal/OS-specific logic. Exported so every colorized command
// output shares one palette instead of each package defining its own.
const (
	AnsiReset   = "\x1b[0m"
	AnsiGray    = "\x1b[90m"
	AnsiCyan    = "\x1b[36m"
	AnsiBlue    = "\x1b[34m"
	AnsiGreen   = "\x1b[32m"
	AnsiYellow  = "\x1b[33m"
	AnsiRed     = "\x1b[31m"
	AnsiRedBold = "\x1b[1;31m"
)

func levelColor(l slog.Level) string {
	switch {
	case l < LevelDebug:
		return AnsiGray
	case l < LevelInfo:
		return AnsiCyan
	case l < LevelSuccess:
		return AnsiBlue
	case l < LevelWarn:
		return AnsiGreen
	case l < LevelError:
		return AnsiYellow
	case l < LevelFatal:
		return AnsiRed
	default:
		return AnsiRedBold
	}
}

// consoleHandler is a slog.Handler that renders human-readable, optionally
// colorized lines in the form:
//
//	[LEVEL] message  key: value  key2: value2
//
// This is deliberately different from slog's built-in TextHandler (which
// renders `time=... level=INFO msg="..." key=value`) — the bracketed
// level tag and "key: value" spacing is what Git Flow Plus's production
// logging format looks like across every command.
type consoleHandler struct {
	w     io.Writer
	level slog.Level
	color bool
	attrs []slog.Attr
	mu    *sync.Mutex
}

func newConsoleHandler(w io.Writer, level slog.Level, color bool) *consoleHandler {
	return &consoleHandler{w: w, level: level, color: color, mu: &sync.Mutex{}}
}

func (h *consoleHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *consoleHandler) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder

	label := levelLabel(r.Level)
	if h.color {
		b.WriteString(levelColor(r.Level))
		b.WriteByte('[')
		b.WriteString(label)
		b.WriteByte(']')
		b.WriteString(AnsiReset)
	} else {
		b.WriteByte('[')
		b.WriteString(label)
		b.WriteByte(']')
	}
	b.WriteByte(' ')
	b.WriteString(r.Message)

	for _, a := range h.attrs {
		writeAttr(&b, a)
	}
	r.Attrs(func(a slog.Attr) bool {
		writeAttr(&b, a)
		return true
	})
	b.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.w, b.String())
	return err
}

func writeAttr(b *strings.Builder, a slog.Attr) {
	if a.Equal(slog.Attr{}) {
		return
	}
	fmt.Fprintf(b, "  %s: %v", a.Key, a.Value.Any())
}

func (h *consoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	next := *h
	next.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &next
}

// WithGroup is a no-op passthrough: no call site in this codebase uses
// slog groups (verified — every logger call is a flat set of key/value
// pairs), so full group-attribute nesting isn't implemented. If a future
// caller does start grouping attrs, this will silently flatten them
// instead of prefixing keys, which is the one behavioral gap versus
// slog's built-in handlers.
func (h *consoleHandler) WithGroup(_ string) slog.Handler {
	return h
}
