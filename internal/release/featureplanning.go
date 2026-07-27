package release

import (
	"context"
	"fmt"

	"github.com/hulhub/git-flow-plus/internal/feature"
)

// Feature-planning operations all check out staging before touching
// features.json: the Feature Registry is a single, permanent record that
// must stay visible to Release Planning regardless of which release cycle
// (if any) is active, so — like release.json and version.json — its
// canonical home is staging rather than develop, even though a feature's
// code is merged into develop first. This mirrors how cli/init.go commits
// config.json onto staging in addition to develop.

// RegisterFeatureMerged records id as merged into develop, creating a new
// Feature Registry entry if this is the first time id has been seen. It
// preserves any existing approval/assignment state, so re-registering an
// already-approved or already-released feature (e.g. after a follow-up fix
// on the same feature branch) doesn't silently reset its progress.
func (s *service) RegisterFeatureMerged(ctx context.Context, id, branch, mergeCommit string) error {
	if _, err := s.checkoutStaging(ctx); err != nil {
		return err
	}

	r, err := s.deps.FeatureLoader.Load(s.deps.RepoPath)
	if err != nil {
		return fmt.Errorf("loading feature registry: %w", err)
	}

	f, _ := r.Find(id)
	f.ID = id
	f.Branch = branch
	f.MergedIntoDevelop = true
	f.MergeCommit = mergeCommit
	r.Upsert(f)

	if err := s.deps.FeatureLoader.Save(s.deps.RepoPath, r); err != nil {
		return fmt.Errorf("saving feature registry: %w", err)
	}
	if err := s.commitMetadata(ctx, fmt.Sprintf("Register feature '%s' as merged into develop", id)); err != nil {
		return err
	}

	s.deps.Logger.Info("feature registered as merged into develop", "id", id, "branch", branch)
	return nil
}

// ApproveFeature marks a feature approved after Unit Testing — the gate
// that makes it eligible for Release Planning (AddFeatureToRelease /
// DeferFeature). There is no separate "mark unit tested" command, so
// approval implies unit testing happened out of band (e.g. in CI).
func (s *service) ApproveFeature(ctx context.Context, id string) error {
	if _, err := s.checkoutStaging(ctx); err != nil {
		return err
	}

	r, err := s.deps.FeatureLoader.Load(s.deps.RepoPath)
	if err != nil {
		return fmt.Errorf("loading feature registry: %w", err)
	}
	f, ok := r.Find(id)
	if !ok {
		return fmt.Errorf("%w: %q", ErrFeatureNotFound, id)
	}
	if !f.MergedIntoDevelop {
		return fmt.Errorf("%w: %q", ErrFeatureNotMergedIntoDevelop, id)
	}
	if f.Approved {
		return fmt.Errorf("%w: %q", ErrFeatureAlreadyApproved, id)
	}

	f.UnitTested = true
	f.Approved = true
	r.Upsert(f)

	if err := s.deps.FeatureLoader.Save(s.deps.RepoPath, r); err != nil {
		return fmt.Errorf("saving feature registry: %w", err)
	}
	if err := s.commitMetadata(ctx, fmt.Sprintf("Approve feature '%s'", id)); err != nil {
		return err
	}

	s.deps.Logger.Info("feature approved", "id", id)
	return nil
}

// AddFeatureToRelease is a Release Planning decision: it assigns an
// approved, unshipped feature to the active release cycle. Git Flow Plus
// never assumes every approved feature belongs to the next release — this
// is always an explicit call. It only updates release.json bookkeeping; the
// feature's code must already be reachable from staging by whatever means
// the team uses to promote it there (Git Flow Plus deliberately performs no
// automated cherry-picking of feature code).
func (s *service) AddFeatureToRelease(ctx context.Context, id string) error {
	if err := s.requireActiveRelease(ctx); err != nil {
		return err
	}

	r, err := s.deps.FeatureLoader.Load(s.deps.RepoPath)
	if err != nil {
		return fmt.Errorf("loading feature registry: %w", err)
	}
	f, ok := r.Find(id)
	if !ok {
		return fmt.Errorf("%w: %q", ErrFeatureNotFound, id)
	}
	if !f.Approved {
		return fmt.Errorf("%w: %q", ErrFeatureNotApproved, id)
	}
	if f.IncludedInRelease {
		return fmt.Errorf("%w: %q already shipped in release %q", ErrFeatureAlreadyAssigned, id, f.Release)
	}

	m, err := s.deps.ManifestLoader.Load(s.deps.RepoPath)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}
	if containsString(m.Features.Included, id) {
		return fmt.Errorf("%w: %q is already included in release %q", ErrFeatureAlreadyAssigned, id, m.Release)
	}

	m.Features.Included = appendUnique(m.Features.Included, id)
	m.Features.Deferred = removeString(m.Features.Deferred, id)
	m.Features.Pending = pendingFeatureIDs(r, m.Features)

	if err := s.deps.ManifestLoader.Save(s.deps.RepoPath, m); err != nil {
		return fmt.Errorf("saving manifest: %w", err)
	}
	if err := s.commitMetadata(ctx, fmt.Sprintf("Add feature '%s' to release %s", id, m.Release)); err != nil {
		return err
	}

	s.deps.Logger.Info("feature added to release", "id", id, "release", m.Release)
	return nil
}

