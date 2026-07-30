package git_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	gitpkg "github.com/tabnaeem/git-flow-plus/internal/git"
)

type call struct {
	dir  string
	args []string
}

type fakeRunner struct {
	calls   []call
	runFunc func(dir string, args []string) (string, error)
}

func (f *fakeRunner) Run(_ context.Context, dir string, args ...string) (string, error) {
	f.calls = append(f.calls, call{dir: dir, args: args})
	if f.runFunc != nil {
		return f.runFunc(dir, args)
	}
	return "", nil
}

func (f *fakeRunner) lastArgs(t *testing.T) []string {
	t.Helper()
	if len(f.calls) == 0 {
		t.Fatal("no calls recorded")
	}
	return f.calls[len(f.calls)-1].args
}

func equalArgs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestClientArgConstruction(t *testing.T) {
	cases := []struct {
		name string
		call func(t *testing.T, c gitpkg.Client)
		want []string
	}{
		{"Init", func(t *testing.T, c gitpkg.Client) { _ = c.Init(context.Background()) }, []string{"init"}},
		{"CreateBranchWithStartPoint", func(t *testing.T, c gitpkg.Client) {
			_ = c.CreateBranch(context.Background(), "feature/x", "develop")
		}, []string{"checkout", "-b", "feature/x", "develop"}},
		{"CreateBranchNoStartPoint", func(t *testing.T, c gitpkg.Client) {
			_ = c.CreateBranch(context.Background(), "feature/x", "")
		}, []string{"checkout", "-b", "feature/x"}},
		{"RenameCurrentBranch", func(t *testing.T, c gitpkg.Client) {
			_ = c.RenameCurrentBranch(context.Background(), "main")
		}, []string{"branch", "-M", "main"}},
		{"Checkout", func(t *testing.T, c gitpkg.Client) {
			_ = c.Checkout(context.Background(), "develop")
		}, []string{"checkout", "develop"}},
		{"Add", func(t *testing.T, c gitpkg.Client) {
			_ = c.Add(context.Background(), ".gitflowplus", "README.md")
		}, []string{"add", ".gitflowplus", "README.md"}},
		{"TagCommit", func(t *testing.T, c gitpkg.Client) {
			_ = c.TagCommit(context.Background(), "v1.0", "abc123", "Release v1.0")
		}, []string{"tag", "-a", "v1.0", "-m", "Release v1.0", "abc123"}},
		{"CommitSHA", func(t *testing.T, c gitpkg.Client) {
			_, _ = c.CommitSHA(context.Background())
		}, []string{"rev-parse", "HEAD"}},
		{"ConfigValue", func(t *testing.T, c gitpkg.Client) {
			_, _ = c.ConfigValue(context.Background(), "user.name")
		}, []string{"config", "--get", "user.name"}},
		{"DeleteBranchSoft", func(t *testing.T, c gitpkg.Client) {
			_ = c.DeleteBranch(context.Background(), "feature/x", false)
		}, []string{"branch", "-d", "feature/x"}},
		{"DeleteBranchForce", func(t *testing.T, c gitpkg.Client) {
			_ = c.DeleteBranch(context.Background(), "feature/x", true)
		}, []string{"branch", "-D", "feature/x"}},
		{"MergeNoFF", func(t *testing.T, c gitpkg.Client) {
			_ = c.MergeNoFF(context.Background(), "feature/x", "Merge feature/x")
		}, []string{"merge", "--no-ff", "-m", "Merge feature/x", "feature/x"}},
		{"Tag", func(t *testing.T, c gitpkg.Client) {
			_ = c.Tag(context.Background(), "v1.0", "Release v1.0")
		}, []string{"tag", "-a", "v1.0", "-m", "Release v1.0"}},
		{"CommitPlain", func(t *testing.T, c gitpkg.Client) {
			_ = c.Commit(context.Background(), "msg", false)
		}, []string{"commit", "-m", "msg"}},
		{"CommitAllowEmpty", func(t *testing.T, c gitpkg.Client) {
			_ = c.Commit(context.Background(), "msg", true)
		}, []string{"commit", "-m", "msg", "--allow-empty"}},
		{"Push", func(t *testing.T, c gitpkg.Client) {
			_ = c.Push(context.Background(), "origin", "main")
		}, []string{"push", "origin", "main"}},
		{"Pull", func(t *testing.T, c gitpkg.Client) {
			_ = c.Pull(context.Background(), "origin", "main")
		}, []string{"pull", "origin", "main"}},
		{"Status", func(t *testing.T, c gitpkg.Client) {
			_, _ = c.Status(context.Background())
		}, []string{"status", "--porcelain"}},
		{"Remove", func(t *testing.T, c gitpkg.Client) {
			_ = c.Remove(context.Background(), "a.txt", "b.txt")
		}, []string{"rm", "-r", "a.txt", "b.txt"}},
		{"ListBranches", func(t *testing.T, c gitpkg.Client) {
			_, _ = c.ListBranches(context.Background(), "release/*")
		}, []string{"for-each-ref", "--format=%(refname:short)", "refs/heads/release/*"}},
		{"CurrentBranch", func(t *testing.T, c gitpkg.Client) {
			_, _ = c.CurrentBranch(context.Background())
		}, []string{"rev-parse", "--abbrev-ref", "HEAD"}},
		{"Version", func(t *testing.T, c gitpkg.Client) {
			_, _ = c.Version(context.Background())
		}, []string{"--version"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runner := &fakeRunner{}
			c := gitpkg.NewClient(runner, "/repo")
			tc.call(t, c)

			got := runner.lastArgs(t)
			if !equalArgs(got, tc.want) {
				t.Errorf("args = %v, want %v", got, tc.want)
			}
			if runner.calls[len(runner.calls)-1].dir != "/repo" {
				t.Errorf("dir = %q, want %q", runner.calls[len(runner.calls)-1].dir, "/repo")
			}
		})
	}
}

