package release_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/hulhub/git-flow-plus/internal/feature"
	gitpkg "github.com/hulhub/git-flow-plus/internal/git"
	"github.com/hulhub/git-flow-plus/internal/gitflow"
	"github.com/hulhub/git-flow-plus/internal/release"
	"github.com/hulhub/git-flow-plus/internal/version"
)

var errBoom = errors.New("boom")

func contains(err error, substr string) bool {
	return err != nil && strings.Contains(err.Error(), substr)
}

// --- StartRelease ---

func TestStartReleasePropagatesEnsureInitializedError(t *testing.T) {
	deps, _, gf, _, _ := happyDeps()
	gf.ensureInitialized = func() error { return errBoom }

	_, err := release.NewService(deps).StartRelease(context.Background(), "5.2")
	if !errors.Is(err, errBoom) {
		t.Errorf("StartRelease() error = %v, want it to wrap errBoom", err)
	}
}

func TestStartReleasePropagatesCheckoutError(t *testing.T) {
	deps, git, _, _, _ := happyDeps()
	git.checkout = func(string) error { return errBoom }

	_, err := release.NewService(deps).StartRelease(context.Background(), "5.2")
	if !contains(err, "checking out") {
		t.Errorf("StartRelease() error = %v, want 'checking out'", err)
	}
}

func TestStartReleaseAlreadyActiveErrors(t *testing.T) {
	deps, _, _, ml, _ := happyDeps()
	ml.exists = func() bool { return true }

	_, err := release.NewService(deps).StartRelease(context.Background(), "5.2")
	if !errors.Is(err, release.ErrReleaseAlreadyActive) {
		t.Errorf("StartRelease() error = %v, want ErrReleaseAlreadyActive", err)
	}
}

func TestStartReleasePropagatesVersionSaveError(t *testing.T) {
	deps, _, _, ml, vl := happyDeps()
	ml.exists = func() bool { return false }
	vl.save = func(*version.Version) error { return errBoom }

	_, err := release.NewService(deps).StartRelease(context.Background(), "5.2")
	if !contains(err, "saving version") {
		t.Errorf("StartRelease() error = %v, want 'saving version'", err)
	}
}

func TestStartReleasePropagatesManifestSaveError(t *testing.T) {
	deps, _, _, ml, _ := happyDeps()
	ml.exists = func() bool { return false }
	ml.save = func(*release.Manifest) error { return errBoom }

	_, err := release.NewService(deps).StartRelease(context.Background(), "5.2")
	if !contains(err, "saving manifest") {
		t.Errorf("StartRelease() error = %v, want 'saving manifest'", err)
	}
}

func TestStartReleasePropagatesCommitStatusError(t *testing.T) {
	deps, git, _, ml, _ := happyDeps()
	ml.exists = func() bool { return false }
	git.status = func() (gitpkg.Status, error) { return gitpkg.Status{}, errBoom }

	_, err := release.NewService(deps).StartRelease(context.Background(), "5.2")
	if !contains(err, "checking working tree status") {
		t.Errorf("StartRelease() error = %v, want 'checking working tree status'", err)
	}
}

func TestStartReleaseSkipsCommitWhenClean(t *testing.T) {
	deps, git, _, ml, _ := happyDeps()
	ml.exists = func() bool { return false }
	committed := false
	git.status = func() (gitpkg.Status, error) { return gitpkg.Status{Clean: true}, nil }
	git.commit = func(string) error { committed = true; return nil }

	if _, err := release.NewService(deps).StartRelease(context.Background(), "5.2"); err != nil {
		t.Fatalf("StartRelease() error = %v", err)
	}
	if committed {
		t.Error("commitMetadata committed despite a clean working tree")
	}
}

// --- FinishRelease ---

func TestFinishReleasePropagatesCheckoutError(t *testing.T) {
	deps, git, _, _, _ := happyDeps()
	git.checkout = func(string) error { return errBoom }

	_, err := release.NewService(deps).FinishRelease(context.Background(), "5.2")
	if !contains(err, "checking out") {
		t.Errorf("FinishRelease() error = %v, want 'checking out'", err)
	}
}

func TestFinishReleaseNoActiveReleaseErrors(t *testing.T) {
	deps, _, _, ml, _ := happyDeps()
	ml.exists = func() bool { return false }

	_, err := release.NewService(deps).FinishRelease(context.Background(), "5.2")
	if !errors.Is(err, release.ErrNoActiveRelease) {
		t.Errorf("FinishRelease() error = %v, want ErrNoActiveRelease", err)
	}
}

func TestFinishReleasePropagatesManifestLoadError(t *testing.T) {
	deps, _, _, ml, _ := happyDeps()
	ml.load = func() (*release.Manifest, error) { return nil, errBoom }

	_, err := release.NewService(deps).FinishRelease(context.Background(), "5.2")
	if !contains(err, "loading manifest") {
		t.Errorf("FinishRelease() error = %v, want 'loading manifest'", err)
	}
}

