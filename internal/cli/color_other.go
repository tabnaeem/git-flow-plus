//go:build !windows

package cli

import "os"

// enableVirtualTerminal is a no-op outside Windows — every other
// supported platform's terminal already interprets ANSI escape codes
// natively, with no console-mode flag to opt into.
func enableVirtualTerminal(*os.File) {}
