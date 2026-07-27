package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/hulhub/git-flow-plus/internal/config"
)

func newConfigCmd(app *App) *cobra.Command {
	var asJSON bool

	list := &cobra.Command{
		Use:   "list",
		Short: "Show the effective Git Flow Plus configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigList(app, asJSON)
		},
	}
	list.Flags().BoolVar(&asJSON, "json", false, "print the configuration as JSON")

	path := &cobra.Command{
		Use:   "path",
		Short: "Print the path to config.json for this repository",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintln(app.Out, config.Path(app.RepoPath))
			return err
		},
	}

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect Git Flow Plus configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigList(app, asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the configuration as JSON")
	cmd.AddCommand(list, path)

	return cmd
}

func runConfigList(app *App, asJSON bool) error {
	cfg, err := app.LoadConfig()
	if err != nil {
		return err
	}

	app.Logger.Debug("loaded configuration", "path", config.Path(app.RepoPath), "exists", app.ConfigLoader.Exists(app.RepoPath))

	if asJSON {
		enc := json.NewEncoder(app.Out)
		enc.SetIndent("", "  ")
		return enc.Encode(cfg)
	}

	lines := []string{
		fmt.Sprintf("Environment: %s", cfg.Environment),
		"Branches:",
		fmt.Sprintf("  main:        %s", cfg.Branches.Main),
		fmt.Sprintf("  staging:     %s", cfg.Branches.Staging),
		fmt.Sprintf("  develop:     %s", cfg.Branches.Develop),
		"Prefixes:",
		fmt.Sprintf("  feature:     %s", cfg.Prefixes.Feature),
		fmt.Sprintf("  hotfix:      %s", cfg.Prefixes.Hotfix),
		fmt.Sprintf("  support:     %s", cfg.Prefixes.Support),
		fmt.Sprintf("  releaseFix:  %s", cfg.Prefixes.ReleaseFix),
		fmt.Sprintf("  devops:      %s", cfg.Prefixes.DevOps),
		fmt.Sprintf("  versionTag:  %s", cfg.Prefixes.VersionTag),
		"Logging:",
		fmt.Sprintf("  level:       %s", cfg.Logging.Level),
		fmt.Sprintf("  format:      %s", cfg.Logging.Format),
		fmt.Sprintf("  color:       %v", cfg.Logging.Color),
		fmt.Sprintf("  verbose:     %v", cfg.Logging.Verbose),
		fmt.Sprintf("  debug:       %v", cfg.Logging.Debug),
	}
	for _, line := range lines {
		if _, err := fmt.Fprintln(app.Out, line); err != nil {
			return err
		}
	}

	return nil
}
