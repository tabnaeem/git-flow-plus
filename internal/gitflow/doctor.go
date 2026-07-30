package gitflow

import (
	"context"
	"fmt"

	"github.com/tabnaeem/git-flow-plus/internal/git"
)

// Doctor runs Git-level health checks: that a git binary is available (and
// its version), the target path is a Git repository, the configured
// main/staging/develop branches exist, the working tree state, and whether
// the repository directory is writable. It stops early once a prerequisite
// check fails, since later checks would be meaningless (e.g. branch checks
// without a repository). Installation-level checks that aren't specific to
// a Git repository (CLI version, PATH resolution, release configuration)
// live in internal/cli's doctor command instead — see that package's doc
// comment for why.
func (s *service) Doctor(ctx context.Context) DoctorReport {
	var checks []DoctorCheck

	if !git.Available() {
		return DoctorReport{Checks: append(checks, DoctorCheck{
			Name: "git binary", OK: false, Detail: "not found on PATH",
		})}
	}
	checks = append(checks, DoctorCheck{Name: "git binary", OK: true, Detail: "found on PATH"})

	if v, err := s.git.Version(ctx); err != nil {
		checks = append(checks, DoctorCheck{Name: "git version", OK: false, Detail: err.Error()})
	} else {
		checks = append(checks, DoctorCheck{Name: "git version", OK: true, Detail: v})
	}

	if !s.git.IsRepo(ctx) {
		return DoctorReport{Checks: append(checks, DoctorCheck{
			Name: "git repository", OK: false, Detail: "current path is not inside a Git working tree",
		})}
	}
	checks = append(checks, DoctorCheck{Name: "git repository", OK: true, Detail: "current path is inside a Git working tree"})

	for _, b := range []struct{ label, name string }{
		{"main branch", s.cfg.Branches.Main},
		{"staging branch", s.cfg.Branches.Staging},
		{"develop branch", s.cfg.Branches.Develop},
	} {
		exists, err := s.git.BranchExists(ctx, b.name)
		switch {
		case err != nil:
			checks = append(checks, DoctorCheck{Name: b.label, OK: false, Detail: err.Error()})
		case exists:
			checks = append(checks, DoctorCheck{Name: b.label, OK: true, Detail: fmt.Sprintf("%q exists", b.name)})
		default:
			checks = append(checks, DoctorCheck{Name: b.label, OK: false, Detail: fmt.Sprintf("%q not found; run 'git flow init'", b.name)})
		}
	}

	status, err := s.git.Status(ctx)
	switch {
	case err != nil:
		checks = append(checks, DoctorCheck{Name: "working tree", OK: false, Detail: err.Error()})
	case status.Clean:
		checks = append(checks, DoctorCheck{Name: "working tree", OK: true, Detail: "clean"})
	default:
		checks = append(checks, DoctorCheck{Name: "working tree", OK: true, Detail: "has uncommitted changes"})
	}

	writable, err := s.git.Writable(ctx)
	switch {
	case err != nil:
		checks = append(checks, DoctorCheck{Name: "permissions", OK: false, Detail: err.Error()})
	case writable:
		checks = append(checks, DoctorCheck{Name: "permissions", OK: true, Detail: "repository directory is writable"})
	default:
		checks = append(checks, DoctorCheck{Name: "permissions", OK: false, Detail: "repository directory is not writable"})
	}

	return DoctorReport{Checks: checks}
}
