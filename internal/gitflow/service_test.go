package gitflow_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/tabnaeem/git-flow-plus/internal/config"
	gitpkg "github.com/tabnaeem/git-flow-plus/internal/git"
	"github.com/tabnaeem/git-flow-plus/internal/gitflow"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// setGitTestEnv scopes a committer identity and disables GPG signing for the
// current test via environment variables (auto-restored by t.Setenv), so
// commits succeed regardless of the host's global Git config and without
// mutating it.
func setGitTestEnv(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_AUTHOR_NAME", "Test User")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Test User")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "commit.gpgsign")
	t.Setenv("GIT_CONFIG_VALUE_0", "false")
}

// newRepo initializes a bare, empty (commit-less) real Git repository and
// returns its directory plus a ready-to-use gitflow.Service.
func newRepo(t *testing.T) (dir string, gitClient gitpkg.Client, svc gitflow.Service, cfg *config.Config) {
	t.Helper()
	if !gitpkg.Available() {
		t.Skip("git binary not available on PATH")
	}
	setGitTestEnv(t)

	dir = t.TempDir()
	runner := gitpkg.NewExecRunner()
	if _, err := runner.Run(context.Background(), dir, "init"); err != nil {
		t.Fatalf("git init: %v", err)
	}

	gitClient = gitpkg.NewClient(runner, dir)
	cfg = config.Default()
	svc = gitflow.NewService(gitClient, cfg, discardLogger())
	return dir, gitClient, svc, cfg
}

// newInitializedRepo is newRepo followed by a successful gitflow Init, so
// tests can start directly from a ready-to-use main/develop repository.
func newInitializedRepo(t *testing.T) (dir string, gitClient gitpkg.Client, svc gitflow.Service, cfg *config.Config) {
	t.Helper()
	dir, gitClient, svc, cfg = newRepo(t)
	if _, err := svc.Init(context.Background()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	return dir, gitClient, svc, cfg
}

func writeAndCommit(t *testing.T, dir string, gitClient gitpkg.Client, name, contents, message string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", name, err)
	}
	if err := gitClient.Add(context.Background(), name); err != nil {
		t.Fatalf("Add(%s) error = %v", name, err)
	}
	if err := gitClient.Commit(context.Background(), message, false); err != nil {
		t.Fatalf("Commit(%s): %v", message, err)
	}
}

func fileExists(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}

// --- Init ---

func TestInitFreshEmptyDirectory(t *testing.T) {
	if !gitpkg.Available() {
		t.Skip("git binary not available on PATH")
	}
	setGitTestEnv(t)

	dir := t.TempDir()
	gitClient := gitpkg.NewClient(gitpkg.NewExecRunner(), dir)
	cfg := config.Default()
	svc := gitflow.NewService(gitClient, cfg, discardLogger())
	ctx := context.Background()

	result, err := svc.Init(ctx)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if !result.RepoCreated {
		t.Error("RepoCreated = false, want true (directory had no .git yet)")
	}
	if !result.MainCreated {
		t.Error("MainCreated = false, want true")
	}
	if !result.StagingCreated {
		t.Error("StagingCreated = false, want true")
	}
	if !result.DevelopCreated {
		t.Error("DevelopCreated = false, want true")
	}

	for _, branch := range []string{cfg.Branches.Main, cfg.Branches.Staging, cfg.Branches.Develop} {
		exists, err := gitClient.BranchExists(ctx, branch)
		if err != nil || !exists {
			t.Errorf("BranchExists(%q) = %v, %v, want true, nil", branch, exists, err)
		}
	}

	current, err := gitClient.CurrentBranch(ctx)
	if err != nil {
		t.Fatalf("CurrentBranch() error = %v", err)
	}
	if current != cfg.Branches.Develop {
		t.Errorf("CurrentBranch() = %q, want %q", current, cfg.Branches.Develop)
	}
}

