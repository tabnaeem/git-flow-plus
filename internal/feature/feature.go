// Package feature implements Git Flow Plus's Feature Registry
// (.gitflowplus/features.json): a persistent, cross-release-cycle ledger
// of every feature that has completed development, whether it's been
// unit-tested and approved, and which release (if any) it ultimately
// shipped in.
//
// The registry is deliberately separate from internal/release's manifest
// (release.json): a feature's lifecycle spans many release cycles (it can
// sit "pending" through several planning rounds before a Release Manager
// assigns it to one), while release.json is reset at the start of every
// cycle. Registry entries are never deleted — Release is the permanent
// record of which release a feature shipped in, once it has.
package feature

// Feature is the tracked lifecycle state of a single completed feature.
type Feature struct {
	// ID is the feature's identifier, e.g. "LOGIN". Also used to derive
	// its branch name (feature/<ID>).
	ID string `json:"id"`
	// Branch is the feature branch this was developed on.
	Branch string `json:"branch"`
	// MergedIntoDevelop is true once `git flow feature finish` has merged
	// this feature into develop. Set automatically — there is no manual
	// command for it.
	MergedIntoDevelop bool `json:"mergedIntoDevelop"`
	// MergeCommit is the merge commit SHA recorded when this feature was
	// finished into develop.
	MergeCommit string `json:"mergeCommit"`
	// UnitTested is set together with Approved by `release feature
	// approve` — there is no separate command for it, since Git Flow Plus
	// has no way to observe test results itself; approving is the
	// Release Manager vouching that testing happened.
	UnitTested bool `json:"unitTested"`
	// Approved is set by `release feature approve`.
	Approved bool `json:"approved"`
	// IncludedInRelease is true once `release feature add` has assigned
	// this feature to a release.
	IncludedInRelease bool `json:"includedInRelease"`
	// Release is the name of the release this feature is assigned to (or
	// has shipped in). Empty means unassigned. Once set, it is permanent
	// — a feature ships in at most one release.
	Release string `json:"release"`
}

// Registry is the full set of tracked features.
type Registry struct {
	Features []Feature `json:"features"`
}

// New returns an empty Registry.
func New() *Registry {
	return &Registry{Features: []Feature{}}
}

// Find returns the feature with the given ID, or false if it isn't
// registered.
func (r *Registry) Find(id string) (Feature, bool) {
	for _, f := range r.Features {
		if f.ID == id {
			return f, true
		}
	}
	return Feature{}, false
}

// Upsert inserts f if its ID isn't already present, or replaces the
// existing entry with the same ID.
func (r *Registry) Upsert(f Feature) {
	for i, existing := range r.Features {
		if existing.ID == f.ID {
			r.Features[i] = f
			return
		}
	}
	r.Features = append(r.Features, f)
}

// Approved returns every feature marked Approved, in registry order.
func (r *Registry) Approved() []Feature {
	var out []Feature
	for _, f := range r.Features {
		if f.Approved {
			out = append(out, f)
		}
	}
	return out
}
