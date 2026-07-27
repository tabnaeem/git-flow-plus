package version_test

import (
	"errors"
	"testing"

	"github.com/hulhub/git-flow-plus/internal/version"
)

func TestNewIsSprintReleaseZeroZeroOne(t *testing.T) {
	v := version.New(5, 2)
	want := "5.2.0.0.1"
	if got := v.String(); got != want {
		t.Errorf("New(5, 2).String() = %q, want %q", got, want)
	}
	if v.QA != 1 {
		t.Errorf("New(5, 2).QA = %d, want 1 (first QA build is automatic)", v.QA)
	}
}

func TestApplyBuildFixesOnly(t *testing.T) {
	v := version.New(5, 2)
	v.ApplyBuild(4, 0)
	if v.Fixes != 4 || v.DevOps != 0 || v.QA != 2 {
		t.Errorf("after ApplyBuild(4, 0): Fixes=%d DevOps=%d QA=%d, want Fixes=4 DevOps=0 QA=2", v.Fixes, v.DevOps, v.QA)
	}
	if got, want := v.String(), "5.2.4.0.2"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestApplyBuildFixesAndDevOps(t *testing.T) {
	v := version.New(5, 2)
	v.ApplyBuild(4, 0)
	v.ApplyBuild(2, 1)
	if v.Fixes != 6 || v.DevOps != 1 || v.QA != 3 {
		t.Errorf("after two ApplyBuild calls: Fixes=%d DevOps=%d QA=%d, want Fixes=6 DevOps=1 QA=3", v.Fixes, v.DevOps, v.QA)
	}
	if got, want := v.String(), "5.2.6.1.3"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestApplyBuildZeroChangeStillAdvancesQA(t *testing.T) {
	// A build with nothing new still represents a QA iteration if called;
	// callers (internal/release) are expected to guard against calling
	// this with nothing pending, but the primitive itself just does the
	// arithmetic it's told to.
	v := version.New(5, 2)
	v.ApplyBuild(0, 0)
	if v.Fixes != 0 || v.DevOps != 0 || v.QA != 2 {
		t.Errorf("after ApplyBuild(0, 0): Fixes=%d DevOps=%d QA=%d, want Fixes=0 DevOps=0 QA=2", v.Fixes, v.DevOps, v.QA)
	}
}

func TestParseRoundTrip(t *testing.T) {
	v, err := version.Parse("5.2.3.1.4")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	want := version.Version{Sprint: 5, Release: 2, Fixes: 3, DevOps: 1, QA: 4}
	if *v != want {
		t.Errorf("Parse() = %+v, want %+v", *v, want)
	}
	if v.String() != "5.2.3.1.4" {
		t.Errorf("round-trip String() = %q, want %q", v.String(), "5.2.3.1.4")
	}
}

func TestParseRejectsInvalidFormats(t *testing.T) {
	cases := []string{"", "5.2", "5.2.3.1", "5.2.3.1.4.5", "a.2.3.1.4", "5.2.3.1.x"}
	for _, s := range cases {
		if _, err := version.Parse(s); !errors.Is(err, version.ErrInvalidFormat) {
			t.Errorf("Parse(%q) error = %v, want ErrInvalidFormat", s, err)
		}
	}
}

func TestParseReleaseName(t *testing.T) {
	sprint, release, err := version.ParseReleaseName("5.2")
	if err != nil {
		t.Fatalf("ParseReleaseName() error = %v", err)
	}
	if sprint != 5 || release != 2 {
		t.Errorf("ParseReleaseName() = (%d, %d), want (5, 2)", sprint, release)
	}
}

func TestParseReleaseNameRejectsInvalidFormats(t *testing.T) {
	cases := []string{"", "5", "5.2.3", "checkout-redesign", "x.2"}
	for _, s := range cases {
		if _, _, err := version.ParseReleaseName(s); !errors.Is(err, version.ErrInvalidReleaseName) {
			t.Errorf("ParseReleaseName(%q) error = %v, want ErrInvalidReleaseName", s, err)
		}
	}
}
