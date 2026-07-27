package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/hulhub/git-flow-plus/internal/config"
	"github.com/hulhub/git-flow-plus/internal/gitflow"
)

func newDoctorCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check that Git, the repository, and Git Flow Plus metadata are healthy",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := app.LoadConfig()
			if err != nil {
				return err
			}

			svc := gitflow.NewService(app.GitClient(), cfg, app.Logger)
			report := svc.Doctor(cmd.Context())

			checks := append([]gitflow.DoctorCheck{}, report.Checks...)
			checks = append(checks, gitflow.DoctorCheck{
				Name:   "gitflowplus config",
				OK:     app.ConfigLoader.Exists(app.RepoPath),
				Detail: descConfigCheck(app),
			})

			healthy := true
			for _, c := range checks {
				status := "ok"
				if !c.OK {
					status = "FAIL"
					healthy = false
				}
				if err := app.Printf("[%s] %-20s %s\n", status, c.Name, c.Detail); err != nil {
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

func descConfigCheck(app *App) string {
	if app.ConfigLoader.Exists(app.RepoPath) {
		return fmt.Sprintf("found at %s", config.Path(app.RepoPath))
	}
	return fmt.Sprintf("not found at %s; run 'git flow init'", config.Path(app.RepoPath))
}
