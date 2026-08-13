package release_test

import (
	"context"
	"strings"
	"testing"

	gitpkg "github.com/tabnaeem/git-flow-plus/internal/git"
	"github.com/tabnaeem/git-flow-plus/internal/release"
	"github.com/tabnaeem/git-flow-plus/internal/version"
)

// cleanStatus is happyDeps' one surprising default: fakeGit.Status()
// defaults to Clean: false (used elsewhere to exercise "has uncommitted
// changes" paths), so every Validate() test that isn't specifically about
// a dirty working tree needs to override it explicitly.
func cleanStatus() (gitpkg.Status, error) {
	return gitpkg.Status{Clean: true}, nil
}

func mustHaveFailingCheck(t *testing.T, checks []release.ValidationCheck, substr string) {
	t.Helper()
	for _, c := range checks {
		if !c.OK && strings.Contains(c.Detail, substr) {
			return
		}
	}
	t.Errorf("no failing check contains %q; checks = %+v", substr, checks)
}

// --- 1. Valid release ---

func TestValidateValidReleasePasses(t *testing.T) {
	deps, git, _, _, _ := happyDeps()
	git.status = cleanStatus

	svc := release.NewService(deps)
	report, err := svc.Validate(context.Background())
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !report.Active {
		t.Fatal("Validate().Active = false, want true")
	}
	if !report.Ready {
		t.Fatalf("Validate().Ready = false, want true; checks = %+v", report.Checks)
	}
	for _, c := range report.Checks {
		if !c.OK {
			t.Errorf("check %q failed unexpectedly: %s", c.Name, c.Detail)
		}
	}
}

// --- 2. Missing release ---

func TestValidateNoActiveReleaseFails(t *testing.T) {
	deps, _, _, ml, _ := happyDeps()
	ml.exists = func() bool { return false }

	svc := release.NewService(deps)
	report, err := svc.Validate(context.Background())
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if report.Active {
		t.Error("Validate().Active = true, want false")
	}
	if report.Ready {
		t.Error("Validate().Ready = true, want false")
	}
	if len(report.Checks) != 1 || report.Checks[0].OK {
		t.Errorf("Checks = %+v, want exactly one failing check", report.Checks)
	}
}

// --- 3. Pending feature ---

func TestValidatePendingFeatureFails(t *testing.T) {
	deps, git, _, ml, _ := happyDeps()
	git.status = cleanStatus
	m := release.New("5.2", "staging", "5.2.0.0.1")
	m.Features.Pending = []string{"REPORT"}
	ml.load = func() (*release.Manifest, error) { return m, nil }

	svc := release.NewService(deps)
	report, err := svc.Validate(context.Background())
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if report.Ready {
		t.Fatal("Validate().Ready = true, want false (REPORT is pending)")
	}
	mustHaveFailingCheck(t, report.Checks, "REPORT")
}

// --- 4. Unapproved feature (registry/manifest drift) ---

func TestValidateFeatureIncludedButNotApprovedInRegistryFails(t *testing.T) {
	deps, git, _, ml, _ := happyDeps()
	git.status = cleanStatus
	m := release.New("5.2", "staging", "5.2.0.0.1")
	m.Features.Included = []string{"LOGIN"}
	ml.load = func() (*release.Manifest, error) { return m, nil }
	// Registry left at its default (empty) - LOGIN is Included in the
	// manifest but not present in the registry at all, let alone Approved.

	svc := release.NewService(deps)
	report, err := svc.Validate(context.Background())
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if report.Ready {
		t.Fatal("Validate().Ready = true, want false (LOGIN is Included but not in the registry)")
	}
	mustHaveFailingCheck(t, report.Checks, "LOGIN")
}

// --- 5. Pending release fix ---

func TestValidatePendingReleaseFixFails(t *testing.T) {
	deps, git, _, ml, _ := happyDeps()
	git.status = cleanStatus
	m := release.New("5.2", "staging", "5.2.0.0.1")
	m.ReleaseFixes.Pending = []string{"FIX-101"}
	ml.load = func() (*release.Manifest, error) { return m, nil }

	svc := release.NewService(deps)
	report, err := svc.Validate(context.Background())
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if report.Ready {
		t.Fatal("Validate().Ready = true, want false (FIX-101 is pending a QA build)")
	}
	mustHaveFailingCheck(t, report.Checks, "FIX-101")
}

// --- 6. Invalid QA state ---

func TestValidateQABuildHistoryMismatchFails(t *testing.T) {
	deps, git, _, ml, vl := happyDeps()
	git.status = cleanStatus
	m := release.New("5.2", "staging", "5.2.0.0.1") // History[0].Build = 1
	ml.load = func() (*release.Manifest, error) { return m, nil }
	// version.json says QA build 2, but the manifest's history was never
	// updated to match - a build interrupted partway through its
	// two-commit save, or hand-edited state.
	vl.load = func() (*version.Version, error) {
		return &version.Version{Sprint: 5, Release: 2, Fixes: 0, DevOps: 0, QA: 2}, nil
	}

	svc := release.NewService(deps)
	report, err := svc.Validate(context.Background())
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if report.Ready {
		t.Fatal("Validate().Ready = true, want false (build history doesn't match version.json's QA counter)")
	}
	mustHaveFailingCheck(t, report.Checks, "QA build #1")
}

