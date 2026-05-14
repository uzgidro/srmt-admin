// Package cutoffs computes the list of "day-boundary" timestamps a backdated
// record must be rotated across. Pure helper, no DB or other deps — kept in
// its own subpackage to avoid an import cycle (storage/repo needs this; the
// parent dayrotation package needs storage/repo for the ticker).
package cutoffs

import (
	"errors"
	"time"
)

// MaxBackdate caps how many cutoffs Compute will return before refusing the
// input. 90 ≈ one quarter — a typo'd year (e.g. 2025 vs 2026) is much further
// back than this and gets rejected at the boundary.
const MaxBackdate = 90

// ErrBackdateTooOld is returned by Compute when start_time is more than
// MaxBackdate cutoffs in the past. Handlers translate this into HTTP 400 to
// protect against runaway transactions and obvious data-entry errors. The
// day-rotation ticker is not subject to this limit; it always processes a
// single cutoff.
var ErrBackdateTooOld = errors.New("start_time is too far in the past for backdate rotation")

// Compute returns every "day-boundary" timestamp in `loc` at hour
// `cutoffHour` that satisfies start < cutoff <= now. The slice is in
// ascending order. Empty result means no rotation is needed (start is on or
// after the most recent cutoff, or in the future).
//
// Used by both POST /discharges and POST /shutdowns to emulate the
// dayrotation ticker for backdated records: every cutoff in the result
// corresponds to one missed rotation that needs to be applied synchronously.
//
// Mirrors the ticker's SQL filter `start_time < cutoff` (strict): a record
// starting exactly at cutoff is NOT rotated.
func Compute(start, now time.Time, cutoffHour int, loc *time.Location) ([]time.Time, error) {
	if !start.Before(now) {
		return nil, nil
	}

	// First candidate cutoff = start's local-day at cutoffHour. If start is
	// already at or past that hour, the first cutoff is the next day.
	startLocal := start.In(loc)
	cutoff := time.Date(startLocal.Year(), startLocal.Month(), startLocal.Day(), cutoffHour, 0, 0, 0, loc)
	if !cutoff.After(start) {
		// time.Date with day+1 is DST-safe; +24h would be wrong on DST boundaries.
		cutoff = time.Date(startLocal.Year(), startLocal.Month(), startLocal.Day()+1, cutoffHour, 0, 0, 0, loc)
	}

	var out []time.Time
	for !cutoff.After(now) {
		out = append(out, cutoff)
		if len(out) > MaxBackdate {
			return nil, ErrBackdateTooOld
		}
		cutoff = time.Date(cutoff.Year(), cutoff.Month(), cutoff.Day()+1, cutoffHour, 0, 0, 0, loc)
	}
	return out, nil
}
