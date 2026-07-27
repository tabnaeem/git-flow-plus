package release_test

import (
	"context"

	"github.com/hulhub/git-flow-plus/internal/config"
	"github.com/hulhub/git-flow-plus/internal/feature"
	gitpkg "github.com/hulhub/git-flow-plus/internal/git"
	"github.com/hulhub/git-flow-plus/internal/gitflow"
	"github.com/hulhub/git-flow-plus/internal/release"
	"github.com/hulhub/git-flow-plus/internal/version"
)

// fakeGit is a minimal git.Client test double. Only the methods
// release.Service actually calls directly (BranchExists, Checkout, Status,
// Add, Remove, Commit, ListBranches) have configurable hooks; the rest
// exist solely to satisfy the interface, since release.Service.FinishRelease
// etc. drive gitflow through fakeGitFlow instead.
type fakeGit struct {
	branchExists  func(name string) (bool, error)
	checkout      func(name string) error
	status        func() (gitpkg.Status, error)
	add           func(paths []string) error
	remove        func(paths []string) error
	commit        func(message string) error
	listBranches  func(pattern string) ([]string, error)
	currentBranch func() (string, error)
	tag           func(name, message string) error
	tagCommit     func(name, commit, message string) error
	tagExists     func(name string) (bool, error)
	commitSHA     func() (string, error)
	configValue   func(key string) (string, error)
}

func (f *fakeGit) IsRepo(context.Context) bool     { return true }
func (f *fakeGit) Init(context.Context) error      { return nil }
func (f *fakeGit) HasCommits(context.Context) bool { return true }

func (f *fakeGit) CurrentBranch(context.Context) (string, error) {
	if f.currentBranch != nil {
		return f.currentBranch()
	}
	return "staging", nil
}

func (f *fakeGit) BranchExists(_ context.Context, name string) (bool, error) {
	if f.branchExists != nil {
		return f.branchExists(name)
	}
	return true, nil
}

func (f *fakeGit) CreateBranch(context.Context, string, string) error { return nil }
func (f *fakeGit) RenameCurrentBranch(context.Context, string) error  { return nil }

func (f *fakeGit) Checkout(_ context.Context, name string) error {
	if f.checkout != nil {
		return f.checkout(name)
	}
	return nil
}

func (f *fakeGit) Add(_ context.Context, paths ...string) error {
	if f.add != nil {
		return f.add(paths)
	}
	return nil
}

func (f *fakeGit) DeleteBranch(context.Context, string, bool) error { return nil }
func (f *fakeGit) MergeNoFF(context.Context, string, string) error  { return nil }

func (f *fakeGit) Tag(_ context.Context, name, message string) error {
	if f.tag != nil {
		return f.tag(name, message)
	}
	return nil
}

func (f *fakeGit) TagCommit(_ context.Context, name, commit, message string) error {
	if f.tagCommit != nil {
		return f.tagCommit(name, commit, message)
	}
	return nil
}

func (f *fakeGit) TagExists(_ context.Context, name string) (bool, error) {
	if f.tagExists != nil {
		return f.tagExists(name)
	}
	return false, nil
}

func (f *fakeGit) CommitSHA(context.Context) (string, error) {
	if f.commitSHA != nil {
		return f.commitSHA()
	}
	return "abc123", nil
}

func (f *fakeGit) ConfigValue(_ context.Context, key string) (string, error) {
	if f.configValue != nil {
		return f.configValue(key)
	}
	return "Test User", nil
}

func (f *fakeGit) Commit(_ context.Context, message string, _ bool) error {
	if f.commit != nil {
		return f.commit(message)
	}
	return nil
}

func (f *fakeGit) Push(context.Context, string, string) error { return nil }
func (f *fakeGit) Pull(context.Context, string, string) error { return nil }

func (f *fakeGit) Status(context.Context) (gitpkg.Status, error) {
	if f.status != nil {
		return f.status()
	}
	return gitpkg.Status{Clean: false}, nil
}

func (f *fakeGit) Remove(_ context.Context, paths ...string) error {
	if f.remove != nil {
		return f.remove(paths)
	}
	return nil
}

func (f *fakeGit) ListBranches(_ context.Context, pattern string) ([]string, error) {
	if f.listBranches != nil {
		return f.listBranches(pattern)
	}
	return nil, nil
}

// fakeGitFlow is a minimal gitflow.Service test double. Only the methods
// release.Service calls have configurable hooks.
type fakeGitFlow struct {
	ensureInitialized func() error
	releaseFinish     func() (gitflow.BranchResult, error)
	releaseFixFinish  func(name string) (gitflow.BranchResult, error)
	devOpsFinish      func(name string) (gitflow.BranchResult, error)
}

func (f *fakeGitFlow) EnsureInitialized(context.Context) error {
	if f.ensureInitialized != nil {
		return f.ensureInitialized()
	}
	return nil
}

func (f *fakeGitFlow) Init(context.Context) (gitflow.InitResult, error) {
	return gitflow.InitResult{}, nil
}

