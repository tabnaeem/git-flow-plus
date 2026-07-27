package release

import (
	"context"
	"fmt"
)

// ReleaseFixFinish merges the release-fix branch via gitflow, then records
// name as pending in release.json. It deliberately does not touch
// version.json — only Build (`git flow release build`) folds pending
// merges into the version.
//
// The active-release check happens before gitflow performs the merge/
// delete, so a missing manifest fails the command with no side effects —
// the release-fix branch is left untouched rather than merged and deleted
// without ever being recorded.
func (s *service) ReleaseFixFinish(ctx context.Context, name string) (FixResult, error) {
	if err := s.requireActiveRelease(ctx); err != nil {
		return FixResult{}, err
	}

	branchResult, err := s.deps.GitFlow.ReleaseFixFinish(ctx, name)
	if err != nil {
		return FixResult{}, err
	}

	m, err := s.deps.ManifestLoader.Load(s.deps.RepoPath)
	if err != nil {
		return FixResult{}, fmt.Errorf("loading manifest: %w", err)
	}
	m.ReleaseFixes.Pending = append(m.ReleaseFixes.Pending, name)

	if err := s.deps.ManifestLoader.Save(s.deps.RepoPath, m); err != nil {
		return FixResult{}, fmt.Errorf("saving manifest: %w", err)
	}
	if err := s.commitMetadata(ctx, fmt.Sprintf("Record release fix '%s' as pending", name)); err != nil {
		return FixResult{}, err
	}

	s.deps.Logger.Info("release fix merged (pending next QA build)", "name", name, "release", m.Release)
	return FixResult{Release: m.Release, Branch: branchResult.Branch}, nil
}

// DevOpsFinish merges the release-devops branch via gitflow, then records
// name as pending in release.json. Like ReleaseFixFinish, it does not
// touch version.json, and checks for an active release before letting
// gitflow merge/delete the branch.
func (s *service) DevOpsFinish(ctx context.Context, name string) (FixResult, error) {
	if err := s.requireActiveRelease(ctx); err != nil {
		return FixResult{}, err
	}

	branchResult, err := s.deps.GitFlow.DevOpsFinish(ctx, name)
	if err != nil {
		return FixResult{}, err
	}

	m, err := s.deps.ManifestLoader.Load(s.deps.RepoPath)
	if err != nil {
		return FixResult{}, fmt.Errorf("loading manifest: %w", err)
	}
	m.DevOps.Pending = append(m.DevOps.Pending, name)

	if err := s.deps.ManifestLoader.Save(s.deps.RepoPath, m); err != nil {
		return FixResult{}, fmt.Errorf("saving manifest: %w", err)
	}
	if err := s.commitMetadata(ctx, fmt.Sprintf("Record DevOps change '%s' as pending", name)); err != nil {
		return FixResult{}, err
	}

	s.deps.Logger.Info("devops change merged (pending next QA build)", "name", name, "release", m.Release)
	return FixResult{Release: m.Release, Branch: branchResult.Branch}, nil
}
