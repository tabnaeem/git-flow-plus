package release

import (
	"context"
	"fmt"
	"time"

	"github.com/hulhub/git-flow-plus/internal/feature"
)

// Feature-planning operations all check out staging before touching
// features.json: the Feature Registry is a single, permanent record that
// must stay visible to Release Planning regardless of which release cycle
// (if any) is active, so — like release.json and version.json — its
// canonical home is staging. Unlike release.json, features.json is never
// reset between cycles.

// RegisterFeatureCreated records a newly started feature branch in the
// Feature Registry as StateCreated, called by the CLI immediately after
// `git flow feature start` succeeds. There is no equivalent "finish"
// call — a developer never merges their own feature branch; only a
// Release Manager does, via AddFeatureToRelease.
//
// This temporarily checks out staging to record the entry, then checks
// back out to branch — the developer should end up exactly where
// `feature start` would otherwise have left them, on their new feature
// branch, ready to work.
func (s *service) RegisterFeatureCreated(ctx context.Context, id, branch string) error {
	if _, err := s.checkoutStaging(ctx); err != nil {
		return err
	}

	r, err := s.deps.FeatureLoader.Load(s.deps.RepoPath)
	if err != nil {
		return fmt.Errorf("loading feature registry: %w", err)
	}

	if _, ok := r.Find(id); !ok {
		r.Upsert(feature.Feature{ID: id, Branch: branch, State: feature.StateCreated})
	}

	if err := s.deps.FeatureLoader.Save(s.deps.RepoPath, r); err != nil {
		return fmt.Errorf("saving feature registry: %w", err)
	}
	if err := s.commitMetadata(ctx, fmt.Sprintf("Register feature '%s' as created", id)); err != nil {
		return err
	}

	if err := s.deps.Git.Checkout(ctx, branch); err != nil {
		return fmt.Errorf("checking out %q: %w", branch, err)
	}

	s.deps.Logger.Info("feature registered", "id", id, "branch", branch)
	return nil
}

// ApproveFeature marks a feature approved — the gate that makes it
// eligible for Release Planning (AddFeatureToRelease/DeferFeature). There
// is no separate "mark unit tested" command, so approval implies unit
// testing happened out of band (e.g. in CI, or manually).
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
	if f.State.AtLeast(feature.StateApproved) {
		return fmt.Errorf("%w: %q", ErrFeatureAlreadyApproved, id)
	}

	f.State = feature.StateApproved
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

