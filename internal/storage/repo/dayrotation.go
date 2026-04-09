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
	InfraEventsRotated      int
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

	// Step 3: Rotate ongoing infra events
	infraCount, err := rotateInfraEvents(ctx, tx, cutoff, op)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("%s: commit: %w", op, err)
	}

	return &DayRotationResult{
		LinkedDischargesRotated: linkedCount,
		DischargesRotated:       dischargeCount,
		InfraEventsRotated:      infraCount,
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

func rotateInfraEvents(ctx context.Context, tx *sql.Tx, cutoff time.Time, op string) (int, error) {
	const selectQuery = `
		SELECT id, category_id, organization_id, description, remediation, notes, created_by_user_id
		FROM sc_infra_events
		WHERE restored_at IS NULL
		  AND occurred_at < $1`

	rows, err := tx.QueryContext(ctx, selectQuery, cutoff)
	if err != nil {
		return 0, fmt.Errorf("%s: select ongoing infra events: %w", op, err)
	}
	defer rows.Close()

	type infraRow struct {
		ID          int64
		CategoryID  int64
		OrgID       int64
		Description string
		Remediation *string
		Notes       *string
		CreatedBy   *int64
	}

	var events []infraRow
	for rows.Next() {
		var e infraRow
		var remediation, notes sql.NullString
		var createdBy sql.NullInt64
		if err := rows.Scan(&e.ID, &e.CategoryID, &e.OrgID, &e.Description, &remediation, &notes, &createdBy); err != nil {
			return 0, fmt.Errorf("%s: scan infra event: %w", op, err)
		}
		if remediation.Valid {
			e.Remediation = &remediation.String
		}
		if notes.Valid {
			e.Notes = &notes.String
		}
		if createdBy.Valid {
			e.CreatedBy = &createdBy.Int64
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("%s: rows error: %w", op, err)
	}

	for _, e := range events {
		// Close old event
		_, err := tx.ExecContext(ctx,
			"UPDATE sc_infra_events SET restored_at = $1 WHERE id = $2",
			cutoff, e.ID)
		if err != nil {
			return 0, fmt.Errorf("%s: close infra event %d: %w", op, e.ID, err)
		}

		// Clone event
		var newID int64
		err = tx.QueryRowContext(ctx, `
			INSERT INTO sc_infra_events (category_id, organization_id, occurred_at, description, remediation, notes, created_by_user_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id`,
			e.CategoryID, e.OrgID, cutoff, e.Description, e.Remediation, e.Notes, e.CreatedBy).Scan(&newID)
		if err != nil {
			return 0, fmt.Errorf("%s: clone infra event %d: %w", op, e.ID, err)
		}

		// Copy file links
		_, err = tx.ExecContext(ctx, `
			INSERT INTO sc_infra_event_file_links (event_id, file_id)
			SELECT $1, file_id FROM sc_infra_event_file_links WHERE event_id = $2`,
			newID, e.ID)
		if err != nil {
			return 0, fmt.Errorf("%s: copy infra event file links %d: %w", op, e.ID, err)
		}
	}

	return len(events), nil
}
