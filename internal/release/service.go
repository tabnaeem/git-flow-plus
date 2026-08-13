package release

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/tabnaeem/git-flow-plus/internal/config"
	"github.com/tabnaeem/git-flow-plus/internal/feature"
	"github.com/tabnaeem/git-flow-plus/internal/git"
	"github.com/tabnaeem/git-flow-plus/internal/gitflow"
	"github.com/tabnaeem/git-flow-plus/internal/hooks"
	"github.com/tabnaeem/git-flow-plus/internal/version"
)

// Dependencies are the collaborators Service composes: gitflow.Service for
// branch mechanics, git.Client for staging/committing the metadata files
// (and tagging — see tagmessage.go) directly, the manifest/version/feature
// loaders for persistence, and Hooks for CI/CD-agnostic lifecycle
// notifications.
type Dependencies struct {
	GitFlow        gitflow.Service
	Git            git.Client
	RepoPath       string
	Config         *config.Config
	ManifestLoader Loader
	VersionLoader  version.Loader
	FeatureLoader  feature.Loader
	Hooks          hooks.Runner
	Logger         *slog.Logger
}

// Service implements Git Flow Plus release management on top of staging,
// the permanent release branch: starting and finishing release cycles
// (with version/manifest bootstrapping and archiving), recording release
// fixes and DevOps changes as pending, and cutting QA builds that fold
// pending changes into the version. Only Build changes a Version — merging
// a release-fix or DevOps branch never does.
type Service interface {
	// StartRelease starts a release named "Sprint.Release" (e.g. "5.2") on
	// staging: bootstraps version.json (Sprint.Release.0.0.1) and
	// release.json. Returns ErrReleaseAlreadyActive if a previous release's
	// manifest is still present (i.e. FinishRelease hasn't been run).
	StartRelease(ctx context.Context, name string) (StartResult, error)
	// FinishRelease is the Production Release step: archives the manifest,
	// merges staging into main and develop, and tags the exact commit that
	// merged into main with a rich, traceable message. Returns
	// ErrPendingChangesNotBuilt if any release fix or DevOps change is
	// merged but not yet folded into the version by a QA build. On success
	// it triggers the "post-production-tag" lifecycle hook.
	FinishRelease(ctx context.Context, name string) (FinishResult, error)
	// Build is `git flow release build`: the only operation that changes
	// the version. It folds every pending release fix and DevOps change
	// into Fixes/DevOps, advances QA by one, records a history entry, and
	// always creates an annotated QA tag (deployments key off these tags,
	// so tagging is not optional) with a message detailing what's
	// included. On success it triggers the "post-qa-tag" lifecycle hook.
	// Returns ErrNotOnStaging if not run from staging, and
	// ErrNothingToBuild if nothing is pending.
	Build(ctx context.Context) (BuildResult, error)

	// ReleaseFixFinish merges a release-fix branch into staging and records
	// it as pending in the manifest. It does not touch the version.
	ReleaseFixFinish(ctx context.Context, name string) (FixResult, error)
	// DevOpsFinish merges a release-devops branch into staging and records
	// it as pending in the manifest. It does not touch the version.
	DevOpsFinish(ctx context.Context, name string) (FixResult, error)

	// Status reports the release active on staging, if any.
	Status(ctx context.Context) (StatusReport, error)
	// Validate checks whether the release active on staging is ready for
	// FinishRelease: every check FinishRelease itself would enforce
	// (pending release fixes/DevOps changes, the production tag not
	// already existing), plus structural-integrity checks against the
	// manifest, version, and Feature Registry. Read-only — it never
	// mutates any state, and reports ErrNoActiveRelease-equivalent
	// information via ValidationReport.Active rather than an error, so a
	// missing release is a normal "not ready" result, not a failure to
	// even run the check.
	Validate(ctx context.Context) (ValidationReport, error)
	// Manifest returns the active release's manifest.
	Manifest(ctx context.Context) (*Manifest, error)
	// Version returns the active release's version.
	Version(ctx context.Context) (*version.Version, error)

	// RegisterFeatureCreated records a newly started feature branch in the
	// Feature Registry as StateCreated, called by the CLI immediately
	// after `git flow feature start` succeeds. It does not require an
	// active release — the registry persists across release cycles. There
	// is no equivalent "registered as finished" call: a developer never
	// merges their own feature branch, only a Release Manager does, via
	// AddFeatureToRelease.
	RegisterFeatureCreated(ctx context.Context, id, branch string) error
	// ApproveFeature marks a feature as approved and ready for Release
	// Planning. Returns ErrFeatureNotFound if the feature isn't
	// registered, or ErrFeatureAlreadyApproved if it has already reached
	// at least Approved.
	ApproveFeature(ctx context.Context, id string) error
	// AddFeatureToRelease is the Release Manager's Release Planning
	// decision, and performs the real merge: validates the feature branch
	// exists, merges it into staging (without deleting it — it stays
	// alive through the QA cycle), advances the version's feature
	// counter, updates the manifest, and marks the feature
	// StateIncludedInRelease. Requires an active release
	// (ErrNoActiveRelease) and an approved, unassigned feature
	// (ErrFeatureNotFound, ErrFeatureNotApproved, ErrFeatureAlreadyAssigned).
	AddFeatureToRelease(ctx context.Context, id string) error
	// DeferFeature marks an approved feature as held back for a future
	// release. Requires an active release and an approved, unassigned
	// feature (same error semantics as AddFeatureToRelease). Never
	// touches Git — deferral is a planning decision made before any merge.
	DeferFeature(ctx context.Context, id string) error
	// ListApprovedFeatures lists every approved feature in the registry,
	// regardless of release assignment.
	ListApprovedFeatures(ctx context.Context) ([]feature.Feature, error)
	// FeatureStatus reports Approved/Included/Deferred/Pending feature IDs
	// for the active release.
	FeatureStatus(ctx context.Context) (FeatureStatusReport, error)
}

