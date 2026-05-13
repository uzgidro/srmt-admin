package repo

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type DayRotationResult struct {
	LinkedDischargesRotated int
	DischargesRotated       int
}

// RotateDayBoundary closes all ongoing shutdowns and independent discharges at cutoff,
// clones them with new start_time, and copies file links. All in a single transaction.
func (r *Repo) RotateDayBoundary(ctx context.Context, cutoff time.Time) (*DayRotationResult, error) {
	const op = "storage.repo.dayrotation.RotateDayBoundary"

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: begin tx: %w", op, err)
	}
	defer tx.Rollback()

	// Step 1: Rotate discharges linked to ongoing shutdowns (shutdowns themselves stay open)
	linkedCount, err := rotateLinkedDischarges(ctx, tx, cutoff, op)
	if err != nil {
		return nil, err
	}

	// Step 2: Rotate independent discharges (not linked to shutdowns)
	dischargeCount, err := rotateDischarges(ctx, tx, cutoff, op)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("%s: commit: %w", op, err)
	}

	return &DayRotationResult{
		LinkedDischargesRotated: linkedCount,
		DischargesRotated:       dischargeCount,
	}, nil
}

// rotateLinkedDischarges rotates idle discharges that are linked to ongoing shutdowns.
// Shutdowns themselves stay open (not closed/cloned).
func rotateLinkedDischarges(ctx context.Context, tx *sql.Tx, cutoff time.Time, op string) (int, error) {
	const selectQuery = `
		SELECT s.id, s.idle_discharge_id
		FROM shutdowns s
		JOIN idle_water_discharges d ON d.id = s.idle_discharge_id
		WHERE s.end_time IS NULL
		  AND s.idle_discharge_id IS NOT NULL
		  AND d.start_time < $1`

	rows, err := tx.QueryContext(ctx, selectQuery, cutoff)
	if err != nil {
		return 0, fmt.Errorf("%s: select shutdowns with linked discharges: %w", op, err)
	}
	defer rows.Close()

	type linkedRow struct {
		ShutdownID      int64
		IdleDischargeID int64
	}

	var linked []linkedRow
	for rows.Next() {
		var r linkedRow
		if err := rows.Scan(&r.ShutdownID, &r.IdleDischargeID); err != nil {
			return 0, fmt.Errorf("%s: scan linked row: %w", op, err)
		}
		linked = append(linked, r)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("%s: rows error: %w", op, err)
	}

	for _, r := range linked {
		// Close old discharge
		_, err := tx.ExecContext(ctx,
			"UPDATE idle_water_discharges SET end_time = $1 WHERE id = $2",
			cutoff, r.IdleDischargeID)
		if err != nil {
			return 0, fmt.Errorf("%s: close idle_discharge %d: %w", op, r.IdleDischargeID, err)
		}

		// Clone discharge (shutdown keeps pointing to the old one)
		_, err = tx.ExecContext(ctx, `
			INSERT INTO idle_water_discharges (organization_id, start_time, flow_rate_m3_s, reason, created_by)
			SELECT organization_id, $1, flow_rate_m3_s, reason, created_by
			FROM idle_water_discharges WHERE id = $2`,
			cutoff, r.IdleDischargeID)
		if err != nil {
			return 0, fmt.Errorf("%s: clone idle_discharge %d: %w", op, r.IdleDischargeID, err)
		}
	}

	return len(linked), nil
}

// RotateBackdatedDischarge applies the day-rotation logic to a freshly-inserted
// independent discharge whose start_time predates one or more 05:00 cutoffs.
// For each cutoff in ascending order: close the current ongoing record at
// cutoff, clone it with start_time=cutoff (file_links carried over), then
// continue the chain from the new clone. Returns the final clone's ID — that
// is the row the frontend will see as the "current ongoing" discharge.
//
// cutoffs MUST be in ascending order (as produced by dayrotation.computeCutoffs).
// Empty cutoffs → returns dischargeID unchanged with no DB activity.
//
// This mirrors rotateDischarges (independent path of the dayrotation ticker).
func (r *Repo) RotateBackdatedDischarge(ctx context.Context, dischargeID int64, cutoffs []time.Time) (int64, error) {
	const op = "storage.repo.dayrotation.RotateBackdatedDischarge"
	if len(cutoffs) == 0 {
		return dischargeID, nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("%s: begin tx: %w", op, err)
	}
	defer tx.Rollback()

	finalID, err := rotateBackdatedDischargeChainTx(ctx, tx, dischargeID, cutoffs, true /*copyFiles*/, op)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("%s: commit: %w", op, err)
	}
	return finalID, nil
}