// --- 7. Invalid version ---

func TestValidateInvalidVersionFails(t *testing.T) {
	deps, git, _, ml, vl := happyDeps()
	git.status = cleanStatus
	m := release.New("5.2", "staging", "5.2.0.0.1")
	ml.load = func() (*release.Manifest, error) { return m, nil }
	vl.load = func() (*version.Version, error) {
		return &version.Version{Sprint: 5, Release: 2, Fixes: 0, DevOps: 0, QA: 0}, nil // QA must be >= 1
	}

	svc := release.NewService(deps)
	report, err := svc.Validate(context.Background())
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if report.Ready {
		t.Fatal("Validate().Ready = true, want false (QA=0 is invalid)")
	}
	mustHaveFailingCheck(t, report.Checks, "QA must be at least 1")
}

func TestValidateVersionMismatchWithManifestFails(t *testing.T) {
	deps, git, _, ml, vl := happyDeps()
	git.status = cleanStatus
	m := release.New("5.2", "staging", "5.2.0.0.1")
	ml.load = func() (*release.Manifest, error) { return m, nil }
	vl.load = func() (*version.Version, error) {
		return &version.Version{Sprint: 5, Release: 2, Fixes: 1, DevOps: 0, QA: 1}, nil // != m.CurrentVersion
	}

	svc := release.NewService(deps)
	report, err := svc.Validate(context.Background())
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if report.Ready {
		t.Fatal("Validate().Ready = true, want false (version.json disagrees with release.json)")
	}
	mustHaveFailingCheck(t, report.Checks, "does not match")
}

// --- 8. Dirty working tree ---

func TestValidateDirtyWorkingTreeFails(t *testing.T) {
	deps, git, _, _, _ := happyDeps()
	git.status = func() (gitpkg.Status, error) { return gitpkg.Status{Clean: false}, nil }

	svc := release.NewService(deps)
	report, err := svc.Validate(context.Background())
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if report.Ready {
		t.Fatal("Validate().Ready = true, want false (working tree is dirty)")
	}
	mustHaveFailingCheck(t, report.Checks, "uncommitted changes")
}

// --- 9. Existing release tag ---

func TestValidateExistingReleaseTagFails(t *testing.T) {
	deps, git, _, _, _ := happyDeps()
	git.status = cleanStatus
	git.tagExists = func(name string) (bool, error) { return name == "v5.2", nil }

	svc := release.NewService(deps)
	report, err := svc.Validate(context.Background())
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if report.Ready {
		t.Fatal("Validate().Ready = true, want false (tag v5.2 already exists)")
	}
	mustHaveFailingCheck(t, report.Checks, "v5.2")
}

// --- 10. Multiple validation failures ---

func TestValidateReportsMultipleFailuresIndependently(t *testing.T) {
	deps, git, _, ml, _ := happyDeps()
	git.status = func() (gitpkg.Status, error) { return gitpkg.Status{Clean: false}, nil }
	git.tagExists = func(name string) (bool, error) { return name == "v5.2", nil }
	m := release.New("5.2", "staging", "5.2.0.0.1")
	m.Features.Pending = []string{"REPORT"}
	m.ReleaseFixes.Pending = []string{"FIX-101"}
	ml.load = func() (*release.Manifest, error) { return m, nil }

	svc := release.NewService(deps)
	report, err := svc.Validate(context.Background())
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if report.Ready {
		t.Fatal("Validate().Ready = true, want false")
	}

	var failing int
	for _, c := range report.Checks {
		if !c.OK {
			failing++
		}
	}
	// REPORT pending, FIX-101 pending, dirty tree, existing tag - four
	// independent problems, each its own check, not collapsed into one.
	if failing < 4 {
		t.Errorf("got %d failing checks, want at least 4 independent failures; checks = %+v", failing, report.Checks)
	}
	mustHaveFailingCheck(t, report.Checks, "REPORT")
	mustHaveFailingCheck(t, report.Checks, "FIX-101")
	mustHaveFailingCheck(t, report.Checks, "uncommitted changes")
	mustHaveFailingCheck(t, report.Checks, "v5.2")
}

// --- extra coverage for the two checks this milestone added beyond the
// requested checklist's illustrated example (branch existence, config
// drift) ---

func TestValidateMissingReleaseBranchFails(t *testing.T) {
	deps, git, _, _, _ := happyDeps()
	git.status = cleanStatus
	git.branchExists = func(name string) (bool, error) { return false, nil }

	svc := release.NewService(deps)
	report, err := svc.Validate(context.Background())
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if report.Ready {
		t.Fatal("Validate().Ready = true, want false (staging branch missing)")
	}
	mustHaveFailingCheck(t, report.Checks, "staging")
}

func TestValidateBranchConfigDriftFails(t *testing.T) {
	deps, git, _, ml, _ := happyDeps()
	git.status = cleanStatus
	m := release.New("5.2", "release-branch-that-no-longer-matches-config", "5.2.0.0.1")
	ml.load = func() (*release.Manifest, error) { return m, nil }

	svc := release.NewService(deps)
	report, err := svc.Validate(context.Background())
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if report.Ready {
		t.Fatal("Validate().Ready = true, want false (manifest branch != configured staging branch)")
	}
	mustHaveFailingCheck(t, report.Checks, "configured staging branch")
}
