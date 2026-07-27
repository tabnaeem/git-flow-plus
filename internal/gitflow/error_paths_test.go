package gitflow_test

import (
	"context"
	"errors"
	"testing"

	"github.com/hulhub/git-flow-plus/internal/config"
	gitpkg "github.com/hulhub/git-flow-plus/internal/git"
	"github.com/hulhub/git-flow-plus/internal/gitflow"
)

var errBoom = errors.New("boom")

func newFakeService(f *fakeClient) gitflow.Service {
	return gitflow.NewService(f, config.Default(), discardLogger())
}

// --- Init error paths ---

func TestInitPropagatesGitInitError(t *testing.T) {
	f := &fakeClient{
		isRepo: func() bool { return false },
		initFn: func() error { return errBoom },
	}
	_, err := newFakeService(f).Init(context.Background())
	if !containsMsg(err, "initializing repository") || !errors.Is(err, errBoom) {
		t.Errorf("Init() error = %v, want it to wrap errBoom with 'initializing repository'", err)
	}
}

func TestInitPropagatesCommitError(t *testing.T) {
	f := &fakeClient{
		hasCommits: func() bool { return false },
		commit:     func(string, bool) error { return errBoom },
	}
	_, err := newFakeService(f).Init(context.Background())
	if !containsMsg(err, "creating initial commit") {
		t.Errorf("Init() error = %v, want 'creating initial commit'", err)
	}
}

func TestInitPropagatesCurrentBranchError(t *testing.T) {
	f := &fakeClient{
		hasCommits:    func() bool { return false },
		currentBranch: func() (string, error) { return "", errBoom },
	}
	_, err := newFakeService(f).Init(context.Background())
	if !containsMsg(err, "resolving current branch") {
		t.Errorf("Init() error = %v, want 'resolving current branch'", err)
	}
}

func TestInitPropagatesRenameError(t *testing.T) {
	f := &fakeClient{
		hasCommits:          func() bool { return false },
		currentBranch:       func() (string, error) { return "master", nil },
		renameCurrentBranch: func(string) error { return errBoom },
	}
	_, err := newFakeService(f).Init(context.Background())
	if !containsMsg(err, "renaming") {
		t.Errorf("Init() error = %v, want 'renaming'", err)
	}
}

func TestInitPropagatesMainBranchExistsError(t *testing.T) {
	f := &fakeClient{
		branchExists: func(name string) (bool, error) {
			if name == "main" {
				return false, errBoom
			}
			return branchExistsFor(name)
		},
	}
	_, err := newFakeService(f).Init(context.Background())
	if !containsMsg(err, "checking branch") {
		t.Errorf("Init() error = %v, want 'checking branch'", err)
	}
}

func TestInitPropagatesDevelopBranchExistsError(t *testing.T) {
	f := &fakeClient{
		branchExists: func(name string) (bool, error) {
			if name == "develop" {
				return false, errBoom
			}
			return branchExistsFor(name)
		},
	}
	_, err := newFakeService(f).Init(context.Background())
	if !containsMsg(err, "checking branch") {
		t.Errorf("Init() error = %v, want 'checking branch'", err)
	}
}

func TestInitPropagatesCreateDevelopError(t *testing.T) {
	f := &fakeClient{
		branchExists: func(name string) (bool, error) {
			if name == "develop" {
				return false, nil
			}
			return true, nil
		},
		createBranch: func(string, string) error { return errBoom },
	}
	_, err := newFakeService(f).Init(context.Background())
	if !containsMsg(err, "creating branch") {
		t.Errorf("Init() error = %v, want 'creating branch'", err)
	}
}

func TestInitPropagatesCheckoutDevelopError(t *testing.T) {
	f := &fakeClient{
		checkout: func(name string) error {
			if name == "develop" {
				return errBoom
			}
			return nil
		},
	}
	_, err := newFakeService(f).Init(context.Background())
	if !containsMsg(err, "checking out") {
		t.Errorf("Init() error = %v, want 'checking out'", err)
	}
}

// --- Start family error paths (feature/hotfix/support/releasefix/devops share shape) ---

func TestStartOperationsPropagateBranchExistsAndCreateErrors(t *testing.T) {
	type startCall struct {
		name string
		call func(gitflow.Service) (gitflow.BranchResult, error)
	}
	calls := []startCall{
		{"FeatureStart", func(s gitflow.Service) (gitflow.BranchResult, error) {
			return s.FeatureStart(context.Background(), "x")
		}},
		{"HotfixStart", func(s gitflow.Service) (gitflow.BranchResult, error) { return s.HotfixStart(context.Background(), "x") }},
		{"SupportStart", func(s gitflow.Service) (gitflow.BranchResult, error) {
			return s.SupportStart(context.Background(), "x")
		}},
		{"ReleaseFixStart", func(s gitflow.Service) (gitflow.BranchResult, error) {
			return s.ReleaseFixStart(context.Background(), "x")
		}},
		{"DevOpsStart", func(s gitflow.Service) (gitflow.BranchResult, error) {
			return s.DevOpsStart(context.Background(), "x")
		}},
	}

	for _, c := range calls {
		t.Run(c.name+"/BranchExistsError", func(t *testing.T) {
			f := &fakeClient{branchExists: func(name string) (bool, error) {
				if name == "main" || name == "staging" || name == "develop" {
					return true, nil
				}
				return false, errBoom
			}}
			_, err := c.call(newFakeService(f))
			if !containsMsg(err, "checking branch") {
				t.Errorf("%s error = %v, want 'checking branch'", c.name, err)
			}
		})

		t.Run(c.name+"/CreateBranchError", func(t *testing.T) {
			f := &fakeClient{createBranch: func(string, string) error { return errBoom }}
			_, err := c.call(newFakeService(f))
			if !containsMsg(err, "creating branch") {
				t.Errorf("%s error = %v, want 'creating branch'", c.name, err)
			}
		})
	}
}

