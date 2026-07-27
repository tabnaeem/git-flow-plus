package release

// StartResult describes the outcome of starting a release.
type StartResult struct {
	Release string
	Branch  string
	Version string
	// Tag is the QA Build 1 tag created for the initial version.
	Tag string
}

// FinishResult describes the outcome of finishing a release (the
// Production Release step: merging staging into main and develop).
type FinishResult struct {
	Release string
	Branch  string
	Tag     string
	Version string
}

// FixResult describes the outcome of merging a release fix or DevOps
// change. It carries no version — merges never change the version; only
// Build does.
type FixResult struct {
	Release string
	Branch  string
}

// BuildResult describes the outcome of a QA build (`git flow release
// build`): how many previously-pending release fixes and DevOps changes
// were folded in, and the resulting version/QA iteration.
type BuildResult struct {
	Release   string
	Version   string
	QABuild   int
	NewFixes  int
	NewDevOps int
	// Tag is the QA tag created for this build. Every build is tagged.
	Tag string
}

// StatusReport is the full picture of the release active on staging, per
// `git flow release status`.
type StatusReport struct {
	// Active is false when staging has no release.json, in which case
	// every other field is zero-valued.
	Active                 bool
	Release                string
	Branch                 string
	Version                string
	QABuild                int
	IncludedReleaseFixes   []string
	PendingReleaseFixes    []string
	IncludedDevOps         []string
	PendingDevOps          []string
	IncludedFeatures       []string
	DeferredFeatures       []string
	PendingFeatures        []string
	OpenReleaseFixBranches []string
	OpenDevOpsBranches     []string
}

// FeatureStatusReport is the picture of the Feature Registry relative to
// the active release, per `git flow release feature status`: which
// features are approved (unit-tested and ready for release planning), and
// among those, which are Included in the active release, Deferred to a
// future one, or still Pending a decision.
type FeatureStatusReport struct {
	Approved []string
	Included []string
	Deferred []string
	Pending  []string
}
