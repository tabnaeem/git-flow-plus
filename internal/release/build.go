package release

import (
	"context"
	"fmt"
	"time"

	"github.com/hulhub/git-flow-plus/internal/version"
)

// tagQABuild creates the annotated tag for a QA build at HEAD (the commit
// that was just made for this version.json/release.json update), used
// identically by StartRelease (QA Build 1, the initial version) and Build
// (every subsequent QA build) via finalizeQABuild. Every QA build gets a
// tag — deployments key off these tags, so this is never optional. It
// returns the resolved commit SHA alongside the tag name so the caller can
// backfill the just-tagged history entry (a commit can't contain the SHA
// of the commit it results in).
func (s *service) tagQABuild(ctx context.Context, m *Manifest, v *version.Version) (tagName, commit string, err error) {
	tagName = s.deps.Config.Prefixes.VersionTag + v.String()

	exists, err := s.deps.Git.TagExists(ctx, tagName)
	if err != nil {
		return "", "", fmt.Errorf("checking tag %q: %w", tagName, err)
	}
	if exists {
		return "", "", fmt.Errorf("tag %q already exists", tagName)
	}

	commit, err = s.deps.Git.CommitSHA(ctx)
	if err != nil {
		return "", "", fmt.Errorf("resolving commit: %w", err)
	}

	message := formatTagMessage(tagMessageInput{
		Release:        m.Release,
		Version:        v.String(),
		QABuild:        v.QA,
		Features:       m.Features.Included,
		ReleaseFixes:   m.ReleaseFixes.Included,
		DevOps:         m.DevOps.Included,
		Branch:         s.deps.Config.Branches.Staging,
		Commit:         commit,
		ReleaseManager: s.releaseManagerIdentity(ctx),
		ReleaseDate:    time.Now(),
	})
	if err := s.deps.Git.Tag(ctx, tagName, message); err != nil {
		return "", "", fmt.Errorf("tagging %q: %w", tagName, err)
	}

	return tagName, commit, nil
}

// finalizeQABuild commits the manifest as prepared by the caller (m.History
// must already have its new entry appended, with Tag/Commit left zero),
// tags the resulting commit as a QA build, then backfills that history
// entry's Tag and Commit and commits the correction. This two-commit
// pattern is required because a commit can't record the SHA it will end up
// with; only the second, untagged commit carries the complete record.
// Shared by StartRelease (QA Build 1) and Build (every subsequent build) so
// the tagging/backfill logic exists in exactly one place.
func (s *service) finalizeQABuild(ctx context.Context, m *Manifest, v *version.Version, commitMessage string) (string, error) {
	if err := s.deps.ManifestLoader.Save(s.deps.RepoPath, m); err != nil {
		return "", fmt.Errorf("saving manifest: %w", err)
	}
	if err := s.commitMetadata(ctx, commitMessage); err != nil {
		return "", err
	}

	tagName, commit, err := s.tagQABuild(ctx, m, v)
	if err != nil {
		return "", err
	}

	last := &m.History[len(m.History)-1]
	last.Tag = tagName
	last.Commit = commit

	if err := s.deps.ManifestLoader.Save(s.deps.RepoPath, m); err != nil {
		return "", fmt.Errorf("saving manifest: %w", err)
	}
	if err := s.commitMetadata(ctx, fmt.Sprintf("Record tag %q for QA Build #%d", tagName, v.QA)); err != nil {
		return "", err
	}

	return tagName, nil
}

// Build is `git flow release build`: the Release Manager's QA build step,
// and the only operation that changes the version. It requires the current
// branch to be staging (ErrNotOnStaging otherwise — this is a deliberate
// check, not an auto-checkout: cutting a build should be a conscious
// action taken from staging, not something that silently switches
// branches for you) and requires at least one pending release fix or
// DevOps change (ErrNothingToBuild otherwise).
//
// It folds every pending release fix/DevOps change into the version
// (Fixes += len(pending fixes), DevOps += len(pending devops), QA += 1),
// moves them from pending to included in the manifest, appends a build
// history entry recording the currently included Features alongside the
// newly-included release fixes/DevOps changes, creates the QA tag, and
// triggers the "post-qa-tag" hook. Features are not folded by Build —
// they're assigned to the release directly by AddFeatureToRelease/
// DeferFeature (see featureplanning.go); Build only records a snapshot of
// whichever features are Included at the time.
func (s *service) Build(ctx context.Context) (BuildResult, error) {
	current, err := s.deps.Git.CurrentBranch(ctx)
	if err != nil {
		return BuildResult{}, fmt.Errorf("resolving current branch: %w", err)
	}
	staging := s.deps.Config.Branches.Staging
	if current != staging {
		return BuildResult{}, fmt.Errorf("%w: currently on %q", ErrNotOnStaging, current)
	}

	if !s.deps.ManifestLoader.Exists(s.deps.RepoPath) {
		return BuildResult{}, ErrNoActiveRelease
	}
	m, err := s.deps.ManifestLoader.Load(s.deps.RepoPath)
	if err != nil {
		return BuildResult{}, fmt.Errorf("loading manifest: %w", err)
	}
	v, err := s.deps.VersionLoader.Load(s.deps.RepoPath)
	if err != nil {
		return BuildResult{}, fmt.Errorf("loading version: %w", err)
	}

	newFixes := len(m.ReleaseFixes.Pending)
	newDevOps := len(m.DevOps.Pending)
	if newFixes == 0 && newDevOps == 0 {
		return BuildResult{}, ErrNothingToBuild
	}

	v.ApplyBuild(newFixes, newDevOps)

	m.ReleaseFixes.Included = append(m.ReleaseFixes.Included, m.ReleaseFixes.Pending...)
	m.ReleaseFixes.Pending = []string{}
	m.DevOps.Included = append(m.DevOps.Included, m.DevOps.Pending...)
	m.DevOps.Pending = []string{}
	m.CurrentVersion = v.String()
	m.CurrentQABuild = v.QA
	m.History = append(m.History, BuildRecord{
		Build:                v.QA,
		Version:              v.String(),
		IncludedFeatures:     append([]string{}, m.Features.Included...),
		IncludedReleaseFixes: append([]string{}, m.ReleaseFixes.Included...),
		IncludedDevOps:       append([]string{}, m.DevOps.Included...),
		BuildDate:            time.Now().UTC(),
	})

	if err := s.deps.VersionLoader.Save(s.deps.RepoPath, v); err != nil {
		return BuildResult{}, fmt.Errorf("saving version: %w", err)
	}

	tagName, err := s.finalizeQABuild(ctx, m, v, fmt.Sprintf("QA Build #%d: version %s", v.QA, v.String()))
	if err != nil {
		return BuildResult{}, err
	}

	result := BuildResult{
		Release:   m.Release,
		Version:   v.String(),
		QABuild:   v.QA,
		NewFixes:  newFixes,
		NewDevOps: newDevOps,
		Tag:       tagName,
	}

	s.runHook(ctx, "post-qa-tag", map[string]string{
		"GITFLOWPLUS_EVENT":    "qa-build",
		"GITFLOWPLUS_RELEASE":  m.Release,
		"GITFLOWPLUS_VERSION":  v.String(),
		"GITFLOWPLUS_TAG":      tagName,
		"GITFLOWPLUS_QA_BUILD": fmt.Sprintf("%d", v.QA),
		"GITFLOWPLUS_BRANCH":   staging,
	})

	s.deps.Logger.Info("QA build created",
		"release", m.Release, "version", v.String(), "qaBuild", v.QA,
		"newFixes", newFixes, "newDevOps", newDevOps, "tag", tagName)
	return result, nil
}
