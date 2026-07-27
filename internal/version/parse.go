package version

import (
	"fmt"
	"strconv"
	"strings"
)

// Parse parses a "Sprint.Release.Fix.DevOps.QABuild" string, e.g. "5.2.3.1.4".
func Parse(s string) (*Version, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 5 {
		return nil, fmt.Errorf("%w: %q (expected Sprint.Release.Fix.DevOps.QABuild)", ErrInvalidFormat, s)
	}

	fields := make([]int, 5)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("%w: %q is not a valid integer in %q", ErrInvalidFormat, p, s)
		}
		fields[i] = n
	}

	return &Version{
		Sprint:  fields[0],
		Release: fields[1],
		Fixes:   fields[2],
		DevOps:  fields[3],
		QA:      fields[4],
	}, nil
}

// ParseReleaseName parses a Git Flow Plus release name, "Sprint.Release"
// (e.g. "5.2"), as used in `git flow release start <name>`.
func ParseReleaseName(name string) (sprint, release int, err error) {
	parts := strings.Split(name, ".")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("%w: %q (expected Sprint.Release, e.g. \"5.2\")", ErrInvalidReleaseName, name)
	}

	sprint, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("%w: sprint %q is not a valid integer", ErrInvalidReleaseName, parts[0])
	}
	release, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("%w: release %q is not a valid integer", ErrInvalidReleaseName, parts[1])
	}

	return sprint, release, nil
}