func TestFinishReleaseNameMismatchErrors(t *testing.T) {
	deps, _, _, ml, _ := happyDeps()
	ml.load = func() (*release.Manifest, error) { return release.New("5.2", "staging", "5.2.0.0.1"), nil }

	_, err := release.NewService(deps).FinishRelease(context.Background(), "5.3")
	if err == nil || !contains(err, "5.2") {
		t.Errorf("FinishRelease(\"5.3\") with active release 5.2 error = %v, want a name-mismatch error mentioning 5.2", err)
	}
}

func TestFinishReleasePendingChangesNotBuiltErrors(t *testing.T) {
	deps, _, _, ml, _ := happyDeps()
	ml.load = func() (*release.Manifest, error) {
		m := release.New("5.2", "staging", "5.2.0.0.1")
		m.ReleaseFixes.Pending = []string{"BUG-101"}
		return m, nil
	}

	_, err := release.NewService(deps).FinishRelease(context.Background(), "5.2")
	if !errors.Is(err, release.ErrPendingChangesNotBuilt) {
		t.Errorf("FinishRelease() with pending fixes error = %v, want ErrPendingChangesNotBuilt", err)
	}
}

func TestFinishReleasePropagatesVersionLoadError(t *testing.T) {
	deps, _, _, ml, vl := happyDeps()
	ml.load = func() (*release.Manifest, error) { return release.New("5.2", "staging", "5.2.0.0.1"), nil }
	vl.load = func() (*version.Version, error) { return nil, errBoom }

	_, err := release.NewService(deps).FinishRelease(context.Background(), "5.2")
	if !contains(err, "loading version") {
		t.Errorf("FinishRelease() error = %v, want 'loading version'", err)
	}
}

func TestFinishReleasePropagatesArchiveError(t *testing.T) {
	deps, _, _, ml, _ := happyDeps()
	ml.load = func() (*release.Manifest, error) { return release.New("5.2", "staging", "5.2.0.0.1"), nil }
	ml.saveArchive = func(string, *release.Manifest) error { return errBoom }

	_, err := release.NewService(deps).FinishRelease(context.Background(), "5.2")
	if !contains(err, "archiving manifest") {
		t.Errorf("FinishRelease() error = %v, want 'archiving manifest'", err)
	}
}

func TestFinishReleasePropagatesAddError(t *testing.T) {
	deps, git, _, ml, _ := happyDeps()
	ml.load = func() (*release.Manifest, error) { return release.New("5.2", "staging", "5.2.0.0.1"), nil }
	git.add = func([]string) error { return errBoom }

	_, err := release.NewService(deps).FinishRelease(context.Background(), "5.2")
	if !contains(err, "staging archived manifest") {
		t.Errorf("FinishRelease() error = %v, want 'staging archived manifest'", err)
	}
}

func TestFinishReleasePropagatesRemoveError(t *testing.T) {
	deps, git, _, ml, _ := happyDeps()
	ml.load = func() (*release.Manifest, error) { return release.New("5.2", "staging", "5.2.0.0.1"), nil }
	git.remove = func([]string) error { return errBoom }

	_, err := release.NewService(deps).FinishRelease(context.Background(), "5.2")
	if !contains(err, "removing live release metadata") {
		t.Errorf("FinishRelease() error = %v, want 'removing live release metadata'", err)
	}
}

func TestFinishReleasePropagatesArchiveCommitError(t *testing.T) {
	deps, git, _, ml, _ := happyDeps()
	ml.load = func() (*release.Manifest, error) { return release.New("5.2", "staging", "5.2.0.0.1"), nil }
	git.commit = func(string) error { return errBoom }

	_, err := release.NewService(deps).FinishRelease(context.Background(), "5.2")
	if !contains(err, "committing archived manifest") {
		t.Errorf("FinishRelease() error = %v, want 'committing archived manifest'", err)
	}
}

func TestFinishReleasePropagatesGitFlowFinishError(t *testing.T) {
	deps, _, gf, ml, _ := happyDeps()
	ml.load = func() (*release.Manifest, error) { return release.New("5.2", "staging", "5.2.0.0.1"), nil }
	gf.releaseFinish = func() (gitflow.BranchResult, error) { return gitflow.BranchResult{}, errBoom }

	_, err := release.NewService(deps).FinishRelease(context.Background(), "5.2")
	if !errors.Is(err, errBoom) {
		t.Errorf("FinishRelease() error = %v, want it to wrap errBoom", err)
	}
}

// --- ReleaseFixFinish / DevOpsFinish ---

func TestReleaseFixFinishPropagatesGitFlowError(t *testing.T) {
	deps, _, gf, _, _ := happyDeps()
	gf.releaseFixFinish = func(string) (gitflow.BranchResult, error) { return gitflow.BranchResult{}, errBoom }

	_, err := release.NewService(deps).ReleaseFixFinish(context.Background(), "BUG-101")
	if !errors.Is(err, errBoom) {
		t.Errorf("ReleaseFixFinish() error = %v, want it to wrap errBoom", err)
	}
}

