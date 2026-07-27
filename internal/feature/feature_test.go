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
	r.Upsert(feature.Feature{ID: "LOGIN", Branch: "feature/LOGIN"})

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
	r.Upsert(feature.Feature{ID: "LOGIN", Approved: false})
	r.Upsert(feature.Feature{ID: "LOGIN", Approved: true})

	if len(r.Features) != 1 {
		t.Fatalf("len(Features) = %d, want 1 (Upsert should replace, not duplicate)", len(r.Features))
	}
	got, _ := r.Find("LOGIN")
	if !got.Approved {
		t.Error("Approved = false, want true (Upsert should have replaced the entry)")
	}
}

func TestApprovedFiltersCorrectly(t *testing.T) {
	r := feature.New()
	r.Upsert(feature.Feature{ID: "LOGIN", Approved: true})
	r.Upsert(feature.Feature{ID: "PROFILE", Approved: false})
	r.Upsert(feature.Feature{ID: "PAYMENT", Approved: true})

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
		t.Error("Approved() included PROFILE, which is not approved")
	}
}