// --- Finish family error paths ---

// finishHappyFake returns a fakeClient configured so that Finish operations
// for a branch named branchName succeed by default.
func finishHappyFake(branchName string) *fakeClient {
	return &fakeClient{
		branchExists: func(name string) (bool, error) {
			if name == "main" || name == "staging" || name == "develop" || name == branchName {
				return true, nil
			}
			return false, nil
		},
	}
}

func TestFeatureMergePropagatesErrors(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(f *fakeClient)
		wantMsg string
	}{
		{"StatusError", func(f *fakeClient) { f.status = func() (gitpkg.Status, error) { return gitpkg.Status{}, errBoom } }, "boom"},
		{"CheckoutError", func(f *fakeClient) { f.checkout = func(string) error { return errBoom } }, "checking out"},
		{"MergeError", func(f *fakeClient) { f.mergeNoFF = func(string, string) error { return errBoom } }, "merging"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := finishHappyFake("feature/x")
			tc.mutate(f)
			_, err := newFakeService(f).FeatureMerge(context.Background(), "x")
			if !containsMsg(err, tc.wantMsg) {
				t.Errorf("FeatureMerge() error = %v, want it to contain %q", err, tc.wantMsg)
			}
		})
	}
}

func TestFeatureMergeMissingBranchErrors(t *testing.T) {
	f := &fakeClient{branchExists: func(name string) (bool, error) {
		return name == "main" || name == "staging" || name == "develop", nil
	}}
	_, err := newFakeService(f).FeatureMerge(context.Background(), "never-started")
	if !errors.Is(err, gitflow.ErrBranchMissing) {
		t.Errorf("FeatureMerge() error = %v, want ErrBranchMissing", err)
	}
}

func TestHotfixFinishPropagatesErrors(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(f *fakeClient)
		wantMsg string
	}{
		{"TagExistsError", func(f *fakeClient) { f.tagExists = func(string) (bool, error) { return false, errBoom } }, "checking tag"},
		{"CheckoutMainError", func(f *fakeClient) {
			f.checkout = func(name string) error {
				if name == "main" {
					return errBoom
				}
				return nil
			}
		}, "checking out"},
		{"TagError", func(f *fakeClient) { f.tag = func(string, string) error { return errBoom } }, "tagging"},
		{"CheckoutDevelopError", func(f *fakeClient) {
			f.checkout = func(name string) error {
				if name == "develop" {
					return errBoom
				}
				return nil
			}
		}, "checking out"},
		{"DeleteError", func(f *fakeClient) { f.deleteBranch = func(string, bool) error { return errBoom } }, "deleting branch"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := finishHappyFake("hotfix/x")
			tc.mutate(f)
			_, err := newFakeService(f).HotfixFinish(context.Background(), "x")
			if !containsMsg(err, tc.wantMsg) {
				t.Errorf("HotfixFinish() error = %v, want it to contain %q", err, tc.wantMsg)
			}
		})
	}
}

func TestReleaseFinishPropagatesErrors(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(f *fakeClient)
		wantMsg string
	}{
		{"CheckoutMainError", func(f *fakeClient) {
			f.checkout = func(name string) error {
				if name == "main" {
					return errBoom
				}
				return nil
			}
		}, "checking out"},
		{"MergeMainError", func(f *fakeClient) { f.mergeNoFF = func(string, string) error { return errBoom } }, "merging"},
		{"CommitSHAError", func(f *fakeClient) { f.commitSHA = func() (string, error) { return "", errBoom } }, "resolving commit"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeClient{}
			tc.mutate(f)
			_, err := newFakeService(f).ReleaseFinish(context.Background())
			if !containsMsg(err, tc.wantMsg) {
				t.Errorf("ReleaseFinish() error = %v, want it to contain %q", err, tc.wantMsg)
			}
		})
	}
}

func TestReleaseFinishRequiresInitializedRepo(t *testing.T) {
	f := &fakeClient{branchExists: func(string) (bool, error) { return false, nil }}
	_, err := newFakeService(f).ReleaseFinish(context.Background())
	if !errors.Is(err, gitflow.ErrNotInitialized) {
		t.Errorf("ReleaseFinish() on an uninitialized repo error = %v, want ErrNotInitialized", err)
	}
}