func TestReleaseFixFinishNoActiveReleaseErrors(t *testing.T) {
	deps, _, _, ml, _ := happyDeps()
	ml.exists = func() bool { return false }

	_, err := release.NewService(deps).ReleaseFixFinish(context.Background(), "BUG-101")
	if !errors.Is(err, release.ErrNoActiveRelease) {
		t.Errorf("ReleaseFixFinish() error = %v, want ErrNoActiveRelease", err)
	}
}

func TestReleaseFixFinishPropagatesManifestLoadError(t *testing.T) {
	deps, _, _, ml, _ := happyDeps()
	ml.load = func() (*release.Manifest, error) { return nil, errBoom }

	_, err := release.NewService(deps).ReleaseFixFinish(context.Background(), "BUG-101")
	if !contains(err, "loading manifest") {
		t.Errorf("ReleaseFixFinish() error = %v, want 'loading manifest'", err)
	}
}

func TestReleaseFixFinishRecordsPendingNotIncluded(t *testing.T) {
	deps, _, _, ml, vl := happyDeps()
	var saved *release.Manifest
	ml.save = func(m *release.Manifest) error { saved = m; return nil }
	versionSaved := false
	vl.save = func(*version.Version) error { versionSaved = true; return nil }

	if _, err := release.NewService(deps).ReleaseFixFinish(context.Background(), "BUG-101"); err != nil {
		t.Fatalf("ReleaseFixFinish() error = %v", err)
	}
	if versionSaved {
		t.Error("ReleaseFixFinish() saved version.json; it must never touch the version")
	}
	if saved == nil {
		t.Fatal("ReleaseFixFinish() did not save the manifest")
	}
	if len(saved.ReleaseFixes.Pending) != 1 || saved.ReleaseFixes.Pending[0] != "BUG-101" {
		t.Errorf("ReleaseFixes.Pending = %v, want [\"BUG-101\"]", saved.ReleaseFixes.Pending)
	}
	if len(saved.ReleaseFixes.Included) != 0 {
		t.Errorf("ReleaseFixes.Included = %v, want empty (not included until a QA build)", saved.ReleaseFixes.Included)
	}
}

func TestDevOpsFinishPropagatesGitFlowError(t *testing.T) {
	deps, _, gf, _, _ := happyDeps()
	gf.devOpsFinish = func(string) (gitflow.BranchResult, error) { return gitflow.BranchResult{}, errBoom }

	_, err := release.NewService(deps).DevOpsFinish(context.Background(), "redis-cache")
	if !errors.Is(err, errBoom) {
		t.Errorf("DevOpsFinish() error = %v, want it to wrap errBoom", err)
	}
}

func TestDevOpsFinishRecordsPendingNotIncluded(t *testing.T) {
	deps, _, _, ml, vl := happyDeps()
	var saved *release.Manifest
	ml.save = func(m *release.Manifest) error { saved = m; return nil }
	versionSaved := false
	vl.save = func(*version.Version) error { versionSaved = true; return nil }

	if _, err := release.NewService(deps).DevOpsFinish(context.Background(), "redis-cache"); err != nil {
		t.Fatalf("DevOpsFinish() error = %v", err)
	}
	if versionSaved {
		t.Error("DevOpsFinish() saved version.json; it must never touch the version")
	}
	if len(saved.DevOps.Pending) != 1 || saved.DevOps.Pending[0] != "redis-cache" {
		t.Errorf("DevOps.Pending = %v, want [\"redis-cache\"]", saved.DevOps.Pending)
	}
	if len(saved.DevOps.Included) != 0 {
		t.Errorf("DevOps.Included = %v, want empty (not included until a QA build)", saved.DevOps.Included)
	}
}

// --- Build ---

func TestBuildRequiresStagingBranch(t *testing.T) {
	deps, git, _, _, _ := happyDeps()
	git.currentBranch = func() (string, error) { return "develop", nil }

	_, err := release.NewService(deps).Build(context.Background())
	if !errors.Is(err, release.ErrNotOnStaging) {
		t.Errorf("Build() off staging error = %v, want ErrNotOnStaging", err)
	}
}

func TestBuildPropagatesCurrentBranchError(t *testing.T) {
	deps, git, _, _, _ := happyDeps()
	git.currentBranch = func() (string, error) { return "", errBoom }

	_, err := release.NewService(deps).Build(context.Background())
	if !contains(err, "resolving current branch") {
		t.Errorf("Build() error = %v, want 'resolving current branch'", err)
	}
}

