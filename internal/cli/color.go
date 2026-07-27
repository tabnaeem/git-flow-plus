package cli

import (
	"io"
	"os"

	"golang.org/x/term"
)

// isTerminal reports whether w is a real terminal that can plausibly
// render ANSI color codes. Only *os.File writers can be terminals —
// anything else (a bytes.Buffer in tests, a redirected file, a pipe) is
// never treated as one, which is exactly the behavior wanted: color must
// never leak into captured/redirected output.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// resolveColor decides whether color output should actually be used,
// combining (in order of precedence) an explicit --no-color flag, the
// NO_COLOR convention (https://no-color.org), and terminal detection.
// wantColor is the caller's starting preference (from config.json's
// logging.color, or its environment-derived default).
func resolveColor(w io.Writer, wantColor, noColorFlag bool) bool {
	if noColorFlag || !wantColor {
		return false
	}
	if _, set := os.LookupEnv("NO_COLOR"); set {
		return false
	}
	if !isTerminal(w) {
		return false
	}
	if f, ok := w.(*os.File); ok {
		enableVirtualTerminal(f)
	}
	return true
}
