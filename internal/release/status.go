package release

import (
	"context"
	"fmt"

	"github.com/hulhub/git-flow-plus/internal/version"
)

// Status reports the release active on the current branch (i.e. whichever
// release, if any, has a release.json/version.json present in the working
// tree right now), including which release-fix and DevOps branches are
// still open.
func (s *service) Status(ctx context.Context) (StatusReport, error) {
	if !s.deps.ManifestLoader.Exists(s.deps.RepoPath) {
		return StatusReport{Active: false}, nil
	}

	m, err := s.deps.ManifestLoader.Load(s.deps.RepoPath)
	if err != nil {
		return StatusReport{}, fmt.Errorf("loading manifest: %w", err)
	}
	v, err := s.deps.VersionLoader.Load(s.deps.RepoPath)
	if err != nil {
		return StatusReport{}, fmt.Errorf("loading version: %w", err)
	}

	fixBranches, err := s.deps.Git.ListBranches(ctx, s.deps.Config.Prefixes.ReleaseFix+"*")
	if err != nil {
		return StatusReport{}, fmt.Errorf("listing release-fix branches: %w", err)
	}
	devopsBranches, err := s.deps.Git.ListBranches(ctx, s.deps.Config.Prefixes.DevOps+"*")
	if err != nil {
		return StatusReport{}, fmt.Errorf("listing release-devops branches: %w", err)
	}

	return StatusReport{
		Active:                 true,
		Release:                m.Release,
		Branch:                 m.Branch,
		Version:                v.String(),
		QABuild:                v.QA,
		IncludedReleaseFixes:   m.ReleaseFixes.Included,
		PendingReleaseFixes:    m.ReleaseFixes.Pending,
		IncludedDevOps:         m.DevOps.Included,
		PendingDevOps:          m.DevOps.Pending,
		IncludedFeatures:       m.Features.Included,
		DeferredFeatures:       m.Features.Deferred,
		PendingFeatures:        m.Features.Pending,
		OpenReleaseFixBranches: fixBranches,
		OpenDevOpsBranches:     devopsBranches,
	}, nil
}

// Manifest returns the active release's manifest, or ErrNoActiveRelease if
// the current branch has none.
func (s *service) Manifest(ctx context.Context) (*Manifest, error) {
	if !s.deps.ManifestLoader.Exists(s.deps.RepoPath) {
		return nil, ErrNoActiveRelease
	}
	return s.deps.ManifestLoader.Load(s.deps.RepoPath)
}

// Version returns the active release's version, or ErrNoActiveRelease if
// the current branch has none.
func (s *service) Version(ctx context.Context) (*version.Version, error) {
	if !s.deps.VersionLoader.Exists(s.deps.RepoPath) {
		return nil, ErrNoActiveRelease
	}
	return s.deps.VersionLoader.Load(s.deps.RepoPath)
}
