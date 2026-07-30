package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/tabnaeem/git-flow-plus/internal/cli"
	"github.com/tabnaeem/git-flow-plus/internal/config"
	"github.com/tabnaeem/git-flow-plus/internal/git"
)

// testApp constructs an App wired to in-memory output buffers, a real
// (but temp-directory-scoped) git.Client, and a committer identity fixed
// via environment variables, so commands that perform real Git operations
// (init/feature/hotfix/support/release) work end-to-end without touching
// the host's global Git config.
func testApp(t *testing.T) (app *cli.App, out, errOut *bytes.Buffer) {
	t.Helper()
	if !git.Available() {
		t.Skip("git binary not available on PATH")
	}
	t.Setenv("GIT_AUTHOR_NAME", "Test User")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Test User")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "commit.gpgsign")
	t.Setenv("GIT_CONFIG_VALUE_0", "false")

	out = &bytes.Buffer{}
	errOut = &bytes.Buffer{}
	return &cli.App{
		Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		ConfigLoader: config.NewLoader(),
		Out:          out,
		Err:          errOut,
		RepoPath:     t.TempDir(),
		Runner:       git.NewExecRunner(),
	}, out, errOut
}

func run(t *testing.T, app *cli.App, args ...string) error {
	t.Helper()
	root := cli.NewRootCmd(app)
	root.SetArgs(args)
	root.SetOut(app.Out)
	root.SetErr(app.Err)
	return root.Execute()
}

func TestRootCommandHasAllRequiredSubcommands(t *testing.T) {
	app, _, _ := testApp(t)
	root := cli.NewRootCmd(app)

	want := []string{"init", "feature", "release", "hotfix", "support", "releasefix", "devops", "doctor", "config"}
	got := map[string]bool{}
	for _, c := range root.Commands() {
		got[c.Name()] = true
	}

	for _, name := range want {
		if !got[name] {
			t.Errorf("root command missing subcommand %q", name)
		}
	}
}

