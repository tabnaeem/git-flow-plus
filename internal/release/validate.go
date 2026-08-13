package release

import (
	"context"
	"fmt"

	"github.com/tabnaeem/git-flow-plus/internal/feature"
	"github.com/tabnaeem/git-flow-plus/internal/version"
)

// Validate is `git flow release validate`: a read-only pre-flight for
// FinishRelease. It never mutates release.json, version.json, or
// features.json, and it never touches Git beyond read-only queries
// (BranchExists, TagExists, Status).
//
// Every check reuses data FinishRelease/Build/Status already consult —
// there is no second approval or state-tracking system here. Where the
// requested checklist describes a concept Git Flow Plus has no persisted
// flag for (a "QA build approval", a general "required approvals" gate),
// the check is instead grounded in the closest real, derivable fact — see
// each check function's own comment.
func (s *service) Validate(ctx context.Context) (ValidationReport, error) {
	if !s.deps.ManifestLoader.Exists(s.deps.RepoPath) {
		return ValidationReport{
			Active: false,
			Checks: []ValidationCheck{
				{
					Name:   "Release exists",
					OK:     false,
					Detail: "No active release exists on staging; run 'git flow release start' first",
				},
			},
		}, nil
	}

	m, err := s.deps.ManifestLoader.Load(s.deps.RepoPath)
	if err != nil {
		return ValidationReport{}, fmt.Errorf("loading manifest: %w", err)
	}
	v, err := s.deps.VersionLoader.Load(s.deps.RepoPath)
	if err != nil {
		return ValidationReport{}, fmt.Errorf("loading version: %w", err)
	}
	registry, err := s.deps.FeatureLoader.Load(s.deps.RepoPath)
	if err != nil {
		return ValidationReport{}, fmt.Errorf("loading feature registry: %w", err)
	}

	var checks []ValidationCheck
	checks = append(checks, ValidationCheck{Name: "Release exists", OK: true})
	checks = append(checks, s.validateReleaseBranch(ctx, m))
	checks = append(checks, validateReleaseState(m))
	checks = append(checks, validateFeaturesFinalized(m)...)
	checks = append(checks, validateFeaturesApproved(m, registry)...)
	checks = append(checks, validateChangeSetFinalized("Release fix", "Release fixes finalized", m.ReleaseFixes)...)
	checks = append(checks, validateChangeSetFinalized("DevOps change", "DevOps changes finalized", m.DevOps)...)
	checks = append(checks, validateQAStatus(m, v))
	checks = append(checks, validateVersion(m, v))
	checks = append(checks, s.validateWorkingTreeClean(ctx))
	checks = append(checks, s.validateTagAvailable(ctx, m))
	checks = append(checks, validateNoConflictingState(s.deps.Config.Branches.Staging, m))

	ready := true
	for _, c := range checks {
		if !c.OK {
			ready = false
			break
		}
	}

	return ValidationReport{Active: true, Checks: checks, Ready: ready}, nil
}

// validateReleaseBranch checks that the branch release.json itself
// records (normally staging) actually exists — the branch every other
// release operation checks out before touching release state.
func (s *service) validateReleaseBranch(ctx context.Context, m *Manifest) ValidationCheck {
	const name = "Release branch exists"
	exists, err := s.deps.Git.BranchExists(ctx, m.Branch)
	if err != nil {
		return ValidationCheck{Name: name, Detail: fmt.Sprintf("Could not check branch %q: %v", m.Branch, err)}
	}
	if !exists {
		return ValidationCheck{Name: name, Detail: fmt.Sprintf("Release branch %q does not exist", m.Branch)}
	}
	return ValidationCheck{Name: name, OK: true}
}

// validateReleaseState is a structural sanity check on the manifest
// itself — the fields every other release operation assumes are
// populated. Load() already guarantees the JSON parsed; this guarantees
// it parsed into something semantically usable (catches a hand-edited or
// partially-written release.json a JSON parser alone wouldn't reject).
func validateReleaseState(m *Manifest) ValidationCheck {
	const name = "Release state valid"
	switch {
	case m.Release == "":
		return ValidationCheck{Name: name, Detail: "Release manifest has no release name recorded"}
	case m.Branch == "":
		return ValidationCheck{Name: name, Detail: "Release manifest has no branch recorded"}
	case m.CurrentVersion == "":
		return ValidationCheck{Name: name, Detail: "Release manifest has no current version recorded"}
	case m.CurrentQABuild < 1:
		return ValidationCheck{Name: name, Detail: fmt.Sprintf("Release manifest's QA build counter is %d, want at least 1", m.CurrentQABuild)}
	}
	return ValidationCheck{Name: name, OK: true}
}