func TestBuildNoActiveReleaseErrors(t *testing.T) {
	deps, _, _, ml, _ := happyDeps()
	ml.exists = func() bool { return false }

	_, err := release.NewService(deps).Build(context.Background())
	if !errors.Is(err, release.ErrNoActiveRelease) {
		t.Errorf("Build() error = %v, want ErrNoActiveRelease", err)
	}
}

func TestBuildNothingPendingErrors(t *testing.T) {
	deps, _, _, ml, _ := happyDeps()
	ml.load = func() (*release.Manifest, error) { return release.New("5.2", "staging", "5.2.0.0.1"), nil }

	_, err := release.NewService(deps).Build(context.Background())
	if !errors.Is(err, release.ErrNothingToBuild) {
		t.Errorf("Build() with nothing pending error = %v, want ErrNothingToBuild", err)
	}
}

func TestBuildFoldsPendingIntoVersionAndHistory(t *testing.T) {
	deps, _, _, ml, vl := happyDeps()
	ml.load = func() (*release.Manifest, error) {
		m := release.New("5.2", "staging", "5.2.0.0.1")
		m.ReleaseFixes.Pending = []string{"BUG-101", "BUG-102"}
		m.DevOps.Pending = []string{"redis-cache"}
		return m, nil
	}
	var savedManifest *release.Manifest
	ml.save = func(m *release.Manifest) error { savedManifest = m; return nil }
	var savedVersion *version.Version
	vl.save = func(v *version.Version) error { savedVersion = v; return nil }

	result, err := release.NewService(deps).Build(context.Background())
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if result.Version != "5.2.2.1.2" {
		t.Errorf("Version = %q, want %q", result.Version, "5.2.2.1.2")
	}
	if result.NewFixes != 2 || result.NewDevOps != 1 {
		t.Errorf("NewFixes=%d NewDevOps=%d, want NewFixes=2 NewDevOps=1", result.NewFixes, result.NewDevOps)
	}
	if result.QABuild != 2 {
		t.Errorf("QABuild = %d, want 2", result.QABuild)
	}
	if savedVersion.Fixes != 2 || savedVersion.DevOps != 1 || savedVersion.QA != 2 {
		t.Errorf("saved version = %+v, want Fixes=2 DevOps=1 QA=2", savedVersion)
	}
	if len(savedManifest.ReleaseFixes.Pending) != 0 || len(savedManifest.DevOps.Pending) != 0 {
		t.Errorf("pending lists not cleared: fixes=%v devops=%v", savedManifest.ReleaseFixes.Pending, savedManifest.DevOps.Pending)
	}
	if len(savedManifest.ReleaseFixes.Included) != 2 || len(savedManifest.DevOps.Included) != 1 {
		t.Errorf("included lists = fixes=%v devops=%v, want 2 fixes and 1 devops", savedManifest.ReleaseFixes.Included, savedManifest.DevOps.Included)
	}
	if len(savedManifest.History) != 2 {
		t.Fatalf("History has %d entries, want 2 (initial + this build)", len(savedManifest.History))
	}
	last := savedManifest.History[1]
	if last.Build != 2 || last.Version != "5.2.2.1.2" || len(last.IncludedReleaseFixes) != 2 || len(last.IncludedDevOps) != 1 {
		t.Errorf("last history entry = %+v, want {Build:2 Version:5.2.2.1.2 IncludedReleaseFixes:len 2 IncludedDevOps:len 1}", last)
	}
}

func TestBuildWithTagCreatesAnnotatedTag(t *testing.T) {
	deps, git, _, ml, _ := happyDeps()
	ml.load = func() (*release.Manifest, error) {
		m := release.New("5.2", "staging", "5.2.0.0.1")
		m.ReleaseFixes.Pending = []string{"BUG-101"}
		return m, nil
	}
	var taggedName string
	git.tag = func(name, message string) error { taggedName = name; return nil }

	result, err := release.NewService(deps).Build(context.Background())
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	want := "v5.2.1.0.2"
	if taggedName != want {
		t.Errorf("tagged name = %q, want %q", taggedName, want)
	}
	if result.Tag != want {
		t.Errorf("result.Tag = %q, want %q", result.Tag, want)
	}
}

func TestBuildTagCollisionErrors(t *testing.T) {
	deps, git, _, ml, _ := happyDeps()
	ml.load = func() (*release.Manifest, error) {
		m := release.New("5.2", "staging", "5.2.0.0.1")
		m.ReleaseFixes.Pending = []string{"BUG-101"}
		return m, nil
	}
	git.tagExists = func(string) (bool, error) { return true, nil }

	_, err := release.NewService(deps).Build(context.Background())
	if err == nil {
		t.Fatal("Build() with a pre-existing tag = nil error, want failure")
	}
}

func TestBuildPropagatesTagError(t *testing.T) {
	deps, git, _, ml, _ := happyDeps()
	ml.load = func() (*release.Manifest, error) {
		m := release.New("5.2", "staging", "5.2.0.0.1")
		m.ReleaseFixes.Pending = []string{"BUG-101"}
		return m, nil
	}
	git.tag = func(string, string) error { return errBoom }

	_, err := release.NewService(deps).Build(context.Background())
	if !contains(err, "tagging") {
		t.Errorf("Build() error = %v, want 'tagging'", err)
	}
}

