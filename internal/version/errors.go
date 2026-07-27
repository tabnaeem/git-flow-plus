package version

import "errors"

var (
	// ErrInvalidFormat indicates a string did not parse as a
	// Sprint.Release.Fix.DevOps.QABuild version.
	ErrInvalidFormat = errors.New("invalid version format")
	// ErrInvalidReleaseName indicates a string did not parse as a
	// Sprint.Release release name.
	ErrInvalidReleaseName = errors.New("invalid release name")
)
