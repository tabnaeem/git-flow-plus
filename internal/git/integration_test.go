package git_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	gitpkg "github.com/hulhub/git-flow-plus/internal/git"
)

// newRealRepo initializes a real Git repository (via the actual git binary)
// in a temp directory, with a committer identity configured locally so
// commits succeed regardless of the host's global Git config.
func newRealRepo(t *testing.T) (dir string, client gitpkg.Client) {
	t.Helper()
	if !gitpkg.Available() {
		t.Skip("git binary not available on PATH")
	}

	dir = t.TempDir()
	runner := gitpkg.NewExecRunner()
	ctx := context.Background()

	if _, err := runner.Run(ctx, dir, "init"); err != nil {
		t.Fatalf("git init: %v", err)
	}
	if _, err := runner.Run(ctx, dir, "config", "user.email", "test@example.com"); err != nil {
		t.Fatalf("git config user.email: %v", err)
	}
	if _, err := runner.Run(ctx, dir, "config", "user.name", "Test User"); err != nil {
		t.Fatalf("git config user.name: %v", err)
	}
	if _, err := runner.Run(ctx, dir, "config", "commit.gpgsign", "false"); err != nil {
		t.Fatalf("git config commit.gpgsign: %v", err)
	}

	return dir, gitpkg.NewClient(runner, dir)
}

func writeFile(t *testing.T, dir, name, contents string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", name, err)
	}
}

func TestIntegrationFullFeatureLifecycle(t *testing.T) {
	dir, c := newRealRepo(t)
	ctx := context.Background()

	if !c.IsRepo(ctx) {
		t.Fatal("IsRepo() = false after git init")
	}
	if c.HasCommits(ctx) {
		t.Fatal("HasCommits() = true for a fresh repo")
	}

	writeFile(t, dir, "README.md", "# Test\n")
	if err := c.Add(ctx, "README.md"); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if err := c.Commit(ctx, "Initial commit", false); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if !c.HasCommits(ctx) {
		t.Fatal("HasCommits() = false after first commit")
	}

	if err := c.RenameCurrentBranch(ctx, "main"); err != nil {
		t.Fatalf("RenameCurrentBranch() error = %v", err)
	}

	if err := c.CreateBranch(ctx, "develop", "main"); err != nil {
		t.Fatalf("CreateBranch(develop) error = %v", err)
	}
	branch, err := c.CurrentBranch(ctx)
	if err != nil {
		t.Fatalf("CurrentBranch() error = %v", err)
	}
	if branch != "develop" {
		t.Fatalf("CurrentBranch() = %q, want %q", branch, "develop")
	}

	if err := c.CreateBranch(ctx, "feature/widgets", "develop"); err != nil {
		t.Fatalf("CreateBranch(feature/widgets) error = %v", err)
	}

	writeFile(t, dir, "widget.go", "package widget\n")
	if err := c.Add(ctx, "widget.go"); err != nil {
		t.Fatalf("Add(widget.go) error = %v", err)
	}
	status, err := c.Status(ctx)
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status.Clean {
		t.Fatal("Status().Clean = true, want false after staging a new file")
	}
	if err := c.Commit(ctx, "Add widget", false); err != nil {
		t.Fatalf("Commit(widget) error = %v", err)
	}

	status, err = c.Status(ctx)
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if !status.Clean {
		t.Fatalf("Status().Clean = false after commit, porcelain = %q", status.Porcelain)
	}

	exists, err := c.BranchExists(ctx, "feature/widgets")
	if err != nil || !exists {
		t.Fatalf("BranchExists(feature/widgets) = %v, %v, want true, nil", exists, err)
	}

	if err := c.Checkout(ctx, "develop"); err != nil {
		t.Fatalf("Checkout(develop) error = %v", err)
	}
	if err := c.MergeNoFF(ctx, "feature/widgets", "Merge feature/widgets into develop"); err != nil {
		t.Fatalf("MergeNoFF() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "widget.go")); err != nil {
		t.Fatalf("widget.go missing on develop after merge: %v", err)
	}

	if err := c.DeleteBranch(ctx, "feature/widgets", false); err != nil {
		t.Fatalf("DeleteBranch() error = %v", err)
	}
	exists, err = c.BranchExists(ctx, "feature/widgets")
	if err != nil || exists {
		t.Fatalf("BranchExists(feature/widgets) after delete = %v, %v, want false, nil", exists, err)
	}

	if err := c.Checkout(ctx, "main"); err != nil {
		t.Fatalf("Checkout(main) error = %v", err)
	}
	if err := c.MergeNoFF(ctx, "develop", "Merge develop into main"); err != nil {
		t.Fatalf("MergeNoFF(develop->main) error = %v", err)
	}
	if err := c.Tag(ctx, "v1.0.0", "Release v1.0.0"); err != nil {
		t.Fatalf("Tag() error = %v", err)
	}
	tagExists, err := c.TagExists(ctx, "v1.0.0")
	if err != nil || !tagExists {
		t.Fatalf("TagExists(v1.0.0) = %v, %v, want true, nil", tagExists, err)
	}
}

func TestIntegrationRemoveStagesDeletion(t *testing.T) {
	dir, c := newRealRepo(t)
	ctx := context.Background()

	writeFile(t, dir, "keep.txt", "keep")
	writeFile(t, dir, "drop.txt", "drop")
	if err := c.Add(ctx, "keep.txt", "drop.txt"); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if err := c.Commit(ctx, "add files", false); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	if err := c.Remove(ctx, "drop.txt"); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "drop.txt")); !os.IsNotExist(err) {
		t.Fatalf("drop.txt still exists on disk after Remove(): err = %v", err)
	}

	status, err := c.Status(ctx)
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status.Clean {
		t.Fatal("Status().Clean = true, want false after a staged removal")
	}
	if err := c.Commit(ctx, "remove drop.txt", false); err != nil {
		t.Fatalf("Commit(remove) error = %v", err)
	}
}

