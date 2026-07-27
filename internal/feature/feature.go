// Package feature implements Git Flow Plus's Feature Registry
// (.gitflowplus/features.json): a persistent, cross-release-cycle ledger
// of every feature branch ever started, its current lifecycle state, and
// which release (if any) it ultimately shipped in.
//
// The registry is deliberately separate from internal/release's manifest
// (release.json): a feature's lifecycle spans many release cycles (it can
// sit "pending" through several planning rounds before a Release Manager
// merges it into one), while release.json is reset at the start of every
// cycle. Registry entries are never deleted — Release is the permanent
// record of which release a feature shipped in, once it has.
package feature

// State is a feature's position in its lifecycle. States are ordered —
// see State.AtLeast — a feature only ever moves forward.
type State string

const (
	// StateCreated is set the moment `git flow feature start` registers
	// the branch. No automatic Git Flow Plus command drives a feature
	// past this into StateInDevelopment/StateAwaitingReview — those exist
	// for completeness (a team may track them manually, or a future
	// command may), since Git Flow Plus has no visibility into individual
	// commits or pull-request review status, both of which happen on
	// whatever Git host is in use, not in this tool.
	StateCreated State = "Created"
	// StateInDevelopment: the developer is actively committing to the
	// feature branch. Not automatically set by any current command.
	StateInDevelopment State = "InDevelopment"
	// StateAwaitingReview: a pull request is open. Not automatically set
	// by any current command.
	StateAwaitingReview State = "AwaitingReview"
	// StateApproved is set by `git flow release feature approve` — the
	// Release Manager vouching that the feature is unit-tested and ready
	// for Release Planning.
	StateApproved State = "Approved"
	// StateIncludedInRelease is set by `git flow release feature add`,
	// the moment the feature branch is actually merged into staging.
	StateIncludedInRelease State = "IncludedInRelease"
	// StateReleased is set by `git flow release finish` for every feature
	// that was included — the release has now shipped.
	StateReleased State = "Released"
	// StateArchived is reserved for a possible future cleanup pass (e.g.
	// archiving very old releases); no current command sets it
	// automatically.
	StateArchived State = "Archived"
)

// rank orders the lifecycle states so AtLeast can compare them without a
// hand-written switch at every call site.
var rank = map[State]int{
	StateCreated:           0,
	StateInDevelopment:     1,
	StateAwaitingReview:    2,
	StateApproved:          3,
	StateIncludedInRelease: 4,
	StateReleased:          5,
	StateArchived:          6,
}

// AtLeast reports whether s has reached at least other in the lifecycle
// (e.g. StateIncludedInRelease.AtLeast(StateApproved) is true — a feature
// that's already merged in was necessarily approved first). An empty or
// otherwise unrecognized State ranks below every named state.
func (s State) AtLeast(other State) bool {
	return rank[s] >= rank[other]
}

// Feature is the tracked lifecycle state of a single feature branch.
type Feature struct {
	// ID is the feature's identifier, e.g. "LOGIN". Also used to derive
	// its branch name (feature/<ID>).
	ID string `json:"id"`
	// Branch is the feature branch this was developed on. The branch is
	// never deleted by Git Flow Plus until the release that included it
	// finishes — see internal/release/finish.go.
	Branch string `json:"branch"`
	// MergeCommit is the merge commit SHA recorded when `release feature
	// add` merged this feature into staging. Empty until then.
	MergeCommit string `json:"mergeCommit"`
	// State is the feature's current lifecycle position — see the State
	// constants.
	State State `json:"state"`
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

// Approved returns every feature that has reached at least StateApproved
// (i.e. is or was eligible for Release Planning), regardless of how much
// further along it's since gotten, in registry order.
func (r *Registry) Approved() []Feature {
	var out []Feature
	for _, f := range r.Features {
		if f.State.AtLeast(StateApproved) {
			out = append(out, f)
		}
	}
	return out
}
