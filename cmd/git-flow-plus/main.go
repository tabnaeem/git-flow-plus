// Command git-flow-plus is the Git Flow Plus CLI entrypoint. When installed
// as `git-flow` (or `git-flow-plus`) on the PATH, Git resolves `git flow ...`
// to this binary and forwards the remaining arguments unchanged.
package main

import (
	"os"

	"github.com/hulhub/git-flow-plus/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
