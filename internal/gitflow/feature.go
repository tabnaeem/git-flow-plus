package gitflow

import (
	"context"
	"fmt"
)

// FeatureStart creates and checks out a new feature branch from staging —
// Git Flow Plus's permanent release branch, not develop. Features are no
// longer developed against develop at all: develop is not part of the
// release lifecycle, it's only a temporary integration branch used for
// unit testing outside Git Flow Plus's view.
//
// There is deliberately no FeatureFinish. A developer never merges their
// own feature branch — they commit, push, and open a pull request on
// whatever Git host is in use. Only a Release Manager decides if and when
// a feature becomes part of a release, via `git flow release feature
// add`, which performs the actual merge (see FeatureMerge).
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

	if err := s.git.CreateBranch(ctx, branch, s.cfg.Branches.Staging); err != nil {
		return BranchResult{}, wrapf(err, "creating branch %q", branch)
	}

	s.logger.Info("started feature branch", "branch", branch, "from", s.cfg.Branches.Staging)
	return BranchResult{
		Branch:  branch,
		Base:    s.cfg.Branches.Staging,
		Message: fmt.Sprintf("Switched to a new branch %q, based on %q", branch, s.cfg.Branches.Staging),
	}, nil
}

// FeatureMerge merges an existing feature branch into staging on behalf
// of the Release Manager (`git flow release feature add`). Unlike
// ReleaseFixFinish/DevOpsFinish, it deliberately does NOT delete the
// branch afterward: a feature branch must stay alive through the entire
// QA cycle so a developer can push follow-up commits if QA reports an
// issue, without starting a new branch. Branches are only deleted later,
// in bulk, by ReleaseFinish, once the release they belong to completes.
func (s *service) FeatureMerge(ctx context.Context, name string) (BranchResult, error) {
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

	staging := s.cfg.Branches.Staging
	if err := s.git.Checkout(ctx, staging); err != nil {
		return BranchResult{}, wrapf(err, "checking out %q", staging)
	}
	message := fmt.Sprintf("Merge feature '%s' into %s", name, staging)
	if err := s.git.MergeNoFF(ctx, branch, message); err != nil {
		return BranchResult{}, wrapf(err, "merging %q into %q", branch, staging)
	}

	s.logger.Info("merged feature branch into staging", "branch", branch)
	return BranchResult{
		Branch:  branch,
		Base:    staging,
		Message: fmt.Sprintf("Merged %q into %q (branch kept alive for the QA cycle)", branch, staging),
	}, nil
}
