package repo

import (
	"context"
	"database/sql"
	"fmt"
	reservoirdevicesummary "srmt-admin/internal/lib/model/reservoir-device-summary"
	"srmt-admin/internal/storage"
	"time"

	"srmt-admin/internal/lib/dto"
)

// GetReservoirDeviceSummary retrieves the latest version of each reservoir device summary.
// If date is non-nil, returns the latest version on or before that date.
func (r *Repo) GetReservoirDeviceSummary(ctx context.Context, date *time.Time) ([]*reservoirdevicesummary.ResponseModel, error) {
	const op = "storage.repo.GetReservoirDeviceSummary"

	query := `
		SELECT DISTINCT ON (rds.organization_id)
			rds.id,
			rds.organization_id,
			COALESCE(o.name, '') AS organization_name,
			rds.count_total,
			rds.count_installed,
			rds.count_operational,
			rds.count_faulty,
			rds.count_active,
			rds.count_automation_scope,
			rds.criterion_1,
			rds.criterion_2,
			rds.created_at,
			rds.updated_at,
			rds.updated_by_user_id
		FROM reservoir_device_summary rds
		LEFT JOIN organizations o ON rds.organization_id = o.id
		WHERE ($1::timestamptz IS NULL OR rds.created_at <= $1)
		ORDER BY rds.organization_id, rds.created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, date)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query reservoir device summaries: %w", op, err)
	}
	defer rows.Close()

	var summaries []*reservoirdevicesummary.ResponseModel
	for rows.Next() {
		m, err := scanReservoirDeviceSummaryRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan reservoir device summary row: %w", op, err)
		}
		summaries = append(summaries, m)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}
	if summaries == nil {
		summaries = make([]*reservoirdevicesummary.ResponseModel, 0)
	}
	return summaries, nil
}

// PatchReservoirDeviceSummary creates new versioned rows for reservoir device summaries.
// For each item, it fetches the current (latest) version, merges non-nil fields, and INSERTs a new row.
func (r *Repo) PatchReservoirDeviceSummary(ctx context.Context, req dto.PatchReservoirDeviceSummaryRequest, updatedByUserID int64) error {
	const op = "storage.repo.PatchReservoirDeviceSummary"

	if len(req.Updates) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: failed to begin transaction: %w", op, err)
	}
	defer tx.Rollback()

	for _, item := range req.Updates {
		// Fetch the latest version for this organization
		var cur reservoirdevicesummary.ResponseModel
		var criterion1, criterion2 sql.NullFloat64

		err := tx.QueryRowContext(ctx, `
			SELECT
				organization_id,
				count_total, count_installed, count_operational,
				count_faulty, count_active, count_automation_scope,
				criterion_1, criterion_2
			FROM reservoir_device_summary
			WHERE organization_id = $1
			ORDER BY created_at DESC
			LIMIT 1`, item.OrganizationID,
		).Scan(
			&cur.OrganizationID,
			&cur.CountTotal, &cur.CountInstalled, &cur.CountOperational,
			&cur.CountFaulty, &cur.CountActive, &cur.CountAutomationScope,
			&criterion1, &criterion2,
		)
		if err != nil {
			if err == sql.ErrNoRows {
				return storage.ErrNotFound
			}
			return fmt.Errorf("%s: failed to fetch current version: %w", op, err)
		}

		if criterion1.Valid {
			cur.Criterion1 = &criterion1.Float64
		}
		if criterion2.Valid {
			cur.Criterion2 = &criterion2.Float64
		}

		// Merge: apply non-nil fields from the request
		if item.CountTotal != nil {
			cur.CountTotal = *item.CountTotal
		}
		if item.CountInstalled != nil {
			cur.CountInstalled = *item.CountInstalled
		}
		if item.CountOperational != nil {
			cur.CountOperational = *item.CountOperational
		}
		if item.CountFaulty != nil {
			cur.CountFaulty = *item.CountFaulty
		}
		if item.CountActive != nil {
			cur.CountActive = *item.CountActive
		}
		if item.CountAutomationScope != nil {
			cur.CountAutomationScope = *item.CountAutomationScope
		}
		if item.Criterion1 != nil {
			cur.Criterion1 = item.Criterion1
		}
		if item.Criterion2 != nil {
			cur.Criterion2 = item.Criterion2
		}

		// INSERT new versioned row
		_, err = tx.ExecContext(ctx, `
			INSERT INTO reservoir_device_summary (
				organization_id,
				count_total, count_installed, count_operational,
				count_faulty, count_active, count_automation_scope,
				criterion_1, criterion_2,
				updated_by_user_id
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			item.OrganizationID,
			cur.CountTotal, cur.CountInstalled, cur.CountOperational,
			cur.CountFaulty, cur.CountActive, cur.CountAutomationScope,
			cur.Criterion1, cur.Criterion2,
			updatedByUserID,
		)
		if err != nil {
			if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
				return translatedErr
			}
			return fmt.Errorf("%s: failed to insert reservoir device summary version: %w", op, err)
		}
	}

	return tx.Commit()
}

func scanReservoirDeviceSummaryRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*reservoirdevicesummary.ResponseModel, error) {
	var m reservoirdevicesummary.ResponseModel
	var (
		criterion1, criterion2 sql.NullFloat64
		updatedAt              sql.NullTime
		updatedByUserID        sql.NullInt64
	)

	err := scanner.Scan(
		&m.ID,
		&m.OrganizationID,
		&m.OrganizationName,
		&m.CountTotal,
		&m.CountInstalled,
		&m.CountOperational,
		&m.CountFaulty,
		&m.CountActive,
		&m.CountAutomationScope,
		&criterion1,
		&criterion2,
		&m.CreatedAt,
		&updatedAt,
		&updatedByUserID,
	)
	if err != nil {
		return nil, err
	}

	if criterion1.Valid {
		m.Criterion1 = &criterion1.Float64
	}
	if criterion2.Valid {
		m.Criterion2 = &criterion2.Float64
	}
	if updatedAt.Valid {
		m.UpdatedAt = &updatedAt.Time
	}
	if updatedByUserID.Valid {
		m.UpdatedByUserID = &updatedByUserID.Int64
	}

	return &m, nil
}
