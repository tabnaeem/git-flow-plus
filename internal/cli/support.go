package cli

import (
	"github.com/spf13/cobra"

	"github.com/tabnaeem/git-flow-plus/internal/gitflow"
)

func newSupportCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "support",
		Short: "Manage support branches",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "start <name>",
			Short: "Start a new support branch from main",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := app.LoadConfig()
				if err != nil {
					return err
				}
				svc := gitflow.NewService(app.GitClient(), cfg, app.Logger)
				result, err := svc.SupportStart(cmd.Context(), args[0])
				if err != nil {
					return err
				}
				return app.Println(result.Message)
			},
		},
		&cobra.Command{
			Use:   "finish <name>",
			Short: "Close out a support branch (support branches are long-lived and are not merged back)",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := app.LoadConfig()
				if err != nil {
					return err
				}
				svc := gitflow.NewService(app.GitClient(), cfg, app.Logger)
				result, err := svc.SupportFinish(cmd.Context(), args[0])
				if err != nil {
					return err
				}
				return app.Println(result.Message)
			},
		},
	)

	return cmd
}
