package repo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"

	"srmt-admin/internal/lib/model/solar"
	"srmt-admin/internal/storage"
)

// --- Solar Config CRUD ---

// UpsertSolarConfig inserts or updates the per-organization solar config row.
// Conflict key is organization_id (UNIQUE in the table).
func (r *Repo) UpsertSolarConfig(ctx context.Context, req solar.UpsertConfigRequest) error {
	const op = "storage.repo.Solar.UpsertConfig"

	const query = `
		INSERT INTO solar_config (organization_id, installed_capacity_kw, sort_order)
		VALUES ($1, $2, $3)
		ON CONFLICT (organization_id) DO UPDATE SET
			installed_capacity_kw = EXCLUDED.installed_capacity_kw,
			sort_order            = EXCLUDED.sort_order,
			updated_at            = NOW()`

	if _, err := r.db.ExecContext(ctx, query, req.OrganizationID, req.InstalledCapacityKW, req.SortOrder); err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

// GetAllSolarConfigs returns all solar configs joined with organization name,
// ordered by sort_order then organization name.
func (r *Repo) GetAllSolarConfigs(ctx context.Context) ([]solar.Config, error) {
	const op = "storage.repo.Solar.GetAllConfigs"

	const query = `
		SELECT c.id, c.organization_id, COALESCE(o.name, ''),
		       c.installed_capacity_kw, c.sort_order, c.updated_at
		FROM solar_config c
		LEFT JOIN organizations o ON o.id = c.organization_id
		ORDER BY c.sort_order, COALESCE(o.name, '')`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: query: %w", op, err)
	}
	defer rows.Close()

	out := make([]solar.Config, 0)
	for rows.Next() {
		var c solar.Config
		if err := rows.Scan(
			&c.ID, &c.OrganizationID, &c.OrganizationName,
			&c.InstalledCapacityKW, &c.SortOrder, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows: %w", op, err)
	}
	return out, nil
}

// DeleteSolarConfig removes the solar config for the given org.
// Returns storage.ErrNotFound if no row matched.
func (r *Repo) DeleteSolarConfig(ctx context.Context, organizationID int64) error {
	const op = "storage.repo.Solar.DeleteConfig"

	res, err := r.db.ExecContext(ctx,
		`DELETE FROM solar_config WHERE organization_id = $1`, organizationID)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: rows affected: %w", op, err)
	}
	if n == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// --- Solar Daily Data ---

// UpsertSolarDailyData performs a bulk upsert of solar daily readings inside a
// single transaction. Date on each item is a YYYY-MM-DD string parsed in the
// repo before write. Per-field overrides use the optional.Optional Set flag via
// CASE flag parameters so absent fields preserve existing values while
// explicit nulls/values are written through.
func (r *Repo) UpsertSolarDailyData(ctx context.Context, items []solar.UpsertDailyDataRequest, userID int64) error {
	const op = "storage.repo.Solar.UpsertDailyData"

	if len(items) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: begin tx: %w", op, err)
	}
	defer tx.Rollback()

	const query = `
		INSERT INTO solar_daily_data (
			organization_id, date,
			generation_kwh, grid_export_kwh,
			created_by_user_id, updated_by_user_id, created_at, updated_at
		)
		VALUES (
			$1, $2::date,
			$3, $4,
			$5, $5, NOW(), NOW()
		)
		ON CONFLICT (organization_id, date) DO UPDATE SET
			generation_kwh     = CASE WHEN $6::boolean THEN EXCLUDED.generation_kwh     ELSE solar_daily_data.generation_kwh     END,
			grid_export_kwh    = CASE WHEN $7::boolean THEN EXCLUDED.grid_export_kwh    ELSE solar_daily_data.grid_export_kwh    END,
			updated_by_user_id = EXCLUDED.updated_by_user_id,
			updated_at         = NOW()`

	for _, it := range items {
		if _, execErr := tx.ExecContext(ctx, query,
			it.OrganizationID,       // $1
			it.Date,                 // $2
			it.GenerationKWh.Value,  // $3
			it.GridExportKWh.Value,  // $4
			userID,                  // $5
			it.GenerationKWh.Set,    // $6
			it.GridExportKWh.Set,    // $7
		); execErr != nil {
			if translatedErr := r.translator.Translate(execErr, op); translatedErr != nil {
				return translatedErr
			}
			return fmt.Errorf("%s: upsert org=%d date=%s: %w",
				op, it.OrganizationID, it.Date, execErr)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: commit: %w", op, err)
	}
	return nil
}

// GetSolarDailyDataRange returns solar daily readings for the given orgIDs
// in the inclusive date range [start, end]. When orgIDs is empty/nil, returns
// readings for ALL organizations that have a row in solar_config.
func (r *Repo) GetSolarDailyDataRange(ctx context.Context, orgIDs []int64, start, end time.Time) ([]solar.DailyData, error) {
	const op = "storage.repo.Solar.GetDailyDataRange"

	var (
		rows *sql.Rows
		err  error
	)

	// Half-open window [start, end). Caller passes end = start + 24h for a
	// single-day query; the strict `<` upper bound prevents Postgres' date
	// cast of the end timestamp from inadvertently including the next
	// calendar day.
	if len(orgIDs) == 0 {
		const query = `
			SELECT d.id, d.organization_id, COALESCE(o.name, ''),
			       d.date,
			       d.generation_kwh, d.grid_export_kwh,
			       d.updated_at
			FROM solar_daily_data d
			JOIN solar_config c ON c.organization_id = d.organization_id
			LEFT JOIN organizations o ON o.id = d.organization_id
			WHERE d.date >= $1::date AND d.date < $2::date
			ORDER BY d.organization_id, d.date`
		rows, err = r.db.QueryContext(ctx, query, start, end)
	} else {
		const query = `
			SELECT d.id, d.organization_id, COALESCE(o.name, ''),
			       d.date,
			       d.generation_kwh, d.grid_export_kwh,
			       d.updated_at
			FROM solar_daily_data d
			LEFT JOIN organizations o ON o.id = d.organization_id
			WHERE d.date >= $1::date AND d.date < $2::date
			  AND d.organization_id = ANY($3)
			ORDER BY d.organization_id, d.date`
		rows, err = r.db.QueryContext(ctx, query, start, end, pq.Array(orgIDs))
	}
	if err != nil {
		return nil, fmt.Errorf("%s: query: %w", op, err)
	}
	defer rows.Close()

	out := make([]solar.DailyData, 0)
	for rows.Next() {
		var rec solar.DailyData
		var generation, gridExport sql.NullFloat64
		if err := rows.Scan(
			&rec.ID, &rec.OrganizationID, &rec.OrganizationName,
			&rec.Date,
			&generation, &gridExport,
			&rec.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		if generation.Valid {
			v := generation.Float64
			rec.GenerationKWh = &v
		}
		if gridExport.Valid {
			v := gridExport.Float64
			rec.GridExportKWh = &v
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows: %w", op, err)
	}
	return out, nil
}

// --- Solar Plans ---

// BulkUpsertSolarPlan upserts multiple monthly solar plan rows in a single
// transaction. Conflict key is (organization_id, year, month).
func (r *Repo) BulkUpsertSolarPlan(ctx context.Context, plans []solar.UpsertPlanRequest, userID int64) error {
	const op = "storage.repo.Solar.BulkUpsertPlan"

	if len(plans) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: begin tx: %w", op, err)
	}
	defer tx.Rollback()

	const query = `
		INSERT INTO solar_production_plan (
			organization_id, year, month, plan_thousand_kwh,
			created_by_user_id, updated_by_user_id, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $5, NOW(), NOW())
		ON CONFLICT (organization_id, year, month) DO UPDATE SET
			plan_thousand_kwh  = EXCLUDED.plan_thousand_kwh,
			updated_by_user_id = EXCLUDED.updated_by_user_id,
			updated_at         = NOW()`

	for _, p := range plans {
		if _, err := tx.ExecContext(ctx, query, p.OrganizationID, p.Year, p.Month, p.PlanThousandKWh, userID); err != nil {
			if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
				return translatedErr
			}
			return fmt.Errorf("%s: upsert plan (org=%d year=%d month=%d): %w",
				op, p.OrganizationID, p.Year, p.Month, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: commit: %w", op, err)
	}
	return nil
}

// GetSolarPlans retrieves all solar plans for a given year, joined with the
// organization name.
func (r *Repo) GetSolarPlans(ctx context.Context, year int) ([]solar.ProductionPlan, error) {
	const op = "storage.repo.Solar.GetPlans"

	const query = `
		SELECT p.id, p.organization_id, COALESCE(o.name, ''),
		       p.year, p.month, p.plan_thousand_kwh
		FROM solar_production_plan p
		LEFT JOIN organizations o ON o.id = p.organization_id
		WHERE p.year = $1
		ORDER BY p.organization_id, p.month`

	rows, err := r.db.QueryContext(ctx, query, year)
	if err != nil {
		return nil, fmt.Errorf("%s: query: %w", op, err)
	}
	defer rows.Close()

	out := make([]solar.ProductionPlan, 0)
	for rows.Next() {
		var p solar.ProductionPlan
		if err := rows.Scan(
			&p.ID, &p.OrganizationID, &p.OrganizationName,
			&p.Year, &p.Month, &p.PlanThousandKWh,
		); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows: %w", op, err)
	}
	return out, nil
}
