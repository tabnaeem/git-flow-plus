package git

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Status is a simplified view of `git status --porcelain`.
type Status struct {
	// Clean is true when the working tree has no staged or unstaged changes.
	Clean bool
	// Porcelain is the raw `git status --porcelain` output.
	Porcelain string
}

// Client performs Git operations against a single working directory. All
// methods shell out to a real git binary via the injected Runner, so
// behavior (hooks, credential helpers, ignore rules, etc.) matches plain Git
// exactly.
type Client interface {
	// IsRepo reports whether Dir is inside a Git working tree.
	IsRepo(ctx context.Context) bool
	// Init runs `git init` in Dir.
	Init(ctx context.Context) error
	// HasCommits reports whether HEAD resolves to a commit, i.e. the
	// repository is not a freshly initialized, commit-less repo.
	HasCommits(ctx context.Context) bool
	// CurrentBranch returns the name of the currently checked-out branch.
	CurrentBranch(ctx context.Context) (string, error)
	// BranchExists reports whether a local branch named name exists.
	BranchExists(ctx context.Context, name string) (bool, error)
	// CreateBranch creates and checks out a new branch named name. If
	// startPoint is empty, it branches from the current HEAD.
	CreateBranch(ctx context.Context, name, startPoint string) error
	// RenameCurrentBranch renames the currently checked-out branch to name.
	RenameCurrentBranch(ctx context.Context, name string) error
	// Checkout switches the working tree to the named branch.
	Checkout(ctx context.Context, name string) error
	// Add stages paths for the next commit.
	Add(ctx context.Context, paths ...string) error
	// DeleteBranch deletes the named local branch. If force is false, Git
	// refuses to delete branches with unmerged commits.
	DeleteBranch(ctx context.Context, name string, force bool) error
	// MergeNoFF merges branch into the current branch with --no-ff, always
	// creating a merge commit, using message as the commit message.
	MergeNoFF(ctx context.Context, branch, message string) error
	// Tag creates an annotated tag named name at HEAD with message.
	Tag(ctx context.Context, name, message string) error
	// TagCommit creates an annotated tag named name at the given commit
	// (rather than HEAD) with message.
	TagCommit(ctx context.Context, name, commit, message string) error
	// TagExists reports whether a tag named name exists.
	TagExists(ctx context.Context, name string) (bool, error)
	// CommitSHA returns the full SHA of HEAD.
	CommitSHA(ctx context.Context) (string, error)
	// ConfigValue returns the value of a git config key (e.g. "user.name"),
	// or "" if it is not set. Only a genuine command failure (not "unset")
	// is returned as an error.
	ConfigValue(ctx context.Context, key string) (string, error)
	// Commit creates a commit with message. If allowEmpty is true, the
	// commit is created even if there are no staged changes.
	Commit(ctx context.Context, message string, allowEmpty bool) error
	// Push pushes branch to remote.
	Push(ctx context.Context, remote, branch string) error
	// Pull fetches and merges branch from remote.
	Pull(ctx context.Context, remote, branch string) error
	// Status reports the working tree's clean/dirty state.
	Status(ctx context.Context) (Status, error)
	// Remove removes paths from the working tree and stages the removal.
	Remove(ctx context.Context, paths ...string) error
	// ListBranches returns the local branch names matching pattern (a glob
	// understood by `git for-each-ref`, e.g. "release/*").
	ListBranches(ctx context.Context, pattern string) ([]string, error)
}

type client struct {
	runner Runner
	dir    string
}

// NewClient returns a Client that runs git commands in dir via runner.
func NewClient(runner Runner, dir string) Client {
	return &client{runner: runner, dir: dir}
}

func (c *client) IsRepo(ctx context.Context) bool {
	_, err := c.runner.Run(ctx, c.dir, "rev-parse", "--is-inside-work-tree")
	return err == nil
}

func (c *client) Init(ctx context.Context) error {
	_, err := c.runner.Run(ctx, c.dir, "init")
	return err
}

func (c *client) HasCommits(ctx context.Context) bool {
	_, err := c.runner.Run(ctx, c.dir, "rev-parse", "--verify", "-q", "HEAD")
	return err == nil
}

func (c *client) CurrentBranch(ctx context.Context) (string, error) {
	return c.runner.Run(ctx, c.dir, "rev-parse", "--abbrev-ref", "HEAD")
}

