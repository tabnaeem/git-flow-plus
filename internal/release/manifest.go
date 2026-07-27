// Package release manages the Git Flow Plus release manifest
// (.gitflowplus/release.json) and orchestrates release-level operations —
// release planning, starting/finishing a release, merging release fixes
// and DevOps changes, and cutting QA builds — by composing internal/gitflow
// (branch mechanics), internal/version (version numbering), internal/feature
// (the Feature Registry), and internal/git (staging, committing, and
// tagging).
package release

import "time"

// ChangeSet splits a collection of changes (release fixes or DevOps
// changes) into what's already folded into the current version
// ("included") and what's merged but awaiting the next QA build
// ("pending"). A merge only ever appends to Pending — only Build moves
// entries from Pending to Included and advances the version; that
// separation of "merged" from "released" is the whole point: many changes
// can land before a Release Manager decides to cut the next QA build.
type ChangeSet struct {
	Included []string `json:"included"`
	Pending  []string `json:"pending"`
}

// FeatureSet is like ChangeSet but for features, which have a third state:
// a feature can be explicitly held back for a future release ("deferred")
// rather than merely awaiting a decision. Unlike release fixes/DevOps
// changes, Pending here is always derived (approved features minus
// Included minus Deferred) and re-materialized on every write — it is
// never hand-maintained, so it can't drift out of sync with the Feature
// Registry.
type FeatureSet struct {
	Included []string `json:"included"`
	Deferred []string `json:"deferred"`
	Pending  []string `json:"pending"`
}

// Manifest is the persisted record of a single release cycle on the
// staging branch.
type Manifest struct {
	Release        string        `json:"release"`
	Branch         string        `json:"branch"`
	CurrentVersion string        `json:"currentVersion"`
	CurrentQABuild int           `json:"currentQABuild"`
	Features       FeatureSet    `json:"features"`
	ReleaseFixes   ChangeSet     `json:"releaseFixes"`
	DevOps         ChangeSet     `json:"devops"`
	History        []BuildRecord `json:"history"`
}

// BuildRecord is one entry in a release's QA build history: a permanent,
// self-contained record of exactly what a given QA build (or the initial
// release-start build) contained.
type BuildRecord struct {
	Build                int       `json:"build"`
	Version              string    `json:"version"`
	IncludedFeatures     []string  `json:"includedFeatures"`
	IncludedReleaseFixes []string  `json:"includedReleaseFixes"`
	IncludedDevOps       []string  `json:"includedDevOps"`
	Tag                  string    `json:"tag"`
	Commit               string    `json:"commit"`
	BuildDate            time.Time `json:"buildDate"`
}

// New returns a fresh Manifest for a release just started: QA build 1,
// with a single history entry recording the initial version (before any
// features, fixes, or DevOps changes have been included). The initial
// history entry's Tag and Commit are filled in by the caller once known
// (a commit can't reference its own resulting SHA).
func New(release, branch, version string) *Manifest {
	return &Manifest{
		Release:        release,
		Branch:         branch,
		CurrentVersion: version,
		CurrentQABuild: 1,
		Features:       FeatureSet{Included: []string{}, Deferred: []string{}, Pending: []string{}},
		ReleaseFixes:   ChangeSet{Included: []string{}, Pending: []string{}},
		DevOps:         ChangeSet{Included: []string{}, Pending: []string{}},
		History: []BuildRecord{
			{
				Build:                1,
				Version:              version,
				IncludedFeatures:     []string{},
				IncludedReleaseFixes: []string{},
				IncludedDevOps:       []string{},
			},
		},
	}
}
