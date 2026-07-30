package feature_test

import (
	"path/filepath"
	"testing"

	"github.com/tabnaeem/git-flow-plus/internal/feature"
)

func TestPath(t *testing.T) {
	got := feature.Path("/repo")
	want := filepath.Join("/repo", ".gitflowplus", "features.json")
	if got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
}

func TestRelPath(t *testing.T) {
	if got, want := feature.RelPath(), ".gitflowplus/features.json"; got != want {
		t.Errorf("RelPath() = %q, want %q", got, want)
	}
}

func TestLoaderExistsFalseForFreshRepo(t *testing.T) {
	dir := t.TempDir()
	if feature.NewLoader().Exists(dir) {
		t.Fatal("Exists() = true for a repo with no features.json")
	}
}

func TestLoaderLoadReturnsEmptyRegistryWhenMissing(t *testing.T) {
	dir := t.TempDir()
	r, err := feature.NewLoader().Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(r.Features) != 0 {
		t.Errorf("Load() on missing file = %v, want empty registry", r.Features)
	}
}

func TestLoaderSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	loader := feature.NewLoader()

	r := feature.New()
	r.Upsert(feature.Feature{
		ID: "LOGIN", Branch: "feature/LOGIN", MergeCommit: "abc123",
		State: feature.StateReleased, Release: "5.3",
	})

	if err := loader.Save(dir, r); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if !loader.Exists(dir) {
		t.Fatal("Exists() = false after Save()")
	}

	got, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	f, ok := got.Find("LOGIN")
	if !ok {
		t.Fatal("Find(LOGIN) = false after round trip")
	}
	if f.Release != "5.3" || f.State != feature.StateReleased || f.MergeCommit != "abc123" {
		t.Errorf("round-tripped feature = %+v, want Release=5.3 State=Released MergeCommit=abc123", f)
	}
}
