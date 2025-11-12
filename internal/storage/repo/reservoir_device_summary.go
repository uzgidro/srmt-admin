package repo

import (
	"context"
	"database/sql"
	"fmt"
	"srmt-admin/internal/lib/dto"
	reservoirdevicesummary "srmt-admin/internal/lib/model/reservoir-device-summary"
	"srmt-admin/internal/storage"
	"strings"
)

// GetReservoirDeviceSummary retrieves all reservoir device summaries with organization name
func (r *Repo) GetReservoirDeviceSummary(ctx context.Context) ([]*reservoirdevicesummary.ResponseModel, error) {
	const op = "storage.repo.GetReservoirDeviceSummary"

	query := selectReservoirDeviceSummaryFields + fromReservoirDeviceSummaryJoins +
		`ORDER BY rds.organization_id, rds.device_type_name`

	rows, err := r.db.QueryContext(ctx, query)
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

// PatchReservoirDeviceSummary updates multiple reservoir device summaries in a single transaction
func (r *Repo) PatchReservoirDeviceSummary(ctx context.Context, req dto.PatchReservoirDeviceSummaryRequest, updatedByUserID int64) error {
	const op = "storage.repo.PatchReservoirDeviceSummary"

	if len(req.Updates) == 0 {
		return nil // Nothing to update
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: failed to begin transaction: %w", op, err)
	}
	defer tx.Rollback()

	for _, item := range req.Updates {
		// Check if record exists
		var exists bool
		err := tx.QueryRowContext(ctx,
			"SELECT EXISTS(SELECT 1 FROM reservoir_device_summary WHERE organization_id = $1 AND device_type_name = $2)",
			item.OrganizationID, item.DeviceTypeName,
		).Scan(&exists)
		if err != nil {
			return fmt.Errorf("%s: failed to check existence: %w", op, err)
		}
		if !exists {
			return storage.ErrNotFound
		}

		// Build dynamic UPDATE query
		var updates []string
		var args []interface{}
		argID := 1

		if item.CountTotal != nil {
			updates = append(updates, fmt.Sprintf("count_total = $%d", argID))
			args = append(args, *item.CountTotal)
			argID++
		}
		if item.CountInstalled != nil {
			updates = append(updates, fmt.Sprintf("count_installed = $%d", argID))
			args = append(args, *item.CountInstalled)
			argID++
		}
		if item.CountOperational != nil {
			updates = append(updates, fmt.Sprintf("count_operational = $%d", argID))
			args = append(args, *item.CountOperational)
			argID++
		}
		if item.CountFaulty != nil {
			updates = append(updates, fmt.Sprintf("count_faulty = $%d", argID))
			args = append(args, *item.CountFaulty)
			argID++
		}
		if item.CountActive != nil {
			updates = append(updates, fmt.Sprintf("count_active = $%d", argID))
			args = append(args, *item.CountActive)
			argID++
		}
		if item.CountAutomationScope != nil {
			updates = append(updates, fmt.Sprintf("count_automation_scope = $%d", argID))
			args = append(args, *item.CountAutomationScope)
			argID++
		}
		if item.Criterion1 != nil {
			updates = append(updates, fmt.Sprintf("criterion_1 = $%d", argID))
			args = append(args, *item.Criterion1)
			argID++
		}
		if item.Criterion2 != nil {
			updates = append(updates, fmt.Sprintf("criterion_2 = $%d", argID))
			args = append(args, *item.Criterion2)
			argID++
		}

		if len(updates) == 0 {
			continue // Nothing to update for this item
		}

		// Add updated_at and updated_by_user_id
		updates = append(updates, fmt.Sprintf("updated_at = NOW()"))
		updates = append(updates, fmt.Sprintf("updated_by_user_id = $%d", argID))
		args = append(args, updatedByUserID)
		argID++

		// Add WHERE clause parameters
		query := fmt.Sprintf(
			"UPDATE reservoir_device_summary SET %s WHERE organization_id = $%d AND device_type_name = $%d",
			strings.Join(updates, ", "),
			argID,
			argID+1,
		)
		args = append(args, item.OrganizationID, item.DeviceTypeName)

		res, err := tx.ExecContext(ctx, query, args...)
		if err != nil {
			if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
				return translatedErr
			}
			return fmt.Errorf("%s: failed to update reservoir device summary: %w", op, err)
		}

		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 0 {
			return storage.ErrNotFound
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
		&m.DeviceTypeName,
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

const (
	selectReservoirDeviceSummaryFields = `
		SELECT
			rds.id,
			rds.organization_id,
			COALESCE(o.name, '') as organization_name,
			rds.device_type_name,
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
	`
	fromReservoirDeviceSummaryJoins = `
		FROM
			reservoir_device_summary rds
		LEFT JOIN
			organizations o ON rds.organization_id = o.id
	`
)
