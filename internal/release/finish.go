package release

import (
	"context"
	"fmt"
	"time"

	"github.com/hulhub/git-flow-plus/internal/feature"
)

// FinishRelease is the Production Release step. It requires every merged
// release fix and DevOps change to already be included in a QA build
// (ErrPendingChangesNotBuilt otherwise) — nothing should ship without ever
// having moved the version. It then archives the manifest (a permanent
// snapshot under .gitflowplus/archive/), removes the live release.json/
// version.json so they don't linger as stale "current release" files
// (resetting the counters for the next release cycle), merges staging into
// main and main into develop, and tags the exact commit that merged into
// main — the commit that carries exactly the code that passed QA — with a
// message detailing what shipped.
//
// The production tag is named after the release itself (e.g. "v5.2"), not
// the full QA version string: that full-version tag (e.g. "v5.2.4.0.2")
// already exists on staging from the last QA build, and Git tag names must
// be globally unique, so the same name can't be reused for the different
// commit on main. "v5.2" marks the one production release of 5.2; its
// message still records the exact version and commit that shipped.
//
// Unlike the old ephemeral release/<name> branch, staging is never
// deleted. On success this triggers the "post-production-tag" lifecycle
// hook.
func (s *service) FinishRelease(ctx context.Context, name string) (FinishResult, error) {
	if err := s.requireActiveRelease(ctx); err != nil {
		return FinishResult{}, err
	}

	m, err := s.deps.ManifestLoader.Load(s.deps.RepoPath)
	if err != nil {
		return FinishResult{}, fmt.Errorf("loading manifest: %w", err)
	}
	if m.Release != name {
		return FinishResult{}, fmt.Errorf("active release on staging is %q, not %q", m.Release, name)
	}
	if len(m.ReleaseFixes.Pending) > 0 || len(m.DevOps.Pending) > 0 {
		return FinishResult{}, fmt.Errorf("%w: %d release fix(es) and %d devops change(s) pending; run 'git flow release build' first",
			ErrPendingChangesNotBuilt, len(m.ReleaseFixes.Pending), len(m.DevOps.Pending))
	}

	v, err := s.deps.VersionLoader.Load(s.deps.RepoPath)
	if err != nil {
		return FinishResult{}, fmt.Errorf("loading version: %w", err)
	}

	tagName := s.deps.Config.Prefixes.VersionTag + name
	tagExists, err := s.deps.Git.TagExists(ctx, tagName)
	if err != nil {
		return FinishResult{}, fmt.Errorf("checking tag %q: %w", tagName, err)
	}
	if tagExists {
		return FinishResult{}, fmt.Errorf("tag %q already exists", tagName)
	}

	if len(m.Features.Included) > 0 {
		r, err := s.deps.FeatureLoader.Load(s.deps.RepoPath)
		if err != nil {
			return FinishResult{}, fmt.Errorf("loading feature registry: %w", err)
		}
		for _, id := range m.Features.Included {
			f, ok := r.Find(id)
			if !ok {
				continue
			}
			f.IncludedInRelease = true
			f.Release = name
			r.Upsert(f)
		}
		if err := s.deps.FeatureLoader.Save(s.deps.RepoPath, r); err != nil {
			return FinishResult{}, fmt.Errorf("saving feature registry: %w", err)
		}
		if err := s.deps.Git.Add(ctx, feature.RelPath()); err != nil {
			return FinishResult{}, fmt.Errorf("staging feature registry: %w", err)
		}
	}

	if err := s.deps.ManifestLoader.SaveArchive(s.deps.RepoPath, name, m); err != nil {
		return FinishResult{}, fmt.Errorf("archiving manifest: %w", err)
	}
	if err := s.deps.Git.Add(ctx, archiveRelPath(name)); err != nil {
		return FinishResult{}, fmt.Errorf("staging archived manifest: %w", err)
	}
	if err := s.deps.Git.Remove(ctx, releaseRelPath(), versionRelPath()); err != nil {
		return FinishResult{}, fmt.Errorf("removing live release metadata: %w", err)
	}
	if err := s.deps.Git.Commit(ctx, fmt.Sprintf("Archive release %s manifest", name), false); err != nil {
		return FinishResult{}, fmt.Errorf("committing archived manifest: %w", err)
	}

	branchResult, err := s.deps.GitFlow.ReleaseFinish(ctx)
	if err != nil {
		return FinishResult{}, err
	}

	message := formatTagMessage(tagMessageInput{
		Release:        name,
		Version:        v.String(),
		Features:       m.Features.Included,
		ReleaseFixes:   m.ReleaseFixes.Included,
		DevOps:         m.DevOps.Included,
		Branch:         s.deps.Config.Branches.Main,
		Commit:         branchResult.Commit,
		ReleaseManager: s.releaseManagerIdentity(ctx),
		ReleaseDate:    time.Now(),
	})
	if err := s.deps.Git.TagCommit(ctx, tagName, branchResult.Commit, message); err != nil {
		return FinishResult{}, fmt.Errorf("tagging %q: %w", tagName, err)
	}

	s.runHook(ctx, "post-production-tag", map[string]string{
		"GITFLOWPLUS_EVENT":   "production-release",
		"GITFLOWPLUS_RELEASE": name,
		"GITFLOWPLUS_VERSION": v.String(),
		"GITFLOWPLUS_TAG":     tagName,
		"GITFLOWPLUS_COMMIT":  branchResult.Commit,
		"GITFLOWPLUS_BRANCH":  s.deps.Config.Branches.Main,
	})

	s.deps.Logger.Info("finished release", "release", name, "version", v.String(), "tag", tagName)
	return FinishResult{
		Release: name,
		Branch:  branchResult.Branch,
		Tag:     tagName,
		Version: v.String(),
	}, nil
}
