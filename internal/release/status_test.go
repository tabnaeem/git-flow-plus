package release_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tabnaeem/git-flow-plus/internal/release"
	"github.com/tabnaeem/git-flow-plus/internal/version"
)

// --- Status(): VersionInfo/Builds population (used by `release status`) ---

func TestStatusPopulatesVersionInfoFromLoadedVersion(t *testing.T) {
	deps, _, _, ml, vl := happyDeps()

	ml.load = func() (*release.Manifest, error) {
		return release.New("5.2", "staging", "5.2.1.0.2"), nil
	}
	vl.load = func() (*version.Version, error) {
		return &version.Version{Sprint: 5, Release: 2, Fixes: 1, DevOps: 0, QA: 2}, nil
	}

	svc := release.NewService(deps)
	report, err := svc.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	want := version.Version{Sprint: 5, Release: 2, Fixes: 1, DevOps: 0, QA: 2}
	if report.VersionInfo != want {
		t.Errorf("VersionInfo = %+v, want %+v", report.VersionInfo, want)
	}
}

func TestStatusPopulatesBuildsFromManifestHistory(t *testing.T) {
	deps, _, _, ml, _ := happyDeps()

	m := release.New("5.2", "staging", "5.2.0.0.1")
	m.History[0].Tag = "v5.2.0.0.1"
	m.History = append(m.History, release.BuildRecord{
		Build: 2, Version: "5.2.1.0.2", Tag: "v5.2.1.0.2", BuildDate: time.Now(),
	})
	m.CurrentQABuild = 2
	ml.load = func() (*release.Manifest, error) { return m, nil }

	svc := release.NewService(deps)
	report, err := svc.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	if len(report.Builds) != 2 {
		t.Fatalf("len(Builds) = %d, want 2", len(report.Builds))
	}
	if report.Builds[0].Build != 1 || report.Builds[1].Build != 2 {
		t.Errorf("Builds = %+v, want build #1 then #2 in order", report.Builds)
	}
}

// TestStatusReflectsIncludedAndPendingChanges exercises a release with
// features, release fixes, and DevOps changes in a mix of Included/
// Deferred/Pending — the exact buckets `release status` renders.
func TestStatusReflectsIncludedAndPendingChanges(t *testing.T) {
	deps, _, _, ml, _ := happyDeps()

	m := release.New("5.2", "staging", "5.2.0.0.1")
	m.Features.Included = []string{"LOGIN"}
	m.Features.Pending = []string{"REPORT"}
	m.Features.Deferred = []string{"BILLING"}
	m.ReleaseFixes.Included = []string{"FIX-101"}
	m.ReleaseFixes.Pending = []string{"FIX-102"}
	m.DevOps.Included = []string{"REDIS"}
	ml.load = func() (*release.Manifest, error) { return m, nil }

	svc := release.NewService(deps)
	report, err := svc.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	checks := []struct {
		name string
		got  []string
		want []string
	}{
		{"IncludedFeatures", report.IncludedFeatures, []string{"LOGIN"}},
		{"PendingFeatures", report.PendingFeatures, []string{"REPORT"}},
		{"DeferredFeatures", report.DeferredFeatures, []string{"BILLING"}},
		{"IncludedReleaseFixes", report.IncludedReleaseFixes, []string{"FIX-101"}},
		{"PendingReleaseFixes", report.PendingReleaseFixes, []string{"FIX-102"}},
		{"IncludedDevOps", report.IncludedDevOps, []string{"REDIS"}},
	}
	for _, c := range checks {
		if len(c.got) != len(c.want) || (len(c.got) > 0 && c.got[0] != c.want[0]) {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}

// TestStatusErrorsOnCorruptManifest covers the "invalid/missing state"
// case: a release.json that fails to load must fail Status(), not report
// a zero-valued or partially-populated release.
func TestStatusErrorsOnCorruptManifest(t *testing.T) {
	deps, _, _, ml, _ := happyDeps()
	ml.load = func() (*release.Manifest, error) {
		return nil, errors.New("release: parsing release.json: unexpected end of JSON input")
	}

	svc := release.NewService(deps)
	if _, err := svc.Status(context.Background()); err == nil {
		t.Fatal("Status() error = nil, want an error for an unreadable manifest")
	}
}
