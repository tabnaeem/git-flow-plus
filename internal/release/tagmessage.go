package release

import (
	"fmt"
	"strings"
	"time"
)

// tagMessageInput carries everything formatTagMessage needs to build an
// annotated tag's message. QABuild is omitted from the message when zero
// (the production tag doesn't have its own QA build number — it's the
// build the release finished at, already implied by Version).
type tagMessageInput struct {
	Release        string
	Version        string
	QABuild        int
	Features       []string
	ReleaseFixes   []string
	DevOps         []string
	Branch         string
	Commit         string
	ReleaseManager string
	ReleaseDate    time.Time
}

// formatTagMessage builds the multi-line annotated tag message used for
// both QA build tags and the production tag, so every tag Git Flow Plus
// creates is traceable back to exactly what it contains without needing to
// dig through commit history: which release, which version, who cut it,
// when, and which release fixes/DevOps changes are included.
func formatTagMessage(in tagMessageInput) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Release: %s\n", in.Release)
	fmt.Fprintf(&b, "Version: %s\n", in.Version)
	if in.QABuild > 0 {
		fmt.Fprintf(&b, "QA Build: %d\n", in.QABuild)
	}
	fmt.Fprintf(&b, "Release Date: %s\n", in.ReleaseDate.UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "Release Manager: %s\n", in.ReleaseManager)

	b.WriteString("\nFeatures:\n")
	writeList(&b, in.Features)

	b.WriteString("\nRelease Fixes:\n")
	writeList(&b, in.ReleaseFixes)

	b.WriteString("\nDevOps:\n")
	writeList(&b, in.DevOps)

	fmt.Fprintf(&b, "\nBranch: %s\n", in.Branch)
	fmt.Fprintf(&b, "Commit: %s\n", in.Commit)

	return b.String()
}

func writeList(b *strings.Builder, items []string) {
	if len(items) == 0 {
		b.WriteString("None\n")
		return
	}
	for _, item := range items {
		fmt.Fprintf(b, "%s\n", item)
	}
}
