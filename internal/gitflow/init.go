package gitflow

import "context"

// Init ensures a Git repository exists with main, staging, and develop
// branches present, mirroring `git flow init` from the original tooling
// with a staging branch added for Git Flow Plus's process (main = production,
// staging = release, develop = development). It only creates or renames
// branches when doing so is unambiguous and safe:
//
//   - A brand-new (or still commit-less) repository gets an initial empty
//     commit and its current branch renamed to the configured main branch.
//   - An existing repository with history is left untouched if its main
//     branch is missing; Init returns an error instead of guessing.
//
// In both cases, missing staging and develop branches are created from
// main, and the working tree is left checked out on develop.
func (s *service) Init(ctx context.Context) (InitResult, error) {
	result := InitResult{Main: s.cfg.Branches.Main, Develop: s.cfg.Branches.Develop, Staging: s.cfg.Branches.Staging}

	if !s.git.IsRepo(ctx) {
		if err := s.git.Init(ctx); err != nil {
			return InitResult{}, wrapf(err, "initializing repository")
		}
		result.RepoCreated = true
		s.logger.Info("initialized new git repository")
	}

	if !s.git.HasCommits(ctx) {
		if err := s.git.Commit(ctx, "Initial commit", true); err != nil {
			return InitResult{}, wrapf(err, "creating initial commit")
		}

		current, err := s.git.CurrentBranch(ctx)
		if err != nil {
			return InitResult{}, wrapf(err, "resolving current branch")
		}
		if current != s.cfg.Branches.Main {
			if err := s.git.RenameCurrentBranch(ctx, s.cfg.Branches.Main); err != nil {
				return InitResult{}, wrapf(err, "renaming %q to %q", current, s.cfg.Branches.Main)
			}
		}
		result.MainCreated = true
		s.logger.Info("created initial commit and main branch", "branch", s.cfg.Branches.Main)
	} else {
		mainExists, err := s.git.BranchExists(ctx, s.cfg.Branches.Main)
		if err != nil {
			return InitResult{}, wrapf(err, "checking branch %q", s.cfg.Branches.Main)
		}
		if !mainExists {
			return InitResult{}, wrapf(nil,
				"main branch %q not found; this repository already has history, so Git Flow Plus will not create or rename branches automatically — create %q manually and re-run init",
				s.cfg.Branches.Main, s.cfg.Branches.Main)
		}
	}

	stagingExists, err := s.git.BranchExists(ctx, s.cfg.Branches.Staging)
	if err != nil {
		return InitResult{}, wrapf(err, "checking branch %q", s.cfg.Branches.Staging)
	}
	if !stagingExists {
		if err := s.git.CreateBranch(ctx, s.cfg.Branches.Staging, s.cfg.Branches.Main); err != nil {
			return InitResult{}, wrapf(err, "creating branch %q", s.cfg.Branches.Staging)
		}
		result.StagingCreated = true
		s.logger.Info("created staging branch", "branch", s.cfg.Branches.Staging, "from", s.cfg.Branches.Main)
	}

	developExists, err := s.git.BranchExists(ctx, s.cfg.Branches.Develop)
	if err != nil {
		return InitResult{}, wrapf(err, "checking branch %q", s.cfg.Branches.Develop)
	}
	if !developExists {
		if err := s.git.CreateBranch(ctx, s.cfg.Branches.Develop, s.cfg.Branches.Main); err != nil {
			return InitResult{}, wrapf(err, "creating branch %q", s.cfg.Branches.Develop)
		}
		result.DevelopCreated = true
		s.logger.Info("created develop branch", "branch", s.cfg.Branches.Develop, "from", s.cfg.Branches.Main)
	} else if err := s.git.Checkout(ctx, s.cfg.Branches.Develop); err != nil {
		return InitResult{}, wrapf(err, "checking out %q", s.cfg.Branches.Develop)
	}

	return result, nil
}