func (c *client) refExists(ctx context.Context, ref string) (bool, error) {
	_, err := c.runner.Run(ctx, c.dir, "show-ref", "--verify", "--quiet", ref)
	if err == nil {
		return true, nil
	}

	var cmdErr *CommandError
	if errors.As(err, &cmdErr) && cmdErr.ExitCode == 1 {
		return false, nil
	}
	return false, err
}

func (c *client) BranchExists(ctx context.Context, name string) (bool, error) {
	return c.refExists(ctx, "refs/heads/"+name)
}

func (c *client) TagExists(ctx context.Context, name string) (bool, error) {
	return c.refExists(ctx, "refs/tags/"+name)
}

func (c *client) CreateBranch(ctx context.Context, name, startPoint string) error {
	args := []string{"checkout", "-b", name}
	if startPoint != "" {
		args = append(args, startPoint)
	}
	_, err := c.runner.Run(ctx, c.dir, args...)
	return err
}

func (c *client) RenameCurrentBranch(ctx context.Context, name string) error {
	_, err := c.runner.Run(ctx, c.dir, "branch", "-M", name)
	return err
}

func (c *client) Checkout(ctx context.Context, name string) error {
	_, err := c.runner.Run(ctx, c.dir, "checkout", name)
	return err
}

func (c *client) Add(ctx context.Context, paths ...string) error {
	args := append([]string{"add"}, paths...)
	_, err := c.runner.Run(ctx, c.dir, args...)
	return err
}

func (c *client) DeleteBranch(ctx context.Context, name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := c.runner.Run(ctx, c.dir, "branch", flag, name)
	return err
}

func (c *client) MergeNoFF(ctx context.Context, branch, message string) error {
	_, err := c.runner.Run(ctx, c.dir, "merge", "--no-ff", "-m", message, branch)
	return err
}

func (c *client) Tag(ctx context.Context, name, message string) error {
	_, err := c.runner.Run(ctx, c.dir, "tag", "-a", name, "-m", message)
	return err
}

func (c *client) TagCommit(ctx context.Context, name, commit, message string) error {
	_, err := c.runner.Run(ctx, c.dir, "tag", "-a", name, "-m", message, commit)
	return err
}

func (c *client) CommitSHA(ctx context.Context) (string, error) {
	return c.runner.Run(ctx, c.dir, "rev-parse", "HEAD")
}

func (c *client) ConfigValue(ctx context.Context, key string) (string, error) {
	out, err := c.runner.Run(ctx, c.dir, "config", "--get", key)
	if err == nil {
		return out, nil
	}

	var cmdErr *CommandError
	if errors.As(err, &cmdErr) && cmdErr.ExitCode == 1 {
		return "", nil
	}
	return "", err
}

func (c *client) Commit(ctx context.Context, message string, allowEmpty bool) error {
	args := []string{"commit", "-m", message}
	if allowEmpty {
		args = append(args, "--allow-empty")
	}
	_, err := c.runner.Run(ctx, c.dir, args...)
	return err
}

func (c *client) Push(ctx context.Context, remote, branch string) error {
	_, err := c.runner.Run(ctx, c.dir, "push", remote, branch)
	return err
}

func (c *client) Pull(ctx context.Context, remote, branch string) error {
	_, err := c.runner.Run(ctx, c.dir, "pull", remote, branch)
	return err
}

func (c *client) Status(ctx context.Context) (Status, error) {
	out, err := c.runner.Run(ctx, c.dir, "status", "--porcelain")
	if err != nil {
		return Status{}, fmt.Errorf("git status: %w", err)
	}
	return Status{Clean: out == "", Porcelain: out}, nil
}

func (c *client) Remove(ctx context.Context, paths ...string) error {
	args := append([]string{"rm", "-r"}, paths...)
	_, err := c.runner.Run(ctx, c.dir, args...)
	return err
}

func (c *client) ListBranches(ctx context.Context, pattern string) ([]string, error) {
	out, err := c.runner.Run(ctx, c.dir, "for-each-ref", "--format=%(refname:short)", "refs/heads/"+pattern)
	if err != nil {
		return nil, fmt.Errorf("listing branches matching %q: %w", pattern, err)
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}