func TestIntegrationListBranches(t *testing.T) {
	dir, c := newRealRepo(t)
	ctx := context.Background()

	writeFile(t, dir, "a.txt", "a")
	if err := c.Add(ctx, "a.txt"); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if err := c.Commit(ctx, "init", false); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if err := c.RenameCurrentBranch(ctx, "main"); err != nil {
		t.Fatalf("RenameCurrentBranch() error = %v", err)
	}

	none, err := c.ListBranches(ctx, "release/*")
	if err != nil {
		t.Fatalf("ListBranches() error = %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("ListBranches(release/*) = %v, want none yet", none)
	}

	if err := c.CreateBranch(ctx, "release/5.2", "main"); err != nil {
		t.Fatalf("CreateBranch(release/5.2) error = %v", err)
	}
	if err := c.CreateBranch(ctx, "release/5.3", "main"); err != nil {
		t.Fatalf("CreateBranch(release/5.3) error = %v", err)
	}
	if err := c.CreateBranch(ctx, "feature/unrelated", "main"); err != nil {
		t.Fatalf("CreateBranch(feature/unrelated) error = %v", err)
	}

	releases, err := c.ListBranches(ctx, "release/*")
	if err != nil {
		t.Fatalf("ListBranches() error = %v", err)
	}
	want := map[string]bool{"release/5.2": true, "release/5.3": true}
	if len(releases) != 2 {
		t.Fatalf("ListBranches(release/*) = %v, want exactly 2 entries", releases)
	}
	for _, r := range releases {
		if !want[r] {
			t.Errorf("ListBranches(release/*) included unexpected branch %q", r)
		}
	}
}

func TestIntegrationCommitSHAAndTagCommit(t *testing.T) {
	dir, c := newRealRepo(t)
	ctx := context.Background()

	writeFile(t, dir, "a.txt", "a")
	if err := c.Add(ctx, "a.txt"); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if err := c.Commit(ctx, "first", false); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	firstSHA, err := c.CommitSHA(ctx)
	if err != nil {
		t.Fatalf("CommitSHA() error = %v", err)
	}
	if len(firstSHA) != 40 {
		t.Fatalf("CommitSHA() = %q, want a 40-character SHA", firstSHA)
	}

	writeFile(t, dir, "b.txt", "b")
	if err := c.Add(ctx, "b.txt"); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if err := c.Commit(ctx, "second", false); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	secondSHA, err := c.CommitSHA(ctx)
	if err != nil {
		t.Fatalf("CommitSHA() error = %v", err)
	}
	if secondSHA == firstSHA {
		t.Fatal("CommitSHA() did not change after a new commit")
	}

	// Tag the *first* commit, not HEAD, to prove TagCommit targets an
	// arbitrary commit rather than always tagging HEAD.
	if err := c.TagCommit(ctx, "v1.0", firstSHA, "First release"); err != nil {
		t.Fatalf("TagCommit() error = %v", err)
	}
	exists, err := c.TagExists(ctx, "v1.0")
	if err != nil || !exists {
		t.Fatalf("TagExists(v1.0) = %v, %v, want true, nil", exists, err)
	}

	out, err := gitpkg.NewExecRunner().Run(ctx, dir, "rev-list", "-n", "1", "v1.0")
	if err != nil {
		t.Fatalf("rev-list v1.0: %v", err)
	}
	if out != firstSHA {
		t.Errorf("tag v1.0 points at %q, want the first commit %q", out, firstSHA)
	}
}

func TestIntegrationConfigValue(t *testing.T) {
	_, c := newRealRepo(t)
	ctx := context.Background()

	// newRealRepo sets user.email via GIT_CONFIG_* env vars, which git
	// config --get honors just like an on-disk config entry.
	value, err := c.ConfigValue(ctx, "user.email")
	if err != nil {
		t.Fatalf("ConfigValue() error = %v", err)
	}
	if value != "test@example.com" {
		t.Errorf("ConfigValue(user.email) = %q, want %q", value, "test@example.com")
	}

	value, err = c.ConfigValue(ctx, "gitflowplus.does-not-exist")
	if err != nil {
		t.Fatalf("ConfigValue() for unset key error = %v", err)
	}
	if value != "" {
		t.Errorf("ConfigValue() for unset key = %q, want empty", value)
	}
}

func TestIntegrationBranchExistsFalseForUnknownBranch(t *testing.T) {
	_, c := newRealRepo(t)
	ctx := context.Background()

	exists, err := c.BranchExists(ctx, "does-not-exist")
	if err != nil {
		t.Fatalf("BranchExists() error = %v", err)
	}
	if exists {
		t.Error("BranchExists() = true for a branch that was never created")
	}
}

func TestIntegrationDeleteUnmergedBranchFailsWithoutForce(t *testing.T) {
	dir, c := newRealRepo(t)
	ctx := context.Background()

	writeFile(t, dir, "a.txt", "a")
	if err := c.Add(ctx, "a.txt"); err != nil {
		t.Fatalf("Add(a.txt) error = %v", err)
	}
	if err := c.Commit(ctx, "init", false); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if err := c.RenameCurrentBranch(ctx, "main"); err != nil {
		t.Fatalf("RenameCurrentBranch() error = %v", err)
	}
	if err := c.CreateBranch(ctx, "feature/orphan", "main"); err != nil {
		t.Fatalf("CreateBranch() error = %v", err)
	}
	writeFile(t, dir, "b.txt", "b")
	if err := c.Add(ctx, "b.txt"); err != nil {
		t.Fatalf("Add(b.txt) error = %v", err)
	}
	if err := c.Commit(ctx, "unmerged work", false); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if err := c.Checkout(ctx, "main"); err != nil {
		t.Fatalf("Checkout(main) error = %v", err)
	}

	if err := c.DeleteBranch(ctx, "feature/orphan", false); err == nil {
		t.Fatal("DeleteBranch(force=false) on an unmerged branch = nil error, want failure")
	}
	if err := c.DeleteBranch(ctx, "feature/orphan", true); err != nil {
		t.Fatalf("DeleteBranch(force=true) error = %v", err)
	}
}
