package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tabnaeem/git-flow-plus/internal/config"
	"github.com/tabnaeem/git-flow-plus/internal/gitflow"
	"github.com/tabnaeem/git-flow-plus/internal/logging"
)

// sectionReleaseTooling/sectionReleaseState are this file's own sections,
// alongside gitflow.SectionEnvironment/SectionBranchModel — kept as plain
// constants here (not in internal/gitflow) since neither concept is
// scoped to a Git repository at all: Release Tooling depends on this
// project's own files (go.mod, .goreleaser.yaml), and Release State
// depends on internal/release, which internal/gitflow deliberately never
// imports — see Architecture.md.
const (
	sectionReleaseTooling = "Release Tooling"
	sectionReleaseState   = "Release State"
)

// doctorSectionOrder controls display order; a section with no checks in
// it (e.g. "Release Tooling" for a non-Go project) is simply omitted.
var doctorSectionOrder = []string{
	gitflow.SectionEnvironment,
	gitflow.SectionBranchModel,
	sectionReleaseTooling,
	sectionReleaseState,
}

// newDoctorCmd builds `git flow doctor`. Checks are assembled from three
// sources:
//   - internal/gitflow's Doctor() covers everything scoped to the Git
//     repository itself (binary present and at a supported version, the
//     repository, remote, branches, working tree, permissions).
//   - This file covers Git Flow Plus's own installation (PATH resolution,
//     config.json existence/validity) and, conditionally, this project's
//     own release tooling (Go/GoReleaser/Syft — see releaseToolingChecks).
//   - Release State reuses release.Service.Status() directly (Phase 1) -
//     no separate manifest-validation logic lives here.
//
// Unlike the config-loading pattern used elsewhere in this package,
// doctor deliberately does NOT call app.LoadConfig() and bail out on
// error: a corrupt config.json becomes a failing "Configuration" check
// instead of aborting before any other check even runs. Showing the full
// picture, even when something's broken, is the entire point of a
// diagnostic command.
func newDoctorCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check whether the local environment is correctly configured for Git Flow Plus",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			loadedCfg, loadErr := app.ConfigLoader.Load(app.RepoPath)
			cfg := loadedCfg
			if loadErr != nil || cfg == nil {
				cfg = config.Default()
			}
			config.ApplyEnvOverrides(cfg)

			var checks []gitflow.DoctorCheck
			checks = append(checks, gitflow.NewService(app.GitClient(), cfg, app.Logger).Doctor(ctx).Checks...)
			checks = append(checks, gitFlowPlusCheck(), configurationCheck(app, loadErr))
			checks = append(checks, releaseToolingChecks(app.RepoPath)...)
			checks = append(checks, releaseStateCheck(ctx, app, cfg))

			return printDoctorReport(app, checks)
		},
	}
}

// gitFlowPlusCheck reports whether `git flow ...` will resolve as a Git
// subcommand (requires a binary literally named "git-flow", not
// "git-flow-plus", on PATH), folding in the running build's own version
// for context. Reuses pathCheck()'s existing, unchanged PATH-resolution
// logic - only the label/section and the version-prefixed Detail are new.
func gitFlowPlusCheck() gitflow.DoctorCheck {
	c := pathCheck()
	c.Section = gitflow.SectionEnvironment
	c.Name = "Git Flow Plus"
	c.Detail = fmt.Sprintf("%s (build %s, commit %s) - %s", Version, BuildNumber, GitCommit, c.Detail)
	return c
}

// pathCheck reports whether `git flow ...` will resolve as a Git
// subcommand, which requires a binary literally named "git-flow" (not
// "git-flow-plus") somewhere on PATH — see InstallationGuide.md/the
// Windows installer's PATH step, which both install under that name for
// exactly this reason.
func pathCheck() gitflow.DoctorCheck {
	if _, err := exec.LookPath("git-flow"); err == nil {
		return gitflow.DoctorCheck{
			Name: "PATH", OK: true,
			Detail: "'git-flow' resolves on PATH; 'git flow ...' works as a Git subcommand",
		}
	}
	if _, err := exec.LookPath("git-flow-plus"); err == nil {
		return gitflow.DoctorCheck{
			Name: "PATH", OK: false,
			Detail: "'git-flow-plus' is on PATH but 'git-flow' is not, so 'git flow ...' will not resolve; see WindowsInstallation.md/Troubleshooting.md",
		}
	}
	return gitflow.DoctorCheck{
		Name: "PATH", OK: false,
		Detail: "neither 'git-flow' nor 'git-flow-plus' found on PATH; reinstall or add the install directory to PATH",
	}
}

// configurationCheck distinguishes "missing" from "exists but invalid"
// from "exists and valid" — app.ConfigLoader.Exists/Load are called by
// the caller once and passed in here (loadErr), rather than re-loading,
// so this stays a pure formatting step.
func configurationCheck(app *App, loadErr error) gitflow.DoctorCheck {
	const name = "Configuration"
	switch {
	case !app.ConfigLoader.Exists(app.RepoPath):
		return gitflow.DoctorCheck{
			Section: gitflow.SectionEnvironment, Name: name, OK: false,
			Detail: fmt.Sprintf("Configuration file is missing. Run 'git flow init'. (expected at %s)", config.Path(app.RepoPath)),
		}
	case loadErr != nil:
		return gitflow.DoctorCheck{
			Section: gitflow.SectionEnvironment, Name: name, OK: false,
			Detail: fmt.Sprintf("%s exists but could not be parsed: %v", config.Path(app.RepoPath), loadErr),
		}
	default:
		return gitflow.DoctorCheck{
			Section: gitflow.SectionEnvironment, Name: name, OK: true,
			Detail: fmt.Sprintf("found at %s", config.Path(app.RepoPath)),
		}
	}
}

