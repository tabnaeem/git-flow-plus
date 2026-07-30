package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Build metadata, overridden at build time via
// -ldflags "-X github.com/tabnaeem/git-flow-plus/internal/cli.<Var>=<value>"
// (see scripts/build.sh / scripts/build.ps1 / the Makefile). Defaults
// describe an unstamped development build (e.g. `go build` run directly,
// without going through the build scripts).
var (
	// BuildNumber is typically a CI run number or a monotonically
	// increasing counter, distinct from the semantic Version.
	BuildNumber = "dev"
	// GitCommit is the short SHA of the commit the binary was built from.
	GitCommit = "unknown"
	// GitBranch is the branch the binary was built from.
	GitBranch = "unknown"
	// BuildDate is an RFC 3339 UTC timestamp set at build time.
	BuildDate = "unknown"
)

func newVersionCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print Git Flow Plus version and build information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			lines := []string{
				"Git Flow Plus",
				"",
				fmt.Sprintf("Version      : %s", Version),
				fmt.Sprintf("Build Number : %s", BuildNumber),
				fmt.Sprintf("Git Commit   : %s", GitCommit),
				fmt.Sprintf("Git Branch   : %s", GitBranch),
				fmt.Sprintf("Build Date   : %s", BuildDate),
				fmt.Sprintf("Go Version   : %s", runtime.Version()),
				fmt.Sprintf("OS           : %s", runtime.GOOS),
				fmt.Sprintf("Architecture : %s", runtime.GOARCH),
			}
			for _, l := range lines {
				if err := app.Println(l); err != nil {
					return err
				}
			}
			return nil
		},
	}
}
