package gitflow

import (
	"context"
	"fmt"
)

// FeatureStart creates and checks out a new feature branch from develop.
func (s *service) FeatureStart(ctx context.Context, name string) (BranchResult, error) {
	name, err := validateName(name)
	if err != nil {
		return BranchResult{}, err
	}
	if err := s.ensureInitialized(ctx); err != nil {
		return BranchResult{}, err
	}

	branch := s.cfg.Prefixes.Feature + name
	exists, err := s.git.BranchExists(ctx, branch)
	if err != nil {
		return BranchResult{}, wrapf(err, "checking branch %q", branch)
	}
	if exists {
		return BranchResult{}, fmt.Errorf("%w: %q", ErrBranchAlreadyExists, branch)
	}

	if err := s.git.CreateBranch(ctx, branch, s.cfg.Branches.Develop); err != nil {
		return BranchResult{}, wrapf(err, "creating branch %q", branch)
	}

	s.logger.Info("started feature branch", "branch", branch, "from", s.cfg.Branches.Develop)
	return BranchResult{
		Branch:  branch,
		Base:    s.cfg.Branches.Develop,
		Message: fmt.Sprintf("Switched to a new branch %q, based on %q", branch, s.cfg.Branches.Develop),
	}, nil
}

// FeatureFinish merges a feature branch into develop and deletes it.
func (s *service) FeatureFinish(ctx context.Context, name string) (BranchResult, error) {
	name, err := validateName(name)
	if err != nil {
		return BranchResult{}, err
	}
	if err := s.ensureInitialized(ctx); err != nil {
		return BranchResult{}, err
	}

	branch := s.cfg.Prefixes.Feature + name
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

	if err := s.git.Checkout(ctx, s.cfg.Branches.Develop); err != nil {
		return BranchResult{}, wrapf(err, "checking out %q", s.cfg.Branches.Develop)
	}

	message := fmt.Sprintf("Merge feature '%s' into %s", name, s.cfg.Branches.Develop)
	if err := s.git.MergeNoFF(ctx, branch, message); err != nil {
		return BranchResult{}, wrapf(err, "merging %q into %q", branch, s.cfg.Branches.Develop)
	}
	if err := s.git.DeleteBranch(ctx, branch, false); err != nil {
		return BranchResult{}, wrapf(err, "deleting branch %q", branch)
	}

	s.logger.Info("finished feature branch", "branch", branch, "mergedInto", s.cfg.Branches.Develop)
	return BranchResult{
		Branch:  branch,
		Base:    s.cfg.Branches.Develop,
		Message: fmt.Sprintf("Merged %q into %q and deleted it", branch, s.cfg.Branches.Develop),
	}, nil
}