func (f *fakeGitFlow) FeatureStart(context.Context, string) (gitflow.BranchResult, error) {
	return gitflow.BranchResult{}, nil
}
func (f *fakeGitFlow) FeatureFinish(context.Context, string) (gitflow.BranchResult, error) {
	return gitflow.BranchResult{}, nil
}
func (f *fakeGitFlow) HotfixStart(context.Context, string) (gitflow.BranchResult, error) {
	return gitflow.BranchResult{}, nil
}
func (f *fakeGitFlow) HotfixFinish(context.Context, string) (gitflow.BranchResult, error) {
	return gitflow.BranchResult{}, nil
}
func (f *fakeGitFlow) SupportStart(context.Context, string) (gitflow.BranchResult, error) {
	return gitflow.BranchResult{}, nil
}
func (f *fakeGitFlow) SupportFinish(context.Context, string) (gitflow.BranchResult, error) {
	return gitflow.BranchResult{}, nil
}

func (f *fakeGitFlow) ReleaseFinish(context.Context) (gitflow.BranchResult, error) {
	if f.releaseFinish != nil {
		return f.releaseFinish()
	}
	return gitflow.BranchResult{Branch: "staging", Base: "main", Commit: "deadbeef"}, nil
}

func (f *fakeGitFlow) ReleaseFixStart(context.Context, string) (gitflow.BranchResult, error) {
	return gitflow.BranchResult{}, nil
}

func (f *fakeGitFlow) ReleaseFixFinish(_ context.Context, name string) (gitflow.BranchResult, error) {
	if f.releaseFixFinish != nil {
		return f.releaseFixFinish(name)
	}
	return gitflow.BranchResult{Branch: "staging"}, nil
}

func (f *fakeGitFlow) DevOpsStart(context.Context, string) (gitflow.BranchResult, error) {
	return gitflow.BranchResult{}, nil
}

func (f *fakeGitFlow) DevOpsFinish(_ context.Context, name string) (gitflow.BranchResult, error) {
	if f.devOpsFinish != nil {
		return f.devOpsFinish(name)
	}
	return gitflow.BranchResult{Branch: "staging"}, nil
}

func (f *fakeGitFlow) Doctor(context.Context) gitflow.DoctorReport { return gitflow.DoctorReport{} }

// fakeManifestLoader is a release.Loader test double.
type fakeManifestLoader struct {
	load        func() (*release.Manifest, error)
	save        func(m *release.Manifest) error
	exists      func() bool
	saveArchive func(name string, m *release.Manifest) error
}

func (f *fakeManifestLoader) Load(string) (*release.Manifest, error) {
	if f.load != nil {
		return f.load()
	}
	return release.New("5.2", "staging", "5.2.0.0.1"), nil
}

func (f *fakeManifestLoader) Save(_ string, m *release.Manifest) error {
	if f.save != nil {
		return f.save(m)
	}
	return nil
}

func (f *fakeManifestLoader) Exists(string) bool {
	if f.exists != nil {
		return f.exists()
	}
	return true
}

func (f *fakeManifestLoader) SaveArchive(_ string, name string, m *release.Manifest) error {
	if f.saveArchive != nil {
		return f.saveArchive(name, m)
	}
	return nil
}

// fakeVersionLoader is a version.Loader test double.
type fakeVersionLoader struct {
	load   func() (*version.Version, error)
	save   func(v *version.Version) error
	exists func() bool
}

func (f *fakeVersionLoader) Load(string) (*version.Version, error) {
	if f.load != nil {
		return f.load()
	}
	return version.New(5, 2), nil
}

func (f *fakeVersionLoader) Save(_ string, v *version.Version) error {
	if f.save != nil {
		return f.save(v)
	}
	return nil
}

func (f *fakeVersionLoader) Exists(string) bool {
	if f.exists != nil {
		return f.exists()
	}
	return true
}

// fakeFeatureLoader is a feature.Loader test double.
type fakeFeatureLoader struct {
	load   func() (*feature.Registry, error)
	save   func(r *feature.Registry) error
	exists func() bool
}

func (f *fakeFeatureLoader) Load(string) (*feature.Registry, error) {
	if f.load != nil {
		return f.load()
	}
	return feature.New(), nil
}

func (f *fakeFeatureLoader) Save(_ string, r *feature.Registry) error {
	if f.save != nil {
		return f.save(r)
	}
	return nil
}

func (f *fakeFeatureLoader) Exists(string) bool {
	if f.exists != nil {
		return f.exists()
	}
	return true
}

// happyDeps returns a Dependencies struct wired with all fakes at their
// zero-value ("happy path") behavior, ready for individual tests to
// override one field.
func happyDeps() (deps release.Dependencies, git *fakeGit, gf *fakeGitFlow, ml *fakeManifestLoader, vl *fakeVersionLoader) {
	git = &fakeGit{}
	gf = &fakeGitFlow{}
	ml = &fakeManifestLoader{}
	vl = &fakeVersionLoader{}
	fl := &fakeFeatureLoader{}
	deps = release.Dependencies{
		GitFlow:        gf,
		Git:            git,
		RepoPath:       "/repo",
		Config:         config.Default(),
		ManifestLoader: ml,
		VersionLoader:  vl,
		FeatureLoader:  fl,
		Logger:         discardLogger(),
	}
	return deps, git, gf, ml, vl
}
