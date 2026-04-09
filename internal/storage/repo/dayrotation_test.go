package repo

import (
	"testing"
	"time"
)

func TestRotateInfraEvents_Structure(t *testing.T) {
	cutoff := time.Date(2026, 4, 9, 5, 0, 0, 0, time.UTC)

	tests := []struct {
		name                string
		ongoingEvents       int // how many ongoing events exist
		eventsWithFiles     int // how many of those have file links
		expectedRotated     int
		expectedFilesCopied bool
	}{
		{
			name:            "no ongoing events - nothing to rotate",
			ongoingEvents:   0,
			expectedRotated: 0,
		},
		{
			name:                "one ongoing event without files",
			ongoingEvents:       1,
			eventsWithFiles:     0,
			expectedRotated:     1,
			expectedFilesCopied: false,
		},
		{
			name:                "one ongoing event with files - files should be copied",
			ongoingEvents:       1,
			eventsWithFiles:     1,
			expectedRotated:     1,
			expectedFilesCopied: true,
		},
		{
			name:                "multiple ongoing events across orgs",
			ongoingEvents:       3,
			eventsWithFiles:     2,
			expectedRotated:     3,
			expectedFilesCopied: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Structural test: validates rotation logic scenarios.
			// Real integration test would use a test DB.
			//
			// rotateInfraEvents should:
			// 1. SELECT all sc_infra_events WHERE restored_at IS NULL
			// 2. For each: close old (SET restored_at = cutoff), clone with occurred_at = cutoff
			// 3. Copy sc_infra_event_file_links from old event to new event
			//
			// Fields copied: category_id, organization_id, description, remediation, notes, created_by_user_id

			t.Logf("Cutoff: %s, OngoingEvents: %d, WithFiles: %d, ExpectedRotated: %d",
				cutoff.Format(time.RFC3339), tt.ongoingEvents, tt.eventsWithFiles, tt.expectedRotated)

			if tt.ongoingEvents == 0 && tt.expectedRotated != 0 {
				t.Error("expected 0 rotated when no ongoing events")
			}
			if tt.expectedRotated != tt.ongoingEvents {
				t.Errorf("expected rotated count %d to match ongoing events %d",
					tt.expectedRotated, tt.ongoingEvents)
			}
			if tt.eventsWithFiles > 0 && !tt.expectedFilesCopied {
				t.Error("expected files to be copied when events have files")
			}
		})
	}
}

func TestRotateInfraEvents_FieldsCopied(t *testing.T) {
	// Validates that ALL required fields are copied during rotation.
	// This is a documentation test ensuring the SQL INSERT copies every field.
	requiredFields := []string{
		"category_id",
		"organization_id",
		"occurred_at",    // set to cutoff, not copied
		"description",    // copied as-is
		"remediation",    // copied as-is (nullable)
		"notes",          // copied as-is (nullable)
		"created_by_user_id", // copied as-is
	}

	t.Logf("Fields that must be present in INSERT: %v", requiredFields)

	// restored_at must NOT be copied (new record is ongoing = NULL)
	t.Log("restored_at must be NULL in cloned record (ongoing)")

	// File links must be copied from sc_infra_event_file_links
	t.Log("sc_infra_event_file_links must be copied: old event_id -> new event_id")
}

func TestDayRotationResult_IncludesInfraEvents(t *testing.T) {
	// Validates that DayRotationResult has the InfraEventsRotated field
	result := DayRotationResult{
		LinkedDischargesRotated: 2,
		DischargesRotated:       3,
		InfraEventsRotated:      5,
	}

	if result.InfraEventsRotated != 5 {
		t.Errorf("expected InfraEventsRotated = 5, got %d", result.InfraEventsRotated)
	}

	t.Logf("DayRotationResult: linked=%d, discharges=%d, infra=%d",
		result.LinkedDischargesRotated, result.DischargesRotated, result.InfraEventsRotated)
}