// RotateBackdatedLinkedDischarge applies the day-rotation logic to a
// freshly-inserted discharge that is linked to a shutdown. Identical to
// RotateBackdatedDischarge except file_links are NOT copied to the clones —
// the shutdown stays pointing at the original (now closed) discharge,
// symmetric with rotateLinkedDischarges in the dayrotation ticker.
//
// Note: when called as part of a larger transaction (AddShutdown), use the
// private rotateBackdatedDischargeChainTx directly to keep atomicity.
func (r *Repo) RotateBackdatedLinkedDischarge(ctx context.Context, dischargeID int64, cutoffs []time.Time) (int64, error) {
	const op = "storage.repo.dayrotation.RotateBackdatedLinkedDischarge"
	if len(cutoffs) == 0 {
		return dischargeID, nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("%s: begin tx: %w", op, err)
	}
	defer tx.Rollback()

	finalID, err := rotateBackdatedDischargeChainTx(ctx, tx, dischargeID, cutoffs, false /*copyFiles*/, op)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("%s: commit: %w", op, err)
	}
	return finalID, nil
}

// rotateBackdatedDischargeChainTx is the shared close+clone[+copyFiles] loop.
// Exposed at package scope so AddShutdown can run rotation inside its own tx.
//
// Per cutoff: UPDATE end_time=cutoff on prevID, INSERT clone selecting from
// prevID with start_time=cutoff, optionally INSERT discharge_file_links
// rows from prevID to newID. Then prevID := newID and continue.
//
// The unique partial index idx_one_ongoing_discharge_per_org is satisfied at
// every step because the close happens before the insert within the same
// transaction snapshot.
func rotateBackdatedDischargeChainTx(ctx context.Context, tx *sql.Tx, dischargeID int64, cutoffs []time.Time, copyFiles bool, op string) (int64, error) {
	prevID := dischargeID
	for _, cutoff := range cutoffs {
		if _, err := tx.ExecContext(ctx,
			"UPDATE idle_water_discharges SET end_time = $1 WHERE id = $2 AND end_time IS NULL",
			cutoff, prevID); err != nil {
			return 0, fmt.Errorf("%s: close %d at %s: %w", op, prevID, cutoff.Format(time.RFC3339), err)
		}

		var newID int64
		if err := tx.QueryRowContext(ctx, `
			INSERT INTO idle_water_discharges (organization_id, start_time, flow_rate_m3_s, reason, created_by)
			SELECT organization_id, $1, flow_rate_m3_s, reason, created_by
			FROM idle_water_discharges WHERE id = $2
			RETURNING id`,
			cutoff, prevID).Scan(&newID); err != nil {
			return 0, fmt.Errorf("%s: clone from %d at %s: %w", op, prevID, cutoff.Format(time.RFC3339), err)
		}

		if copyFiles {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO discharge_file_links (discharge_id, file_id)
				SELECT $1, file_id FROM discharge_file_links WHERE discharge_id = $2`,
				newID, prevID); err != nil {
				return 0, fmt.Errorf("%s: copy file links %d->%d: %w", op, prevID, newID, err)
			}
		}

		prevID = newID
	}
	return prevID, nil
}

func rotateDischarges(ctx context.Context, tx *sql.Tx, cutoff time.Time, op string) (int, error) {
	const selectQuery = `
		SELECT id, organization_id, flow_rate_m3_s, reason, created_by
		FROM idle_water_discharges
		WHERE end_time IS NULL
		  AND start_time < $1
		  AND id NOT IN (SELECT idle_discharge_id FROM shutdowns WHERE idle_discharge_id IS NOT NULL)`

	rows, err := tx.QueryContext(ctx, selectQuery, cutoff)
	if err != nil {
		return 0, fmt.Errorf("%s: select ongoing discharges: %w", op, err)
	}
	defer rows.Close()

	type dischargeRow struct {
		ID        int64
		OrgID     int64
		FlowRate  float64
		Reason    *string
		CreatedBy int64
	}

	var discharges []dischargeRow
	for rows.Next() {
		var d dischargeRow
		var reason sql.NullString
		if err := rows.Scan(&d.ID, &d.OrgID, &d.FlowRate, &reason, &d.CreatedBy); err != nil {
			return 0, fmt.Errorf("%s: scan discharge: %w", op, err)
		}
		if reason.Valid {
			d.Reason = &reason.String
		}
		discharges = append(discharges, d)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("%s: rows error: %w", op, err)
	}

	for _, d := range discharges {
		// Close old discharge
		_, err := tx.ExecContext(ctx,
			"UPDATE idle_water_discharges SET end_time = $1 WHERE id = $2",
			cutoff, d.ID)
		if err != nil {
			return 0, fmt.Errorf("%s: close discharge %d: %w", op, d.ID, err)
		}

		// Clone discharge
		var newID int64
		err = tx.QueryRowContext(ctx, `
			INSERT INTO idle_water_discharges (organization_id, start_time, flow_rate_m3_s, reason, created_by)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id`,
			d.OrgID, cutoff, d.FlowRate, d.Reason, d.CreatedBy).Scan(&newID)
		if err != nil {
			return 0, fmt.Errorf("%s: clone discharge %d: %w", op, d.ID, err)
		}

		// Copy file links
		_, err = tx.ExecContext(ctx, `
			INSERT INTO discharge_file_links (discharge_id, file_id)
			SELECT $1, file_id FROM discharge_file_links WHERE discharge_id = $2`,
			newID, d.ID)
		if err != nil {
			return 0, fmt.Errorf("%s: copy discharge file links %d: %w", op, d.ID, err)
		}
	}

	return len(discharges), nil
}