// validateFeaturesFinalized reports every feature still Pending a Release
// Planning decision (release.Manifest.Features.Pending — approved, but
// neither added to nor deferred from this release yet) as its own failing
// check, one per feature. A feature that has never been approved at all
// never appears here, or anywhere in the manifest's feature tracking — it
// isn't part of Release Planning for this release until it is.
func validateFeaturesFinalized(m *Manifest) []ValidationCheck {
	if len(m.Features.Pending) == 0 {
		return []ValidationCheck{{Name: "Features finalized", OK: true}}
	}
	checks := make([]ValidationCheck, 0, len(m.Features.Pending))
	for _, id := range m.Features.Pending {
		checks = append(checks, ValidationCheck{
			Detail: fmt.Sprintf("Feature %s is pending a Release Planning decision (approve and add, or defer, it)", id),
		})
	}
	return checks
}

// validateFeaturesApproved is an integrity check, distinct from
// validateFeaturesFinalized: every feature the manifest already claims is
// Included must still resolve, in the Feature Registry, to a state that
// has actually reached Approved — AddFeatureToRelease only ever
// transitions Approved -> IncludedInRelease, so this should always hold
// under normal operation. It exists to catch registry/manifest drift
// (e.g. hand-edited files), not a scenario Git Flow Plus's own commands
// can produce on their own.
func validateFeaturesApproved(m *Manifest, registry *feature.Registry) []ValidationCheck {
	var checks []ValidationCheck
	for _, id := range m.Features.Included {
		f, ok := registry.Find(id)
		switch {
		case !ok:
			checks = append(checks, ValidationCheck{
				Detail: fmt.Sprintf("Feature %s is included in the release but is not present in the Feature Registry", id),
			})
		case !f.State.AtLeast(feature.StateApproved):
			checks = append(checks, ValidationCheck{
				Detail: fmt.Sprintf("Feature %s is included in the release but its registry state is %q, not Approved or later", id, f.State),
			})
		}
	}
	if len(checks) == 0 {
		return []ValidationCheck{{Name: "Features approved", OK: true}}
	}
	return checks
}

// validateChangeSetFinalized reports every release fix or DevOps change
// still merged-but-Pending a QA build as its own failing check — the
// exact condition FinishRelease itself rejects with
// ErrPendingChangesNotBuilt, surfaced here per item instead of as one
// combined error. itemLabel names one item (e.g. "Release fix", used in
// each per-item Detail sentence); checkName is the already-pluralized
// success-path label (e.g. "Release fixes finalized") — kept separate
// rather than derived from itemLabel, since naive "+s" pluralization
// gets "Release fix" -> "Release fixs" wrong.
func validateChangeSetFinalized(itemLabel, checkName string, cs ChangeSet) []ValidationCheck {
	name := checkName
	if len(cs.Pending) == 0 {
		return []ValidationCheck{{Name: name, OK: true}}
	}
	checks := make([]ValidationCheck, 0, len(cs.Pending))
	for _, id := range cs.Pending {
		checks = append(checks, ValidationCheck{
			Detail: fmt.Sprintf("%s %s has been merged but is not yet included in a QA build; run 'git flow release build'", itemLabel, id),
		})
	}
	return checks
}

