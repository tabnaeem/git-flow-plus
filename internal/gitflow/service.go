package gitflow

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/hulhub/git-flow-plus/internal/config"
	"github.com/hulhub/git-flow-plus/internal/git"
)

// Service implements the Git Flow branching model against a single
// repository. Every method is safe to call concurrently only insofar as the
// underlying working tree is not shared across goroutines — like Git
// itself, operations mutate repository-wide state (the current branch).
type Service interface {
	// Init ensures a Git repository exists at the configured location with
	// main and develop branches present, creating them if necessary.
	Init(ctx context.Context) (InitResult, error)

	// FeatureStart branches feature/<name> from staging. There is no
	// FeatureFinish — see FeatureMerge.
	FeatureStart(ctx context.Context, name string) (BranchResult, error)
	// FeatureMerge merges feature/<name> into staging without deleting
	// it, called by internal/release when a Release Manager runs `git
	// flow release feature add`.
	FeatureMerge(ctx context.Context, name string) (BranchResult, error)

	HotfixStart(ctx context.Context, name string) (BranchResult, error)
	HotfixFinish(ctx context.Context, name string) (BranchResult, error)

	SupportStart(ctx context.Context, name string) (BranchResult, error)
	SupportFinish(ctx context.Context, name string) (BranchResult, error)

	// ReleaseFinish is the "Production Release" step: merge staging into
	// main. staging is never deleted — it is a permanent branch reused by
	// every release cycle. develop is not part of the release lifecycle
	// (it's only a temporary integration branch for unit testing), so
	// this does not touch it. It does not tag; internal/release creates
	// the production tag itself, referencing the returned
	// BranchResult.Commit.
	ReleaseFinish(ctx context.Context) (BranchResult, error)

	// ReleaseFixStart and ReleaseFixFinish manage a release-fix branch
	// against staging, Git Flow Plus's permanent release branch.
	ReleaseFixStart(ctx context.Context, name string) (BranchResult, error)
	ReleaseFixFinish(ctx context.Context, name string) (BranchResult, error)

	// DevOpsStart and DevOpsFinish manage a release-devops branch against
	// staging, mirroring ReleaseFixStart/Finish.
	DevOpsStart(ctx context.Context, name string) (BranchResult, error)
	DevOpsFinish(ctx context.Context, name string) (BranchResult, error)

	// EnsureInitialized confirms the main, staging, and develop branches
	// exist, returning ErrNotInitialized if any is missing. Exposed so
	// internal/release can check repository readiness before bootstrapping
	// a release without duplicating the branch list here.
	EnsureInitialized(ctx context.Context) error

	// Doctor runs Git-level health checks (binary present, repository
	// valid, main/staging/develop branches present).
	Doctor(ctx context.Context) DoctorReport
}

type service struct {
	git    git.Client
	cfg    *config.Config
	logger *slog.Logger
}

// NewService returns a Service that operates against gitClient using cfg's
// branch and prefix conventions. logger receives structured diagnostics for
// each operation; pass slog.New(discard handler) to silence it.
func NewService(gitClient git.Client, cfg *config.Config, logger *slog.Logger) Service {
	return &service{git: gitClient, cfg: cfg, logger: logger}
}

func validateName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", ErrEmptyName
	}
	return trimmed, nil
}

// EnsureInitialized confirms the main, staging, and develop branches exist,
// returning ErrNotInitialized if any is missing.
func (s *service) EnsureInitialized(ctx context.Context) error {
	for _, branch := range []string{s.cfg.Branches.Main, s.cfg.Branches.Staging, s.cfg.Branches.Develop} {
		exists, err := s.git.BranchExists(ctx, branch)
		if err != nil {
			return fmt.Errorf("checking branch %q: %w", branch, err)
		}
		if !exists {
			return fmt.Errorf("%w (missing branch %q)", ErrNotInitialized, branch)
		}
	}
	return nil
}

// ensureInitialized is the private form used by every other method in this
// package; EnsureInitialized is the exported alias internal/release uses.
func (s *service) ensureInitialized(ctx context.Context) error {
	return s.EnsureInitialized(ctx)
}

// ensureClean confirms the working tree has no uncommitted changes.
func (s *service) ensureClean(ctx context.Context) error {
	status, err := s.git.Status(ctx)
	if err != nil {
		return err
	}
	if !status.Clean {
		return ErrDirtyWorkingTree
	}
	return nil
}