func TestBranchExistsTrue(t *testing.T) {
	runner := &fakeRunner{runFunc: func(dir string, args []string) (string, error) {
		return "", nil
	}}
	c := gitpkg.NewClient(runner, "/repo")

	ok, err := c.BranchExists(context.Background(), "develop")
	if err != nil {
		t.Fatalf("BranchExists() error = %v", err)
	}
	if !ok {
		t.Error("BranchExists() = false, want true")
	}
	if !equalArgs(runner.lastArgs(t), []string{"show-ref", "--verify", "--quiet", "refs/heads/develop"}) {
		t.Errorf("args = %v", runner.lastArgs(t))
	}
}

func TestBranchExistsFalseOnExitCode1(t *testing.T) {
	runner := &fakeRunner{runFunc: func(dir string, args []string) (string, error) {
		return "", &gitpkg.CommandError{Args: args, ExitCode: 1, Err: errors.New("exit status 1")}
	}}
	c := gitpkg.NewClient(runner, "/repo")

	ok, err := c.BranchExists(context.Background(), "missing")
	if err != nil {
		t.Fatalf("BranchExists() error = %v, want nil (not-found is not an error)", err)
	}
	if ok {
		t.Error("BranchExists() = true, want false")
	}
}

func TestBranchExistsPropagatesRealErrors(t *testing.T) {
	wantErr := &gitpkg.CommandError{ExitCode: 128, Stderr: "fatal: not a git repository"}
	runner := &fakeRunner{runFunc: func(dir string, args []string) (string, error) {
		return "", wantErr
	}}
	c := gitpkg.NewClient(runner, "/repo")

	_, err := c.BranchExists(context.Background(), "develop")
	if !errors.Is(err, wantErr) && err.Error() != wantErr.Error() {
		t.Errorf("BranchExists() error = %v, want it to propagate %v", err, wantErr)
	}
}

func TestTagExistsUsesTagRef(t *testing.T) {
	runner := &fakeRunner{}
	c := gitpkg.NewClient(runner, "/repo")

	_, _ = c.TagExists(context.Background(), "v1.0")

	if !equalArgs(runner.lastArgs(t), []string{"show-ref", "--verify", "--quiet", "refs/tags/v1.0"}) {
		t.Errorf("args = %v", runner.lastArgs(t))
	}
}

func TestIsRepoTrueAndFalse(t *testing.T) {
	okRunner := &fakeRunner{runFunc: func(string, []string) (string, error) { return "true", nil }}
	if !gitpkg.NewClient(okRunner, "/repo").IsRepo(context.Background()) {
		t.Error("IsRepo() = false, want true")
	}

	failRunner := &fakeRunner{runFunc: func(string, []string) (string, error) {
		return "", &gitpkg.CommandError{ExitCode: 128}
	}}
	if gitpkg.NewClient(failRunner, "/repo").IsRepo(context.Background()) {
		t.Error("IsRepo() = true, want false")
	}
}

func TestHasCommitsTrueAndFalse(t *testing.T) {
	okRunner := &fakeRunner{}
	if !gitpkg.NewClient(okRunner, "/repo").HasCommits(context.Background()) {
		t.Error("HasCommits() = false, want true")
	}

	failRunner := &fakeRunner{runFunc: func(string, []string) (string, error) {
		return "", &gitpkg.CommandError{ExitCode: 1}
	}}
	if gitpkg.NewClient(failRunner, "/repo").HasCommits(context.Background()) {
		t.Error("HasCommits() = true, want false")
	}
}

