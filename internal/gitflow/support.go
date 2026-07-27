package gitflow

import (
	"context"
	"fmt"
)

// SupportStart creates and checks out a new support branch from main.
// Support branches maintain older release lines and, unlike feature and
// hotfix branches, are intentionally long-lived.
func (s *service) SupportStart(ctx context.Context, name string) (BranchResult, error) {
	name, err := validateName(name)
	if err != nil {
		return BranchResult{}, err
	}
	if err := s.ensureInitialized(ctx); err != nil {
		return BranchResult{}, err
	}

	branch := s.cfg.Prefixes.Support + name
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

	s.logger.Info("started support branch", "branch", branch, "from", s.cfg.Branches.Main)
	return BranchResult{
		Branch:  branch,
		Base:    s.cfg.Branches.Main,
		Message: fmt.Sprintf("Switched to a new branch %q, based on %q", branch, s.cfg.Branches.Main),
	}, nil
}

// SupportFinish closes out a support branch. Support branches represent
// long-lived maintenance lines for older releases and are never merged back
// into main or develop — merging old support work forward would silently
// reintroduce outdated code. Finish therefore only validates the branch and
// leaves it in place; it performs no merge or delete.
func (s *service) SupportFinish(ctx context.Context, name string) (BranchResult, error) {
	name, err := validateName(name)
	if err != nil {
		return BranchResult{}, err
	}
	if err := s.ensureInitialized(ctx); err != nil {
		return BranchResult{}, err
	}

	branch := s.cfg.Prefixes.Support + name
	exists, err := s.git.BranchExists(ctx, branch)
	if err != nil {
		return BranchResult{}, wrapf(err, "checking branch %q", branch)
	}
	if !exists {
		return BranchResult{}, fmt.Errorf("%w: %q", ErrBranchMissing, branch)
	}

	s.logger.Info("closed out support branch (left in place; support branches are not merged back)", "branch", branch)
	return BranchResult{
		Branch: branch,
		Base:   s.cfg.Branches.Main,
		Message: fmt.Sprintf(
			"Support branch %q remains active; support branches are long-lived maintenance lines and are not merged back into %q or %q",
			branch, s.cfg.Branches.Main, s.cfg.Branches.Develop,
		),
	}, nil
}
