package repo

import "testing"

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
