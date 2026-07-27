package feature_test

import (
	"testing"

	"github.com/hulhub/git-flow-plus/internal/feature"
)

func TestNewIsEmpty(t *testing.T) {
	r := feature.New()
	if len(r.Features) != 0 {
		t.Errorf("New().Features = %v, want empty", r.Features)
	}
}

func TestFindMissingReturnsFalse(t *testing.T) {
	r := feature.New()
	_, ok := r.Find("LOGIN")
	if ok {
		t.Error("Find() on empty registry = true, want false")
	}
}

func TestUpsertInsertsNew(t *testing.T) {
	r := feature.New()
	r.Upsert(feature.Feature{ID: "LOGIN", Branch: "feature/LOGIN", State: feature.StateCreated})

	got, ok := r.Find("LOGIN")
	if !ok {
		t.Fatal("Find(LOGIN) = false after Upsert, want true")
	}
	if got.Branch != "feature/LOGIN" {
		t.Errorf("Branch = %q, want %q", got.Branch, "feature/LOGIN")
	}
}

func TestUpsertReplacesExisting(t *testing.T) {
	r := feature.New()
	r.Upsert(feature.Feature{ID: "LOGIN", State: feature.StateCreated})
	r.Upsert(feature.Feature{ID: "LOGIN", State: feature.StateApproved})

	if len(r.Features) != 1 {
		t.Fatalf("len(Features) = %d, want 1 (Upsert should replace, not duplicate)", len(r.Features))
	}
	got, _ := r.Find("LOGIN")
	if got.State != feature.StateApproved {
		t.Errorf("State = %q, want %q (Upsert should have replaced the entry)", got.State, feature.StateApproved)
	}
}

func TestApprovedFiltersCorrectly(t *testing.T) {
	r := feature.New()
	r.Upsert(feature.Feature{ID: "LOGIN", State: feature.StateApproved})
	r.Upsert(feature.Feature{ID: "PROFILE", State: feature.StateCreated})
	r.Upsert(feature.Feature{ID: "PAYMENT", State: feature.StateIncludedInRelease})

	approved := r.Approved()
	if len(approved) != 2 {
		t.Fatalf("len(Approved()) = %d, want 2", len(approved))
	}
	ids := map[string]bool{}
	for _, f := range approved {
		ids[f.ID] = true
	}
	if !ids["LOGIN"] || !ids["PAYMENT"] {
		t.Errorf("Approved() = %v, want LOGIN and PAYMENT", approved)
	}
	if ids["PROFILE"] {
		t.Error("Approved() included PROFILE, which hasn't reached StateApproved")
	}
}

func TestStateAtLeast(t *testing.T) {
	cases := []struct {
		state, other feature.State
		want         bool
	}{
		{feature.StateCreated, feature.StateCreated, true},
		{feature.StateApproved, feature.StateCreated, true},
		{feature.StateCreated, feature.StateApproved, false},
		{feature.StateIncludedInRelease, feature.StateApproved, true},
		{feature.StateApproved, feature.StateIncludedInRelease, false},
		{feature.StateArchived, feature.StateReleased, true},
		{"", feature.StateCreated, true}, // empty ranks as StateCreated (rank 0)
		{"", feature.StateApproved, false},
	}
	for _, c := range cases {
		if got := c.state.AtLeast(c.other); got != c.want {
			t.Errorf("State(%q).AtLeast(%q) = %v, want %v", c.state, c.other, got, c.want)
		}
	}
}