// validateQAStatus is a structural-integrity check on the QA build
// history, not an "approval" gate — Git Flow Plus records no such flag
// for a QA build. It confirms the manifest's own build history is
// internally consistent with the live version: at least one build is
// recorded, and the most recent one matches the version's current QA
// iteration and version string. A mismatch here means release.json and
// version.json have drifted apart (e.g. one was hand-edited, or a build
// was interrupted partway through its two-commit save), not merely that
// a QA build hasn't been "approved."
func validateQAStatus(m *Manifest, v *version.Version) ValidationCheck {
	const name = "QA status valid"
	if len(m.History) == 0 {
		return ValidationCheck{Name: name, Detail: "No QA build history recorded for this release"}
	}
	last := m.History[len(m.History)-1]
	if last.Build != v.QA {
		return ValidationCheck{Name: name, Detail: fmt.Sprintf(
			"QA build #%d does not match the current QA iteration (%d) recorded in version.json", last.Build, v.QA)}
	}
	if last.Version != v.String() {
		return ValidationCheck{Name: name, Detail: fmt.Sprintf(
			"QA build #%d is recorded at version %s, but version.json now reports %s", last.Build, last.Version, v.String())}
	}
	return ValidationCheck{Name: name, OK: true}
}

// validateVersion checks version.json's own internal consistency (no
// negative field, QA build counter at least 1 - a release always starts
// at QA build 1 and only ever increments) and that it agrees with the
// manifest's own recorded current version - the same "single source of
// truth" invariant every mutating release operation maintains by writing
// both files together.
func validateVersion(m *Manifest, v *version.Version) ValidationCheck {
	const name = "Version valid"
	if v.Sprint < 0 || v.Release < 0 || v.Fixes < 0 || v.DevOps < 0 || v.QA < 1 {
		return ValidationCheck{Name: name, Detail: fmt.Sprintf(
			"version.json has an invalid field (Sprint=%d Release=%d Fixes=%d DevOps=%d QA=%d); none may be negative and QA must be at least 1",
			v.Sprint, v.Release, v.Fixes, v.DevOps, v.QA)}
	}
	if v.String() != m.CurrentVersion {
		return ValidationCheck{Name: name, Detail: fmt.Sprintf(
			"version.json (%s) does not match the current version recorded in release.json (%s)", v.String(), m.CurrentVersion)}
	}
	return ValidationCheck{Name: name, OK: true}
}

// validateWorkingTreeClean reuses the same working-tree check
// commitMetadata already performs before every metadata commit.
func (s *service) validateWorkingTreeClean(ctx context.Context) ValidationCheck {
	const name = "Working tree clean"
	status, err := s.deps.Git.Status(ctx)
	if err != nil {
		return ValidationCheck{Name: name, Detail: fmt.Sprintf("Could not check working tree status: %v", err)}
	}
	if !status.Clean {
		return ValidationCheck{Name: name, Detail: "Working tree has uncommitted changes; commit or stash them first"}
	}
	return ValidationCheck{Name: name, OK: true}
}

// validateTagAvailable pre-flights the exact tag-existence check
// FinishRelease performs right before tagging main - the same tag name
// ("<VersionTag prefix><release name>", e.g. "v5.2"), computed the same
// way.
func (s *service) validateTagAvailable(ctx context.Context, m *Manifest) ValidationCheck {
	const name = "Release tag available"
	tagName := s.deps.Config.Prefixes.VersionTag + m.Release
	exists, err := s.deps.Git.TagExists(ctx, tagName)
	if err != nil {
		return ValidationCheck{Name: name, Detail: fmt.Sprintf("Could not check tag %q: %v", tagName, err)}
	}
	if exists {
		return ValidationCheck{Name: name, Detail: fmt.Sprintf(
			"Tag %q already exists; this release may already have been finished, or the tag was created manually", tagName)}
	}
	return ValidationCheck{Name: name, OK: true}
}

// validateNoConflictingState checks the manifest's own recorded branch
// against the currently-configured staging branch. Git Flow Plus only
// ever tracks one active release at a time (StartRelease refuses a
// second one outright), so there is no *other* release.json to conflict
// with by construction - this instead catches the manifest and the
// live config disagreeing about which branch this release even lives on
// (e.g. config.json's staging branch was changed after this release
// started).
func validateNoConflictingState(configuredStaging string, m *Manifest) ValidationCheck {
	const name = "No conflicting release state"
	if m.Branch != configuredStaging {
		return ValidationCheck{Name: name, Detail: fmt.Sprintf(
			"Release manifest is recorded against branch %q, but the configured staging branch is %q", m.Branch, configuredStaging)}
	}
	return ValidationCheck{Name: name, OK: true}
}