// AddFeatureToRelease is the Release Manager's Release Planning decision,
// and — unlike the earlier bookkeeping-only design — now performs the
// real work of getting a feature into the release:
//
//  1. Validates the feature branch exists (via GitFlow.FeatureMerge).
//  2. Merges feature/<id> into staging (GitFlow.FeatureMerge). The branch
//     is deliberately NOT deleted — it must stay alive through the QA
//     cycle so follow-up commits can land if QA reports an issue.
//  3. Updates the release manifest (Features.Included/Pending).
//  4. Marks the feature StateIncludedInRelease in the registry.
//  5. Advances the version's feature counter (version.Version.Release —
//     see ApplyFeature) by one — only the first time; see below.
//  6. Persists the updated version.json.
//  7. Records the action in the manifest's FeatureHistory.
//
// Requires an active release (ErrNoActiveRelease) and a feature that has
// been approved (ErrFeatureNotFound, ErrFeatureNotApproved). It is safe to
// call again for a feature already StateIncludedInRelease in the *current*
// cycle: this is how a developer's QA follow-up commits (pushed to the
// still-alive branch after the first merge) actually make it into
// staging — Git Flow Plus performs no automatic re-sync, so the Release
// Manager re-runs this command to pull them in. A second call merges again
// (a no-op if there's genuinely nothing new) without double-counting the
// feature in the version's feature counter. ErrFeatureAlreadyAssigned is
// reserved for a feature that has already shipped in a *finished* release
// (StateReleased or StateArchived) — that's the case a repeat call can't
// recover from.
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
	resync := f.State == feature.StateIncludedInRelease
	if f.State.AtLeast(feature.StateReleased) {
		return fmt.Errorf("%w: %q", ErrFeatureAlreadyAssigned, id)
	}
	if !resync && f.State != feature.StateApproved {
		return fmt.Errorf("%w: %q", ErrFeatureNotApproved, id)
	}

	m, err := s.deps.ManifestLoader.Load(s.deps.RepoPath)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	branchResult, err := s.deps.GitFlow.FeatureMerge(ctx, id)
	if err != nil {
		return err
	}
	mergeCommit, err := s.deps.Git.CommitSHA(ctx)
	if err != nil {
		return fmt.Errorf("resolving merge commit: %w", err)
	}

	versionStr := m.CurrentVersion
	if !resync {
		v, err := s.deps.VersionLoader.Load(s.deps.RepoPath)
		if err != nil {
			return fmt.Errorf("loading version: %w", err)
		}
		v.ApplyFeature()
		if err := s.deps.VersionLoader.Save(s.deps.RepoPath, v); err != nil {
			return fmt.Errorf("saving version: %w", err)
		}
		versionStr = v.String()
	}

	f.State = feature.StateIncludedInRelease
	f.MergeCommit = mergeCommit
	r.Upsert(f)
	if err := s.deps.FeatureLoader.Save(s.deps.RepoPath, r); err != nil {
		return fmt.Errorf("saving feature registry: %w", err)
	}

	m.Features.Included = appendUnique(m.Features.Included, id)
	m.Features.Deferred = removeString(m.Features.Deferred, id)
	m.Features.Pending = pendingFeatureIDs(r, m.Features)
	m.CurrentVersion = versionStr
	m.FeatureHistory = append(m.FeatureHistory, FeatureEvent{
		ID:          id,
		Branch:      branchResult.Branch,
		MergeCommit: mergeCommit,
		Version:     versionStr,
		AddedAt:     time.Now().UTC(),
	})
	if err := s.deps.ManifestLoader.Save(s.deps.RepoPath, m); err != nil {
		return fmt.Errorf("saving manifest: %w", err)
	}

	msg := fmt.Sprintf("Add feature '%s' to release %s (version %s)", id, m.Release, versionStr)
	if resync {
		msg = fmt.Sprintf("Merge additional commits for feature '%s' in release %s", id, m.Release)
	}
	if err := s.commitMetadata(ctx, msg); err != nil {
		return err
	}

	s.deps.Logger.Info("feature merged into staging and added to release",
		"id", id, "release", m.Release, "version", versionStr, "mergeCommit", mergeCommit, "resync", resync)
	return nil
}

// DeferFeature marks an approved feature as explicitly held back for a
// future release. Unlike AddFeatureToRelease, this never touches Git —
// deferral is purely a planning decision made before any merge happens.
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
	if f.State.AtLeast(feature.StateIncludedInRelease) {
		return fmt.Errorf("%w: %q", ErrFeatureAlreadyAssigned, id)
	}
	if f.State != feature.StateApproved {
		return fmt.Errorf("%w: %q", ErrFeatureNotApproved, id)
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

// ListApprovedFeatures lists every feature that has reached at least
// StateApproved in the registry on the current branch, regardless of
// release assignment. Like Status/Manifest/Version, it does not force a
// checkout — the registry is canonically kept on staging by the mutating
// operations above, so callers should generally be on staging when they
// want the full picture.
func (s *service) ListApprovedFeatures(ctx context.Context) ([]feature.Feature, error) {
	r, err := s.deps.FeatureLoader.Load(s.deps.RepoPath)
	if err != nil {
		return nil, fmt.Errorf("loading feature registry: %w", err)
	}
	return r.Approved(), nil
}

// FeatureStatus reports Approved/Included/Deferred/Pending feature IDs. If
// no release is active, Included and Deferred are empty and Pending is
// every approved feature that hasn't shipped yet.
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
// approved feature that hasn't yet reached StateIncludedInRelease (in any
// cycle, past or present) and hasn't been explicitly deferred in the
// current one. This is never hand-maintained so it can't drift out of
// sync with the registry.
func pendingFeatureIDs(r *feature.Registry, fs FeatureSet) []string {
	deferred := toSet(fs.Deferred)

	pending := []string{}
	for _, f := range r.Approved() {
		if f.State.AtLeast(feature.StateIncludedInRelease) || deferred[f.ID] {
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
