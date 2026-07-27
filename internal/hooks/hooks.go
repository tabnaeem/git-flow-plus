// Package hooks lets Git Flow Plus trigger external automation (CI/CD
// pipelines, deployment scripts, notifications) at release lifecycle
// events, without the tool itself knowing anything about any specific
// CI/CD system (GitHub Actions, GitLab CI, Azure DevOps, Jenkins,
// Bitbucket Pipelines, or a custom system).
//
// The mechanism is deliberately simple and file-based, mirroring Git's own
// hooks: a repository can drop an executable script under
// .gitflowplus/hooks/<event-name> (with a platform-appropriate extension),
// and Git Flow Plus runs it after that event, passing context via
// environment variables. If no script is present, nothing happens — hooks
// are entirely opt-in. Since any CI/CD system can also simply trigger off
// the pushed Git tag itself, a hook script is only needed for local or
// non-tag-triggered automation (e.g. calling a webhook directly).
package hooks

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
)

// DirName is the hooks directory inside the Git Flow Plus metadata
// directory (config.DirName).
const DirName = "hooks"

// interpreter maps a script extension to the command used to run it.
type interpreter struct {
	ext   string
	build func(path string) (cmd string, args []string)
}

// candidates is checked in order; the first matching file on disk wins.
var candidates = []interpreter{
	{"", func(path string) (string, []string) { return path, nil }},
	{".sh", func(path string) (string, []string) { return "sh", []string{path} }},
	{".ps1", func(path string) (string, []string) {
		return "powershell", []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", path}
	}},
	{".bat", func(path string) (string, []string) { return "cmd", []string{"/C", path} }},
	{".cmd", func(path string) (string, []string) { return "cmd", []string{"/C", path} }},
}

// Runner looks for and executes lifecycle hook scripts.
type Runner interface {
	// Run looks for a hook script named name (trying, in order, no
	// extension then .sh/.ps1/.bat/.cmd) under
	// <repoPath>/.gitflowplus/hooks/, and executes it if found from
	// repoPath as the working directory, with env merged into the
	// process environment. ran is false (with a nil error) if no script
	// exists; a script that runs but exits non-zero is reported as err
	// with ran true.
	Run(ctx context.Context, repoPath, name string, env map[string]string) (ran bool, err error)
}

type execRunner struct{}

// NewRunner returns a Runner backed by the real OS process/filesystem.
func NewRunner() Runner {
	return execRunner{}
}

func (execRunner) Run(ctx context.Context, repoPath, name string, env map[string]string) (bool, error) {
	dir := filepath.Join(repoPath, ".gitflowplus", DirName)

	for _, c := range candidates {
		path := filepath.Join(dir, name+c.ext)
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue
		}

		cmdName, args := c.build(path)
		cmd := exec.CommandContext(ctx, cmdName, args...)
		cmd.Dir = repoPath
		cmd.Env = append(os.Environ(), envSlice(env)...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		return true, cmd.Run()
	}

	return false, nil
}

func envSlice(env map[string]string) []string {
	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, k+"="+v)
	}
	return out
}
