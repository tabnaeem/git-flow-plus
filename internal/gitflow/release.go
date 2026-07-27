package gitflow

import (
	"context"
	"fmt"
)

// ReleaseFinish is the "Production Release" step: it merges staging into
// main. Unlike feature/hotfix branches, staging is a permanent branch and
// is never deleted — the next release cycle reuses it. develop is
// deliberately untouched here: it is not part of the release lifecycle,
// only a temporary integration branch used for unit testing, and the
// release lifecycle begins and ends on staging/main.
//
// This method does not create a tag. The production tag must reference the
// exact commit that passed QA, and its message needs release data (which
// features/fixes/DevOps changes shipped) that only internal/release has —
// so internal/release builds and creates that tag itself, using the merge
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

	s.logger.Info("finished release", "commit", commit)
	return BranchResult{
		Branch:  staging,
		Base:    main,
		Commit:  commit,
		Message: fmt.Sprintf("Merged %q into %q", staging, main),
	}, nil
}