func TestBuildPropagatesVersionSaveError(t *testing.T) {
	deps, _, _, ml, vl := happyDeps()
	ml.load = func() (*release.Manifest, error) {
		m := release.New("5.2", "staging", "5.2.0.0.1")
		m.ReleaseFixes.Pending = []string{"BUG-101"}
		return m, nil
	}
	vl.save = func(*version.Version) error { return errBoom }

	_, err := release.NewService(deps).Build(context.Background())
	if !contains(err, "saving version") {
		t.Errorf("Build() error = %v, want 'saving version'", err)
	}
}

// --- Status / Manifest / Version ---

func TestStatusInactiveWhenManifestMissing(t *testing.T) {
	deps, _, _, ml, _ := happyDeps()
	ml.exists = func() bool { return false }

	report, err := release.NewService(deps).Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if report.Active {
		t.Error("Status().Active = true when manifest missing, want false")
	}
}

func TestStatusPropagatesManifestLoadError(t *testing.T) {
	deps, _, _, ml, _ := happyDeps()
	ml.load = func() (*release.Manifest, error) { return nil, errBoom }

	_, err := release.NewService(deps).Status(context.Background())
	if !contains(err, "loading manifest") {
		t.Errorf("Status() error = %v, want 'loading manifest'", err)
	}
}

func TestStatusPropagatesVersionLoadError(t *testing.T) {
	deps, _, _, _, vl := happyDeps()
	vl.load = func() (*version.Version, error) { return nil, errBoom }

	_, err := release.NewService(deps).Status(context.Background())
	if !contains(err, "loading version") {
		t.Errorf("Status() error = %v, want 'loading version'", err)
	}
}

func TestStatusPropagatesListReleaseFixBranchesError(t *testing.T) {
	deps, git, _, _, _ := happyDeps()
	git.listBranches = func(pattern string) ([]string, error) {
		if pattern == "release-fix/*" {
			return nil, errBoom
		}
		return nil, nil
	}

	_, err := release.NewService(deps).Status(context.Background())
	if !contains(err, "listing release-fix branches") {
		t.Errorf("Status() error = %v, want 'listing release-fix branches'", err)
	}
}

func TestStatusPropagatesListDevOpsBranchesError(t *testing.T) {
	deps, git, _, _, _ := happyDeps()
	git.listBranches = func(pattern string) ([]string, error) {
		if pattern == "release-devops/*" {
			return nil, errBoom
		}
		return nil, nil
	}

	_, err := release.NewService(deps).Status(context.Background())
	if !contains(err, "listing release-devops branches") {
		t.Errorf("Status() error = %v, want 'listing release-devops branches'", err)
	}
}

func TestManifestErrorsWhenMissing(t *testing.T) {
	deps, _, _, ml, _ := happyDeps()
	ml.exists = func() bool { return false }

	_, err := release.NewService(deps).Manifest(context.Background())
	if !errors.Is(err, release.ErrNoActiveRelease) {
		t.Errorf("Manifest() error = %v, want ErrNoActiveRelease", err)
	}
}

func TestVersionErrorsWhenMissing(t *testing.T) {
	deps, _, _, _, vl := happyDeps()
	vl.exists = func() bool { return false }

	_, err := release.NewService(deps).Version(context.Background())
	if !errors.Is(err, release.ErrNoActiveRelease) {
		t.Errorf("Version() error = %v, want ErrNoActiveRelease", err)
	}
}

// --- RegisterFeatureMerged ---

func TestRegisterFeatureMergedPropagatesCheckoutError(t *testing.T) {
	deps, git, _, _, _ := happyDeps()
	git.checkout = func(string) error { return errBoom }

	err := release.NewService(deps).RegisterFeatureMerged(context.Background(), "LOGIN", "feature/LOGIN", "abc123")
	if !contains(err, "checking out") {
		t.Errorf("RegisterFeatureMerged() error = %v, want 'checking out'", err)
	}
}

func TestRegisterFeatureMergedCreatesNewEntry(t *testing.T) {
	deps, _, _, _, _ := happyDeps()
	fl := &fakeFeatureLoader{}
	var saved *feature.Registry
	fl.save = func(r *feature.Registry) error { saved = r; return nil }
	deps.FeatureLoader = fl

	if err := release.NewService(deps).RegisterFeatureMerged(context.Background(), "LOGIN", "feature/LOGIN", "abc123"); err != nil {
		t.Fatalf("RegisterFeatureMerged() error = %v", err)
	}
	f, ok := saved.Find("LOGIN")
	if !ok {
		t.Fatal("registry saved without LOGIN entry")
	}
	if !f.MergedIntoDevelop || f.Branch != "feature/LOGIN" || f.MergeCommit != "abc123" {
		t.Errorf("saved feature = %+v, want MergedIntoDevelop=true Branch=feature/LOGIN MergeCommit=abc123", f)
	}
}

