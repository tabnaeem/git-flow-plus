package gitflow

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/tabnaeem/git-flow-plus/internal/git"
)

// Section labels used to group DoctorCheck results for display — see
// internal/cli's doctor renderer, which reads this field to build
// `git flow doctor`'s "Environment"/"Branch Model" headings.
const (
	SectionEnvironment = "Environment"
	SectionBranchModel = "Branch Model"
)

// minGitMajor/minGitMinor is the minimum supported Git version. Chosen as
// a conservative floor: old enough (December 2018) that failing it means
// a genuinely unmaintained install, not a real constraint — nothing in
// git.Client actually requires anything newer than Git's long-stable
// core plumbing (checkout, branch, merge, tag, commit, status, config,
// rev-parse). This is a supportability check, not a functional one.
const (
	minGitMajor = 2
	minGitMinor = 20
)

// Doctor runs Git-level health checks: that a git binary is available and
// at a supported version, the target path is a Git repository, whether a
// remote is configured (informational only — Git Flow Plus never
// requires one; every push/pull step is a manual, separate action), the
// configured main/staging/develop branches exist, the working tree
// state, and whether the repository directory is writable. It stops
// early once a prerequisite check fails, since later checks would be
// meaningless (e.g. branch checks without a repository).
//
// Installation-level checks that aren't specific to a Git repository
// (CLI version, PATH resolution, Git Flow Plus configuration, release
// tooling) live in internal/cli's doctor command instead — see that
// package's doc comment for why.
func (s *service) Doctor(ctx context.Context) DoctorReport {
	var checks []DoctorCheck

	if !git.Available() {
		return DoctorReport{Checks: append(checks, DoctorCheck{
			Section: SectionEnvironment, Name: "Git", OK: false,
			Detail: "git binary not found on PATH",
		})}
	}

	raw, err := s.git.Version(ctx)
	switch {
	case err != nil:
		checks = append(checks, DoctorCheck{Section: SectionEnvironment, Name: "Git", Detail: fmt.Sprintf("could not determine git version: %v", err)})
	default:
		major, minor := parseGitVersion(raw)
		if gitVersionSupported(major, minor) {
			checks = append(checks, DoctorCheck{Section: SectionEnvironment, Name: "Git", OK: true, Detail: raw})
		} else {
			checks = append(checks, DoctorCheck{Section: SectionEnvironment, Name: "Git", Detail: fmt.Sprintf(
				"%s is older than the minimum supported version (%d.%d.0); some Git Flow Plus operations may not work reliably",
				raw, minGitMajor, minGitMinor)})
		}
	}

	if !s.git.IsRepo(ctx) {
		return DoctorReport{Checks: append(checks, DoctorCheck{
			Section: SectionEnvironment, Name: "Repository", OK: false,
			Detail: "current path is not inside a Git working tree",
		})}
	}
	checks = append(checks, DoctorCheck{Section: SectionEnvironment, Name: "Repository", OK: true, Detail: "current path is inside a Git working tree"})

	// Informational only: Git Flow Plus never pushes or pulls
	// automatically (every push is a manual step the developer takes
	// themselves - see README.md's "The short version"), so a repository
	// with no remote yet is a completely normal, healthy state (e.g.
	// right after `git flow init`, before it's ever been pushed
	// anywhere) - this must never fail the overall doctor result.
	remoteURL, err := s.git.ConfigValue(ctx, "remote.origin.url")
	switch {
	case err != nil:
		checks = append(checks, DoctorCheck{Section: SectionEnvironment, Name: "Remote", OK: true, Detail: fmt.Sprintf("could not check for a remote: %v", err)})
	case remoteURL == "":
		checks = append(checks, DoctorCheck{Section: SectionEnvironment, Name: "Remote", OK: true, Detail: "no 'origin' remote configured (optional; needed only to push/collaborate)"})
	default:
		checks = append(checks, DoctorCheck{Section: SectionEnvironment, Name: "Remote", OK: true, Detail: fmt.Sprintf("'origin' -> %s", remoteURL)})
	}

	for _, b := range []struct{ label, name string }{
		{"main", s.cfg.Branches.Main},
		{"staging", s.cfg.Branches.Staging},
		{"develop", s.cfg.Branches.Develop},
	} {
		exists, err := s.git.BranchExists(ctx, b.name)
		switch {
		case err != nil:
			checks = append(checks, DoctorCheck{Section: SectionBranchModel, Name: b.label, Detail: err.Error()})
		case exists:
			checks = append(checks, DoctorCheck{Section: SectionBranchModel, Name: b.label, OK: true, Detail: fmt.Sprintf("%q exists", b.name)})
		default:
			checks = append(checks, DoctorCheck{Section: SectionBranchModel, Name: b.label, Detail: fmt.Sprintf("%q not found; run 'git flow init'", b.name)})
		}
	}

	status, err := s.git.Status(ctx)
	switch {
	case err != nil:
		checks = append(checks, DoctorCheck{Section: SectionEnvironment, Name: "Working Tree", Detail: err.Error()})
	case status.Clean:
		checks = append(checks, DoctorCheck{Section: SectionEnvironment, Name: "Working Tree", OK: true, Detail: "clean"})
	default:
		checks = append(checks, DoctorCheck{Section: SectionEnvironment, Name: "Working Tree", OK: true, Detail: "has uncommitted changes"})
	}

	writable, err := s.git.Writable(ctx)
	switch {
	case err != nil:
		checks = append(checks, DoctorCheck{Section: SectionEnvironment, Name: "Permissions", Detail: err.Error()})
	case writable:
		checks = append(checks, DoctorCheck{Section: SectionEnvironment, Name: "Permissions", OK: true, Detail: "repository directory is writable"})
	default:
		checks = append(checks, DoctorCheck{Section: SectionEnvironment, Name: "Permissions", Detail: "repository directory is not writable"})
	}

	return DoctorReport{Checks: checks}
}

// parseGitVersion extracts the numeric major/minor from `git --version`'s
// output (e.g. "git version 2.43.0.windows.1" or "git version 2.43.0" ->
// 2, 43). It scans every whitespace-separated field rather than assuming
// a fixed position, since the exact wording varies by platform/vendor
// (e.g. "git version 2.43.0.windows.1" on Git for Windows). A field that
// can't be parsed as at least two dot-separated integers is skipped;
// nothing parseable at all yields (0, 0), which fails the minimum-version
// check deliberately rather than silently passing an unrecognized string.
func parseGitVersion(raw string) (major, minor int) {
	for _, field := range strings.Fields(raw) {
		parts := strings.SplitN(field, ".", 3)
		if len(parts) < 2 {
			continue
		}
		maj, errMaj := strconv.Atoi(parts[0])
		min, errMin := strconv.Atoi(parts[1])
		if errMaj == nil && errMin == nil {
			return maj, min
		}
	}
	return 0, 0
}

func gitVersionSupported(major, minor int) bool {
	if major != minGitMajor {
		return major > minGitMajor
	}
	return minor >= minGitMinor
}
