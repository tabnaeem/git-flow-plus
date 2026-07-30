package cli

import (
	"encoding/json"
	"strconv"
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
		&cobra.Command{
			Use:   "status",
			Short: "Show the current release, version, pending/included changes, and open release-fix/devops branches",
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
				return printReleaseStatus(app, report)
			},
		},
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

func printReleaseStatus(app *App, report release.StatusReport) error {
	if !report.Active {
		return app.Println("No active release on staging.")
	}

	lines := []struct {
		label string
		value string
	}{
		{"Release", report.Release},
		{"Branch", report.Branch},
		{"Version", report.Version},
		{"QA Build", itoa(report.QABuild)},
		{"Included Release Fixes", formatList(report.IncludedReleaseFixes)},
		{"Pending Release Fixes", formatList(report.PendingReleaseFixes)},
		{"Included DevOps", formatList(report.IncludedDevOps)},
		{"Pending DevOps", formatList(report.PendingDevOps)},
		{"Included Features", formatList(report.IncludedFeatures)},
		{"Deferred Features", formatList(report.DeferredFeatures)},
		{"Pending Features", formatList(report.PendingFeatures)},
		{"Open Release-Fix Branches", formatList(report.OpenReleaseFixBranches)},
		{"Open DevOps Branches", formatList(report.OpenDevOpsBranches)},
	}
	for _, l := range lines {
		if err := app.Printf("%-26s %s\n", l.label+":", l.value); err != nil {
			return err
		}
	}
	return nil
}

func formatList(items []string) string {
	if len(items) == 0 {
		return "(none)"
	}
	return strings.Join(items, ", ")
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
