package cli

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/tabnaeem/git-flow-plus/internal/config"
	"github.com/tabnaeem/git-flow-plus/internal/gitflow"
	"github.com/tabnaeem/git-flow-plus/internal/logging"
	"github.com/tabnaeem/git-flow-plus/internal/version"
)

// newDoctorCmd builds `git flow doctor`. Health checks are assembled from
// two layers: internal/gitflow's Doctor() covers everything scoped to the
// Git repository itself (binary present, its version, branch/permission
// checks — see that package's doc comment for why), and the checks
// appended below cover Git Flow Plus's own installation (config file,
// build version, PATH resolution, whether a release is in progress).
// Those depend on internal/config and internal/version, which
// internal/gitflow deliberately never imports, so they can't live there
// without inverting the layering documented in Architecture.md.
func newDoctorCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check that Git, the repository, and Git Flow Plus itself are healthy",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := app.LoadConfig()
			if err != nil {
				return err
			}

			svc := gitflow.NewService(app.GitClient(), cfg, app.Logger)
			report := svc.Doctor(cmd.Context())

			checks := append([]gitflow.DoctorCheck{}, report.Checks...)
			checks = append(checks,
				gitflow.DoctorCheck{
					Name:   "gitflowplus config",
					OK:     app.ConfigLoader.Exists(app.RepoPath),
					Detail: descConfigCheck(app),
				},
				gitflow.DoctorCheck{
					Name:   "git flow plus version",
					OK:     true,
					Detail: fmt.Sprintf("%s (build %s, commit %s)", Version, BuildNumber, GitCommit),
				},
				pathCheck(),
				releaseConfigCheck(app),
			)

			healthy := true
			for _, c := range checks {
				if !c.OK {
					healthy = false
				}
				if err := app.Println(formatCheck(c, app.ColorEnabled)); err != nil {
					return err
				}
			}

			if !healthy {
				return fmt.Errorf("doctor: one or more checks failed")
			}
			return nil
		},
	}
}

// formatCheck renders a single check as "[STATUS] name  detail", coloring
// the status tag green/red when colorEnabled (resolved once in root.go
// from config, --no-color, NO_COLOR, and whether Out is a real terminal —
// see App.ColorEnabled).
func formatCheck(c gitflow.DoctorCheck, colorEnabled bool) string {
	status, color := "OK  ", logging.AnsiGreen
	if !c.OK {
		status, color = "FAIL", logging.AnsiRed
	}
	tag := fmt.Sprintf("[%s]", status)
	if colorEnabled {
		tag = color + tag + logging.AnsiReset
	}
	return fmt.Sprintf("%s %-22s %s", tag, c.Name, c.Detail)
}

func descConfigCheck(app *App) string {
	if app.ConfigLoader.Exists(app.RepoPath) {
		return fmt.Sprintf("found at %s", config.Path(app.RepoPath))
	}
	return fmt.Sprintf("not found at %s; run 'git flow init'", config.Path(app.RepoPath))
}

// pathCheck reports whether `git flow ...` will resolve as a Git
// subcommand, which requires a binary literally named "git-flow" (not
// "git-flow-plus") somewhere on PATH — see InstallationGuide.md/the
// Windows installer's PATH step, which both install under that name for
// exactly this reason.
func pathCheck() gitflow.DoctorCheck {
	if _, err := exec.LookPath("git-flow"); err == nil {
		return gitflow.DoctorCheck{
			Name: "PATH", OK: true,
			Detail: "'git-flow' resolves on PATH; 'git flow ...' works as a Git subcommand",
		}
	}
	if _, err := exec.LookPath("git-flow-plus"); err == nil {
		return gitflow.DoctorCheck{
			Name: "PATH", OK: false,
			Detail: "'git-flow-plus' is on PATH but 'git-flow' is not, so 'git flow ...' will not resolve; see WindowsInstallation.md/Troubleshooting.md",
		}
	}
	return gitflow.DoctorCheck{
		Name: "PATH", OK: false,
		Detail: "neither 'git-flow' nor 'git-flow-plus' found on PATH; reinstall or add the install directory to PATH",
	}
}

// releaseConfigCheck reports whether a release is currently in progress.
// Having none is a normal, common state, so this is always OK — it's
// informational, like gitflow.Doctor's "working tree" check.
func releaseConfigCheck(app *App) gitflow.DoctorCheck {
	loader := version.NewLoader()
	if !loader.Exists(app.RepoPath) {
		return gitflow.DoctorCheck{
			Name: "release configuration", OK: true,
			Detail: "no active release; run 'git flow release start' to begin one",
		}
	}
	v, err := loader.Load(app.RepoPath)
	if err != nil {
		return gitflow.DoctorCheck{Name: "release configuration", OK: false, Detail: err.Error()}
	}
	return gitflow.DoctorCheck{
		Name: "release configuration", OK: true,
		Detail: fmt.Sprintf("active release, version %s", v.String()),
	}
}
