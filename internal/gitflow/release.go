package gitflow

import (
	"context"
	"fmt"
)

// ReleaseFinish is the "Production Release" step: it merges staging into
// main, then merges main back into develop so both stay in sync from a
// single source of truth. Unlike feature/hotfix branches, staging is a
// permanent branch and is never deleted — the next release cycle reuses
// it.
//
// This method does not create a tag. The production tag must reference the
// exact commit that passed QA, and its message needs release data (which
// fixes/DevOps changes shipped) that only internal/release has — so
// internal/release builds and creates that tag itself, using the merge
// commit SHA returned here via BranchResult.Commit.
func (s *service) ReleaseFinish(ctx context.Context) (BranchResult, error) {
	if err := s.ensureInitialized(ctx); err != nil {
		return BranchResult{}, err
	}
	if err := s.ensureClean(ctx); err != nil {
		return BranchResult{}, err
	}

	staging := s.cfg.Branches.Staging
	main := s.cfg.Branches.Main
	develop := s.cfg.Branches.Develop

	if err := s.git.Checkout(ctx, main); err != nil {
		return BranchResult{}, wrapf(err, "checking out %q", main)
	}
	if err := s.git.MergeNoFF(ctx, staging, fmt.Sprintf("Merge %s into %s", staging, main)); err != nil {
		return BranchResult{}, wrapf(err, "merging %q into %q", staging, main)
	}

	commit, err := s.git.CommitSHA(ctx)
	if err != nil {
		return BranchResult{}, wrapf(err, "resolving commit")
	}

	if err := s.git.Checkout(ctx, develop); err != nil {
		return BranchResult{}, wrapf(err, "checking out %q", develop)
	}
	if err := s.git.MergeNoFF(ctx, main, fmt.Sprintf("Merge %s into %s", main, develop)); err != nil {
		return BranchResult{}, wrapf(err, "merging %q into %q", main, develop)
	}

	s.logger.Info("finished release", "commit", commit)
	return BranchResult{
		Branch:  staging,
		Base:    main,
		Commit:  commit,
		Message: fmt.Sprintf("Merged %q into %q and %q into %q", staging, main, main, develop),
	}, nil
}
