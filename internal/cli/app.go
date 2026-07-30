// Package cli implements the Git Flow Plus command-line interface using
// Cobra. Command handlers in this package are intentionally thin: they parse
// and validate arguments, then delegate to the business-logic packages
// (internal/gitflow, internal/release, internal/version) for everything
// else. No Git or release-management logic lives here.
package cli

import (
	"fmt"
	"io"
	"os"

	"log/slog"

	"github.com/tabnaeem/git-flow-plus/internal/config"
	"github.com/tabnaeem/git-flow-plus/internal/feature"
	"github.com/tabnaeem/git-flow-plus/internal/git"
	"github.com/tabnaeem/git-flow-plus/internal/gitflow"
	"github.com/tabnaeem/git-flow-plus/internal/hooks"
	"github.com/tabnaeem/git-flow-plus/internal/logging"
	"github.com/tabnaeem/git-flow-plus/internal/release"
	"github.com/tabnaeem/git-flow-plus/internal/version"
)

// App bundles the dependencies shared by every CLI command. It is
// constructed once by the root command and passed to each subcommand
// factory, keeping command logic testable and free of package-level state.
type App struct {
	// Logger is used for structured diagnostic output.
	Logger *slog.Logger
	// ConfigLoader reads and writes .gitflowplus/config.json.
	ConfigLoader config.Loader
	// Out is the destination for normal command output.
	Out io.Writer
	// Err is the destination for error/diagnostic output.
	Err io.Writer
	// RepoPath is the filesystem path to the repository the command
	// operates against, resolved from the --path flag (default: cwd).
	RepoPath string
	// Runner executes git subcommands; injected so tests can substitute a
	// fake instead of invoking a real git binary.
	Runner git.Runner
	// Hooks runs CI/CD lifecycle hooks (post-qa-tag, post-production-tag);
	// injected so tests can substitute a fake instead of touching the
	// filesystem/OS process table.
	Hooks hooks.Runner
	// ColorEnabled reports whether commands that print directly to Out
	// (e.g. `git flow doctor`) should use ANSI color. Resolved once in
	// the root command's PersistentPreRunE from config.json's
	// logging.color, --no-color, NO_COLOR, and whether Out is a real
	// terminal; false until then.
	ColorEnabled bool
}

// NewApp constructs an App wired to real OS dependencies (stdout, stderr,
// working directory, and a Viper-backed config loader). A no-op logger is
// installed until the root command's PersistentPreRunE configures the real
// one from parsed flags.
func NewApp() *App {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	return &App{
		// A sane default in case an error occurs during flag parsing,
		// before PersistentPreRunE gets a chance to reconfigure this
		// with the repository's actual logging settings.
		Logger:       logging.New(logging.Options{Writer: os.Stderr, Format: logging.FormatText}),
		ConfigLoader: config.NewLoader(),
		Out:          os.Stdout,
		Err:          os.Stderr,
		RepoPath:     cwd,
		Runner:       git.NewExecRunner(),
		Hooks:        hooks.NewRunner(),
	}
}

// LoadConfig loads the effective Git Flow Plus configuration for RepoPath,
// falling back to config.Default() if no config file has been saved yet,
// then applies any GITFLOWPLUS_* environment variable overrides (see
// config.ApplyEnvOverrides). This is the one blessed path every command
// uses to read configuration, so environment overrides apply uniformly
// everywhere without each command needing to know about them.
func (a *App) LoadConfig() (*config.Config, error) {
	cfg, err := a.ConfigLoader.Load(a.RepoPath)
	if err != nil {
		return nil, logging.NewAppError(fmt.Errorf("loading config: %w", err),
			"the repository's .gitflowplus/config.json could not be read or contains invalid JSON",
			"fix or remove "+config.Path(a.RepoPath)+", or run 'git flow init' to regenerate it")
	}
	config.ApplyEnvOverrides(cfg)
	return cfg, nil
}

// Printf writes a formatted message to Out, returning any write error
// instead of silently discarding it (Out may be a pipe that can fail).
func (a *App) Printf(format string, args ...any) error {
	_, err := fmt.Fprintf(a.Out, format, args...)
	return err
}

// Println writes args to Out followed by a newline, returning any write
// error instead of silently discarding it.
func (a *App) Println(args ...any) error {
	_, err := fmt.Fprintln(a.Out, args...)
	return err
}

// GitClient returns a git.Client bound to RepoPath using Runner.
func (a *App) GitClient() git.Client {
	return git.NewClient(a.Runner, a.RepoPath)
}

// GitFlowService returns a gitflow.Service bound to RepoPath, using cfg's
// branch/prefix conventions.
func (a *App) GitFlowService(cfg *config.Config) gitflow.Service {
	return gitflow.NewService(a.GitClient(), cfg, a.Logger)
}

// ReleaseService returns a release.Service bound to RepoPath, composing a
// fresh gitflow.Service, git.Client, and the manifest/version loaders.
func (a *App) ReleaseService(cfg *config.Config) release.Service {
	return release.NewService(release.Dependencies{
		GitFlow:        a.GitFlowService(cfg),
		Git:            a.GitClient(),
		RepoPath:       a.RepoPath,
		Config:         cfg,
		ManifestLoader: release.NewLoader(),
		VersionLoader:  version.NewLoader(),
		FeatureLoader:  feature.NewLoader(),
		Hooks:          a.Hooks,
		Logger:         a.Logger,
	})
}
