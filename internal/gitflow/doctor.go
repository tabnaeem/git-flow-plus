package gitflow

import (
	"context"
	"fmt"

	"github.com/hulhub/git-flow-plus/internal/git"
)

// Doctor runs Git-level health checks: that a git binary is available, the
// target path is a Git repository, and the configured main/staging/develop
// branches exist. It stops early once a prerequisite check fails, since
// later checks would be meaningless (e.g. branch checks without a
// repository).
func (s *service) Doctor(ctx context.Context) DoctorReport {
	var checks []DoctorCheck

	if !git.Available() {
		return DoctorReport{Checks: append(checks, DoctorCheck{
			Name: "git binary", OK: false, Detail: "not found on PATH",
		})}
	}
	checks = append(checks, DoctorCheck{Name: "git binary", OK: true, Detail: "found on PATH"})

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

	return DoctorReport{Checks: checks}
}
