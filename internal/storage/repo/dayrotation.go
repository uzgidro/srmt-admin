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
		WHERE s.end_time IS NULL
		  AND s.idle_discharge_id IS NOT NULL`

	rows, err := tx.QueryContext(ctx, selectQuery)
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

func rotateDischarges(ctx context.Context, tx *sql.Tx, cutoff time.Time, op string) (int, error) {
	const selectQuery = `
		SELECT id, organization_id, flow_rate_m3_s, reason, created_by
		FROM idle_water_discharges
		WHERE end_time IS NULL
		  AND id NOT IN (SELECT idle_discharge_id FROM shutdowns WHERE idle_discharge_id IS NOT NULL)`

	rows, err := tx.QueryContext(ctx, selectQuery)
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