// --- ReleaseFix / DevOps error paths ---

func TestReleaseFixDevOpsStartPropagateErrors(t *testing.T) {
	type startCall struct {
		name string
		call func(gitflow.Service) (gitflow.BranchResult, error)
	}
	calls := []startCall{
		{"ReleaseFixStart", func(s gitflow.Service) (gitflow.BranchResult, error) {
			return s.ReleaseFixStart(context.Background(), "x")
		}},
		{"DevOpsStart", func(s gitflow.Service) (gitflow.BranchResult, error) { return s.DevOpsStart(context.Background(), "x") }},
	}

	for _, c := range calls {
		t.Run(c.name+"/BranchExistsError", func(t *testing.T) {
			f := &fakeClient{branchExists: func(name string) (bool, error) {
				if name == "main" || name == "staging" || name == "develop" {
					return true, nil
				}
				return false, errBoom
			}}
			_, err := c.call(newFakeService(f))
			if !containsMsg(err, "checking branch") {
				t.Errorf("%s error = %v, want 'checking branch'", c.name, err)
			}
		})

		t.Run(c.name+"/CreateBranchError", func(t *testing.T) {
			f := &fakeClient{createBranch: func(string, string) error { return errBoom }}
			_, err := c.call(newFakeService(f))
			if !containsMsg(err, "creating branch") {
				t.Errorf("%s error = %v, want 'creating branch'", c.name, err)
			}
		})
	}
}

func TestReleaseFixDevOpsFinishPropagateErrors(t *testing.T) {
	type finishCall struct {
		name       string
		branchName string
		call       func(gitflow.Service) (gitflow.BranchResult, error)
	}
	calls := []finishCall{
		{"ReleaseFixFinish", "release-fix/x", func(s gitflow.Service) (gitflow.BranchResult, error) {
			return s.ReleaseFixFinish(context.Background(), "x")
		}},
		{"DevOpsFinish", "release-devops/x", func(s gitflow.Service) (gitflow.BranchResult, error) {
			return s.DevOpsFinish(context.Background(), "x")
		}},
	}

	for _, c := range calls {
		t.Run(c.name+"/CheckoutError", func(t *testing.T) {
			f := finishHappyFake(c.branchName)
			f.checkout = func(string) error { return errBoom }
			_, err := c.call(newFakeService(f))
			if !containsMsg(err, "checking out") {
				t.Errorf("%s error = %v, want 'checking out'", c.name, err)
			}
		})

		t.Run(c.name+"/MergeError", func(t *testing.T) {
			f := finishHappyFake(c.branchName)
			f.mergeNoFF = func(string, string) error { return errBoom }
			_, err := c.call(newFakeService(f))
			if !containsMsg(err, "merging") {
				t.Errorf("%s error = %v, want 'merging'", c.name, err)
			}
		})
	}
}

func TestSupportFinishPropagatesBranchExistsError(t *testing.T) {
	f := &fakeClient{branchExists: func(name string) (bool, error) {
		if name == "main" || name == "staging" || name == "develop" {
			return true, nil
		}
		return false, errBoom
	}}
	_, err := newFakeService(f).SupportFinish(context.Background(), "x")
	if !containsMsg(err, "checking branch") {
		t.Errorf("SupportFinish() error = %v, want 'checking branch'", err)
	}
}

// --- Doctor error paths ---

func TestDoctorReportsBranchExistsError(t *testing.T) {
	f := &fakeClient{branchExists: func(name string) (bool, error) {
		if name == "main" {
			return false, errBoom
		}
		return true, nil
	}}
	report := newFakeService(f).Doctor(context.Background())

	found := false
	for _, c := range report.Checks {
		if c.Name == "main branch" {
			found = true
			if c.OK {
				t.Error("'main branch' check OK = true, want false when BranchExists errors")
			}
		}
	}
	if !found {
		t.Error("Doctor() did not include a 'main branch' check")
	}
}

func TestDoctorReportsStatusError(t *testing.T) {
	f := &fakeClient{status: func() (gitpkg.Status, error) { return gitpkg.Status{}, errBoom }}
	report := newFakeService(f).Doctor(context.Background())

	found := false
	for _, c := range report.Checks {
		if c.Name == "working tree" {
			found = true
			if c.OK {
				t.Error("'working tree' check OK = true, want false when Status errors")
			}
		}
	}
	if !found {
		t.Error("Doctor() did not include a 'working tree' check")
	}
}

func TestDoctorReportsDirtyWorkingTree(t *testing.T) {
	f := &fakeClient{status: func() (gitpkg.Status, error) { return gitpkg.Status{Clean: false, Porcelain: " M x"}, nil }}
	report := newFakeService(f).Doctor(context.Background())

	for _, c := range report.Checks {
		if c.Name == "working tree" && c.Detail != "has uncommitted changes" {
			t.Errorf("'working tree' detail = %q, want %q", c.Detail, "has uncommitted changes")
		}
	}
}
