package release

import "errors"

var (
	// ErrNoActiveRelease indicates a release operation was requested but
	// staging has no release.json — i.e. there is no in-progress release
	// cycle.
	ErrNoActiveRelease = errors.New("no active release on staging; run 'git flow release start' first")
	// ErrReleaseAlreadyActive indicates StartRelease was called while a
	// previous release cycle's manifest is still present on staging.
	ErrReleaseAlreadyActive = errors.New("a release is already active on staging")
	// ErrPendingChangesNotBuilt indicates FinishRelease was called while
	// release fixes or DevOps changes have been merged but not yet folded
	// into the version by a QA build.
	ErrPendingChangesNotBuilt = errors.New("pending release fixes or DevOps changes have not been included in a QA build")
	// ErrNotOnStaging indicates Build was called from a branch other than
	// staging.
	ErrNotOnStaging = errors.New("must be on the staging branch to create a QA build")
	// ErrNothingToBuild indicates Build was called with no release fixes or
	// DevOps changes pending inclusion.
	ErrNothingToBuild = errors.New("no pending release fixes or DevOps changes to build")

	// ErrFeatureNotFound indicates a feature-planning command referenced a
	// feature ID that isn't in the Feature Registry.
	ErrFeatureNotFound = errors.New("feature not found in the registry")
	// ErrFeatureNotMergedIntoDevelop indicates ApproveFeature was called
	// for a feature that hasn't been merged into develop yet.
	ErrFeatureNotMergedIntoDevelop = errors.New("feature has not been merged into develop yet; run 'git flow feature finish' first")
	// ErrFeatureAlreadyApproved indicates ApproveFeature was called for an
	// already-approved feature.
	ErrFeatureAlreadyApproved = errors.New("feature is already approved")
	// ErrFeatureNotApproved indicates a feature-planning command requires
	// approval that hasn't happened yet.
	ErrFeatureNotApproved = errors.New("feature is not approved")
	// ErrFeatureAlreadyAssigned indicates AddFeatureToRelease or
	// DeferFeature was called for a feature already assigned to (or
	// shipped in) a release.
	ErrFeatureAlreadyAssigned = errors.New("feature is already assigned to a release")
	// ErrFeatureNotAssignedToCurrentRelease indicates
	// RemoveFeatureFromRelease was called for a feature not assigned to
	// the currently active release.
	ErrFeatureNotAssignedToCurrentRelease = errors.New("feature is not assigned to the active release")
)
