package cli

import (
	"github.com/spf13/cobra"

	"github.com/hulhub/git-flow-plus/internal/gitflow"
)

func newHotfixCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hotfix",
		Short: "Manage hotfix branches",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "start <name>",
			Short: "Start a new hotfix branch from main",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := app.LoadConfig()
				if err != nil {
					return err
				}
				svc := gitflow.NewService(app.GitClient(), cfg, app.Logger)
				result, err := svc.HotfixStart(cmd.Context(), args[0])
				if err != nil {
					return err
				}
				return app.Println(result.Message)
			},
		},
		&cobra.Command{
			Use:   "finish <name>",
			Short: "Merge a hotfix branch into main and develop, then tag it",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := app.LoadConfig()
				if err != nil {
					return err
				}
				svc := gitflow.NewService(app.GitClient(), cfg, app.Logger)
				result, err := svc.HotfixFinish(cmd.Context(), args[0])
				if err != nil {
					return err
				}
				return app.Println(result.Message)
			},
		},
	)

	return cmd
}