type service struct {
	deps Dependencies
}

// NewService returns a Service backed by deps.
func NewService(deps Dependencies) Service {
	return &service{deps: deps}
}

// checkoutStaging checks out staging, wrapping any failure with context,
// and returns the branch name for convenience. Shared by every operation
// that requires staging to be the current branch before touching
// release.json/features.json.
func (s *service) checkoutStaging(ctx context.Context) (string, error) {
	staging := s.deps.Config.Branches.Staging
	if err := s.deps.Git.Checkout(ctx, staging); err != nil {
		return "", fmt.Errorf("checking out %q: %w", staging, err)
	}
	return staging, nil
}

// requireActiveRelease checks out staging and returns ErrNoActiveRelease
// if release.json isn't present there. Shared by every operation that
// needs an in-progress release cycle to act on (release fixes, DevOps
// changes, feature-planning decisions, and finishing the release itself).
func (s *service) requireActiveRelease(ctx context.Context) error {
	if _, err := s.checkoutStaging(ctx); err != nil {
		return err
	}
	if !s.deps.ManifestLoader.Exists(s.deps.RepoPath) {
		return ErrNoActiveRelease
	}
	return nil
}

// commitMetadata stages and commits the Git Flow Plus metadata directory if
// (and only if) it has uncommitted changes, keeping the working tree clean
// afterward for the next Git Flow operation's safety checks.
func (s *service) commitMetadata(ctx context.Context, message string) error {
	status, err := s.deps.Git.Status(ctx)
	if err != nil {
		return fmt.Errorf("checking working tree status: %w", err)
	}
	if status.Clean {
		return nil
	}
	if err := s.deps.Git.Add(ctx, config.DirName); err != nil {
		return fmt.Errorf("staging %s: %w", config.DirName, err)
	}
	if err := s.deps.Git.Commit(ctx, message, false); err != nil {
		return fmt.Errorf("committing %s: %w", config.DirName, err)
	}
	return nil
}

// releaseManagerIdentity resolves "Name <email>" from git config for the
// tag message's "Release Manager" field, degrading gracefully (never
// failing the release) if either value is unset.
func (s *service) releaseManagerIdentity(ctx context.Context) string {
	name, _ := s.deps.Git.ConfigValue(ctx, "user.name")
	email, _ := s.deps.Git.ConfigValue(ctx, "user.email")

	switch {
	case name != "" && email != "":
		return fmt.Sprintf("%s <%s>", name, email)
	case name != "":
		return name
	case email != "":
		return email
	default:
		return "unknown"
	}
}

// runHook triggers a lifecycle hook by name, logging (but never failing
// the release operation on) a missing runner, a missing script, or a
// script that errors — the tag has already been created by the time hooks
// run, so a broken deployment trigger must not undo or block that.
func (s *service) runHook(ctx context.Context, name string, env map[string]string) {
	if s.deps.Hooks == nil {
		return
	}
	ran, err := s.deps.Hooks.Run(ctx, s.deps.RepoPath, name, env)
	if err != nil {
		s.deps.Logger.Warn("lifecycle hook failed", "hook", name, "error", err)
		return
	}
	if ran {
		s.deps.Logger.Info("lifecycle hook executed", "hook", name)
	}
}
