package gitflow_test

import (
	"context"
	"strings"

	gitpkg "github.com/tabnaeem/git-flow-plus/internal/git"
)

// fakeClient is a git.Client test double whose methods default to a
// "happy path" (repo initialized, target branches present, working tree
// clean) so individual tests only need to override the one method whose
// failure they're exercising.
type fakeClient struct {
	isRepo              func() bool
	initFn              func() error
	hasCommits          func() bool
	currentBranch       func() (string, error)
	branchExists        func(name string) (bool, error)
	createBranch        func(name, start string) error
	renameCurrentBranch func(name string) error
	checkout            func(name string) error
	add                 func(paths []string) error
	deleteBranch        func(name string, force bool) error
	mergeNoFF           func(branch, msg string) error
	tag                 func(name, msg string) error
	tagExists           func(name string) (bool, error)
	commit              func(msg string, allowEmpty bool) error
	push                func(remote, branch string) error
	pull                func(remote, branch string) error
	status              func() (gitpkg.Status, error)
	remove              func(paths []string) error
	listBranches        func(pattern string) ([]string, error)
	tagCommit           func(name, commit, msg string) error
	commitSHA           func() (string, error)
	configValue         func(key string) (string, error)
	version             func() (string, error)
	writable            func() (bool, error)
}

// branchExistsFor returns true for "main"/"staging"/"develop" (so
// ensureInitialized passes) and false for anything else (so Start
// operations proceed by default). Tests exercising Finish operations
// override branchExists.
func branchExistsFor(name string) (bool, error) {
	if name == "main" || name == "staging" || name == "develop" {
		return true, nil
	}
	return false, nil
}

func (f *fakeClient) IsRepo(context.Context) bool {
	if f.isRepo != nil {
		return f.isRepo()
	}
	return true
}

func (f *fakeClient) Init(context.Context) error {
	if f.initFn != nil {
		return f.initFn()
	}
	return nil
}

func (f *fakeClient) HasCommits(context.Context) bool {
	if f.hasCommits != nil {
		return f.hasCommits()
	}
	return true
}

func (f *fakeClient) CurrentBranch(context.Context) (string, error) {
	if f.currentBranch != nil {
		return f.currentBranch()
	}
	return "develop", nil
}

func (f *fakeClient) BranchExists(_ context.Context, name string) (bool, error) {
	if f.branchExists != nil {
		return f.branchExists(name)
	}
	return branchExistsFor(name)
}

func (f *fakeClient) CreateBranch(_ context.Context, name, start string) error {
	if f.createBranch != nil {
		return f.createBranch(name, start)
	}
	return nil
}

func (f *fakeClient) RenameCurrentBranch(_ context.Context, name string) error {
	if f.renameCurrentBranch != nil {
		return f.renameCurrentBranch(name)
	}
	return nil
}

func (f *fakeClient) Checkout(_ context.Context, name string) error {
	if f.checkout != nil {
		return f.checkout(name)
	}
	return nil
}

func (f *fakeClient) Add(_ context.Context, paths ...string) error {
	if f.add != nil {
		return f.add(paths)
	}
	return nil
}

func (f *fakeClient) DeleteBranch(_ context.Context, name string, force bool) error {
	if f.deleteBranch != nil {
		return f.deleteBranch(name, force)
	}
	return nil
}

func (f *fakeClient) MergeNoFF(_ context.Context, branch, msg string) error {
	if f.mergeNoFF != nil {
		return f.mergeNoFF(branch, msg)
	}
	return nil
}

func (f *fakeClient) Tag(_ context.Context, name, msg string) error {
	if f.tag != nil {
		return f.tag(name, msg)
	}
	return nil
}

func (f *fakeClient) TagExists(_ context.Context, name string) (bool, error) {
	if f.tagExists != nil {
		return f.tagExists(name)
	}
	return false, nil
}

func (f *fakeClient) Commit(_ context.Context, msg string, allowEmpty bool) error {
	if f.commit != nil {
		return f.commit(msg, allowEmpty)
	}
	return nil
}

func (f *fakeClient) Push(_ context.Context, remote, branch string) error {
	if f.push != nil {
		return f.push(remote, branch)
	}
	return nil
}

func (f *fakeClient) Pull(_ context.Context, remote, branch string) error {
	if f.pull != nil {
		return f.pull(remote, branch)
	}
	return nil
}

func (f *fakeClient) Status(context.Context) (gitpkg.Status, error) {
	if f.status != nil {
		return f.status()
	}
	return gitpkg.Status{Clean: true}, nil
}

func (f *fakeClient) Remove(_ context.Context, paths ...string) error {
	if f.remove != nil {
		return f.remove(paths)
	}
	return nil
}

func (f *fakeClient) ListBranches(_ context.Context, pattern string) ([]string, error) {
	if f.listBranches != nil {
		return f.listBranches(pattern)
	}
	return nil, nil
}

func (f *fakeClient) TagCommit(_ context.Context, name, commit, msg string) error {
	if f.tagCommit != nil {
		return f.tagCommit(name, commit, msg)
	}
	return nil
}

func (f *fakeClient) CommitSHA(context.Context) (string, error) {
	if f.commitSHA != nil {
		return f.commitSHA()
	}
	return "abc123", nil
}

func (f *fakeClient) ConfigValue(_ context.Context, key string) (string, error) {
	if f.configValue != nil {
		return f.configValue(key)
	}
	return "", nil
}

func (f *fakeClient) Version(context.Context) (string, error) {
	if f.version != nil {
		return f.version()
	}
	return "git version 2.43.0", nil
}

func (f *fakeClient) Writable(context.Context) (bool, error) {
	if f.writable != nil {
		return f.writable()
	}
	return true, nil
}

// containsMsg reports whether err's message contains substr; used to assert
// that a low-level failure was propagated (wrapped) rather than swallowed.
func containsMsg(err error, substr string) bool {
	return err != nil && strings.Contains(err.Error(), substr)
}
