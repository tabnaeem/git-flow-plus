package gitflow

import (
	"context"
	"fmt"
)

// HotfixStart creates and checks out a new hotfix branch from main.
func (s *service) HotfixStart(ctx context.Context, name string) (BranchResult, error) {
	name, err := validateName(name)
	if err != nil {
		return BranchResult{}, err
	}
	if err := s.ensureInitialized(ctx); err != nil {
		return BranchResult{}, err
	}

	branch := s.cfg.Prefixes.Hotfix + name
	exists, err := s.git.BranchExists(ctx, branch)
	if err != nil {
		return BranchResult{}, wrapf(err, "checking branch %q", branch)
	}
	if exists {
		return BranchResult{}, fmt.Errorf("%w: %q", ErrBranchAlreadyExists, branch)
	}

	if err := s.git.CreateBranch(ctx, branch, s.cfg.Branches.Main); err != nil {
		return BranchResult{}, wrapf(err, "creating branch %q", branch)
	}

	s.logger.Info("started hotfix branch", "branch", branch, "from", s.cfg.Branches.Main)
	return BranchResult{
		Branch:  branch,
		Base:    s.cfg.Branches.Main,
		Message: fmt.Sprintf("Switched to a new branch %q, based on %q", branch, s.cfg.Branches.Main),
	}, nil
}

// HotfixFinish merges a hotfix branch into main (tagging the result),
// staging, and develop, then deletes it.
func (s *service) HotfixFinish(ctx context.Context, name string) (BranchResult, error) {
	name, err := validateName(name)
	if err != nil {
		return BranchResult{}, err
	}
	if err := s.ensureInitialized(ctx); err != nil {
		return BranchResult{}, err
	}

	branch := s.cfg.Prefixes.Hotfix + name
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

	tag := s.cfg.Prefixes.VersionTag + name
	tagExists, err := s.git.TagExists(ctx, tag)
	if err != nil {
		return BranchResult{}, wrapf(err, "checking tag %q", tag)
	}
	if tagExists {
		return BranchResult{}, fmt.Errorf("tag %q already exists; choose a different hotfix name", tag)
	}

	if err := s.git.Checkout(ctx, s.cfg.Branches.Main); err != nil {
		return BranchResult{}, wrapf(err, "checking out %q", s.cfg.Branches.Main)
	}
	if err := s.git.MergeNoFF(ctx, branch, fmt.Sprintf("Merge hotfix '%s' into %s", name, s.cfg.Branches.Main)); err != nil {
		return BranchResult{}, wrapf(err, "merging %q into %q", branch, s.cfg.Branches.Main)
	}
	if err := s.git.Tag(ctx, tag, fmt.Sprintf("Hotfix %s", name)); err != nil {
		return BranchResult{}, wrapf(err, "tagging %q", tag)
	}

	if err := s.git.Checkout(ctx, s.cfg.Branches.Staging); err != nil {
		return BranchResult{}, wrapf(err, "checking out %q", s.cfg.Branches.Staging)
	}
	if err := s.git.MergeNoFF(ctx, branch, fmt.Sprintf("Merge hotfix '%s' into %s", name, s.cfg.Branches.Staging)); err != nil {
		return BranchResult{}, wrapf(err, "merging %q into %q", branch, s.cfg.Branches.Staging)
	}

	if err := s.git.Checkout(ctx, s.cfg.Branches.Develop); err != nil {
		return BranchResult{}, wrapf(err, "checking out %q", s.cfg.Branches.Develop)
	}
	if err := s.git.MergeNoFF(ctx, branch, fmt.Sprintf("Merge hotfix '%s' into %s", name, s.cfg.Branches.Develop)); err != nil {
		return BranchResult{}, wrapf(err, "merging %q into %q", branch, s.cfg.Branches.Develop)
	}

	if err := s.git.DeleteBranch(ctx, branch, false); err != nil {
		return BranchResult{}, wrapf(err, "deleting branch %q", branch)
	}

	s.logger.Info("finished hotfix branch", "branch", branch, "tag", tag)
	return BranchResult{
		Branch:  branch,
		Base:    s.cfg.Branches.Main,
		Tag:     tag,
		Message: fmt.Sprintf("Merged %q into %q, %q, and %q, tagged %q, and deleted it", branch, s.cfg.Branches.Main, s.cfg.Branches.Staging, s.cfg.Branches.Develop, tag),
	}, nil
}