// RemoveFeatureFromRelease undoes AddFeatureToRelease, returning the
// feature to Pending (it remains approved and eligible for a future
// decision).
func (s *service) RemoveFeatureFromRelease(ctx context.Context, id string) error {
	if err := s.requireActiveRelease(ctx); err != nil {
		return err
	}

	m, err := s.deps.ManifestLoader.Load(s.deps.RepoPath)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}
	if !containsString(m.Features.Included, id) {
		return fmt.Errorf("%w: %q", ErrFeatureNotAssignedToCurrentRelease, id)
	}

	r, err := s.deps.FeatureLoader.Load(s.deps.RepoPath)
	if err != nil {
		return fmt.Errorf("loading feature registry: %w", err)
	}

	m.Features.Included = removeString(m.Features.Included, id)
	m.Features.Pending = pendingFeatureIDs(r, m.Features)

	if err := s.deps.ManifestLoader.Save(s.deps.RepoPath, m); err != nil {
		return fmt.Errorf("saving manifest: %w", err)
	}
	if err := s.commitMetadata(ctx, fmt.Sprintf("Remove feature '%s' from release %s", id, m.Release)); err != nil {
		return err
	}

	s.deps.Logger.Info("feature removed from release", "id", id, "release", m.Release)
	return nil
}

// DeferFeature marks an approved feature as explicitly held back for a
// future release, moving it out of Included if it was there.
func (s *service) DeferFeature(ctx context.Context, id string) error {
	if err := s.requireActiveRelease(ctx); err != nil {
		return err
	}

	r, err := s.deps.FeatureLoader.Load(s.deps.RepoPath)
	if err != nil {
		return fmt.Errorf("loading feature registry: %w", err)
	}
	f, ok := r.Find(id)
	if !ok {
		return fmt.Errorf("%w: %q", ErrFeatureNotFound, id)
	}
	if !f.Approved {
		return fmt.Errorf("%w: %q", ErrFeatureNotApproved, id)
	}
	if f.IncludedInRelease {
		return fmt.Errorf("%w: %q already shipped in release %q", ErrFeatureAlreadyAssigned, id, f.Release)
	}

	m, err := s.deps.ManifestLoader.Load(s.deps.RepoPath)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}
	if containsString(m.Features.Deferred, id) {
		return fmt.Errorf("%w: %q is already deferred", ErrFeatureAlreadyAssigned, id)
	}

	m.Features.Deferred = appendUnique(m.Features.Deferred, id)
	m.Features.Included = removeString(m.Features.Included, id)
	m.Features.Pending = pendingFeatureIDs(r, m.Features)

	if err := s.deps.ManifestLoader.Save(s.deps.RepoPath, m); err != nil {
		return fmt.Errorf("saving manifest: %w", err)
	}
	if err := s.commitMetadata(ctx, fmt.Sprintf("Defer feature '%s' from release %s", id, m.Release)); err != nil {
		return err
	}

	s.deps.Logger.Info("feature deferred", "id", id, "release", m.Release)
	return nil
}

// ListApprovedFeatures lists every approved feature in the registry on the
// current branch, regardless of release assignment. Like Status/Manifest/
// Version, it does not force a checkout — the registry is canonically kept
// on staging by the mutating operations above, so callers should generally
// be on staging when they want the full picture.
func (s *service) ListApprovedFeatures(ctx context.Context) ([]feature.Feature, error) {
	r, err := s.deps.FeatureLoader.Load(s.deps.RepoPath)
	if err != nil {
		return nil, fmt.Errorf("loading feature registry: %w", err)
	}
	return r.Approved(), nil
}

// FeatureStatus reports Approved/Included/Deferred/Pending feature IDs. If
// no release is active, Included and Deferred are empty and Pending is
// every approved, unshipped feature.
func (s *service) FeatureStatus(ctx context.Context) (FeatureStatusReport, error) {
	r, err := s.deps.FeatureLoader.Load(s.deps.RepoPath)
	if err != nil {
		return FeatureStatusReport{}, fmt.Errorf("loading feature registry: %w", err)
	}
	approved := r.Approved()
	approvedIDs := make([]string, len(approved))
	for i, f := range approved {
		approvedIDs[i] = f.ID
	}

	if !s.deps.ManifestLoader.Exists(s.deps.RepoPath) {
		return FeatureStatusReport{
			Approved: approvedIDs,
			Pending:  pendingFeatureIDs(r, FeatureSet{}),
		}, nil
	}

	m, err := s.deps.ManifestLoader.Load(s.deps.RepoPath)
	if err != nil {
		return FeatureStatusReport{}, fmt.Errorf("loading manifest: %w", err)
	}

	return FeatureStatusReport{
		Approved: approvedIDs,
		Included: m.Features.Included,
		Deferred: m.Features.Deferred,
		Pending:  pendingFeatureIDs(r, m.Features),
	}, nil
}

// pendingFeatureIDs computes the derived "pending decision" set: every
// approved feature that hasn't shipped in a past release and hasn't been
// explicitly included or deferred in the current one. This is never
// hand-maintained so it can't drift out of sync with the registry.
func pendingFeatureIDs(r *feature.Registry, fs FeatureSet) []string {
	included := toSet(fs.Included)
	deferred := toSet(fs.Deferred)

	pending := []string{}
	for _, f := range r.Approved() {
		if f.IncludedInRelease || included[f.ID] || deferred[f.ID] {
			continue
		}
		pending = append(pending, f.ID)
	}
	return pending
}

func toSet(items []string) map[string]bool {
	set := make(map[string]bool, len(items))
	for _, item := range items {
		set[item] = true
	}
	return set
}

func containsString(items []string, id string) bool {
	for _, item := range items {
		if item == id {
			return true
		}
	}
	return false
}

func appendUnique(items []string, id string) []string {
	if containsString(items, id) {
		return items
	}
	return append(items, id)
}

func removeString(items []string, id string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item != id {
			out = append(out, item)
		}
	}
	return out
}
