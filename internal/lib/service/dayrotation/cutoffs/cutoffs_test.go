package cutoffs

import (
	"errors"
	"testing"
	"time"
)

func tashkent(t *testing.T) *time.Location {
	t.Helper()
	return time.FixedZone("Asia/Tashkent", 5*3600)
}

func at(t *testing.T, loc *time.Location, y int, m time.Month, d, h, mn, sec int) time.Time {
	t.Helper()
	return time.Date(y, m, d, h, mn, sec, 0, loc)
}

// TestCompute covers the full backdate-rotation predicate: for a given
// (start, now) the helper enumerates every "day boundary" 05:00 in `loc` such
// that start < cutoff <= now. The resulting slice drives RotateBackdated*.
func TestCompute(t *testing.T) {
	loc := tashkent(t)

	cases := []struct {
		name  string
		start time.Time
		now   time.Time
		want  []time.Time
	}{
		{
			name:  "single cutoff today (start before 05:00, now after)",
			start: at(t, loc, 2026, 5, 12, 4, 30, 0),
			now:   at(t, loc, 2026, 5, 12, 9, 0, 0),
			want:  []time.Time{at(t, loc, 2026, 5, 12, 5, 0, 0)},
		},
		{
			name:  "multi-day backdate: 4 missed cutoffs",
			start: at(t, loc, 2026, 5, 9, 2, 0, 0),
			now:   at(t, loc, 2026, 5, 12, 9, 0, 0),
			want: []time.Time{
				at(t, loc, 2026, 5, 9, 5, 0, 0),
				at(t, loc, 2026, 5, 10, 5, 0, 0),
				at(t, loc, 2026, 5, 11, 5, 0, 0),
				at(t, loc, 2026, 5, 12, 5, 0, 0),
			},
		},
		{
			name:  "no rotation: start after today's cutoff",
			start: at(t, loc, 2026, 5, 12, 8, 0, 0),
			now:   at(t, loc, 2026, 5, 12, 9, 0, 0),
			want:  nil,
		},
		{
			name:  "no rotation: future start",
			start: at(t, loc, 2026, 5, 12, 11, 0, 0),
			now:   at(t, loc, 2026, 5, 12, 9, 0, 0),
			want:  nil,
		},
		{
			name:  "boundary: start one second before cutoff, now one second after",
			start: at(t, loc, 2026, 5, 12, 4, 59, 59),
			now:   at(t, loc, 2026, 5, 12, 5, 0, 1),
			want:  []time.Time{at(t, loc, 2026, 5, 12, 5, 0, 0)},
		},
		{
			name:  "boundary: start exactly at cutoff is NOT rotated (matches SQL `start_time < $1`)",
			start: at(t, loc, 2026, 5, 12, 5, 0, 0),
			now:   at(t, loc, 2026, 5, 12, 9, 0, 0),
			want:  nil,
		},
		{
			name:  "before-cutoff hour, now also before next cutoff: still 0 (need now > cutoff)",
			start: at(t, loc, 2026, 5, 12, 3, 0, 0),
			now:   at(t, loc, 2026, 5, 12, 4, 30, 0),
			want:  nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Compute(tc.start, tc.now, 5, loc)
			if err != nil {
				t.Fatalf("Compute: unexpected err: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("len: want %d, got %d (%v)", len(tc.want), len(got), got)
			}
			for i := range got {
				if !got[i].Equal(tc.want[i]) {
					t.Errorf("[%d]: want %s, got %s", i, tc.want[i], got[i])
				}
			}
		})
	}
}

// TestCompute_TooOld pins the 90-day backdate guard. start_time older than
// MaxBackdate (90) full cutoffs back returns ErrBackdateTooOld; handlers
// translate that to 400 Bad Request to protect against typo'd years and
// runaway transactions.
//
// Counting: each calendar day past start contributes one cutoff at 05:00. So
// from start=Feb 11 04:00 the cutoffs are Feb 11 05:00, Feb 12 05:00, …
// May 12 05:00 — inclusive on both ends. The test pins both the boundary at
// exactly 90 cutoffs (allowed) and 91 cutoffs (rejected).
func TestCompute_TooOld(t *testing.T) {
	loc := tashkent(t)
	now := at(t, loc, 2026, 5, 12, 9, 0, 0)

	allowedStart := at(t, loc, 2026, 2, 12, 4, 0, 0)
	cutoffs, err := Compute(allowedStart, now, 5, loc)
	if err != nil {
		t.Fatalf("Feb 12 04:00 should succeed, got err: %v", err)
	}
	if len(cutoffs) != 90 {
		t.Fatalf("Feb 12 04:00 should produce exactly 90 cutoffs, got %d", len(cutoffs))
	}

	tooOldStart := at(t, loc, 2026, 2, 11, 4, 0, 0)
	got, err := Compute(tooOldStart, now, 5, loc)
	if !errors.Is(err, ErrBackdateTooOld) {
		t.Fatalf("Feb 11 04:00 should give ErrBackdateTooOld, got err=%v slice=%v", err, got)
	}
	if got != nil {
		t.Fatalf("expected nil slice on error, got %v", got)
	}
}
