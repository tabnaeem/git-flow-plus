package cli

import (
	"github.com/spf13/cobra"

	"github.com/hulhub/git-flow-plus/internal/gitflow"
)

func newFeatureCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "feature",
		Short: "Manage feature branches",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "start <name>",
			Short: "Start a new feature branch from staging and register it in the Feature Registry",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := app.LoadConfig()
				if err != nil {
					return err
				}
				svc := gitflow.NewService(app.GitClient(), cfg, app.Logger)
				result, err := svc.FeatureStart(cmd.Context(), args[0])
				if err != nil {
					return err
				}
				if err := app.Println(result.Message); err != nil {
					return err
				}

				if err := app.ReleaseService(cfg).RegisterFeatureCreated(cmd.Context(), args[0], result.Branch); err != nil {
					return err
				}
				return app.Printf("Registered feature %q\n", args[0])
			},
		},
	)

	return cmd
}
