package version_test

import (
	"path/filepath"
	"testing"

	"github.com/hulhub/git-flow-plus/internal/version"
)

func TestPath(t *testing.T) {
	got := version.Path("/repo")
	want := filepath.Join("/repo", ".gitflowplus", "version.json")
	if got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
}

func TestLoaderExistsFalseForFreshRepo(t *testing.T) {
	dir := t.TempDir()
	if version.NewLoader().Exists(dir) {
		t.Fatal("Exists() = true for a repo with no version.json")
	}
}

func TestLoaderSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	loader := version.NewLoader()

	v := version.New(5, 2)
	v.ApplyBuild(4, 1)

	if err := loader.Save(dir, v); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if !loader.Exists(dir) {
		t.Fatal("Exists() = false after Save()")
	}

	got, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if *got != *v {
		t.Errorf("Load() = %+v, want %+v", *got, *v)
	}
}

func TestLoaderLoadMissingFileErrors(t *testing.T) {
	dir := t.TempDir()
	if _, err := version.NewLoader().Load(dir); err == nil {
		t.Fatal("Load() on a missing file = nil error, want failure")
	}
}
