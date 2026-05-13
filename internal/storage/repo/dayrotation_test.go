package repo

import (
	"context"
	"testing"
	"time"
)

func TestDayRotationResult_Fields(t *testing.T) {
	result := DayRotationResult{
		LinkedDischargesRotated: 2,
		DischargesRotated:       3,
	}

	if result.LinkedDischargesRotated != 2 {
		t.Errorf("expected LinkedDischargesRotated = 2, got %d", result.LinkedDischargesRotated)
	}
	if result.DischargesRotated != 3 {
		t.Errorf("expected DischargesRotated = 3, got %d", result.DischargesRotated)
	}
}

// TestRotateBackdatedDischarge_NoCutoffs_NoOp pins the contract that calling
// RotateBackdatedDischarge with an empty cutoffs slice short-circuits without
// any DB work, returning the input ID. Handler code relies on this so it can
// always call the method (even when start_time is current) without conditional
// logic — defense in depth on top of the handler-side `len(cutoffs) > 0` check.
func TestRotateBackdatedDischarge_NoCutoffs_NoOp(t *testing.T) {
	r := &Repo{} // db is nil; if the function ever touches it for empty cutoffs, this panics.
	got, err := r.RotateBackdatedDischarge(context.Background(), 42, nil)
	if err != nil {
		t.Fatalf("nil cutoffs should not error: %v", err)
	}
	if got != 42 {
		t.Errorf("want input ID 42, got %d", got)
	}

	got, err = r.RotateBackdatedDischarge(context.Background(), 42, []time.Time{})
	if err != nil {
		t.Fatalf("empty cutoffs should not error: %v", err)
	}
	if got != 42 {
		t.Errorf("want input ID 42, got %d", got)
	}
}

// TestRotateBackdatedLinkedDischarge_NoCutoffs_NoOp mirrors the above for the
// linked variant. Same contract: empty cutoffs is a no-op, returns input ID.
func TestRotateBackdatedLinkedDischarge_NoCutoffs_NoOp(t *testing.T) {
	r := &Repo{} // db is nil — must not be touched.
	got, err := r.RotateBackdatedLinkedDischarge(context.Background(), 99, nil)
	if err != nil {
		t.Fatalf("nil cutoffs should not error: %v", err)
	}
	if got != 99 {
		t.Errorf("want input ID 99, got %d", got)
	}
}