func TestRegisterFeatureMergedPreservesExistingApproval(t *testing.T) {
	deps, _, _, _, _ := happyDeps()
	fl := &fakeFeatureLoader{
		load: func() (*feature.Registry, error) {
			r := feature.New()
			r.Upsert(feature.Feature{ID: "LOGIN", Approved: true, UnitTested: true})
			return r, nil
		},
	}
	var saved *feature.Registry
	fl.save = func(r *feature.Registry) error { saved = r; return nil }
	deps.FeatureLoader = fl

	if err := release.NewService(deps).RegisterFeatureMerged(context.Background(), "LOGIN", "feature/LOGIN", "def456"); err != nil {
		t.Fatalf("RegisterFeatureMerged() error = %v", err)
	}
	f, _ := saved.Find("LOGIN")
	if !f.Approved {
		t.Error("RegisterFeatureMerged() cleared Approved on re-registration, want it preserved")
	}
}

// --- ApproveFeature ---

func TestApproveFeatureNotFoundErrors(t *testing.T) {
	deps, _, _, _, _ := happyDeps()
	deps.FeatureLoader = &fakeFeatureLoader{}

	err := release.NewService(deps).ApproveFeature(context.Background(), "LOGIN")
	if !errors.Is(err, release.ErrFeatureNotFound) {
		t.Errorf("ApproveFeature() error = %v, want ErrFeatureNotFound", err)
	}
}

func TestApproveFeatureNotMergedErrors(t *testing.T) {
	deps, _, _, _, _ := happyDeps()
	deps.FeatureLoader = &fakeFeatureLoader{
		load: func() (*feature.Registry, error) {
			r := feature.New()
			r.Upsert(feature.Feature{ID: "LOGIN", MergedIntoDevelop: false})
			return r, nil
		},
	}

	err := release.NewService(deps).ApproveFeature(context.Background(), "LOGIN")
	if !errors.Is(err, release.ErrFeatureNotMergedIntoDevelop) {
		t.Errorf("ApproveFeature() error = %v, want ErrFeatureNotMergedIntoDevelop", err)
	}
}

func TestApproveFeatureAlreadyApprovedErrors(t *testing.T) {
	deps, _, _, _, _ := happyDeps()
	deps.FeatureLoader = &fakeFeatureLoader{
		load: func() (*feature.Registry, error) {
			r := feature.New()
			r.Upsert(feature.Feature{ID: "LOGIN", MergedIntoDevelop: true, Approved: true})
			return r, nil
		},
	}

	err := release.NewService(deps).ApproveFeature(context.Background(), "LOGIN")
	if !errors.Is(err, release.ErrFeatureAlreadyApproved) {
		t.Errorf("ApproveFeature() error = %v, want ErrFeatureAlreadyApproved", err)
	}
}

func TestApproveFeatureMarksApprovedAndUnitTested(t *testing.T) {
	deps, _, _, _, _ := happyDeps()
	fl := &fakeFeatureLoader{
		load: func() (*feature.Registry, error) {
			r := feature.New()
			r.Upsert(feature.Feature{ID: "LOGIN", MergedIntoDevelop: true})
			return r, nil
		},
	}
	var saved *feature.Registry
	fl.save = func(r *feature.Registry) error { saved = r; return nil }
	deps.FeatureLoader = fl

	if err := release.NewService(deps).ApproveFeature(context.Background(), "LOGIN"); err != nil {
		t.Fatalf("ApproveFeature() error = %v", err)
	}
	f, _ := saved.Find("LOGIN")
	if !f.Approved || !f.UnitTested {
		t.Errorf("saved feature = %+v, want Approved=true UnitTested=true", f)
	}
}

// --- AddFeatureToRelease ---

func TestAddFeatureToReleaseNoActiveReleaseErrors(t *testing.T) {
	deps, _, _, ml, _ := happyDeps()
	ml.exists = func() bool { return false }

	err := release.NewService(deps).AddFeatureToRelease(context.Background(), "LOGIN")
	if !errors.Is(err, release.ErrNoActiveRelease) {
		t.Errorf("AddFeatureToRelease() error = %v, want ErrNoActiveRelease", err)
	}
}

func TestAddFeatureToReleaseNotApprovedErrors(t *testing.T) {
	deps, _, _, _, _ := happyDeps()
	deps.FeatureLoader = &fakeFeatureLoader{
		load: func() (*feature.Registry, error) {
			r := feature.New()
			r.Upsert(feature.Feature{ID: "LOGIN", Approved: false})
			return r, nil
		},
	}

	err := release.NewService(deps).AddFeatureToRelease(context.Background(), "LOGIN")
	if !errors.Is(err, release.ErrFeatureNotApproved) {
		t.Errorf("AddFeatureToRelease() error = %v, want ErrFeatureNotApproved", err)
	}
}

