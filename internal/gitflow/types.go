// Package gitflow implements the Git Flow branching model — init, feature,
// hotfix, support, and the Git-level parts of release — on top of
// internal/git's low-level operations and internal/config's branch/prefix
// conventions. It is deliberately unaware of Git Flow Plus's release
// versioning and manifest tracking (internal/version, internal/release);
// those packages compose on top of this one.
package gitflow

// BranchResult describes the outcome of a start/finish operation on a
// short-lived Git Flow branch.
type BranchResult struct {
	// Branch is the full branch name the operation acted on
	// (e.g. "feature/checkout-redesign").
	Branch string
	// Base is the branch the operation started from or merged into.
	Base string
	// Tag is the tag created by the operation, if any (e.g. hotfix finish).
	Tag string
	// Commit is the resulting commit SHA, populated where a caller needs
	// to reference the exact commit afterward (e.g. ReleaseFinish's merge
	// commit, which internal/release tags itself).
	Commit string
	// Message is a human-readable summary suitable for direct display.
	Message string
}

// InitResult describes the outcome of initializing a repository for Git
// Flow Plus.
type InitResult struct {
	// RepoCreated is true if `git init` was run because no repository
	// existed yet.
	RepoCreated bool
	// MainCreated is true if the main branch had to be created.
	MainCreated bool
	// DevelopCreated is true if the develop branch had to be created.
	DevelopCreated bool
	// StagingCreated is true if the staging branch had to be created.
	StagingCreated bool
	// Main is the configured main branch name.
	Main string
	// Develop is the configured develop branch name.
	Develop string
	// Staging is the configured staging branch name.
	Staging string
}

// DoctorCheck is a single health check result.
type DoctorCheck struct {
	// Section groups related checks for display (e.g. "Environment",
	// "Branch Model") — see `git flow doctor`'s renderer in internal/cli.
	Section string
	Name    string
	OK      bool
	Detail  string
}

// DoctorReport is the full set of health checks for a repository.
type DoctorReport struct {
	Checks []DoctorCheck
}

// Healthy reports whether every check in the report passed.
func (r DoctorReport) Healthy() bool {
	for _, c := range r.Checks {
		if !c.OK {
			return false
		}
	}
	return true
}
