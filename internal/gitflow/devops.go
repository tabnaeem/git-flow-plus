package gitflow

import (
	"context"
	"fmt"
)

// DevOpsStart creates and checks out a new release-devops branch from
// staging.
func (s *service) DevOpsStart(ctx context.Context, name string) (BranchResult, error) {
	name, err := validateName(name)
	if err != nil {
		return BranchResult{}, err
	}
	if err := s.ensureInitialized(ctx); err != nil {
		return BranchResult{}, err
	}

	branch := s.cfg.Prefixes.DevOps + name
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

	s.logger.Info("started release-devops branch", "branch", branch, "from", s.cfg.Branches.Staging)
	return BranchResult{
		Branch:  branch,
		Base:    s.cfg.Branches.Staging,
		Message: fmt.Sprintf("Switched to a new branch %q, based on %q", branch, s.cfg.Branches.Staging),
	}, nil
}

// DevOpsFinish merges a release-devops branch into staging and deletes it.
// It does not touch version.json or the manifest's included counts —
// internal/release records the merge as pending, and only a QA build
// (`git flow release build`) folds pending merges into the version.
func (s *service) DevOpsFinish(ctx context.Context, name string) (BranchResult, error) {
	name, err := validateName(name)
	if err != nil {
		return BranchResult{}, err
	}
	if err := s.ensureInitialized(ctx); err != nil {
		return BranchResult{}, err
	}

	branch := s.cfg.Prefixes.DevOps + name
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
	message := fmt.Sprintf("Merge release-devops '%s' into %s", name, staging)
	if err := s.git.MergeNoFF(ctx, branch, message); err != nil {
		return BranchResult{}, wrapf(err, "merging %q into %q", branch, staging)
	}
	if err := s.git.DeleteBranch(ctx, branch, false); err != nil {
		return BranchResult{}, wrapf(err, "deleting branch %q", branch)
	}

	s.logger.Info("finished release-devops branch", "branch", branch)
	return BranchResult{
		Branch:  branch,
		Base:    staging,
		Message: fmt.Sprintf("Merged %q into %q and deleted it", branch, staging),
	}, nil
}