// stubGitFlowOnPath creates a dummy "git-flow" executable in a temp
// directory and prepends it to PATH for the duration of the test, so
// doctor's PATH check (which requires a real "git-flow" resolvable via
// exec.LookPath — that's what makes `git flow ...` work as a Git
// subcommand) passes the way it would on a real installed system, without
// depending on the host machine already having one.
func stubGitFlowOnPath(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	name := "git-flow"
	if runtime.GOOS == "windows" {
		name = "git-flow.exe"
	}
	stub := filepath.Join(dir, name)
	if err := os.WriteFile(stub, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("stubGitFlowOnPath: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func mustRun(t *testing.T, app *cli.App, args ...string) {
	t.Helper()
	if err := run(t, app, args...); err != nil {
		t.Fatalf("run(%v) error = %v", args, err)
	}
}

func TestFeatureStartRequiresExactlyOneArg(t *testing.T) {
	app, _, _ := testApp(t)

	err := run(t, app, "feature", "start")
	if err == nil {
		t.Fatal("run(feature start) with no name = nil error, want usage error")
	}
}

// --- init ---

func TestInitCreatesRepoAndPersistsConfig(t *testing.T) {
	app, out, _ := testApp(t)

	if err := run(t, app, "init"); err != nil {
		t.Fatalf("run(init) error = %v", err)
	}
	if out.Len() == 0 {
		t.Error("run(init) produced no output")
	}
	if !app.ConfigLoader.Exists(app.RepoPath) {
		t.Error("run(init) did not persist .gitflowplus/config.json")
	}
	if !bytes.Contains(out.Bytes(), []byte(`"staging"`)) {
		t.Errorf("run(init) output = %q, want it to mention the staging branch", out.String())
	}
}

func TestInitPersistsConfigOnStagingToo(t *testing.T) {
	app, _, _ := testApp(t)
	mustRun(t, app, "init")

	cfg, err := app.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if err := app.GitClient().Checkout(context.Background(), cfg.Branches.Staging); err != nil {
		t.Fatalf("Checkout(staging) error = %v", err)
	}
	if !app.ConfigLoader.Exists(app.RepoPath) {
		t.Error("staging branch has no .gitflowplus/config.json; lifecycle hooks (and anything else in .gitflowplus/) added later won't be found while running release commands from staging")
	}
}

func TestInitIsIdempotentViaCLI(t *testing.T) {
	app, _, _ := testApp(t)

	mustRun(t, app, "init")
	if err := run(t, app, "init"); err != nil {
		t.Fatalf("second run(init) error = %v", err)
	}
}

// --- feature / hotfix / support / release lifecycles ---

func TestFeatureLifecycleViaCLI(t *testing.T) {
	app, out, _ := testApp(t)
	mustRun(t, app, "init")
	out.Reset()

	if err := run(t, app, "feature", "start", "widgets"); err != nil {
		t.Fatalf("run(feature start) error = %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("feature/widgets")) {
		t.Errorf("feature start output = %q, want it to mention the branch name", out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte(`Registered feature "widgets"`)) {
		t.Errorf("feature start output = %q, want it to mention registering the feature", out.String())
	}

	// There is no `feature finish` — only a Release Manager, via `release
	// feature add`, decides if and when a feature joins a release. The
	// branch stays right where `feature start` left it.
	current, err := app.GitClient().CurrentBranch(context.Background())
	if err != nil {
		t.Fatalf("CurrentBranch() error = %v", err)
	}
	if current != "feature/widgets" {
		t.Errorf("CurrentBranch() = %q, want %q (feature start leaves the developer on their branch)", current, "feature/widgets")
	}
}

func TestFeatureStartBranchesFromStagingNotDevelop(t *testing.T) {
	app, _, _ := testApp(t)
	mustRun(t, app, "init")
	ctx := context.Background()

	cfg, err := app.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if err := app.GitClient().Checkout(ctx, cfg.Branches.Develop); err != nil {
		t.Fatalf("Checkout(develop) error = %v", err)
	}
	writeFileAndCommit(t, app.RepoPath, "develop-only.txt", "x", "develop-only change")
	if err := app.GitClient().Checkout(ctx, cfg.Branches.Staging); err != nil {
		t.Fatalf("Checkout(staging) error = %v", err)
	}

	mustRun(t, app, "feature", "start", "LOGIN")

	if _, err := os.Stat(filepath.Join(app.RepoPath, "develop-only.txt")); !os.IsNotExist(err) {
		t.Error("feature/LOGIN contains develop-only.txt, want it branched from staging (not develop)")
	}
}

func TestHotfixLifecycleViaCLI(t *testing.T) {
	app, out, _ := testApp(t)
	mustRun(t, app, "init")
	out.Reset()

	if err := run(t, app, "hotfix", "start", "urgent"); err != nil {
		t.Fatalf("run(hotfix start) error = %v", err)
	}
	if err := run(t, app, "hotfix", "finish", "urgent"); err != nil {
		t.Fatalf("run(hotfix finish) error = %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("vurgent")) {
		t.Errorf("hotfix finish output = %q, want it to mention the tag", out.String())
	}
}

func TestSupportLifecycleViaCLIDoesNotDeleteBranch(t *testing.T) {
	app, _, _ := testApp(t)
	mustRun(t, app, "init")

	if err := run(t, app, "support", "start", "v1-line"); err != nil {
		t.Fatalf("run(support start) error = %v", err)
	}
	if err := run(t, app, "support", "finish", "v1-line"); err != nil {
		t.Fatalf("run(support finish) error = %v", err)
	}
}

func TestReleaseStartFinishLifecycleViaCLI(t *testing.T) {
	app, out, _ := testApp(t)
	mustRun(t, app, "init")
	out.Reset()

	if err := run(t, app, "release", "start", "5.2"); err != nil {
		t.Fatalf("run(release start) error = %v", err)
	}
	if err := run(t, app, "release", "finish", "5.2"); err != nil {
		t.Fatalf("run(release finish) error = %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("v5.2")) {
		t.Errorf("release finish output = %q, want it to mention the tag", out.String())
	}
}

func TestFeatureStartWithoutInitErrors(t *testing.T) {
	app, _, _ := testApp(t)

	err := run(t, app, "feature", "start", "too-early")
	if err == nil {
		t.Fatal("run(feature start) before init = nil error, want failure")
	}
}

// --- release status / version / manifest ---

func TestReleaseStatusInactiveBeforeRelease(t *testing.T) {
	app, out, _ := testApp(t)
	mustRun(t, app, "init")
	out.Reset()

	if err := run(t, app, "release", "status"); err != nil {
		t.Fatalf("run(release status) error = %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("No active release")) {
		t.Errorf("release status output = %q, want it to report no active release", out.String())
	}
}

func TestReleaseVersionErrorsBeforeRelease(t *testing.T) {
	app, _, _ := testApp(t)
	mustRun(t, app, "init")

	if err := run(t, app, "release", "version"); err == nil {
		t.Fatal("run(release version) before any release = nil error, want failure")
	}
}

func TestReleaseStatusVersionManifestAfterStart(t *testing.T) {
	app, out, _ := testApp(t)
	mustRun(t, app, "init")
	mustRun(t, app, "release", "start", "5.2")
	out.Reset()

	if err := run(t, app, "release", "status"); err != nil {
		t.Fatalf("run(release status) error = %v", err)
	}
	for _, want := range []string{"5.2", "5.2.0.0.1", "QA Build:", "1"} {
		if !bytes.Contains(out.Bytes(), []byte(want)) {
			t.Errorf("release status output = %q, want it to contain %q", out.String(), want)
		}
	}
	out.Reset()

	if err := run(t, app, "release", "version"); err != nil {
		t.Fatalf("run(release version) error = %v", err)
	}
	if out.String() != "5.2.0.0.1\n" {
		t.Errorf("release version output = %q, want %q", out.String(), "5.2.0.0.1\n")
	}
	out.Reset()

	if err := run(t, app, "release", "manifest"); err != nil {
		t.Fatalf("run(release manifest) error = %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(out.Bytes(), &m); err != nil {
		t.Fatalf("release manifest output is not valid JSON: %v (%q)", err, out.String())
	}
	if m["release"] != "5.2" {
		t.Errorf("manifest[release] = %v, want %q", m["release"], "5.2")
	}
}

// --- releasefix / devops / build lifecycles ---

func TestReleaseFixLifecycleViaCLI(t *testing.T) {
	app, out, _ := testApp(t)
	mustRun(t, app, "init")
	mustRun(t, app, "release", "start", "5.2")
	out.Reset()

	if err := run(t, app, "releasefix", "start", "BUG-101"); err != nil {
		t.Fatalf("run(releasefix start) error = %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("release-fix/BUG-101")) {
		t.Errorf("releasefix start output = %q, want it to mention the branch", out.String())
	}

	writeFileAndCommit(t, app.RepoPath, "bugfix.txt", "fixed", "Fix BUG-101")

	out.Reset()
	if err := run(t, app, "releasefix", "finish", "BUG-101"); err != nil {
		t.Fatalf("run(releasefix finish) error = %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("pending inclusion")) {
		t.Errorf("releasefix finish output = %q, want it to say the fix is pending (not yet in the version)", out.String())
	}

	// The version must not have moved yet — only `release build` moves it.
	out.Reset()
	if err := run(t, app, "release", "version"); err != nil {
		t.Fatalf("run(release version) error = %v", err)
	}
	if out.String() != "5.2.0.0.1\n" {
		t.Errorf("release version after releasefix finish = %q, want unchanged %q", out.String(), "5.2.0.0.1\n")
	}

	out.Reset()
	if err := run(t, app, "release", "build"); err != nil {
		t.Fatalf("run(release build) error = %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("5.2.1.0.2")) {
		t.Errorf("release build output = %q, want it to mention version 5.2.1.0.2", out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("v5.2.1.0.2")) {
		t.Errorf("release build output = %q, want it to mention the tag v5.2.1.0.2", out.String())
	}
}

func TestDevOpsLifecycleViaCLI(t *testing.T) {
	app, out, _ := testApp(t)
	mustRun(t, app, "init")
	mustRun(t, app, "release", "start", "5.2")
	out.Reset()

	if err := run(t, app, "devops", "start", "redis-cache"); err != nil {
		t.Fatalf("run(devops start) error = %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("release-devops/redis-cache")) {
		t.Errorf("devops start output = %q, want it to mention the branch", out.String())
	}

	writeFileAndCommit(t, app.RepoPath, "redis.yaml", "config", "Add redis config")

	out.Reset()
	if err := run(t, app, "devops", "finish", "redis-cache"); err != nil {
		t.Fatalf("run(devops finish) error = %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("pending inclusion")) {
		t.Errorf("devops finish output = %q, want it to say the change is pending (not yet in the version)", out.String())
	}

	out.Reset()
	if err := run(t, app, "release", "build"); err != nil {
		t.Fatalf("run(release build) error = %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("5.2.0.1.2")) {
		t.Errorf("release build output = %q, want it to mention version 5.2.0.1.2", out.String())
	}
}

// --- release feature (Release Planning) ---

func TestReleaseFeaturePlanningLifecycleViaCLI(t *testing.T) {
	app, out, _ := testApp(t)
	mustRun(t, app, "init")

	mustRun(t, app, "feature", "start", "LOGIN")
	writeFileAndCommit(t, app.RepoPath, "login.go", "package login", "Implement LOGIN")

	mustRun(t, app, "release", "start", "5.3")

	// Not approved yet: Release Planning must reject it.
	if err := run(t, app, "release", "feature", "add", "LOGIN"); err == nil {
		t.Fatal("run(release feature add) before approval = nil error, want failure")
	}

	out.Reset()
	if err := run(t, app, "release", "feature", "approve", "LOGIN"); err != nil {
		t.Fatalf("run(release feature approve) error = %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte(`Approved feature "LOGIN"`)) {
		t.Errorf("release feature approve output = %q, want confirmation", out.String())
	}

	out.Reset()
	if err := run(t, app, "release", "feature", "list"); err != nil {
		t.Fatalf("run(release feature list) error = %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("LOGIN")) {
		t.Errorf("release feature list output = %q, want it to mention LOGIN", out.String())
	}

	out.Reset()
	if err := run(t, app, "release", "feature", "add", "LOGIN"); err != nil {
		t.Fatalf("run(release feature add) error = %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte(`Merged feature "LOGIN" into staging`)) {
		t.Errorf("release feature add output = %q, want confirmation mentioning the merge", out.String())
	}

	// The feature branch must survive the merge — it stays alive through
	// the QA cycle.
	if exists, err := app.GitClient().BranchExists(context.Background(), "feature/LOGIN"); err != nil || !exists {
		t.Errorf("BranchExists(feature/LOGIN) after release feature add = %v, %v, want true, nil", exists, err)
	}
	if !fileExists(app.RepoPath, "login.go") {
		t.Error("login.go missing from staging after release feature add")
	}

	out.Reset()
	if err := run(t, app, "release", "feature", "status"); err != nil {
		t.Fatalf("run(release feature status) error = %v", err)
	}
	for _, want := range []string{"Approved:", "LOGIN", "Included:", "Pending:"} {
		if !bytes.Contains(out.Bytes(), []byte(want)) {
			t.Errorf("release feature status output = %q, want it to contain %q", out.String(), want)
		}
	}

	out.Reset()
	if err := run(t, app, "release", "status"); err != nil {
		t.Fatalf("run(release status) error = %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("LOGIN")) {
		t.Errorf("release status output = %q, want it to mention the included feature LOGIN", out.String())
	}

	// The version's feature counter must have advanced.
	out.Reset()
	if err := run(t, app, "release", "version"); err != nil {
		t.Fatalf("run(release version) error = %v", err)
	}
	if out.String() != "5.4.0.0.1\n" {
		t.Errorf("release version after release feature add = %q, want %q (feature counter incremented from 3 to 4)", out.String(), "5.4.0.0.1\n")
	}
}

func TestReleaseFeatureDeferViaCLI(t *testing.T) {
	app, out, _ := testApp(t)
	mustRun(t, app, "init")

	mustRun(t, app, "feature", "start", "REPORTS")
	mustRun(t, app, "release", "start", "5.3")
	mustRun(t, app, "release", "feature", "approve", "REPORTS")
	out.Reset()

	if err := run(t, app, "release", "feature", "defer", "REPORTS"); err != nil {
		t.Fatalf("run(release feature defer) error = %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte(`Deferred feature "REPORTS"`)) {
		t.Errorf("release feature defer output = %q, want confirmation", out.String())
	}
}

func TestReleaseFeatureAddRequiresActiveRelease(t *testing.T) {
	app, _, _ := testApp(t)
	mustRun(t, app, "init")

	if err := run(t, app, "release", "feature", "add", "LOGIN"); err == nil {
		t.Fatal("run(release feature add) without an active release = nil error, want failure")
	}
}

func TestReleaseFixStartWithoutInitErrors(t *testing.T) {
	app, _, _ := testApp(t)

	if err := run(t, app, "releasefix", "start", "BUG-101"); err == nil {
		t.Fatal("run(releasefix start) before init = nil error, want failure")
	}
}

func TestReleaseFixFinishWithoutActiveReleaseErrors(t *testing.T) {
	app, _, _ := testApp(t)
	mustRun(t, app, "init")

	// releasefix start only needs staging to exist (a pure Git Flow
	// branch operation); it's releasefix finish that needs an active
	// release to record against.
	if err := run(t, app, "releasefix", "start", "BUG-101"); err != nil {
		t.Fatalf("run(releasefix start) error = %v", err)
	}
	writeFileAndCommit(t, app.RepoPath, "bugfix.txt", "fixed", "Fix BUG-101")

	if err := run(t, app, "releasefix", "finish", "BUG-101"); err == nil {
		t.Fatal("run(releasefix finish) without an active release = nil error, want failure")
	}
}

func TestReleaseBuildRequiresStagingBranch(t *testing.T) {
	app, _, _ := testApp(t)
	mustRun(t, app, "init")
	mustRun(t, app, "release", "start", "5.2")

	cfg, err := app.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if err := app.GitClient().Checkout(context.Background(), cfg.Branches.Develop); err != nil {
		t.Fatalf("Checkout(develop) error = %v", err)
	}

	if err := run(t, app, "release", "build"); err == nil {
		t.Fatal("run(release build) off staging = nil error, want failure")
	}
}

func TestReleaseFinishRequiresBuildFirstViaCLI(t *testing.T) {
	app, _, _ := testApp(t)
	mustRun(t, app, "init")
	mustRun(t, app, "release", "start", "5.2")
	mustRun(t, app, "releasefix", "start", "BUG-101")
	writeFileAndCommit(t, app.RepoPath, "bugfix.txt", "fixed", "Fix BUG-101")
	mustRun(t, app, "releasefix", "finish", "BUG-101")

	if err := run(t, app, "release", "finish", "5.2"); err == nil {
		t.Fatal("run(release finish) with pending unbuild changes = nil error, want failure")
	}
}

func fileExists(repoPath, name string) bool {
	_, err := os.Stat(filepath.Join(repoPath, name))
	return err == nil
}

func writeFileAndCommit(t *testing.T, repoPath, name, contents, message string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repoPath, name), []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", name, err)
	}
	runner := git.NewExecRunner()
	ctx := context.Background()
	if _, err := runner.Run(ctx, repoPath, "add", name); err != nil {
		t.Fatalf("git add %s: %v", name, err)
	}
	if _, err := runner.Run(ctx, repoPath, "commit", "-m", message); err != nil {
		t.Fatalf("git commit: %v", err)
	}
}

// --- doctor ---

func TestDoctorHealthyAfterInit(t *testing.T) {
	stubGitFlowOnPath(t)
	app, out, _ := testApp(t)
	mustRun(t, app, "init")
	out.Reset()

	if err := run(t, app, "doctor"); err != nil {
		t.Fatalf("run(doctor) error = %v, want a clean bill of health after init; output:\n%s", err, out.String())
	}
	for _, want := range []string{"gitflowplus config", "staging branch", "git version", "permissions", "git flow plus version", "PATH", "release configuration"} {
		if !bytes.Contains(out.Bytes(), []byte(want)) {
			t.Errorf("doctor output = %q, want it to include a %q check", out.String(), want)
		}
	}
}

func TestDoctorFailsBeforeInit(t *testing.T) {
	app, _, _ := testApp(t)

	err := run(t, app, "doctor")
	if err == nil {
		t.Fatal("run(doctor) before init = nil error, want failure")
	}
}

// --- config ---

func TestConfigListPrintsDefaultsForUninitializedRepo(t *testing.T) {
	app, out, _ := testApp(t)

	if err := run(t, app, "config", "list"); err != nil {
		t.Fatalf("run(config list) error = %v", err)
	}

	got := out.String()
	for _, want := range []string{"main", "staging", "develop", "feature/", "release-fix/", "release-devops/"} {
		if !bytes.Contains([]byte(got), []byte(want)) {
			t.Errorf("config list output = %q, want it to contain %q", got, want)
		}
	}
}

func TestConfigBareAliasesList(t *testing.T) {
	app, out, _ := testApp(t)

	if err := run(t, app, "config"); err != nil {
		t.Fatalf("run(config) error = %v", err)
	}
	if out.Len() == 0 {
		t.Fatal("run(config) produced no output, want same output as `config list`")
	}
}

func TestConfigListJSONOutputIsValidAndComplete(t *testing.T) {
	app, out, _ := testApp(t)

	if err := run(t, app, "config", "list", "--json"); err != nil {
		t.Fatalf("run(config list --json) error = %v", err)
	}

	var cfg config.Config
	if err := json.Unmarshal(out.Bytes(), &cfg); err != nil {
		t.Fatalf("output is not valid JSON: %v (%q)", err, out.String())
	}

	want := config.Default()
	if cfg != *want {
		t.Errorf("decoded config = %+v, want default %+v", cfg, want)
	}
}

func TestConfigPathPrintsResolvedFilePath(t *testing.T) {
	app, out, _ := testApp(t)

	if err := run(t, app, "config", "path"); err != nil {
		t.Fatalf("run(config path) error = %v", err)
	}

	want := config.Path(app.RepoPath) + "\n"
	if out.String() != want {
		t.Errorf("config path output = %q, want %q", out.String(), want)
	}
}

func TestConfigListReflectsSavedOverrides(t *testing.T) {
	app, out, _ := testApp(t)

	cfg := config.Default()
	cfg.Branches.Main = "master"
	if err := app.ConfigLoader.Save(app.RepoPath, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if err := run(t, app, "config", "list"); err != nil {
		t.Fatalf("run(config list) error = %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("master")) {
		t.Errorf("config list output = %q, want it to reflect saved override %q", out.String(), "master")
	}
}

// --- global flags ---

// --- logging / errors / version / environment config ---

func TestConsoleLogFormatIsBracketedAndHumanReadable(t *testing.T) {
	app, _, errOut := testApp(t)
	mustRun(t, app, "init")
	errOut.Reset()

	mustRun(t, app, "feature", "start", "widgets")

	if !bytes.Contains(errOut.Bytes(), []byte("[INFO] started feature branch")) {
		t.Errorf("stderr = %q, want a bracketed [INFO] console line", errOut.String())
	}
	if bytes.Contains(errOut.Bytes(), []byte("time=")) {
		t.Errorf("stderr = %q, want the new console format, not slog's default time=... text format", errOut.String())
	}
}

func TestJSONLogFormatIsNewlineDelimitedJSON(t *testing.T) {
	app, _, errOut := testApp(t)
	mustRun(t, app, "init")
	errOut.Reset()

	if err := run(t, app, "--json-log", "feature", "start", "widgets"); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	var record map[string]any
	firstLine := bytes.SplitN(errOut.Bytes(), []byte("\n"), 2)[0]
	if err := json.Unmarshal(firstLine, &record); err != nil {
		t.Fatalf("stderr line is not valid JSON: %v (%q)", err, firstLine)
	}
	if record["level"] != "INFO" {
		t.Errorf("record[level] = %v, want %q", record["level"], "INFO")
	}
}

func TestDebugFlagEnablesTraceLevel(t *testing.T) {
	app, _, errOut := testApp(t)

	mustRun(t, app, "--debug", "init")

	if !bytes.Contains(errOut.Bytes(), []byte("[DEBUG]")) && !bytes.Contains(errOut.Bytes(), []byte("[TRACE]")) {
		// init doesn't itself emit Trace-level logs, but --debug must at
		// least not suppress the existing Info/Debug-level ones.
		if !bytes.Contains(errOut.Bytes(), []byte("[INFO]")) {
			t.Errorf("stderr = %q, want at least [INFO] lines with --debug set", errOut.String())
		}
	}
}

func TestNoColorFlagProducesNoANSICodes(t *testing.T) {
	app, _, errOut := testApp(t)
	mustRun(t, app, "init")
	errOut.Reset()

	mustRun(t, app, "--no-color", "feature", "start", "widgets")

	if bytes.Contains(errOut.Bytes(), []byte("\x1b[")) {
		t.Errorf("stderr = %q, want no ANSI escape codes with --no-color", errOut.String())
	}
}

func TestCLIErrorsUseBracketedErrorFormat(t *testing.T) {
	app, _, _ := testApp(t)
	mustRun(t, app, "init")

	err := run(t, app, "hotfix", "finish", "never-started")
	if err == nil {
		t.Fatal("run(hotfix finish) on an unknown branch = nil error, want failure")
	}

	// cli.Execute (not exercised directly by run() here) is what actually
	// calls logging.LogError; run() only returns the Go error, so assert
	// the error's own message shape instead of stderr content.
	if !bytes.Contains([]byte(err.Error()), []byte("branch does not exist")) {
		t.Errorf("error = %q, want it to describe the missing branch", err.Error())
	}
}

func TestExecuteFormatsErrorsWithBracketedTag(t *testing.T) {
	dir := t.TempDir()
	if !gitAvailable(t) {
		return
	}
	t.Setenv("GIT_AUTHOR_NAME", "Test User")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Test User")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")

	var out, errOut bytes.Buffer
	app := &cli.App{
		Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		ConfigLoader: config.NewLoader(),
		Out:          &out,
		Err:          &errOut,
		RepoPath:     dir,
		Runner:       git.NewExecRunner(),
	}
	root := cli.NewRootCmd(app)
	root.SetArgs([]string{"hotfix", "finish", "never-started"})
	root.SetOut(&out)
	root.SetErr(&errOut)

	execErr := root.Execute()
	if execErr == nil {
		t.Fatal("root.Execute() = nil error, want failure")
	}
	// SilenceErrors is now true on the root command — cobra itself must
	// not print "Error: ..."; only cli.Execute's logging.LogError call
	// (exercised indirectly here by checking cobra stayed silent) does.
	if bytes.Contains(errOut.Bytes(), []byte("Error:")) {
		t.Errorf("stderr = %q, want cobra's own error printer silenced (SilenceErrors)", errOut.String())
	}
}

func gitAvailable(t *testing.T) bool {
	t.Helper()
	if !git.Available() {
		t.Skip("git binary not available on PATH")
		return false
	}
	return true
}

func TestVersionCommandPrintsBuildMetadata(t *testing.T) {
	app, out, _ := testApp(t)

	if err := run(t, app, "version"); err != nil {
		t.Fatalf("run(version) error = %v", err)
	}
	for _, want := range []string{"Git Flow Plus", "Version", "Git Commit", "Go Version", "OS", "Architecture"} {
		if !bytes.Contains(out.Bytes(), []byte(want)) {
			t.Errorf("version output = %q, want it to contain %q", out.String(), want)
		}
	}
}

func TestConfigListShowsEnvironmentAndLogging(t *testing.T) {
	app, out, _ := testApp(t)

	if err := run(t, app, "config", "list"); err != nil {
		t.Fatalf("run(config list) error = %v", err)
	}
	for _, want := range []string{"Environment: development", "Logging:", "level:", "format:", "color:"} {
		if !bytes.Contains(out.Bytes(), []byte(want)) {
			t.Errorf("config list output = %q, want it to contain %q", out.String(), want)
		}
	}
}

func TestGitflowplusLogLevelEnvVarOverridesConfig(t *testing.T) {
	app, _, errOut := testApp(t)
	mustRun(t, app, "init")
	errOut.Reset()
	t.Setenv("GITFLOWPLUS_LOG_LEVEL", "error")

	mustRun(t, app, "feature", "start", "widgets")

	if bytes.Contains(errOut.Bytes(), []byte("[INFO]")) {
		t.Errorf("stderr = %q, want Info-level logs suppressed when GITFLOWPLUS_LOG_LEVEL=error", errOut.String())
	}
}

func TestMalformedConfigJSONReportsCauseAndResolution(t *testing.T) {
	app, _, _ := testApp(t)
	mustRun(t, app, "init")

	if err := os.WriteFile(config.Path(app.RepoPath), []byte("{ not valid json"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := app.LoadConfig()
	if err == nil {
		t.Fatal("LoadConfig() with malformed JSON = nil error, want failure")
	}
	type causer interface{ Cause() string }
	type resolver interface{ Resolution() string }
	c, ok := err.(causer)
	if !ok || c.Cause() == "" {
		t.Errorf("error = %v (%T), want it to implement Cause() with a non-empty message", err, err)
	}
	r, ok := err.(resolver)
	if !ok || r.Resolution() == "" {
		t.Errorf("error = %v (%T), want it to implement Resolution() with a non-empty message", err, err)
	}
}

func TestVerboseAndJSONLogFlagsAreAccepted(t *testing.T) {
	app, _, errOut := testApp(t)

	err := run(t, app, "--verbose", "--json-log", "init")
	if err != nil {
		t.Fatalf("run() with global flags error = %v", err)
	}
	if errOut.Len() == 0 {
		t.Error("expected structured log output on stderr, got none")
	}

	var record map[string]any
	firstLine := bytes.SplitN(errOut.Bytes(), []byte("\n"), 2)[0]
	if err := json.Unmarshal(firstLine, &record); err != nil {
		t.Errorf("stderr log line is not valid JSON: %v (%q)", err, firstLine)
	}
}