// releaseStateCheck reuses release.Service.Status() (Phase 1) as-is: if
// staging has no release.json, that is a completely normal, healthy
// state (matching the pre-Phase-3 "release configuration" check's own
// philosophy); if one exists but fails to load/parse, that's a real
// failure. No manifest-validation logic is duplicated here.
func releaseStateCheck(ctx context.Context, app *App, cfg *config.Config) gitflow.DoctorCheck {
	const name = "Manifest"
	report, err := app.ReleaseService(cfg).Status(ctx)
	switch {
	case err != nil:
		return gitflow.DoctorCheck{Section: sectionReleaseState, Name: name, OK: false, Detail: err.Error()}
	case !report.Active:
		return gitflow.DoctorCheck{
			Section: sectionReleaseState, Name: name, OK: true,
			Detail: "no active release; run 'git flow release start' to begin one",
		}
	default:
		return gitflow.DoctorCheck{
			Section: sectionReleaseState, Name: name, OK: true,
			Detail: fmt.Sprintf("active release %q, version %s", report.Release, report.Version),
		}
	}
}

// releaseToolingChecks only ever returns checks for tools the current
// project actually needs, detected from its own files — never a blanket
// "GoReleaser/Syft are mandatory" requirement for every Git Flow Plus
// user, most of whom aren't building Git Flow Plus itself:
//   - go.mod present -> check `go` is on PATH.
//   - .goreleaser.yaml/.yml present -> check `goreleaser` is on PATH.
//   - that file's content mentions "sboms:" -> also check `syft`.
//
// The .goreleaser.yaml scan is a plain substring search, not a YAML
// parse — sufficient to detect relevance without adding a YAML parsing
// dependency solely for this diagnostic.
func releaseToolingChecks(repoPath string) []gitflow.DoctorCheck {
	var checks []gitflow.DoctorCheck

	if fileExists(filepath.Join(repoPath, "go.mod")) {
		checks = append(checks, toolingCheck("Go", "go", "this project has a go.mod"))
	}

	goreleaserConfig := firstExistingFile(repoPath, ".goreleaser.yaml", ".goreleaser.yml")
	if goreleaserConfig != "" {
		checks = append(checks, toolingCheck("GoReleaser", "goreleaser", "this project has a "+filepath.Base(goreleaserConfig)))
		if fileContains(goreleaserConfig, "sboms:") {
			checks = append(checks, toolingCheck("Syft", "syft", filepath.Base(goreleaserConfig)+" has an sboms: block"))
		}
	}

	return checks
}

func toolingCheck(label, binary, why string) gitflow.DoctorCheck {
	if _, err := exec.LookPath(binary); err == nil {
		return gitflow.DoctorCheck{
			Section: sectionReleaseTooling, Name: label, OK: true,
			Detail: fmt.Sprintf("found on PATH (%s)", why),
		}
	}
	return gitflow.DoctorCheck{
		Section: sectionReleaseTooling, Name: label, OK: false,
		Detail: fmt.Sprintf("%q not found on PATH, but %s; see Building.md", binary, why),
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func firstExistingFile(dir string, names ...string) string {
	for _, name := range names {
		path := filepath.Join(dir, name)
		if fileExists(path) {
			return path
		}
	}
	return ""
}

func fileContains(path, substr string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), substr)
}

// printDoctorReport renders checks grouped by Section, in
// doctorSectionOrder, matching `git flow doctor`'s specified layout: a
// title, one heading + divider per non-empty section, "✓"/"✗" per check,
// the failing ones' Detail printed underneath, and a final "Environment
// Status: READY"/"NOT READY" line. Returns a non-nil error when any
// check failed, so cobra reports a non-zero exit code.
func printDoctorReport(app *App, checks []gitflow.DoctorCheck) error {
	if err := app.Println("Git Flow Plus Doctor"); err != nil {
		return err
	}
	if err := app.Println(); err != nil {
		return err
	}

	healthy := true
	for _, section := range doctorSectionOrder {
		var inSection []gitflow.DoctorCheck
		for _, c := range checks {
			if c.Section == section {
				inSection = append(inSection, c)
			}
		}
		if len(inSection) == 0 {
			continue
		}

		lines := []string{section, strings.Repeat("─", doctorRuleWidth), ""}
		for _, c := range inSection {
			if !c.OK {
				healthy = false
			}
			lines = append(lines, formatDoctorCheck(c, app.ColorEnabled))
			if !c.OK && c.Detail != "" {
				lines = append(lines, "  "+c.Detail)
			}
		}
		lines = append(lines, "")

		for _, l := range lines {
			if err := app.Println(l); err != nil {
				return err
			}
		}
	}

	if err := app.Println("Environment Status:"); err != nil {
		return err
	}
	if !healthy {
		if err := app.Println("NOT READY"); err != nil {
			return err
		}
		return fmt.Errorf("doctor: one or more checks failed")
	}
	return app.Println("READY")
}

// doctorRuleWidth matches the divider width used in the requested report
// layout.
const doctorRuleWidth = 24

func formatDoctorCheck(c gitflow.DoctorCheck, colorEnabled bool) string {
	symbol, color := "✓", logging.AnsiGreen
	if !c.OK {
		symbol, color = "✗", logging.AnsiRed
	}
	if colorEnabled {
		symbol = color + symbol + logging.AnsiReset
	}
	return fmt.Sprintf("%-20s %s", c.Name, symbol)
}
