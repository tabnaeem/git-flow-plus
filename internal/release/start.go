package release

import (
	"context"
	"fmt"

	"github.com/tabnaeem/git-flow-plus/internal/version"
)

// StartRelease parses name as "Sprint.Release", checks out staging (Git
// Flow Plus's permanent release branch — no new branch is created), and
// bootstraps version.json (Sprint.Release.0.0.1) and release.json,
// committing them to staging.
func (s *service) StartRelease(ctx context.Context, name string) (StartResult, error) {
	sprint, rel, err := version.ParseReleaseName(name)
	if err != nil {
		return StartResult{}, err
	}
	if err := s.deps.GitFlow.EnsureInitialized(ctx); err != nil {
		return StartResult{}, err
	}

	staging, err := s.checkoutStaging(ctx)
	if err != nil {
		return StartResult{}, err
	}

	if s.deps.ManifestLoader.Exists(s.deps.RepoPath) {
		return StartResult{}, fmt.Errorf("%w; run 'git flow release finish' to close it out first", ErrReleaseAlreadyActive)
	}

	v := version.New(sprint, rel)
	m := New(name, staging, v.String())

	if err := s.deps.VersionLoader.Save(s.deps.RepoPath, v); err != nil {
		return StartResult{}, fmt.Errorf("saving version: %w", err)
	}

	tagName, err := s.finalizeQABuild(ctx, m, v, fmt.Sprintf("Initialize release %s (%s)", name, v.String()))
	if err != nil {
		return StartResult{}, err
	}

	s.runHook(ctx, "post-qa-tag", map[string]string{
		"GITFLOWPLUS_EVENT":    "qa-build",
		"GITFLOWPLUS_RELEASE":  name,
		"GITFLOWPLUS_VERSION":  v.String(),
		"GITFLOWPLUS_TAG":      tagName,
		"GITFLOWPLUS_QA_BUILD": fmt.Sprintf("%d", v.QA),
		"GITFLOWPLUS_BRANCH":   staging,
	})

	s.deps.Logger.Info("started release", "release", name, "branch", staging, "version", v.String(), "tag", tagName)
	return StartResult{Release: name, Branch: staging, Version: v.String(), Tag: tagName}, nil
}
