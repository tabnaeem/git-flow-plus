package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tabnaeem/git-flow-plus/internal/release"
)

func newReleaseCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "release",
		Short: "Manage the release cycle on staging and Git Flow Plus release metadata",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "start <name>",
			Short: "Start a new release on staging (name is \"Sprint.Release\", e.g. \"5.2\") and tag QA Build 1",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := app.LoadConfig()
				if err != nil {
					return err
				}
				result, err := app.ReleaseService(cfg).StartRelease(cmd.Context(), args[0])
				if err != nil {
					return err
				}
				if err := app.Printf("Started release %q on %q at version %s\n", result.Release, result.Branch, result.Version); err != nil {
					return err
				}
				return app.Printf("Tagged %q\n", result.Tag)
			},
		},
		&cobra.Command{
			Use:   "finish <name>",
			Short: "Production Release: archive the manifest, merge staging into main and develop, and tag the exact commit that passed QA",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := app.LoadConfig()
				if err != nil {
					return err
				}
				result, err := app.ReleaseService(cfg).FinishRelease(cmd.Context(), args[0])
				if err != nil {
					return err
				}
				return app.Printf("Finished release %q at version %s, tagged %q\n", result.Release, result.Version, result.Tag)
			},
		},
		&cobra.Command{
			Use:   "build",
			Short: "Cut a QA build: fold pending release fixes/DevOps changes into the version and tag it. Must be run from staging.",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := app.LoadConfig()
				if err != nil {
					return err
				}
				result, err := app.ReleaseService(cfg).Build(cmd.Context())
				if err != nil {
					return err
				}
				if err := app.Printf("QA Build #%d: version %s (+%d release fix(es), +%d devops change(s))\n",
					result.QABuild, result.Version, result.NewFixes, result.NewDevOps); err != nil {
					return err
				}
				return app.Printf("Tagged %q\n", result.Tag)
			},
		},
		newReleaseStatusCmd(app),
		&cobra.Command{
			Use:   "version",
			Short: "Print the current Sprint.Release.Fix.DevOps.QAIteration version",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := app.LoadConfig()
				if err != nil {
					return err
				}
				v, err := app.ReleaseService(cfg).Version(cmd.Context())
				if err != nil {
					return err
				}
				return app.Println(v.String())
			},
		},
		&cobra.Command{
			Use:   "manifest",
			Short: "Print the current release manifest",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := app.LoadConfig()
				if err != nil {
					return err
				}
				m, err := app.ReleaseService(cfg).Manifest(cmd.Context())
				if err != nil {
					return err
				}
				enc := json.NewEncoder(app.Out)
				enc.SetIndent("", "  ")
				return enc.Encode(m)
			},
		},
	)

	cmd.AddCommand(newReleaseFeatureCmd(app))

	return cmd
}

func newReleaseFeatureCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "feature",
		Short: "Release Planning: select which approved features belong to the active release",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List every approved feature in the registry",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := app.LoadConfig()
				if err != nil {
					return err
				}
				features, err := app.ReleaseService(cfg).ListApprovedFeatures(cmd.Context())
				if err != nil {
					return err
				}
				if len(features) == 0 {
					return app.Println("No approved features.")
				}
				for _, f := range features {
					if err := app.Printf("%-16s state=%-18s release=%s\n", f.ID, f.State, valueOrNone(f.Release)); err != nil {
						return err
					}
				}
				return nil
			},
		},
		&cobra.Command{
			Use:   "add <id>",
			Short: "Merge an approved feature's branch into staging and include it in the active release (Release Planning)",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := app.LoadConfig()
				if err != nil {
					return err
				}
				if err := app.ReleaseService(cfg).AddFeatureToRelease(cmd.Context(), args[0]); err != nil {
					return err
				}
				return app.Printf("Merged feature %q into staging and added it to the active release\n", args[0])
			},
		},
		&cobra.Command{
			Use:   "approve <id>",
			Short: "Mark a feature as approved after Unit Testing",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := app.LoadConfig()
				if err != nil {
					return err
				}
				if err := app.ReleaseService(cfg).ApproveFeature(cmd.Context(), args[0]); err != nil {
					return err
				}
				return app.Printf("Approved feature %q\n", args[0])
			},
		},
		&cobra.Command{
			Use:   "defer <id>",
			Short: "Defer an approved feature to a future release",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := app.LoadConfig()
				if err != nil {
					return err
				}
				if err := app.ReleaseService(cfg).DeferFeature(cmd.Context(), args[0]); err != nil {
					return err
				}
				return app.Printf("Deferred feature %q\n", args[0])
			},
		},
		&cobra.Command{
			Use:   "status",
			Short: "Show Approved, Included, Deferred, and Pending features",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := app.LoadConfig()
				if err != nil {
					return err
				}
				report, err := app.ReleaseService(cfg).FeatureStatus(cmd.Context())
				if err != nil {
					return err
				}
				lines := []struct {
					label string
					value string
				}{
					{"Approved", formatList(report.Approved)},
					{"Included", formatList(report.Included)},
					{"Deferred", formatList(report.Deferred)},
					{"Pending", formatList(report.Pending)},
				}
				for _, l := range lines {
					if err := app.Printf("%-10s %s\n", l.label+":", l.value); err != nil {
						return err
					}
				}
				return nil
			},
		},
	)

	return cmd
}

func valueOrNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

// newReleaseStatusCmd builds `git flow release status`: a full
// human-readable (or, with --json, machine-readable) summary of the
// release active on staging, assembled entirely from release.Service.
// Status()'s existing StatusReport (release.json + version.json, already
// loaded by every other release subcommand) — no second state-management
// system, no new persisted fields.
func newReleaseStatusCmd(app *App) *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show a full summary of the current release: version, features, fixes, DevOps, QA builds, and readiness",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := app.LoadConfig()
			if err != nil {
				return err
			}
			report, err := app.ReleaseService(cfg).Status(cmd.Context())
			if err != nil {
				return err
			}
			if asJSON {
				return printReleaseStatusJSON(app, report)
			}
			return printReleaseStatus(app, report)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the release status as JSON")
	return cmd
}

// releaseStatusJSON is the `git flow release status --json` output shape.
// Deliberately its own type, not a direct marshal of StatusReport or
// version.Version: the requested field names (release, sprint, features,
// release_fixes, devops, qa_build, status) don't match either type's own
// json tags 1:1 — e.g. version.Version's "release" field is the feature
// counter, not the release name, and StatusReport has no single "status"
// verdict field. Keeping this separate avoids either silently colliding
// field names or coupling the public JSON contract to those types'
// internal shapes.
type releaseStatusJSON struct {
	Release      string `json:"release"`
	Sprint       int    `json:"sprint"`
	Features     int    `json:"features"`
	ReleaseFixes int    `json:"release_fixes"`
	DevOps       int    `json:"devops"`
	QABuild      int    `json:"qa_build"`
	Status       string `json:"status"`
}

func printReleaseStatusJSON(app *App, report release.StatusReport) error {
	enc := json.NewEncoder(app.Out)
	enc.SetIndent("", "  ")

	if !report.Active {
		// No "release"/"sprint"/etc. fields: there is no release to report
		// them for, and a numeric field defaulting to 0 here would be
		// indistinguishable from a genuinely-zero value on an active
		// release (e.g. a release legitimately started as "0.1").
		return enc.Encode(map[string]any{
			"active": false,
			"status": "no_release",
		})
	}

	v := report.VersionInfo
	return enc.Encode(releaseStatusJSON{
		Release:      "v" + report.Version,
		Sprint:       v.Sprint,
		Features:     v.Release,
		ReleaseFixes: v.Fixes,
		DevOps:       v.DevOps,
		QABuild:      v.QA,
		Status:       releaseReadinessStatus(report),
	})
}

// releaseReadinessStatus computes `release status`'s readiness verdict
// purely from release fixes/DevOps changes/features already recorded as
// Pending in the release manifest (release.Manifest's FeatureSet/ChangeSet
// — no new state is stored or invented here). "ready" means every merged
// change has already been folded into a QA build by `git flow release
// build` — it does NOT mean `git flow release finish` has been approved or
// run: Git Flow Plus tracks no such flag, and by definition a release
// `status` can report on hasn't been finished yet (finishing archives and
// removes the manifest status reads from).
func releaseReadinessStatus(report release.StatusReport) string {
	if !report.Active {
		return "no_release"
	}
	if len(report.PendingFeatures) == 0 && len(report.PendingReleaseFixes) == 0 && len(report.PendingDevOps) == 0 {
		return "ready"
	}
	return "not_ready"
}

// statusRuleWidth matches the divider width used in the requested report
// layout.
const statusRuleWidth = 38

