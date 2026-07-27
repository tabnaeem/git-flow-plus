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
			Short: "Start a new feature branch from develop",
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
				return app.Println(result.Message)
			},
		},
		&cobra.Command{
			Use:   "finish <name>",
			Short: "Merge a feature branch into develop, delete it, and register the feature as merged in the Feature Registry",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := app.LoadConfig()
				if err != nil {
					return err
				}
				svc := gitflow.NewService(app.GitClient(), cfg, app.Logger)
				result, err := svc.FeatureFinish(cmd.Context(), args[0])
				if err != nil {
					return err
				}
				if err := app.Println(result.Message); err != nil {
					return err
				}

				mergeCommit, err := app.GitClient().CommitSHA(cmd.Context())
				if err != nil {
					return err
				}
				if err := app.ReleaseService(cfg).RegisterFeatureMerged(cmd.Context(), args[0], result.Branch, mergeCommit); err != nil {
					return err
				}
				return app.Printf("Registered feature %q as merged into develop\n", args[0])
			},
		},
	)

	return cmd
}
