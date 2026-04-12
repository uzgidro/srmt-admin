package repo

import (
	"context"
	"fmt"
	reservoirdata "srmt-admin/internal/lib/model/reservoir-data"
)

// UpsertReservoirData inserts or updates reservoir data records
// If a record with the same organization_id and date exists, it updates the values
// Otherwise, it creates a new record
func (r *Repo) UpsertReservoirData(ctx context.Context, data []reservoirdata.ReservoirDataItem, userID int64) error {
	const op = "storage.repo.UpsertReservoirData"

	if len(data) == 0 {
		return nil
	}

	// Use a transaction to ensure all records are inserted/updated atomically
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: failed to begin transaction: %w", op, err)
	}
	defer tx.Rollback()

	const query = `
		INSERT INTO reservoir_data (
			organization_id, date, income_m3_s, release_m3_s, level_m, volume_mln_m3,
			total_income_volume_mln_m3, total_income_volume_prev_year_mln_m3,
			created_by_user_id, updated_by_user_id, created_at, updated_at
		)
		VALUES (
			$1, $2,
			COALESCE($3, 0), COALESCE($4, 0), COALESCE($5, 0), COALESCE($6, 0),
			$7, $8,
			$9, $9, NOW(), NOW()
		)
		ON CONFLICT (organization_id, date)
		DO UPDATE SET
			income_m3_s = CASE WHEN $10::boolean
				THEN COALESCE(EXCLUDED.income_m3_s, 0)
				ELSE reservoir_data.income_m3_s END,
			release_m3_s = CASE WHEN $11::boolean
				THEN COALESCE(EXCLUDED.release_m3_s, 0)
				ELSE reservoir_data.release_m3_s END,
			level_m = CASE WHEN $12::boolean
				THEN COALESCE(EXCLUDED.level_m, 0)
				ELSE reservoir_data.level_m END,
			volume_mln_m3 = CASE WHEN $13::boolean
				THEN COALESCE(EXCLUDED.volume_mln_m3, 0)
				ELSE reservoir_data.volume_mln_m3 END,
			total_income_volume_mln_m3 = CASE WHEN $14::boolean
				THEN EXCLUDED.total_income_volume_mln_m3
				ELSE reservoir_data.total_income_volume_mln_m3 END,
			total_income_volume_prev_year_mln_m3 = CASE WHEN $15::boolean
				THEN EXCLUDED.total_income_volume_prev_year_mln_m3
				ELSE reservoir_data.total_income_volume_prev_year_mln_m3 END,
			updated_by_user_id = EXCLUDED.updated_by_user_id,
			updated_at = NOW()
	`

	const modsnowQuery = `
		INSERT INTO modsnow (
			organization_id, date, cover,
			created_by_user_id, updated_by_user_id, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $4, NOW(), NOW())
		ON CONFLICT (organization_id, date)
		DO UPDATE SET
			cover = EXCLUDED.cover,
			updated_by_user_id = EXCLUDED.updated_by_user_id,
			updated_at = NOW()
	`

	for _, item := range data {
		_, err := tx.ExecContext(
			ctx,
			query,
			item.OrganizationID,                  // $1
			item.Date,                            // $2
			item.Income.Value,                    // $3
			item.Release.Value,                   // $4
			item.Level.Value,                     // $5
			item.Volume.Value,                    // $6
			item.TotalIncomeVolume.Value,         // $7
			item.TotalIncomeVolumePrevYear.Value, // $8
			userID,                               // $9
			item.Income.Set,                      // $10
			item.Release.Set,                     // $11
			item.Level.Set,                       // $12
			item.Volume.Set,                      // $13
			item.TotalIncomeVolume.Set,           // $14
			item.TotalIncomeVolumePrevYear.Set,   // $15
		)
		if err != nil {
			if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
				return translatedErr
			}
			return fmt.Errorf("%s: failed to upsert reservoir data for org_id=%d, date=%s: %w",
				op, item.OrganizationID, item.Date, err)
		}

		// Upsert modsnow current value if provided
		if item.ModsnowCurrent != nil {
			_, err := tx.ExecContext(
				ctx,
				modsnowQuery,
				item.OrganizationID,
				item.Date,
				*item.ModsnowCurrent,
				userID,
			)
			if err != nil {
				if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
					return translatedErr
				}
				return fmt.Errorf("%s: failed to upsert modsnow current for org_id=%d, date=%s: %w",
					op, item.OrganizationID, item.Date, err)
			}
		}

		// Upsert modsnow year ago value if provided
		if item.ModsnowYearAgo != nil {
			_, err := tx.ExecContext(
				ctx,
				`INSERT INTO modsnow (
					organization_id, date, cover,
					created_by_user_id, updated_by_user_id, created_at, updated_at
				)
				VALUES ($1, $2::date - INTERVAL '1 year', $3, $4, $4, NOW(), NOW())
				ON CONFLICT (organization_id, date)
				DO UPDATE SET
					cover = EXCLUDED.cover,
					updated_by_user_id = EXCLUDED.updated_by_user_id,
					updated_at = NOW()`,
				item.OrganizationID,
				item.Date,
				*item.ModsnowYearAgo,
				userID,
			)
			if err != nil {
				if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
					return translatedErr
				}
				return fmt.Errorf("%s: failed to upsert modsnow year ago for org_id=%d, date=%s: %w",
					op, item.OrganizationID, item.Date, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: failed to commit transaction: %w", op, err)
	}

	return nil
}