func printReleaseStatus(app *App, report release.StatusReport) error {
	if !report.Active {
		return app.Println("No active release on staging.")
	}

	rule := strings.Repeat("─", statusRuleWidth)
	v := report.VersionInfo

	lines := []string{
		"Git Flow Plus Release Status",
		rule,
		"",
		fmt.Sprintf("Release:       v%s", report.Version),
		fmt.Sprintf("Sprint:        %d", v.Sprint),
		fmt.Sprintf("Features:      %d", v.Release),
		fmt.Sprintf("Release Fixes: %d", v.Fixes),
		fmt.Sprintf("DevOps:        %d", v.DevOps),
		fmt.Sprintf("QA Builds:     %d", v.QA),
		"",
		"Features",
		rule,
		"",
	}
	lines = append(lines, featureStatusLines(report)...)
	lines = append(lines, "", "Release Fixes", rule, "")
	lines = append(lines, changeStatusLines(report.IncludedReleaseFixes, report.PendingReleaseFixes)...)
	lines = append(lines, "", "QA", rule, "")
	lines = append(lines, buildStatusLines(report)...)
	lines = append(lines, "", "Release Readiness", rule, "")
	lines = append(lines, readinessStatusLines(report)...)
	lines = append(lines, "", "Status:", statusHeadline(report))

	for _, l := range lines {
		if err := app.Println(l); err != nil {
			return err
		}
	}
	return nil
}

// featureStatusLines renders the Features section: every feature merged
// into the release (Included), explicitly held back (Deferred), or still
// awaiting a Release Planning decision (Pending) — the three real,
// existing buckets release.Manifest.Features already tracks (see
// release.FeatureSet). No per-feature label beyond these three is used,
// since nothing else about a feature's release-planning state is tracked.
func featureStatusLines(report release.StatusReport) []string {
	var out []string
	for _, id := range report.IncludedFeatures {
		out = append(out, fmt.Sprintf("✓ %-12s Included", id))
	}
	for _, id := range report.DeferredFeatures {
		out = append(out, fmt.Sprintf("○ %-12s Deferred", id))
	}
	for _, id := range report.PendingFeatures {
		out = append(out, fmt.Sprintf("○ %-12s Pending", id))
	}
	if len(out) == 0 {
		out = append(out, "(none)")
	}
	return out
}

// changeStatusLines renders a release-fix or DevOps section: entries
// already folded into the version by a QA build (Included) versus merged
// but awaiting the next build (Pending) — release.ChangeSet's two real
// states.
func changeStatusLines(included, pending []string) []string {
	var out []string
	for _, id := range included {
		out = append(out, fmt.Sprintf("✓ %s", id))
	}
	for _, id := range pending {
		out = append(out, fmt.Sprintf("○ %-12s Pending", id))
	}
	if len(out) == 0 {
		out = append(out, "(none)")
	}
	return out
}

// buildStatusLines renders the QA section from the release's build
// history (release.Manifest.History, one release.BuildRecord per `git
// flow release build`, plus the initial build `release start` creates).
// The latest build — the one matching the version's current QA
// iteration — is marked Current; every earlier one is simply done.
func buildStatusLines(report release.StatusReport) []string {
	var out []string
	for _, b := range report.Builds {
		if b.Build == report.QABuild {
			out = append(out, fmt.Sprintf("→ Build #%d Current", b.Build))
		} else {
			out = append(out, fmt.Sprintf("✓ Build #%d", b.Build))
		}
	}
	if len(out) == 0 {
		out = append(out, "(none)")
	}
	return out
}

// readinessStatusLines renders the Release Readiness checklist. Features/
// Release Fixes/DevOps Changes are green the moment nothing is left
// Pending in the manifest. QA and Production Release are always shown "in
// progress"/"Pending": Git Flow Plus records no "QA complete" or
// "production approved" flag anywhere, and an active release is, by
// definition, one `git flow release finish` hasn't been run for yet (that
// operation archives and removes the manifest `status` reads from) — so
// stating otherwise here would mean inventing state that doesn't exist.
func readinessStatusLines(report release.StatusReport) []string {
	return []string{
		fmt.Sprintf("%-21s %s", "Features", readinessMark(len(report.PendingFeatures))),
		fmt.Sprintf("%-21s %s", "Release Fixes", readinessMark(len(report.PendingReleaseFixes))),
		fmt.Sprintf("%-21s %s", "DevOps Changes", readinessMark(len(report.PendingDevOps))),
		fmt.Sprintf("%-21s → Build #%d in progress", "QA", report.QABuild),
		fmt.Sprintf("%-21s ○ Pending (run 'git flow release finish')", "Production Release"),
	}
}

func readinessMark(pendingCount int) string {
	if pendingCount == 0 {
		return "✓"
	}
	return fmt.Sprintf("○ Pending (%d)", pendingCount)
}

func statusHeadline(report release.StatusReport) string {
	if releaseReadinessStatus(report) == "ready" {
		return "READY FOR PRODUCTION RELEASE"
	}
	return "NOT READY FOR PRODUCTION"
}

func formatList(items []string) string {
	if len(items) == 0 {
		return "(none)"
	}
	return strings.Join(items, ", ")
}
