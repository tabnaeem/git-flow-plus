package release_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/tabnaeem/git-flow-plus/internal/config"
	"github.com/tabnaeem/git-flow-plus/internal/feature"
	gitpkg "github.com/tabnaeem/git-flow-plus/internal/git"
	"github.com/tabnaeem/git-flow-plus/internal/gitflow"
	"github.com/tabnaeem/git-flow-plus/internal/hooks"
	"github.com/tabnaeem/git-flow-plus/internal/release"
	"github.com/tabnaeem/git-flow-plus/internal/version"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

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

// newInitializedRepo sets up a real Git repository, initialized for Git
// Flow, and returns a release.Service ready to use plus its dependencies
// for direct assertions. gitflowSvc is returned too so tests can drive
// releasefix/devops start (which stays a pure gitflow operation, with no
// manifest bookkeeping) directly.
func newInitializedRepo(t *testing.T) (dir string, gitClient gitpkg.Client, svc release.Service, gitflowSvc gitflow.Service, cfg *config.Config) {
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
	gitflowSvc = gitflow.NewService(gitClient, cfg, discardLogger())
	if _, err := gitflowSvc.Init(context.Background()); err != nil {
		t.Fatalf("gitflow Init() error = %v", err)
	}

	svc = release.NewService(release.Dependencies{
		GitFlow:        gitflowSvc,
		Git:            gitClient,
		RepoPath:       dir,
		Config:         cfg,
		ManifestLoader: release.NewLoader(),
		VersionLoader:  version.NewLoader(),
		FeatureLoader:  feature.NewLoader(),
		Hooks:          hooks.NewRunner(),
		Logger:         discardLogger(),
	})

	return dir, gitClient, svc, gitflowSvc, cfg
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

// --- StartRelease ---

func TestStartReleaseBootstrapsVersionManifestAndTag(t *testing.T) {
	dir, gitClient, svc, _, cfg := newInitializedRepo(t)
	ctx := context.Background()

	result, err := svc.StartRelease(ctx, "5.2")
	if err != nil {
		t.Fatalf("StartRelease() error = %v", err)
	}
	if result.Version != "5.2.0.0.1" {
		t.Errorf("Version = %q, want %q", result.Version, "5.2.0.0.1")
	}
	if result.Branch != cfg.Branches.Staging {
		t.Errorf("Branch = %q, want %q", result.Branch, cfg.Branches.Staging)
	}
	if result.Tag != "v5.2.0.0.1" {
		t.Errorf("Tag = %q, want %q", result.Tag, "v5.2.0.0.1")
	}

	if !fileExists(dir, ".gitflowplus/version.json") {
		t.Error("version.json missing after StartRelease")
	}
	if !fileExists(dir, ".gitflowplus/release.json") {
		t.Error("release.json missing after StartRelease")
	}

	tagExists, err := gitClient.TagExists(ctx, "v5.2.0.0.1")
	if err != nil || !tagExists {
		t.Errorf("TagExists(v5.2.0.0.1) = %v, %v, want true, nil", tagExists, err)
	}

	status, err := gitClient.Status(ctx)
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if !status.Clean {
		t.Errorf("working tree not clean after StartRelease: %q", status.Porcelain)
	}
}

func TestStartReleaseRejectsInvalidName(t *testing.T) {
	_, _, svc, _, _ := newInitializedRepo(t)

	_, err := svc.StartRelease(context.Background(), "not-a-version")
	if !errors.Is(err, version.ErrInvalidReleaseName) {
		t.Errorf("StartRelease() error = %v, want ErrInvalidReleaseName", err)
	}
}

func TestStartReleaseAlreadyActiveErrorsRealRepo(t *testing.T) {
	_, _, svc, _, _ := newInitializedRepo(t)
	ctx := context.Background()

	if _, err := svc.StartRelease(ctx, "5.2"); err != nil {
		t.Fatalf("first StartRelease() error = %v", err)
	}

	_, err := svc.StartRelease(ctx, "5.3")
	if !errors.Is(err, release.ErrReleaseAlreadyActive) {
		t.Errorf("second StartRelease() error = %v, want ErrReleaseAlreadyActive", err)
	}
}

// --- Status / Manifest / Version before and after start ---

func TestStatusInactiveBeforeRelease(t *testing.T) {
	_, _, svc, _, _ := newInitializedRepo(t)

	report, err := svc.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if report.Active {
		t.Error("Status().Active = true before any release started, want false")
	}
}

func TestManifestAndVersionErrorBeforeRelease(t *testing.T) {
	_, _, svc, _, _ := newInitializedRepo(t)
	ctx := context.Background()

	if _, err := svc.Manifest(ctx); !errors.Is(err, release.ErrNoActiveRelease) {
		t.Errorf("Manifest() error = %v, want ErrNoActiveRelease", err)
	}
	if _, err := svc.Version(ctx); !errors.Is(err, release.ErrNoActiveRelease) {
		t.Errorf("Version() error = %v, want ErrNoActiveRelease", err)
	}
}

func TestStatusActiveAfterStart(t *testing.T) {
	_, _, svc, _, _ := newInitializedRepo(t)
	ctx := context.Background()

	if _, err := svc.StartRelease(ctx, "5.2"); err != nil {
		t.Fatalf("StartRelease() error = %v", err)
	}

	report, err := svc.Status(ctx)
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if !report.Active {
		t.Fatal("Status().Active = false after StartRelease, want true")
	}
	if report.Release != "5.2" {
		t.Errorf("Release = %q, want %q", report.Release, "5.2")
	}
	if report.Version != "5.2.0.0.1" {
		t.Errorf("Version = %q, want %q", report.Version, "5.2.0.0.1")
	}
	if report.QABuild != 1 {
		t.Errorf("QABuild = %d, want 1", report.QABuild)
	}
}

// --- ReleaseFixFinish / DevOpsFinish: pending, never touches the version ---

func TestReleaseFixFinishRecordsPendingWithoutChangingVersion(t *testing.T) {
	dir, gitClient, svc, gitflowSvc, _ := newInitializedRepo(t)
	ctx := context.Background()

	if _, err := svc.StartRelease(ctx, "5.2"); err != nil {
		t.Fatalf("StartRelease() error = %v", err)
	}

	if _, err := gitflowSvc.ReleaseFixStart(ctx, "BUG-101"); err != nil {
		t.Fatalf("ReleaseFixStart() error = %v", err)
	}
	writeAndCommit(t, dir, gitClient, "bugfix.txt", "fixed", "Fix BUG-101")

	if _, err := svc.ReleaseFixFinish(ctx, "BUG-101"); err != nil {
		t.Fatalf("ReleaseFixFinish() error = %v", err)
	}

	v, err := svc.Version(ctx)
	if err != nil {
		t.Fatalf("Version() error = %v", err)
	}
	if v.String() != "5.2.0.0.1" {
		t.Errorf("Version() = %q, want unchanged %q (merges must never move the version)", v.String(), "5.2.0.0.1")
	}

	m, err := svc.Manifest(ctx)
	if err != nil {
		t.Fatalf("Manifest() error = %v", err)
	}
	if len(m.ReleaseFixes.Pending) != 1 || m.ReleaseFixes.Pending[0] != "BUG-101" {
		t.Errorf("ReleaseFixes.Pending = %v, want [\"BUG-101\"]", m.ReleaseFixes.Pending)
	}
	if len(m.ReleaseFixes.Included) != 0 {
		t.Errorf("ReleaseFixes.Included = %v, want empty (not included until a QA build)", m.ReleaseFixes.Included)
	}

	status, err := gitClient.Status(ctx)
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if !status.Clean {
		t.Errorf("working tree not clean after ReleaseFixFinish: %q", status.Porcelain)
	}
}

func TestDevOpsFinishRecordsPendingWithoutChangingVersion(t *testing.T) {
	dir, gitClient, svc, gitflowSvc, _ := newInitializedRepo(t)
	ctx := context.Background()

	if _, err := svc.StartRelease(ctx, "5.2"); err != nil {
		t.Fatalf("StartRelease() error = %v", err)
	}

	if _, err := gitflowSvc.DevOpsStart(ctx, "redis-cache"); err != nil {
		t.Fatalf("DevOpsStart() error = %v", err)
	}
	writeAndCommit(t, dir, gitClient, "redis.yaml", "config", "Add redis config")

	if _, err := svc.DevOpsFinish(ctx, "redis-cache"); err != nil {
		t.Fatalf("DevOpsFinish() error = %v", err)
	}

	m, err := svc.Manifest(ctx)
	if err != nil {
		t.Fatalf("Manifest() error = %v", err)
	}
	if len(m.DevOps.Pending) != 1 || m.DevOps.Pending[0] != "redis-cache" {
		t.Errorf("DevOps.Pending = %v, want [\"redis-cache\"]", m.DevOps.Pending)
	}
	if len(m.DevOps.Included) != 0 {
		t.Errorf("DevOps.Included = %v, want empty (not included until a QA build)", m.DevOps.Included)
	}
}

// --- Build ---

func TestBuildFoldsPendingChangesAndTags(t *testing.T) {
	dir, gitClient, svc, gitflowSvc, cfg := newInitializedRepo(t)
	ctx := context.Background()

	if _, err := svc.StartRelease(ctx, "5.2"); err != nil {
		t.Fatalf("StartRelease() error = %v", err)
	}

	for _, name := range []string{"BUG-101", "BUG-102", "BUG-103", "BUG-104"} {
		if _, err := gitflowSvc.ReleaseFixStart(ctx, name); err != nil {
			t.Fatalf("ReleaseFixStart(%s) error = %v", name, err)
		}
		writeAndCommit(t, dir, gitClient, name+".txt", "fix", "Fix "+name)
		if _, err := svc.ReleaseFixFinish(ctx, name); err != nil {
			t.Fatalf("ReleaseFixFinish(%s) error = %v", name, err)
		}
	}

	// Still checked out on staging (ReleaseFixFinish ends there).
	result, err := svc.Build(ctx)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if result.Version != "5.2.4.0.2" {
		t.Errorf("Version = %q, want %q", result.Version, "5.2.4.0.2")
	}
	if result.NewFixes != 4 || result.NewDevOps != 0 {
		t.Errorf("NewFixes=%d NewDevOps=%d, want NewFixes=4 NewDevOps=0", result.NewFixes, result.NewDevOps)
	}
	if result.QABuild != 2 {
		t.Errorf("QABuild = %d, want 2", result.QABuild)
	}
	if result.Tag != "v5.2.4.0.2" {
		t.Errorf("Tag = %q, want %q", result.Tag, "v5.2.4.0.2")
	}

	tagExists, err := gitClient.TagExists(ctx, "v5.2.4.0.2")
	if err != nil || !tagExists {
		t.Errorf("TagExists(v5.2.4.0.2) = %v, %v, want true, nil", tagExists, err)
	}

	m, err := svc.Manifest(ctx)
	if err != nil {
		t.Fatalf("Manifest() error = %v", err)
	}
	if len(m.ReleaseFixes.Pending) != 0 {
		t.Errorf("ReleaseFixes.Pending = %v, want empty after build", m.ReleaseFixes.Pending)
	}
	if len(m.ReleaseFixes.Included) != 4 {
		t.Errorf("ReleaseFixes.Included = %v, want 4 entries", m.ReleaseFixes.Included)
	}
	if len(m.History) != 2 {
		t.Fatalf("History has %d entries, want 2", len(m.History))
	}

	current, err := gitClient.CurrentBranch(ctx)
	if err != nil {
		t.Fatalf("CurrentBranch() error = %v", err)
	}
	if current != cfg.Branches.Staging {
		t.Errorf("CurrentBranch() = %q, want %q", current, cfg.Branches.Staging)
	}
}

func TestBuildRequiresStagingBranchRealRepo(t *testing.T) {
	_, gitClient, svc, _, cfg := newInitializedRepo(t)
	ctx := context.Background()

	if _, err := svc.StartRelease(ctx, "5.2"); err != nil {
		t.Fatalf("StartRelease() error = %v", err)
	}
	if err := gitClient.Checkout(ctx, cfg.Branches.Develop); err != nil {
		t.Fatalf("Checkout(develop) error = %v", err)
	}

	_, err := svc.Build(ctx)
	if !errors.Is(err, release.ErrNotOnStaging) {
		t.Errorf("Build() off staging error = %v, want ErrNotOnStaging", err)
	}
}

func TestBuildNothingPendingErrorsRealRepo(t *testing.T) {
	_, _, svc, _, _ := newInitializedRepo(t)
	ctx := context.Background()

	if _, err := svc.StartRelease(ctx, "5.2"); err != nil {
		t.Fatalf("StartRelease() error = %v", err)
	}

	_, err := svc.Build(ctx)
	if !errors.Is(err, release.ErrNothingToBuild) {
		t.Errorf("Build() with nothing pending error = %v, want ErrNothingToBuild", err)
	}
}

// --- FinishRelease ---

func TestFinishReleaseRequiresBuildBeforeFinishing(t *testing.T) {
	dir, gitClient, svc, gitflowSvc, _ := newInitializedRepo(t)
	ctx := context.Background()

	if _, err := svc.StartRelease(ctx, "5.2"); err != nil {
		t.Fatalf("StartRelease() error = %v", err)
	}
	if _, err := gitflowSvc.ReleaseFixStart(ctx, "BUG-101"); err != nil {
		t.Fatalf("ReleaseFixStart() error = %v", err)
	}
	writeAndCommit(t, dir, gitClient, "bugfix.txt", "fixed", "Fix BUG-101")
	if _, err := svc.ReleaseFixFinish(ctx, "BUG-101"); err != nil {
		t.Fatalf("ReleaseFixFinish() error = %v", err)
	}

	_, err := svc.FinishRelease(ctx, "5.2")
	if !errors.Is(err, release.ErrPendingChangesNotBuilt) {
		t.Errorf("FinishRelease() with pending unbuild changes error = %v, want ErrPendingChangesNotBuilt", err)
	}
}

func TestFinishReleaseArchivesTagsAndResetsForNextCycle(t *testing.T) {
	dir, gitClient, svc, gitflowSvc, cfg := newInitializedRepo(t)
	ctx := context.Background()

	if _, err := svc.StartRelease(ctx, "5.2"); err != nil {
		t.Fatalf("StartRelease() error = %v", err)
	}
	if _, err := gitflowSvc.ReleaseFixStart(ctx, "BUG-101"); err != nil {
		t.Fatalf("ReleaseFixStart() error = %v", err)
	}
	writeAndCommit(t, dir, gitClient, "bugfix.txt", "fixed", "Fix BUG-101")
	if _, err := svc.ReleaseFixFinish(ctx, "BUG-101"); err != nil {
		t.Fatalf("ReleaseFixFinish() error = %v", err)
	}
	buildResult, err := svc.Build(ctx)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	result, err := svc.FinishRelease(ctx, "5.2")
	if err != nil {
		t.Fatalf("FinishRelease() error = %v", err)
	}
	if result.Version != buildResult.Version {
		t.Errorf("FinishRelease Version = %q, want %q (the built version)", result.Version, buildResult.Version)
	}
	// The production tag names the release, not the QA build's full
	// version string — that name is already taken by the QA tag on a
	// different commit (Git tag names are globally unique).
	if result.Tag != "v5.2" {
		t.Errorf("FinishRelease Tag = %q, want %q", result.Tag, "v5.2")
	}
	if result.Tag == buildResult.Tag {
		t.Errorf("production tag %q must differ from the QA build tag %q (they point at different commits)", result.Tag, buildResult.Tag)
	}

	tagExists, err := gitClient.TagExists(ctx, result.Tag)
	if err != nil || !tagExists {
		t.Errorf("TagExists(%s) = %v, %v, want true, nil", result.Tag, tagExists, err)
	}
	// The QA build tag must still exist too — finishing a release doesn't
	// remove the trail of QA tags that led to it.
	qaTagExists, err := gitClient.TagExists(ctx, buildResult.Tag)
	if err != nil || !qaTagExists {
		t.Errorf("TagExists(%s) [QA tag] = %v, %v, want true, nil", buildResult.Tag, qaTagExists, err)
	}

	// The release-fix branch stayed alive through the QA cycle; only now,
	// once the release has finished, should it be deleted.
	fixExists, err := gitClient.BranchExists(ctx, "release-fix/BUG-101")
	if err != nil || fixExists {
		t.Errorf("BranchExists(release-fix/BUG-101) after release finish = %v, %v, want false, nil", fixExists, err)
	}

	if err := gitClient.Checkout(ctx, cfg.Branches.Main); err != nil {
		t.Fatalf("Checkout(%s) error = %v", cfg.Branches.Main, err)
	}
	if fileExists(dir, ".gitflowplus/release.json") {
		t.Error("release.json still present on main after finish; it should be archived and removed")
	}
	if fileExists(dir, ".gitflowplus/version.json") {
		t.Error("version.json still present on main after finish")
	}
	if !fileExists(dir, ".gitflowplus/archive/5.2.json") {
		t.Error("archive/5.2.json missing on main after finish")
	}
	if !fileExists(dir, "bugfix.txt") {
		t.Error("bugfix.txt missing from main after finish")
	}

	// develop is not part of the release lifecycle — ReleaseFinish must
	// not touch it at all.
	if err := gitClient.Checkout(ctx, cfg.Branches.Develop); err != nil {
		t.Fatalf("Checkout(%s) error = %v", cfg.Branches.Develop, err)
	}
	if fileExists(dir, "bugfix.txt") {
		t.Error("bugfix.txt present on develop after finish, want develop untouched")
	}

	// staging is permanent and must still exist, ready for the next release.
	if err := gitClient.Checkout(ctx, cfg.Branches.Staging); err != nil {
		t.Fatalf("Checkout(staging) error = %v", err)
	}
	report, err := svc.Status(ctx)
	if err != nil {
		t.Fatalf("Status() on staging error = %v", err)
	}
	if report.Active {
		t.Error("Status().Active = true on staging after release finished, want false (counters reset for the next release)")
	}

	// Starting the next release cycle must work cleanly.
	next, err := svc.StartRelease(ctx, "5.3")
	if err != nil {
		t.Fatalf("StartRelease(5.3) after finishing 5.2 error = %v", err)
	}
	if next.Version != "5.3.0.0.1" {
		t.Errorf("next release Version = %q, want %q", next.Version, "5.3.0.0.1")
	}
}

// --- Feature Registry / Release Planning ---

func TestFeaturePlanningLifecycleRealRepo(t *testing.T) {
	dir, gitClient, svc, gitflowSvc, _ := newInitializedRepo(t)
	ctx := context.Background()

	// Start a feature (branches from staging, not develop) and register
	// it — the registry update is CLI-layer orchestration in the real
	// tool (cli/feature.go), so the test drives RegisterFeatureCreated
	// directly, mirroring that.
	startResult, err := gitflowSvc.FeatureStart(ctx, "LOGIN")
	if err != nil {
		t.Fatalf("FeatureStart() error = %v", err)
	}
	if err := svc.RegisterFeatureCreated(ctx, "LOGIN", startResult.Branch); err != nil {
		t.Fatalf("RegisterFeatureCreated() error = %v", err)
	}

	// RegisterFeatureCreated must leave the developer on their feature
	// branch, not staging.
	current, err := gitClient.CurrentBranch(ctx)
	if err != nil {
		t.Fatalf("CurrentBranch() error = %v", err)
	}
	if current != "feature/LOGIN" {
		t.Errorf("CurrentBranch() after RegisterFeatureCreated = %q, want %q", current, "feature/LOGIN")
	}

	writeAndCommit(t, dir, gitClient, "login.go", "package login", "Implement LOGIN")

	// Not approved yet: Release Planning can't touch it.
	if _, err := svc.StartRelease(ctx, "5.3"); err != nil {
		t.Fatalf("StartRelease() error = %v", err)
	}
	if err := svc.AddFeatureToRelease(ctx, "LOGIN"); !errors.Is(err, release.ErrFeatureNotApproved) {
		t.Errorf("AddFeatureToRelease() before approval error = %v, want ErrFeatureNotApproved", err)
	}

	if err := svc.ApproveFeature(ctx, "LOGIN"); err != nil {
		t.Fatalf("ApproveFeature() error = %v", err)
	}

	// Release Planning is explicit: LOGIN stays Pending until a decision.
	statusBeforeDecision, err := svc.FeatureStatus(ctx)
	if err != nil {
		t.Fatalf("FeatureStatus() error = %v", err)
	}
	if len(statusBeforeDecision.Pending) != 1 || statusBeforeDecision.Pending[0] != "LOGIN" {
		t.Errorf("FeatureStatus().Pending = %v, want [\"LOGIN\"] before any planning decision", statusBeforeDecision.Pending)
	}

	versionBefore, err := svc.Version(ctx)
	if err != nil {
		t.Fatalf("Version() error = %v", err)
	}

	if err := svc.AddFeatureToRelease(ctx, "LOGIN"); err != nil {
		t.Fatalf("AddFeatureToRelease() error = %v", err)
	}

	// AddFeatureToRelease performs the real merge into staging.
	if !fileExists(dir, "login.go") {
		t.Error("login.go missing from staging after AddFeatureToRelease")
	}
	versionAfter, err := svc.Version(ctx)
	if err != nil {
		t.Fatalf("Version() error = %v", err)
	}
	if versionAfter.Release != versionBefore.Release+1 {
		t.Errorf("Version().Release = %d, want %d (the feature counter, incremented by AddFeatureToRelease)", versionAfter.Release, versionBefore.Release+1)
	}

	// The feature branch must still exist — it stays alive through QA.
	exists, err := gitClient.BranchExists(ctx, "feature/LOGIN")
	if err != nil || !exists {
		t.Errorf("BranchExists(feature/LOGIN) after AddFeatureToRelease = %v, %v, want true, nil", exists, err)
	}

	m, err := svc.Manifest(ctx)
	if err != nil {
		t.Fatalf("Manifest() error = %v", err)
	}
	if len(m.Features.Included) != 1 || m.Features.Included[0] != "LOGIN" {
		t.Errorf("Manifest().Features.Included = %v, want [\"LOGIN\"]", m.Features.Included)
	}
	if len(m.Features.Pending) != 0 {
		t.Errorf("Manifest().Features.Pending = %v, want empty once LOGIN is included", m.Features.Pending)
	}
	if len(m.FeatureHistory) != 1 || m.FeatureHistory[0].ID != "LOGIN" {
		t.Errorf("Manifest().FeatureHistory = %+v, want one entry for LOGIN", m.FeatureHistory)
	}

	// Finishing the release (no pending fixes/devops) must promote LOGIN to
	// permanently Released in the registry, and delete its branch.
	finishResult, err := svc.FinishRelease(ctx, "5.3")
	if err != nil {
		t.Fatalf("FinishRelease() error = %v", err)
	}
	if finishResult.Release != "5.3" {
		t.Errorf("FinishRelease().Release = %q, want %q", finishResult.Release, "5.3")
	}

	features, err := svc.ListApprovedFeatures(ctx)
	if err != nil {
		t.Fatalf("ListApprovedFeatures() error = %v", err)
	}
	if len(features) != 1 || features[0].State != feature.StateReleased || features[0].Release != "5.3" {
		t.Errorf("ListApprovedFeatures() = %+v, want LOGIN Released in release 5.3", features)
	}

	exists, err = gitClient.BranchExists(ctx, "feature/LOGIN")
	if err != nil || exists {
		t.Errorf("BranchExists(feature/LOGIN) after release finish = %v, %v, want false, nil (deleted once the release completes)", exists, err)
	}
}

func TestDeferFeatureKeepsItOutOfNextReleaseUntilReAdded(t *testing.T) {
	_, _, svc, gitflowSvc, _ := newInitializedRepo(t)
	ctx := context.Background()

	if _, err := gitflowSvc.FeatureStart(ctx, "REPORTS"); err != nil {
		t.Fatalf("FeatureStart() error = %v", err)
	}
	if err := svc.RegisterFeatureCreated(ctx, "REPORTS", "feature/REPORTS"); err != nil {
		t.Fatalf("RegisterFeatureCreated() error = %v", err)
	}
	if err := svc.ApproveFeature(ctx, "REPORTS"); err != nil {
		t.Fatalf("ApproveFeature() error = %v", err)
	}

	if _, err := svc.StartRelease(ctx, "5.3"); err != nil {
		t.Fatalf("StartRelease() error = %v", err)
	}
	if err := svc.DeferFeature(ctx, "REPORTS"); err != nil {
		t.Fatalf("DeferFeature() error = %v", err)
	}

	status, err := svc.FeatureStatus(ctx)
	if err != nil {
		t.Fatalf("FeatureStatus() error = %v", err)
	}
	if len(status.Deferred) != 1 || status.Deferred[0] != "REPORTS" {
		t.Errorf("FeatureStatus().Deferred = %v, want [\"REPORTS\"]", status.Deferred)
	}
	if len(status.Pending) != 0 {
		t.Errorf("FeatureStatus().Pending = %v, want empty (REPORTS is explicitly deferred, not undecided)", status.Pending)
	}

	// PM changes their mind before the release closes: re-adding must work.
	if err := svc.AddFeatureToRelease(ctx, "REPORTS"); err != nil {
		t.Fatalf("AddFeatureToRelease() after defer error = %v", err)
	}
	status, err = svc.FeatureStatus(ctx)
	if err != nil {
		t.Fatalf("FeatureStatus() error = %v", err)
	}
	if len(status.Included) != 1 || status.Included[0] != "REPORTS" {
		t.Errorf("FeatureStatus().Included = %v, want [\"REPORTS\"]", status.Included)
	}
	if len(status.Deferred) != 0 {
		t.Errorf("FeatureStatus().Deferred = %v, want empty after re-adding", status.Deferred)
	}
}

func TestFinishReleaseNoActiveReleaseErrorsRealRepo(t *testing.T) {
	_, _, svc, _, _ := newInitializedRepo(t)

	_, err := svc.FinishRelease(context.Background(), "5.2")
	if !errors.Is(err, release.ErrNoActiveRelease) {
		t.Errorf("FinishRelease() error = %v, want ErrNoActiveRelease", err)
	}
}

// --- Hooks (real hooks.Runner, real script) ---

func TestStartReleaseAndBuildTriggerPostQATagHook(t *testing.T) {
	dir, gitClient, svc, gitflowSvc, _ := newInitializedRepo(t)
	ctx := context.Background()

	// Hook scripts live in .gitflowplus/hooks/ and must be committed on
	// staging (where QA build events run from) to be found.
	hookDir := filepath.Join(dir, ".gitflowplus", "hooks")
	if err := os.MkdirAll(hookDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	markerPath := filepath.Join(dir, "..", "hook-marker.txt")
	markerPath, _ = filepath.Abs(markerPath)
	script := "#!/bin/sh\necho \"$GITFLOWPLUS_EVENT $GITFLOWPLUS_TAG\" >> \"" + markerPath + "\"\n"
	if err := os.WriteFile(filepath.Join(hookDir, "post-qa-tag.sh"), []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	defer os.Remove(markerPath)

	if _, err := svc.StartRelease(ctx, "5.2"); err != nil {
		t.Fatalf("StartRelease() error = %v", err)
	}

	data, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatalf("hook did not run during StartRelease: reading marker: %v", err)
	}
	if !bytes.Contains(data, []byte("qa-build v5.2.0.0.1")) {
		t.Errorf("marker content = %q, want it to record the StartRelease QA tag", data)
	}

	if _, err := gitflowSvc.ReleaseFixStart(ctx, "BUG-101"); err != nil {
		t.Fatalf("ReleaseFixStart() error = %v", err)
	}
	writeAndCommit(t, dir, gitClient, "bugfix.txt", "fixed", "Fix BUG-101")
	if _, err := svc.ReleaseFixFinish(ctx, "BUG-101"); err != nil {
		t.Fatalf("ReleaseFixFinish() error = %v", err)
	}
	if _, err := svc.Build(ctx); err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	data, err = os.ReadFile(markerPath)
	if err != nil {
		t.Fatalf("reading marker after Build(): %v", err)
	}
	if !bytes.Contains(data, []byte("qa-build v5.2.1.0.2")) {
		t.Errorf("marker content = %q, want it to also record the Build QA tag", data)
	}
}
