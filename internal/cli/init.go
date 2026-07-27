package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/hulhub/git-flow-plus/internal/config"
	"github.com/hulhub/git-flow-plus/internal/gitflow"
)

func newInitCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize a new Git Flow Plus repository",
		Long:  "Sets up the standard Git Flow branches (main/develop/staging) and creates the\n.gitflowplus metadata directory (config.json).",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := app.LoadConfig()
			if err != nil {
				return err
			}

			client := app.GitClient()
			svc := gitflow.NewService(client, cfg, app.Logger)
			result, err := svc.Init(cmd.Context())
			if err != nil {
				return err
			}

			if err := app.ConfigLoader.Save(app.RepoPath, cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			// Commit the metadata directory so the working tree stays clean
			// and .gitflowplus/config.json is versioned and shared like any
			// other tracked file, rather than left permanently untracked.
			status, err := client.Status(cmd.Context())
			if err != nil {
				return fmt.Errorf("checking working tree status: %w", err)
			}
			if !status.Clean {
				if err := client.Add(cmd.Context(), config.DirName); err != nil {
					return fmt.Errorf("staging %s: %w", config.DirName, err)
				}
				if err := client.Commit(cmd.Context(), "Add Git Flow Plus configuration", false); err != nil {
					return fmt.Errorf("committing %s: %w", config.DirName, err)
				}
			}

			// staging branches from main before config.json exists, so
			// without this it would never receive .gitflowplus/config.json
			// (or anything else later added there, e.g. lifecycle hook
			// scripts) since it doesn't branch from develop.
			if err := client.Checkout(cmd.Context(), cfg.Branches.Staging); err != nil {
				return fmt.Errorf("checking out %q: %w", cfg.Branches.Staging, err)
			}
			if !app.ConfigLoader.Exists(app.RepoPath) {
				if err := app.ConfigLoader.Save(app.RepoPath, cfg); err != nil {
					return fmt.Errorf("saving config on %q: %w", cfg.Branches.Staging, err)
				}
			}
			stagingStatus, err := client.Status(cmd.Context())
			if err != nil {
				return fmt.Errorf("checking working tree status: %w", err)
			}
			if !stagingStatus.Clean {
				if err := client.Add(cmd.Context(), config.DirName); err != nil {
					return fmt.Errorf("staging %s: %w", config.DirName, err)
				}
				if err := client.Commit(cmd.Context(), "Add Git Flow Plus configuration", false); err != nil {
					return fmt.Errorf("committing %s: %w", config.DirName, err)
				}
			}
			if err := client.Checkout(cmd.Context(), cfg.Branches.Develop); err != nil {
				return fmt.Errorf("checking out %q: %w", cfg.Branches.Develop, err)
			}

			if result.RepoCreated {
				if err := app.Println("Initialized empty Git repository"); err != nil {
					return err
				}
			}
			if result.MainCreated {
				if err := app.Printf("Created branch %q\n", result.Main); err != nil {
					return err
				}
			}
			if result.DevelopCreated {
				if err := app.Printf("Created branch %q from %q\n", result.Develop, result.Main); err != nil {
					return err
				}
			}
			if result.StagingCreated {
				if err := app.Printf("Created branch %q from %q\n", result.Staging, result.Main); err != nil {
					return err
				}
			}
			return app.Printf("Git Flow Plus initialized. You are now on branch %q.\n", result.Develop)
		},
	}
}