func TestStatusCleanAndDirty(t *testing.T) {
	cleanRunner := &fakeRunner{runFunc: func(string, []string) (string, error) { return "", nil }}
	status, err := gitpkg.NewClient(cleanRunner, "/repo").Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if !status.Clean {
		t.Error("Status().Clean = false, want true for empty porcelain output")
	}

	dirtyRunner := &fakeRunner{runFunc: func(string, []string) (string, error) {
		return " M internal/git/client.go", nil
	}}
	status, err = gitpkg.NewClient(dirtyRunner, "/repo").Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status.Clean {
		t.Error("Status().Clean = true, want false for non-empty porcelain output")
	}
}

func TestWritableTrueForWritableDir(t *testing.T) {
	dir := t.TempDir()
	writable, err := gitpkg.NewClient(&fakeRunner{}, dir).Writable(context.Background())
	if err != nil {
		t.Fatalf("Writable() error = %v", err)
	}
	if !writable {
		t.Error("Writable() = false, want true for a fresh temp directory")
	}
}

func TestWritableFalseForNonexistentDir(t *testing.T) {
	writable, err := gitpkg.NewClient(&fakeRunner{}, filepath.Join(t.TempDir(), "does-not-exist")).Writable(context.Background())
	if err != nil {
		t.Fatalf("Writable() error = %v, want a false result rather than an error for a missing directory", err)
	}
	if writable {
		t.Error("Writable() = true, want false for a directory that doesn't exist")
	}
}

func TestListBranchesEmptyReturnsNil(t *testing.T) {
	runner := &fakeRunner{runFunc: func(string, []string) (string, error) { return "", nil }}
	branches, err := gitpkg.NewClient(runner, "/repo").ListBranches(context.Background(), "release/*")
	if err != nil {
		t.Fatalf("ListBranches() error = %v", err)
	}
	if branches != nil {
		t.Errorf("ListBranches() = %v, want nil for no matches", branches)
	}
}

func TestListBranchesSplitsMultipleLines(t *testing.T) {
	runner := &fakeRunner{runFunc: func(string, []string) (string, error) {
		return "release/5.2\nrelease/5.3", nil
	}}
	branches, err := gitpkg.NewClient(runner, "/repo").ListBranches(context.Background(), "release/*")
	if err != nil {
		t.Fatalf("ListBranches() error = %v", err)
	}
	want := []string{"release/5.2", "release/5.3"}
	if !equalArgs(branches, want) {
		t.Errorf("ListBranches() = %v, want %v", branches, want)
	}
}

func TestListBranchesPropagatesError(t *testing.T) {
	runner := &fakeRunner{runFunc: func(string, []string) (string, error) {
		return "", &gitpkg.CommandError{ExitCode: 128, Stderr: "fatal: boom"}
	}}
	_, err := gitpkg.NewClient(runner, "/repo").ListBranches(context.Background(), "release/*")
	if err == nil {
		t.Fatal("ListBranches() error = nil, want failure propagated")
	}
}

func TestConfigValueReturnsEmptyForUnsetKey(t *testing.T) {
	runner := &fakeRunner{runFunc: func(string, []string) (string, error) {
		return "", &gitpkg.CommandError{ExitCode: 1}
	}}
	value, err := gitpkg.NewClient(runner, "/repo").ConfigValue(context.Background(), "user.name")
	if err != nil {
		t.Fatalf("ConfigValue() error = %v, want nil for an unset key", err)
	}
	if value != "" {
		t.Errorf("ConfigValue() = %q, want empty string", value)
	}
}

func TestConfigValuePropagatesRealErrors(t *testing.T) {
	runner := &fakeRunner{runFunc: func(string, []string) (string, error) {
		return "", &gitpkg.CommandError{ExitCode: 128, Stderr: "fatal: boom"}
	}}
	_, err := gitpkg.NewClient(runner, "/repo").ConfigValue(context.Background(), "user.name")
	if err == nil {
		t.Fatal("ConfigValue() error = nil, want failure propagated")
	}
}

func TestCommandErrorMessage(t *testing.T) {
	err := &gitpkg.CommandError{Args: []string{"checkout", "-b", "x"}, Stderr: "fatal: already exists"}
	want := `git checkout -b x: fatal: already exists`
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}

	noStderr := &gitpkg.CommandError{Args: []string{"init"}, Err: errors.New("boom")}
	want = "git init: boom"
	if noStderr.Error() != want {
		t.Errorf("Error() = %q, want %q", noStderr.Error(), want)
	}
}

func TestCommandErrorUnwrap(t *testing.T) {
	inner := errors.New("inner")
	err := &gitpkg.CommandError{Err: inner}
	if !errors.Is(err, inner) {
		t.Error("errors.Is() = false, want true (Unwrap should expose inner error)")
	}
}