func TestInitIsIdempotent(t *testing.T) {
	_, _, svc, _ := newInitializedRepo(t)

	result, err := svc.Init(context.Background())
	if err != nil {
		t.Fatalf("second Init() error = %v", err)
	}
	if result.RepoCreated || result.MainCreated || result.StagingCreated || result.DevelopCreated {
		t.Errorf("second Init() = %+v, want no creation flags set", result)
	}
}

func TestInitExistingHistoryMissingMainErrors(t *testing.T) {
	dir, gitClient, svc, _ := newRepo(t)
	ctx := context.Background()

	writeAndCommit(t, dir, gitClient, "a.txt", "a", "initial")
	if err := gitClient.RenameCurrentBranch(ctx, "trunk"); err != nil {
		t.Fatalf("RenameCurrentBranch() error = %v", err)
	}

	_, err := svc.Init(ctx)
	if err == nil {
		t.Fatal("Init() on existing history without a main branch = nil error, want failure")
	}

	exists, _ := gitClient.BranchExists(ctx, "main")
	if exists {
		t.Error("Init() created a 'main' branch on a repo with unrelated history; it should have refused instead")
	}
}

// --- Feature ---

func TestFeatureStartAndMergeLifecycle(t *testing.T) {
	dir, gitClient, svc, cfg := newInitializedRepo(t)
	ctx := context.Background()

	startResult, err := svc.FeatureStart(ctx, "widgets")
	if err != nil {
		t.Fatalf("FeatureStart() error = %v", err)
	}
	if startResult.Branch != "feature/widgets" {
		t.Errorf("Branch = %q, want %q", startResult.Branch, "feature/widgets")
	}
	if startResult.Base != cfg.Branches.Staging {
		t.Errorf("Base = %q, want %q (features branch from staging, not develop)", startResult.Base, cfg.Branches.Staging)
	}

	current, _ := gitClient.CurrentBranch(ctx)
	if current != "feature/widgets" {
		t.Fatalf("CurrentBranch() = %q, want %q", current, "feature/widgets")
	}

	writeAndCommit(t, dir, gitClient, "widget.go", "package widget\n", "Add widget")

	mergeResult, err := svc.FeatureMerge(ctx, "widgets")
	if err != nil {
		t.Fatalf("FeatureMerge() error = %v", err)
	}
	if mergeResult.Base != cfg.Branches.Staging {
		t.Errorf("Base = %q, want %q", mergeResult.Base, cfg.Branches.Staging)
	}

	current, _ = gitClient.CurrentBranch(ctx)
	if current != cfg.Branches.Staging {
		t.Fatalf("CurrentBranch() after merge = %q, want %q", current, cfg.Branches.Staging)
	}
	if !fileExists(dir, "widget.go") {
		t.Error("widget.go missing from staging after FeatureMerge")
	}

	exists, err := gitClient.BranchExists(ctx, "feature/widgets")
	if err != nil || !exists {
		t.Errorf("BranchExists(feature/widgets) after merge = %v, %v, want true, nil (branch must stay alive for the QA cycle)", exists, err)
	}
}

func TestFeatureStartDuplicateErrors(t *testing.T) {
	_, _, svc, _ := newInitializedRepo(t)
	ctx := context.Background()

	if _, err := svc.FeatureStart(ctx, "dup"); err != nil {
		t.Fatalf("first FeatureStart() error = %v", err)
	}
	_, err := svc.FeatureStart(ctx, "dup")
	if !errors.Is(err, gitflow.ErrBranchAlreadyExists) {
		t.Errorf("second FeatureStart() error = %v, want ErrBranchAlreadyExists", err)
	}
}

func TestFeatureMergeMissingBranchErrorsRealRepo(t *testing.T) {
	_, _, svc, _ := newInitializedRepo(t)

	_, err := svc.FeatureMerge(context.Background(), "never-started")
	if !errors.Is(err, gitflow.ErrBranchMissing) {
		t.Errorf("FeatureMerge() error = %v, want ErrBranchMissing", err)
	}
}