func TestAddFeatureToReleaseAlreadyShippedErrors(t *testing.T) {
	deps, _, _, _, _ := happyDeps()
	deps.FeatureLoader = &fakeFeatureLoader{
		load: func() (*feature.Registry, error) {
			r := feature.New()
			r.Upsert(feature.Feature{ID: "LOGIN", Approved: true, IncludedInRelease: true, Release: "5.1"})
			return r, nil
		},
	}

	err := release.NewService(deps).AddFeatureToRelease(context.Background(), "LOGIN")
	if !errors.Is(err, release.ErrFeatureAlreadyAssigned) {
		t.Errorf("AddFeatureToRelease() error = %v, want ErrFeatureAlreadyAssigned", err)
	}
}

func TestAddFeatureToReleaseMovesFromDeferredToIncluded(t *testing.T) {
	deps, _, _, ml, _ := happyDeps()
	deps.FeatureLoader = &fakeFeatureLoader{
		load: func() (*feature.Registry, error) {
			r := feature.New()
			r.Upsert(feature.Feature{ID: "LOGIN", Approved: true})
			return r, nil
		},
	}
	ml.load = func() (*release.Manifest, error) {
		m := release.New("5.2", "staging", "5.2.0.0.1")
		m.Features.Deferred = []string{"LOGIN"}
		return m, nil
	}
	var saved *release.Manifest
	ml.save = func(m *release.Manifest) error { saved = m; return nil }

	if err := release.NewService(deps).AddFeatureToRelease(context.Background(), "LOGIN"); err != nil {
		t.Fatalf("AddFeatureToRelease() error = %v", err)
	}
	if len(saved.Features.Included) != 1 || saved.Features.Included[0] != "LOGIN" {
		t.Errorf("Features.Included = %v, want [\"LOGIN\"]", saved.Features.Included)
	}
	if len(saved.Features.Deferred) != 0 {
		t.Errorf("Features.Deferred = %v, want empty", saved.Features.Deferred)
	}
}

// --- RemoveFeatureFromRelease ---

func TestRemoveFeatureFromReleaseNotAssignedErrors(t *testing.T) {
	deps, _, _, ml, _ := happyDeps()
	ml.load = func() (*release.Manifest, error) { return release.New("5.2", "staging", "5.2.0.0.1"), nil }

	err := release.NewService(deps).RemoveFeatureFromRelease(context.Background(), "LOGIN")
	if !errors.Is(err, release.ErrFeatureNotAssignedToCurrentRelease) {
		t.Errorf("RemoveFeatureFromRelease() error = %v, want ErrFeatureNotAssignedToCurrentRelease", err)
	}
}

func TestRemoveFeatureFromReleaseReturnsToPending(t *testing.T) {
	deps, _, _, ml, _ := happyDeps()
	deps.FeatureLoader = &fakeFeatureLoader{
		load: func() (*feature.Registry, error) {
			r := feature.New()
			r.Upsert(feature.Feature{ID: "LOGIN", Approved: true})
			return r, nil
		},
	}
	ml.load = func() (*release.Manifest, error) {
		m := release.New("5.2", "staging", "5.2.0.0.1")
		m.Features.Included = []string{"LOGIN"}
		return m, nil
	}
	var saved *release.Manifest
	ml.save = func(m *release.Manifest) error { saved = m; return nil }

	if err := release.NewService(deps).RemoveFeatureFromRelease(context.Background(), "LOGIN"); err != nil {
		t.Fatalf("RemoveFeatureFromRelease() error = %v", err)
	}
	if len(saved.Features.Included) != 0 {
		t.Errorf("Features.Included = %v, want empty", saved.Features.Included)
	}
	if len(saved.Features.Pending) != 1 || saved.Features.Pending[0] != "LOGIN" {
		t.Errorf("Features.Pending = %v, want [\"LOGIN\"] (derived from approved-minus-included-minus-deferred)", saved.Features.Pending)
	}
}

// --- DeferFeature ---

func TestDeferFeatureNoActiveReleaseErrors(t *testing.T) {
	deps, _, _, ml, _ := happyDeps()
	ml.exists = func() bool { return false }

	err := release.NewService(deps).DeferFeature(context.Background(), "LOGIN")
	if !errors.Is(err, release.ErrNoActiveRelease) {
		t.Errorf("DeferFeature() error = %v, want ErrNoActiveRelease", err)
	}
}

func TestDeferFeatureNotApprovedErrors(t *testing.T) {
	deps, _, _, _, _ := happyDeps()
	deps.FeatureLoader = &fakeFeatureLoader{
		load: func() (*feature.Registry, error) {
			r := feature.New()
			r.Upsert(feature.Feature{ID: "LOGIN", Approved: false})
			return r, nil
		},
	}

	err := release.NewService(deps).DeferFeature(context.Background(), "LOGIN")
	if !errors.Is(err, release.ErrFeatureNotApproved) {
		t.Errorf("DeferFeature() error = %v, want ErrFeatureNotApproved", err)
	}
}

