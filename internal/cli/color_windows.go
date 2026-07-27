//go:build windows

package cli

import (
	"os"

	"golang.org/x/sys/windows"
)

// enableVirtualTerminal turns on ANSI escape-code interpretation for f's
// console. Unlike Unix terminals, Windows consoles don't interpret ANSI
// codes by default outside Windows Terminal — this is the same
// ENABLE_VIRTUAL_TERMINAL_PROCESSING flag tools like Git for Windows and
// most modern CLIs set. It's best-effort: any failure (not a console, an
// older Windows build without the flag, ...) just means colors won't
// render, and is not treated as an error.
func enableVirtualTerminal(f *os.File) {
	handle := windows.Handle(f.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return
	}
	_ = windows.SetConsoleMode(handle, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
}
