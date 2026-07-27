package gitflow

import (
	"context"
	"fmt"
)

// ReleaseFixStart creates and checks out a new release-fix branch from
// staging.
func (s *service) ReleaseFixStart(ctx context.Context, name string) (BranchResult, error) {
	name, err := validateName(name)
	if err != nil {
		return BranchResult{}, err
	}
	if err := s.ensureInitialized(ctx); err != nil {
		return BranchResult{}, err
	}

	branch := s.cfg.Prefixes.ReleaseFix + name
	exists, err := s.git.BranchExists(ctx, branch)
	if err != nil {
		return BranchResult{}, wrapf(err, "checking branch %q", branch)
	}
	if exists {
		return BranchResult{}, fmt.Errorf("%w: %q", ErrBranchAlreadyExists, branch)
	}

	if err := s.git.CreateBranch(ctx, branch, s.cfg.Branches.Staging); err != nil {
		return BranchResult{}, wrapf(err, "creating branch %q", branch)
	}

	s.logger.Info("started release-fix branch", "branch", branch, "from", s.cfg.Branches.Staging)
	return BranchResult{
		Branch:  branch,
		Base:    s.cfg.Branches.Staging,
		Message: fmt.Sprintf("Switched to a new branch %q, based on %q", branch, s.cfg.Branches.Staging),
	}, nil
}

// ReleaseFixFinish merges a release-fix branch into staging. The branch
// is deliberately NOT deleted here — like feature branches, it stays
// alive through the rest of the QA cycle so follow-up commits can land on
// it if needed, and is only deleted in bulk by ReleaseFinish once the
// release completes. It does not touch version.json or the manifest's
// included counts — internal/release records the merge as pending, and
// only a QA build (`git flow release build`) folds pending merges into
// the version.
func (s *service) ReleaseFixFinish(ctx context.Context, name string) (BranchResult, error) {
	name, err := validateName(name)
	if err != nil {
		return BranchResult{}, err
	}
	if err := s.ensureInitialized(ctx); err != nil {
		return BranchResult{}, err
	}

	branch := s.cfg.Prefixes.ReleaseFix + name
	exists, err := s.git.BranchExists(ctx, branch)
	if err != nil {
		return BranchResult{}, wrapf(err, "checking branch %q", branch)
	}
	if !exists {
		return BranchResult{}, fmt.Errorf("%w: %q", ErrBranchMissing, branch)
	}
	if err := s.ensureClean(ctx); err != nil {
		return BranchResult{}, err
	}

	staging := s.cfg.Branches.Staging
	if err := s.git.Checkout(ctx, staging); err != nil {
		return BranchResult{}, wrapf(err, "checking out %q", staging)
	}
	message := fmt.Sprintf("Merge release-fix '%s' into %s", name, staging)
	if err := s.git.MergeNoFF(ctx, branch, message); err != nil {
		return BranchResult{}, wrapf(err, "merging %q into %q", branch, staging)
	}

	s.logger.Info("merged release-fix branch into staging", "branch", branch)
	return BranchResult{
		Branch:  branch,
		Base:    staging,
		Message: fmt.Sprintf("Merged %q into %q (branch kept alive for the QA cycle)", branch, staging),
	}, nil
}