func TestFeatureMergeDirtyWorkingTreeErrors(t *testing.T) {
	dir, _, svc, _ := newInitializedRepo(t)
	ctx := context.Background()

	if _, err := svc.FeatureStart(ctx, "dirty"); err != nil {
		t.Fatalf("FeatureStart() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "untracked.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := svc.FeatureMerge(ctx, "dirty")
	if !errors.Is(err, gitflow.ErrDirtyWorkingTree) {
		t.Errorf("FeatureMerge() with uncommitted changes error = %v, want ErrDirtyWorkingTree", err)
	}
}

func TestFeatureStartWithoutInitErrors(t *testing.T) {
	dir, gitClient, svc, _ := newRepo(t)
	writeAndCommit(t, dir, gitClient, "a.txt", "a", "initial")

	_, err := svc.FeatureStart(context.Background(), "too-early")
	if !errors.Is(err, gitflow.ErrNotInitialized) {
		t.Errorf("FeatureStart() before init error = %v, want ErrNotInitialized", err)
	}
}

func TestFeatureStartRejectsEmptyName(t *testing.T) {
	_, _, svc, _ := newInitializedRepo(t)

	_, err := svc.FeatureStart(context.Background(), "   ")
	if !errors.Is(err, gitflow.ErrEmptyName) {
		t.Errorf("FeatureStart(\"   \") error = %v, want ErrEmptyName", err)
	}
}

// --- Hotfix ---

func TestHotfixLifecycleTagsAndMergesBoth(t *testing.T) {
	dir, gitClient, svc, cfg := newInitializedRepo(t)
	ctx := context.Background()

	if _, err := svc.HotfixStart(ctx, "critical-bug"); err != nil {
		t.Fatalf("HotfixStart() error = %v", err)
	}
	current, _ := gitClient.CurrentBranch(ctx)
	if current != "hotfix/critical-bug" {
		t.Fatalf("CurrentBranch() = %q, want %q", current, "hotfix/critical-bug")
	}

	writeAndCommit(t, dir, gitClient, "fix.txt", "fixed", "Fix critical bug")

	result, err := svc.HotfixFinish(ctx, "critical-bug")
	if err != nil {
		t.Fatalf("HotfixFinish() error = %v", err)
	}
	if result.Tag != "vcritical-bug" {
		t.Errorf("Tag = %q, want %q", result.Tag, "vcritical-bug")
	}

	tagExists, err := gitClient.TagExists(ctx, "vcritical-bug")
	if err != nil || !tagExists {
		t.Errorf("TagExists(vcritical-bug) = %v, %v, want true, nil", tagExists, err)
	}

	if err := gitClient.Checkout(ctx, cfg.Branches.Main); err != nil {
		t.Fatalf("Checkout(main) error = %v", err)
	}
	if !fileExists(dir, "fix.txt") {
		t.Error("fix.txt missing from main after HotfixFinish")
	}

	if err := gitClient.Checkout(ctx, cfg.Branches.Staging); err != nil {
		t.Fatalf("Checkout(staging) error = %v", err)
	}
	if !fileExists(dir, "fix.txt") {
		t.Error("fix.txt missing from staging after HotfixFinish")
	}

	if err := gitClient.Checkout(ctx, cfg.Branches.Develop); err != nil {
		t.Fatalf("Checkout(develop) error = %v", err)
	}
	if !fileExists(dir, "fix.txt") {
		t.Error("fix.txt missing from develop after HotfixFinish")
	}

	exists, err := gitClient.BranchExists(ctx, "hotfix/critical-bug")
	if err != nil || exists {
		t.Errorf("BranchExists(hotfix/critical-bug) after finish = %v, %v, want false, nil", exists, err)
	}
}

func TestHotfixFinishTagCollisionErrors(t *testing.T) {
	dir, gitClient, svc, _ := newInitializedRepo(t)
	ctx := context.Background()

	if _, err := gitpkg.NewExecRunner().Run(ctx, dir, "tag", "-a", "vconflict", "-m", "preexisting"); err != nil {
		t.Fatalf("pre-creating tag: %v", err)
	}

	if _, err := svc.HotfixStart(ctx, "conflict"); err != nil {
		t.Fatalf("HotfixStart() error = %v", err)
	}
	writeAndCommit(t, dir, gitClient, "fix.txt", "fixed", "Fix")

	_, err := svc.HotfixFinish(ctx, "conflict")
	if err == nil {
		t.Fatal("HotfixFinish() with a pre-existing tag = nil error, want failure")
	}
}

// --- Support ---

func TestSupportStartAndFinishDoesNotMerge(t *testing.T) {
	dir, gitClient, svc, cfg := newInitializedRepo(t)
	ctx := context.Background()

	if _, err := svc.SupportStart(ctx, "v1-maintenance"); err != nil {
		t.Fatalf("SupportStart() error = %v", err)
	}
	writeAndCommit(t, dir, gitClient, "legacy-patch.txt", "patch", "Legacy patch")

	result, err := svc.SupportFinish(ctx, "v1-maintenance")
	if err != nil {
		t.Fatalf("SupportFinish() error = %v", err)
	}
	if result.Branch != "support/v1-maintenance" {
		t.Errorf("Branch = %q, want %q", result.Branch, "support/v1-maintenance")
	}

	exists, err := gitClient.BranchExists(ctx, "support/v1-maintenance")
	if err != nil || !exists {
		t.Errorf("BranchExists(support/v1-maintenance) after finish = %v, %v, want true, nil (support branches are not deleted)", exists, err)
	}

	if err := gitClient.Checkout(ctx, cfg.Branches.Develop); err != nil {
		t.Fatalf("Checkout(develop) error = %v", err)
	}
	if fileExists(dir, "legacy-patch.txt") {
		t.Error("legacy-patch.txt found on develop; support branches must not be merged back")
	}
}

func TestSupportFinishMissingBranchErrors(t *testing.T) {
	_, _, svc, _ := newInitializedRepo(t)

	_, err := svc.SupportFinish(context.Background(), "never-started")
	if !errors.Is(err, gitflow.ErrBranchMissing) {
		t.Errorf("SupportFinish() error = %v, want ErrBranchMissing", err)
	}
}

// --- Release (production release: staging -> main only; develop is not
// part of the release lifecycle) ---
//
// ReleaseFinish itself no longer tags (internal/release owns tagging, so
// it can reference the exact merge commit with a manifest-derived
// message); these tests only cover the merge mechanics and the returned
// commit SHA.

func TestReleaseFinishLifecycle(t *testing.T) {
	dir, gitClient, svc, cfg := newInitializedRepo(t)
	ctx := context.Background()

	if err := gitClient.Checkout(ctx, cfg.Branches.Staging); err != nil {
		t.Fatalf("Checkout(staging) error = %v", err)
	}
	writeAndCommit(t, dir, gitClient, "CHANGELOG.md", "5.2 notes", "Prep release notes")

	result, err := svc.ReleaseFinish(ctx)
	if err != nil {
		t.Fatalf("ReleaseFinish() error = %v", err)
	}
	if result.Branch != cfg.Branches.Staging {
		t.Errorf("Branch = %q, want %q", result.Branch, cfg.Branches.Staging)
	}
	if result.Commit == "" {
		t.Error("Commit = \"\", want the merge commit SHA on main")
	}

	if err := gitClient.Checkout(ctx, cfg.Branches.Main); err != nil {
		t.Fatalf("Checkout(main) error = %v", err)
	}
	mainSHA, err := gitClient.CommitSHA(ctx)
	if err != nil {
		t.Fatalf("CommitSHA() error = %v", err)
	}
	if mainSHA != result.Commit {
		t.Errorf("result.Commit = %q, want main's HEAD %q", result.Commit, mainSHA)
	}
	if !fileExists(dir, "CHANGELOG.md") {
		t.Error("CHANGELOG.md missing from main after ReleaseFinish")
	}

	// develop is not part of the release lifecycle — ReleaseFinish must
	// not touch it at all.
	if err := gitClient.Checkout(ctx, cfg.Branches.Develop); err != nil {
		t.Fatalf("Checkout(develop) error = %v", err)
	}
	if fileExists(dir, "CHANGELOG.md") {
		t.Error("CHANGELOG.md present on develop after ReleaseFinish, want develop untouched")
	}

	// staging is permanent — it must still exist after finish.
	exists, err := gitClient.BranchExists(ctx, cfg.Branches.Staging)
	if err != nil || !exists {
		t.Errorf("BranchExists(staging) after finish = %v, %v, want true, nil (staging is never deleted)", exists, err)
	}
}

func TestReleaseFinishRequiresCleanWorkingTree(t *testing.T) {
	dir, gitClient, svc, cfg := newInitializedRepo(t)
	ctx := context.Background()

	if err := gitClient.Checkout(ctx, cfg.Branches.Staging); err != nil {
		t.Fatalf("Checkout(staging) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "untracked.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := svc.ReleaseFinish(ctx)
	if !errors.Is(err, gitflow.ErrDirtyWorkingTree) {
		t.Errorf("ReleaseFinish() with uncommitted changes error = %v, want ErrDirtyWorkingTree", err)
	}
}

// --- ReleaseFix (branches from / merges into staging directly) ---

func TestReleaseFixLifecycle(t *testing.T) {
	dir, gitClient, svc, cfg := newInitializedRepo(t)
	ctx := context.Background()

	startResult, err := svc.ReleaseFixStart(ctx, "BUG-101")
	if err != nil {
		t.Fatalf("ReleaseFixStart() error = %v", err)
	}
	if startResult.Branch != "release-fix/BUG-101" {
		t.Errorf("Branch = %q, want %q", startResult.Branch, "release-fix/BUG-101")
	}
	if startResult.Base != cfg.Branches.Staging {
		t.Errorf("Base = %q, want %q", startResult.Base, cfg.Branches.Staging)
	}

	writeAndCommit(t, dir, gitClient, "bugfix.txt", "fixed", "Fix BUG-101")

	finishResult, err := svc.ReleaseFixFinish(ctx, "BUG-101")
	if err != nil {
		t.Fatalf("ReleaseFixFinish() error = %v", err)
	}
	if finishResult.Base != cfg.Branches.Staging {
		t.Errorf("Base = %q, want %q", finishResult.Base, cfg.Branches.Staging)
	}

	current, _ := gitClient.CurrentBranch(ctx)
	if current != cfg.Branches.Staging {
		t.Fatalf("CurrentBranch() = %q, want %q", current, cfg.Branches.Staging)
	}
	if !fileExists(dir, "bugfix.txt") {
		t.Error("bugfix.txt missing from staging after ReleaseFixFinish")
	}

	exists, err := gitClient.BranchExists(ctx, "release-fix/BUG-101")
	if err != nil || !exists {
		t.Errorf("BranchExists(release-fix/BUG-101) after finish = %v, %v, want true, nil (branch stays alive until release finish)", exists, err)
	}
}

func TestReleaseFixFinishMissingBranchErrors(t *testing.T) {
	_, _, svc, _ := newInitializedRepo(t)

	_, err := svc.ReleaseFixFinish(context.Background(), "never-started")
	if !errors.Is(err, gitflow.ErrBranchMissing) {
		t.Errorf("ReleaseFixFinish() error = %v, want ErrBranchMissing", err)
	}
}

func TestReleaseFixStartWithoutInitErrors(t *testing.T) {
	dir, gitClient, svc, _ := newRepo(t)
	writeAndCommit(t, dir, gitClient, "a.txt", "a", "initial")

	_, err := svc.ReleaseFixStart(context.Background(), "BUG-101")
	if !errors.Is(err, gitflow.ErrNotInitialized) {
		t.Errorf("ReleaseFixStart() before init error = %v, want ErrNotInitialized", err)
	}
}

// --- DevOps (branches from / merges into staging directly) ---

func TestDevOpsLifecycle(t *testing.T) {
	dir, gitClient, svc, cfg := newInitializedRepo(t)
	ctx := context.Background()

	startResult, err := svc.DevOpsStart(ctx, "redis-cache")
	if err != nil {
		t.Fatalf("DevOpsStart() error = %v", err)
	}
	if startResult.Branch != "release-devops/redis-cache" {
		t.Errorf("Branch = %q, want %q", startResult.Branch, "release-devops/redis-cache")
	}

	writeAndCommit(t, dir, gitClient, "redis.yaml", "config", "Add redis cache config")

	finishResult, err := svc.DevOpsFinish(ctx, "redis-cache")
	if err != nil {
		t.Fatalf("DevOpsFinish() error = %v", err)
	}
	if finishResult.Base != cfg.Branches.Staging {
		t.Errorf("Base = %q, want %q", finishResult.Base, cfg.Branches.Staging)
	}

	if !fileExists(dir, "redis.yaml") {
		t.Error("redis.yaml missing from staging after DevOpsFinish")
	}
	exists, err := gitClient.BranchExists(ctx, "release-devops/redis-cache")
	if err != nil || !exists {
		t.Errorf("BranchExists(release-devops/redis-cache) after finish = %v, %v, want true, nil (branch stays alive until release finish)", exists, err)
	}
}

func TestDevOpsFinishMissingBranchErrors(t *testing.T) {
	_, _, svc, _ := newInitializedRepo(t)

	_, err := svc.DevOpsFinish(context.Background(), "never-started")
	if !errors.Is(err, gitflow.ErrBranchMissing) {
		t.Errorf("DevOpsFinish() error = %v, want ErrBranchMissing", err)
	}
}

// --- Doctor ---

func TestDoctorHealthyRepo(t *testing.T) {
	_, _, svc, _ := newInitializedRepo(t)

	report := svc.Doctor(context.Background())
	if !report.Healthy() {
		t.Errorf("Doctor().Healthy() = false, want true; checks = %+v", report.Checks)
	}
	if len(report.Checks) == 0 {
		t.Fatal("Doctor() returned no checks")
	}
	for _, name := range []string{"Git", "Permissions"} {
		found := false
		for _, c := range report.Checks {
			if c.Name == name {
				found = true
				if !c.OK {
					t.Errorf("%q check OK = false, want true; detail = %q", name, c.Detail)
				}
			}
		}
		if !found {
			t.Errorf("Doctor() did not include a %q check", name)
		}
	}
}

func TestDoctorReportsUnwritableRepo(t *testing.T) {
	svc := gitflow.NewService(&fakeClient{writable: func() (bool, error) { return false, nil }}, config.Default(), discardLogger())

	report := svc.Doctor(context.Background())
	if report.Healthy() {
		t.Error("Doctor().Healthy() = true with an unwritable repository directory, want false")
	}
	found := false
	for _, c := range report.Checks {
		if c.Name == "Permissions" {
			found = true
			if c.OK {
				t.Error("'Permissions' check OK = true, want false")
			}
		}
	}
	if !found {
		t.Error("Doctor() did not include a 'Permissions' check")
	}
}

func TestDoctorNotARepo(t *testing.T) {
	if !gitpkg.Available() {
		t.Skip("git binary not available on PATH")
	}
	dir := t.TempDir()
	svc := gitflow.NewService(gitpkg.NewClient(gitpkg.NewExecRunner(), dir), config.Default(), discardLogger())

	report := svc.Doctor(context.Background())
	if report.Healthy() {
		t.Error("Doctor().Healthy() = true for a non-repository directory, want false")
	}
	found := false
	for _, c := range report.Checks {
		if c.Name == "Repository" {
			found = true
			if c.OK {
				t.Error("'Repository' check OK = true, want false")
			}
		}
	}
	if !found {
		t.Error("Doctor() did not include a 'Repository' check")
	}
}

func TestDoctorRepoWithoutBranches(t *testing.T) {
	dir, _, _, cfg := newRepo(t)
	svc := gitflow.NewService(gitpkg.NewClient(gitpkg.NewExecRunner(), dir), cfg, discardLogger())

	report := svc.Doctor(context.Background())
	if report.Healthy() {
		t.Error("Doctor().Healthy() = true for an uninitialized repo, want false")
	}
}