func TestDeferFeatureMovesFromIncludedToDeferred(t *testing.T) {
	deps, _, _, ml, _ := happyDeps()
	deps.FeatureLoader = &fakeFeatureLoader{
		load: func() (*feature.Registry, error) {
			r := feature.New()
			r.Upsert(feature.Feature{ID: "REPORTS", Approved: true})
			return r, nil
		},
	}
	ml.load = func() (*release.Manifest, error) {
		m := release.New("5.2", "staging", "5.2.0.0.1")
		m.Features.Included = []string{"REPORTS"}
		return m, nil
	}
	var saved *release.Manifest
	ml.save = func(m *release.Manifest) error { saved = m; return nil }

	if err := release.NewService(deps).DeferFeature(context.Background(), "REPORTS"); err != nil {
		t.Fatalf("DeferFeature() error = %v", err)
	}
	if len(saved.Features.Deferred) != 1 || saved.Features.Deferred[0] != "REPORTS" {
		t.Errorf("Features.Deferred = %v, want [\"REPORTS\"]", saved.Features.Deferred)
	}
	if len(saved.Features.Included) != 0 {
		t.Errorf("Features.Included = %v, want empty", saved.Features.Included)
	}
}

// --- ListApprovedFeatures / FeatureStatus ---

func TestListApprovedFeaturesFiltersUnapproved(t *testing.T) {
	deps, _, _, _, _ := happyDeps()
	deps.FeatureLoader = &fakeFeatureLoader{
		load: func() (*feature.Registry, error) {
			r := feature.New()
			r.Upsert(feature.Feature{ID: "LOGIN", Approved: true})
			r.Upsert(feature.Feature{ID: "DASHBOARD", Approved: false})
			return r, nil
		},
	}

	got, err := release.NewService(deps).ListApprovedFeatures(context.Background())
	if err != nil {
		t.Fatalf("ListApprovedFeatures() error = %v", err)
	}
	if len(got) != 1 || got[0].ID != "LOGIN" {
		t.Errorf("ListApprovedFeatures() = %v, want only LOGIN", got)
	}
}

func TestFeatureStatusNoActiveReleaseReportsAllPending(t *testing.T) {
	deps, _, _, ml, _ := happyDeps()
	ml.exists = func() bool { return false }
	deps.FeatureLoader = &fakeFeatureLoader{
		load: func() (*feature.Registry, error) {
			r := feature.New()
			r.Upsert(feature.Feature{ID: "LOGIN", Approved: true})
			return r, nil
		},
	}

	report, err := release.NewService(deps).FeatureStatus(context.Background())
	if err != nil {
		t.Fatalf("FeatureStatus() error = %v", err)
	}
	if len(report.Included) != 0 || len(report.Deferred) != 0 {
		t.Errorf("FeatureStatus() with no active release = %+v, want empty Included/Deferred", report)
	}
	if len(report.Pending) != 1 || report.Pending[0] != "LOGIN" {
		t.Errorf("FeatureStatus().Pending = %v, want [\"LOGIN\"]", report.Pending)
	}
}

func TestFeatureStatusReflectsManifestAssignments(t *testing.T) {
	deps, _, _, ml, _ := happyDeps()
	deps.FeatureLoader = &fakeFeatureLoader{
		load: func() (*feature.Registry, error) {
			r := feature.New()
			r.Upsert(feature.Feature{ID: "LOGIN", Approved: true})
			r.Upsert(feature.Feature{ID: "PROFILE", Approved: true})
			r.Upsert(feature.Feature{ID: "REPORTS", Approved: true})
			return r, nil
		},
	}
	ml.load = func() (*release.Manifest, error) {
		m := release.New("5.3", "staging", "5.3.0.0.1")
		m.Features.Included = []string{"LOGIN"}
		m.Features.Deferred = []string{"REPORTS"}
		return m, nil
	}

	report, err := release.NewService(deps).FeatureStatus(context.Background())
	if err != nil {
		t.Fatalf("FeatureStatus() error = %v", err)
	}
	if len(report.Approved) != 3 {
		t.Errorf("Approved = %v, want 3 entries", report.Approved)
	}
	if len(report.Included) != 1 || report.Included[0] != "LOGIN" {
		t.Errorf("Included = %v, want [\"LOGIN\"]", report.Included)
	}
	if len(report.Deferred) != 1 || report.Deferred[0] != "REPORTS" {
		t.Errorf("Deferred = %v, want [\"REPORTS\"]", report.Deferred)
	}
	if len(report.Pending) != 1 || report.Pending[0] != "PROFILE" {
		t.Errorf("Pending = %v, want [\"PROFILE\"]", report.Pending)
	}
}
