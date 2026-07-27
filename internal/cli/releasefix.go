package cli

import (
	"github.com/spf13/cobra"
)

func newReleaseFixCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "releasefix",
		Short: "Manage release-fix branches (bug fixes against staging)",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "start <name>",
			Short: "Start a new release-fix branch from staging",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := app.LoadConfig()
				if err != nil {
					return err
				}
				result, err := app.GitFlowService(cfg).ReleaseFixStart(cmd.Context(), args[0])
				if err != nil {
					return err
				}
				return app.Println(result.Message)
			},
		},
		&cobra.Command{
			Use:   "finish <name>",
			Short: "Merge the release-fix branch into staging and record it as pending (does not change the version)",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := app.LoadConfig()
				if err != nil {
					return err
				}
				result, err := app.ReleaseService(cfg).ReleaseFixFinish(cmd.Context(), args[0])
				if err != nil {
					return err
				}
				return app.Printf("Merged release fix %q into staging (release %q); pending inclusion in the next QA build\n", args[0], result.Release)
			},
		},
	)

	return cmd
}
