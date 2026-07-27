// Package version implements Git Flow Plus's internal release-numbering
// scheme: Sprint.Release.Fix.DevOps.QAIteration (e.g. "5.2.4.1.3"). This is
// not Semantic Versioning — it exists purely to track, within a sprint/
// release cycle, how many release fixes and DevOps changes are included in
// the current QA build, independent of the Git tag or any package version.
//
// The version only changes when a QA build is cut (internal/release's
// Build operation, driven by `git flow release build`). Merging a
// release-fix or DevOps branch does not, by itself, move the version —
// those merges only become "included" (and reflected in the version) once
// a Release Manager runs a build.
package version

import "fmt"

// Version is a single Sprint.Release.Fix.DevOps.QABuild value.
type Version struct {
	Sprint  int `json:"sprint"`
	Release int `json:"release"`
	Fixes   int `json:"fixes"`
	DevOps  int `json:"devops"`
	QA      int `json:"qa"`
}

// New returns the version a release starts at: Sprint.Release.0.0.1 — the
// first QA build is created automatically.
func New(sprint, release int) *Version {
	return &Version{Sprint: sprint, Release: release, Fixes: 0, DevOps: 0, QA: 1}
}

// String formats v as "Sprint.Release.Fix.DevOps.QABuild".
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d.%d.%d", v.Sprint, v.Release, v.Fixes, v.DevOps, v.QA)
}

// ApplyBuild records a new QA build: newFixes previously-merged release
// fixes and newDevOps previously-merged DevOps changes are folded into the
// running totals, and the QA iteration advances by one. This is the only
// operation that changes a Version — it must be called exactly once per
// `git flow release build`, never per merge.
func (v *Version) ApplyBuild(newFixes, newDevOps int) {
	v.Fixes += newFixes
	v.DevOps += newDevOps
	v.QA++
}
