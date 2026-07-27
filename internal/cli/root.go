package cli

import (
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hulhub/git-flow-plus/internal/config"
	"github.com/hulhub/git-flow-plus/internal/logging"
)

// Version is the CLI version string. It is overridden at build time via
// -ldflags "-X github.com/hulhub/git-flow-plus/internal/cli.Version=...".
var Version = "0.1.0-dev"

type rootFlags struct {
	verbose  bool
	debug    bool
	jsonLogs bool
	noColor  bool
	path     string
}

// NewRootCmd builds the "git-flow" command tree rooted at app. Invoked as a
// Git subcommand (`git flow ...`), Git resolves `git flow` to this
// executable and forwards the remaining arguments unchanged, so subcommand
// names and flags below mirror the original Git Flow syntax exactly.
func NewRootCmd(app *App) *cobra.Command {
	flags := &rootFlags{}

	root := &cobra.Command{
		Use:           "git-flow",
		Short:         "Git Flow Plus: Git Flow with enterprise release management",
		Long:          "Git Flow Plus extends the standard Git Flow branching model with\nrelease fixes, DevOps changes, and a Sprint.Release.Fix.DevOps.QABuild\nversioning scheme, while remaining fully compatible with plain Git.",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true, // internal/cli.Execute formats and logs errors itself, via logging.LogError
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Only the explicit --path flag overrides App.RepoPath: leaving
			// it unset preserves whatever the caller already configured
			// (NewApp defaults it to the working directory; tests and other
			// embedders may set it directly).
			if flags.path != "" {
				abs, err := filepath.Abs(flags.path)
				if err != nil {
					return err
				}
				app.RepoPath = abs
			}

			app.Logger = buildLogger(app, flags, cmd)
			return nil
		},
	}

	root.PersistentFlags().BoolVarP(&flags.verbose, "verbose", "v", false, "enable debug-level logging")
	root.PersistentFlags().BoolVar(&flags.debug, "debug", false, "enable trace-level logging (most detailed)")
	root.PersistentFlags().BoolVar(&flags.jsonLogs, "json-log", false, "emit structured logs as JSON instead of text (for CI/CD)")
	root.PersistentFlags().BoolVar(&flags.noColor, "no-color", false, "disable colorized console output")
	root.PersistentFlags().StringVar(&flags.path, "path", "", "path to the Git repository to operate on (default: current directory)")

	root.AddCommand(
		newInitCmd(app),
		newFeatureCmd(app),
		newReleaseCmd(app),
		newHotfixCmd(app),
		newSupportCmd(app),
		newReleaseFixCmd(app),
		newDevOpsCmd(app),
		newDoctorCmd(app),
		newConfigCmd(app),
		newVersionCmd(app),
	)

	return root
}

// buildLogger resolves the effective logging configuration — level,
// format, and color — with precedence explicit CLI flag > config.json's
// "logging" block (itself seeded by "environment" and subject to
// GITFLOWPLUS_* env var overrides, see config.ApplyEnvOverrides) >
// built-in default. Config errors here are deliberately non-fatal: a
// broken config.json shouldn't prevent even seeing the error that
// describes it, so this falls back to development defaults and lets the
// command's own app.LoadConfig() call surface the real failure normally.
func buildLogger(app *App, flags *rootFlags, cmd *cobra.Command) *slog.Logger {
	loggingCfg := config.DefaultLogging(config.EnvDevelopment)
	if cfg, err := app.LoadConfig(); err == nil {
		loggingCfg = cfg.Logging
	}

	level := logging.ParseLevel(loggingCfg.Level)
	if loggingCfg.Verbose {
		level = logging.LevelDebug
	}
	if loggingCfg.Debug {
		level = logging.LevelTrace
	}
	if cmd.Flags().Changed("verbose") && flags.verbose {
		level = logging.LevelDebug
	}
	if cmd.Flags().Changed("debug") && flags.debug {
		level = logging.LevelTrace
	}

	format := logging.FormatText
	if strings.EqualFold(loggingCfg.Format, "json") {
		format = logging.FormatJSON
	}
	if cmd.Flags().Changed("json-log") && flags.jsonLogs {
		format = logging.FormatJSON
	}

	color := format == logging.FormatText && resolveColor(app.Err, loggingCfg.Color, flags.noColor)

	return logging.New(logging.Options{
		Writer: app.Err,
		Level:  &level,
		Format: format,
		Color:  color,
	})
}

// Execute runs the Git Flow Plus CLI with real OS dependencies and returns
// the process exit code.
func Execute() int {
	app := NewApp()
	if err := NewRootCmd(app).Execute(); err != nil {
		logging.LogError(app.Logger, err)
		return 1
	}
	return 0
}
