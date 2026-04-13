package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
	gesreport "srmt-admin/internal/lib/model/ges-report"
	"srmt-admin/internal/storage"
)

// --- GES Config CRUD ---

// UpsertGESConfig inserts or updates a GES config record.
func (r *Repo) UpsertGESConfig(ctx context.Context, req gesreport.UpsertConfigRequest) error {
	const op = "storage.repo.GESReport.UpsertGESConfig"

	const query = `
		INSERT INTO ges_config (organization_id, installed_capacity_mwt, total_aggregates, has_reservoir, sort_order)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (organization_id) DO UPDATE SET
			installed_capacity_mwt = EXCLUDED.installed_capacity_mwt,
			total_aggregates = EXCLUDED.total_aggregates,
			has_reservoir = EXCLUDED.has_reservoir,
			sort_order = EXCLUDED.sort_order,
			updated_at = NOW()`

	_, err := r.db.ExecContext(ctx, query,
		req.OrganizationID,
		req.InstalledCapacityMWt,
		req.TotalAggregates,
		req.HasReservoir,
		req.SortOrder,
	)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

// GetAllGESConfigs returns all GES configs with organization names and cascade info.
func (r *Repo) GetAllGESConfigs(ctx context.Context) ([]gesreport.Config, error) {
	const op = "storage.repo.GESReport.GetAllGESConfigs"

	const query = `
		SELECT
			gc.id,
			gc.organization_id,
			o.name AS organization_name,
			cascade_org.id AS cascade_id,
			cascade_org.name AS cascade_name,
			gc.installed_capacity_mwt,
			gc.total_aggregates,
			gc.has_reservoir,
			gc.sort_order
		FROM ges_config gc
		JOIN organizations o ON o.id = gc.organization_id
		LEFT JOIN organizations cascade_org ON cascade_org.id = o.parent_organization_id
		ORDER BY gc.sort_order, o.name`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: query: %w", op, err)
	}
	defer rows.Close()

	result := make([]gesreport.Config, 0)
	for rows.Next() {
		var cfg gesreport.Config
		var cascadeID sql.NullInt64
		var cascadeName sql.NullString

		if err := rows.Scan(
			&cfg.ID,
			&cfg.OrganizationID,
			&cfg.OrganizationName,
			&cascadeID,
			&cascadeName,
			&cfg.InstalledCapacityMWt,
			&cfg.TotalAggregates,
			&cfg.HasReservoir,
			&cfg.SortOrder,
		); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		if cascadeID.Valid {
			cfg.CascadeID = &cascadeID.Int64
		}
		if cascadeName.Valid {
			cfg.CascadeName = &cascadeName.String
		}
		result = append(result, cfg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows: %w", op, err)
	}
	return result, nil
}

// DeleteGESConfig removes a GES config by organization_id.
func (r *Repo) DeleteGESConfig(ctx context.Context, organizationID int64) error {
	const op = "storage.repo.GESReport.DeleteGESConfig"

	res, err := r.db.ExecContext(ctx, "DELETE FROM ges_config WHERE organization_id = $1", organizationID)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: %w", op, err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: rows affected: %w", op, err)
	}
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// --- Cascade Config CRUD ---

// UpsertCascadeConfig inserts or updates a cascade config record.
func (r *Repo) UpsertCascadeConfig(ctx context.Context, req gesreport.UpsertCascadeConfigRequest) error {
	const op = "storage.repo.GESReport.UpsertCascadeConfig"

	const query = `
		INSERT INTO cascade_config (organization_id, latitude, longitude, sort_order)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (organization_id) DO UPDATE SET
			latitude = EXCLUDED.latitude,
			longitude = EXCLUDED.longitude,
			sort_order = EXCLUDED.sort_order,
			updated_at = NOW()`

	_, err := r.db.ExecContext(ctx, query,
		req.OrganizationID,
		req.Latitude,
		req.Longitude,
		req.SortOrder,
	)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

// GetAllCascadeConfigs returns all cascade configs with organization names.
func (r *Repo) GetAllCascadeConfigs(ctx context.Context) ([]gesreport.CascadeConfig, error) {
	const op = "storage.repo.GESReport.GetAllCascadeConfigs"

	const query = `
		SELECT
			cc.id,
			cc.organization_id,
			o.name AS organization_name,
			cc.latitude,
			cc.longitude,
			cc.sort_order
		FROM cascade_config cc
		JOIN organizations o ON o.id = cc.organization_id
		ORDER BY cc.sort_order, o.name`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: query: %w", op, err)
	}
	defer rows.Close()

	result := make([]gesreport.CascadeConfig, 0)
	for rows.Next() {
		var cfg gesreport.CascadeConfig
		var latitude, longitude sql.NullFloat64

		if err := rows.Scan(
			&cfg.ID,
			&cfg.OrganizationID,
			&cfg.OrganizationName,
			&latitude,
			&longitude,
			&cfg.SortOrder,
		); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		if latitude.Valid {
			cfg.Latitude = &latitude.Float64
		}
		if longitude.Valid {
			cfg.Longitude = &longitude.Float64
		}
		result = append(result, cfg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows: %w", op, err)
	}
	return result, nil
}

// DeleteCascadeConfig removes a cascade config by organization_id.
func (r *Repo) DeleteCascadeConfig(ctx context.Context, organizationID int64) error {
	const op = "storage.repo.GESReport.DeleteCascadeConfig"

	res, err := r.db.ExecContext(ctx, "DELETE FROM cascade_config WHERE organization_id = $1", organizationID)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: %w", op, err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: rows affected: %w", op, err)
	}
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// --- GES Daily Data CRUD ---

// UpsertGESDailyData inserts or updates daily operational data.
func (r *Repo) UpsertGESDailyData(ctx context.Context, req gesreport.UpsertDailyDataRequest, userID int64) error {
	const op = "storage.repo.GESReport.UpsertGESDailyData"

	const query = `
		INSERT INTO ges_daily_data (
			organization_id, date,
			daily_production_mln_kwh, working_aggregates,
			water_level_m, water_volume_mln_m3, water_head_m,
			reservoir_income_m3s, total_outflow_m3s, ges_flow_m3s,
			created_by_user_id, updated_by_user_id, created_at, updated_at
		) VALUES (
			$1, $2::date,
			$3, $4,
			$5, $6, $7,
			$8, $9, $10,
			$11, $11, NOW(), NOW()
		)
		ON CONFLICT (organization_id, date) DO UPDATE SET
			daily_production_mln_kwh = EXCLUDED.daily_production_mln_kwh,
			working_aggregates = EXCLUDED.working_aggregates,
			water_level_m = EXCLUDED.water_level_m,
			water_volume_mln_m3 = EXCLUDED.water_volume_mln_m3,
			water_head_m = EXCLUDED.water_head_m,
			reservoir_income_m3s = EXCLUDED.reservoir_income_m3s,
			total_outflow_m3s = EXCLUDED.total_outflow_m3s,
			ges_flow_m3s = EXCLUDED.ges_flow_m3s,
			updated_by_user_id = EXCLUDED.updated_by_user_id,
			updated_at = NOW()`

	_, err := r.db.ExecContext(ctx, query,
		req.OrganizationID,
		req.Date,
		req.DailyProductionMlnKWh,
		req.WorkingAggregates,
		req.WaterLevelM,
		req.WaterVolumeMlnM3,
		req.WaterHeadM,
		req.ReservoirIncomeM3s,
		req.TotalOutflowM3s,
		req.GESFlowM3s,
		userID,
	)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

// GetGESDailyData retrieves daily data for a single GES on a given date.
func (r *Repo) GetGESDailyData(ctx context.Context, organizationID int64, date string) (*gesreport.DailyData, error) {
	const op = "storage.repo.GESReport.GetGESDailyData"

	const query = `
		SELECT
			id, organization_id, date::text,
			daily_production_mln_kwh, working_aggregates,
			water_level_m, water_volume_mln_m3, water_head_m,
			reservoir_income_m3s, total_outflow_m3s, ges_flow_m3s
		FROM ges_daily_data
		WHERE organization_id = $1 AND date = $2::date`

	var d gesreport.DailyData
	err := r.db.QueryRowContext(ctx, query, organizationID, date).Scan(
		&d.ID,
		&d.OrganizationID,
		&d.Date,
		&d.DailyProductionMlnKWh,
		&d.WorkingAggregates,
		&d.WaterLevelM,
		&d.WaterVolumeMlnM3,
		&d.WaterHeadM,
		&d.ReservoirIncomeM3s,
		&d.TotalOutflowM3s,
		&d.GESFlowM3s,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &d, nil
}

// --- GES Production Plan CRUD ---

// BulkUpsertGESPlan upserts multiple plan entries in a transaction.
func (r *Repo) BulkUpsertGESPlan(ctx context.Context, req gesreport.BulkUpsertPlanRequest, userID int64) error {
	const op = "storage.repo.GESReport.BulkUpsertGESPlan"

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: begin tx: %w", op, err)
	}
	defer tx.Rollback()

	const query = `
		INSERT INTO ges_production_plan (organization_id, year, month, plan_mln_kwh, created_by_user_id, updated_by_user_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5, NOW(), NOW())
		ON CONFLICT (organization_id, year, month) DO UPDATE SET
			plan_mln_kwh = EXCLUDED.plan_mln_kwh,
			updated_by_user_id = EXCLUDED.updated_by_user_id,
			updated_at = NOW()`

	for _, p := range req.Plans {
		if _, err := tx.ExecContext(ctx, query, p.OrganizationID, p.Year, p.Month, p.PlanMlnKWh, userID); err != nil {
			if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
				return translatedErr
			}
			return fmt.Errorf("%s: upsert plan (org=%d year=%d month=%d): %w", op, p.OrganizationID, p.Year, p.Month, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: commit: %w", op, err)
	}
	return nil
}

// GetGESPlans retrieves all plans for a given year.
func (r *Repo) GetGESPlans(ctx context.Context, year int) ([]gesreport.ProductionPlan, error) {
	const op = "storage.repo.GESReport.GetGESPlans"

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, organization_id, year, month, plan_mln_kwh
		 FROM ges_production_plan
		 WHERE year = $1
		 ORDER BY organization_id, month`,
		year,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: query: %w", op, err)
	}
	defer rows.Close()

	result := make([]gesreport.ProductionPlan, 0)
	for rows.Next() {
		var p gesreport.ProductionPlan
		if err := rows.Scan(&p.ID, &p.OrganizationID, &p.Year, &p.Month, &p.PlanMlnKWh); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		result = append(result, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows: %w", op, err)
	}
	return result, nil
}

// --- Batch Report Queries ---

// GetGESDailyDataBatch fetches all configured GES data for a date, returning
// all configured stations even if no daily data has been entered yet.
func (r *Repo) GetGESDailyDataBatch(ctx context.Context, date string) ([]gesreport.RawDailyRow, error) {
	const op = "storage.repo.GESReport.GetGESDailyDataBatch"

	const query = `
		SELECT
			c.organization_id,
			o.name,
			o.parent_organization_id,
			po.name,
			COALESCE(d.date::text, $1::text),
			COALESCE(d.daily_production_mln_kwh, 0),
			COALESCE(d.working_aggregates, 0),
			d.water_level_m, d.water_volume_mln_m3, d.water_head_m,
			d.reservoir_income_m3s, d.total_outflow_m3s, d.ges_flow_m3s,
			c.installed_capacity_mwt, c.total_aggregates, c.has_reservoir, c.sort_order
		FROM ges_config c
		JOIN organizations o ON c.organization_id = o.id
		LEFT JOIN organizations po ON o.parent_organization_id = po.id
		LEFT JOIN ges_daily_data d ON d.organization_id = c.organization_id AND d.date = $1::date
		ORDER BY c.sort_order, o.name`

	rows, err := r.db.QueryContext(ctx, query, date)
	if err != nil {
		return nil, fmt.Errorf("%s: query: %w", op, err)
	}
	defer rows.Close()

	result := make([]gesreport.RawDailyRow, 0)
	for rows.Next() {
		var row gesreport.RawDailyRow
		var cascadeID sql.NullInt64
		var cascadeName sql.NullString

		if err := rows.Scan(
			&row.OrganizationID,
			&row.OrganizationName,
			&cascadeID,
			&cascadeName,
			&row.Date,
			&row.DailyProductionMlnKWh,
			&row.WorkingAggregates,
			&row.WaterLevelM,
			&row.WaterVolumeMlnM3,
			&row.WaterHeadM,
			&row.ReservoirIncomeM3s,
			&row.TotalOutflowM3s,
			&row.GESFlowM3s,
			&row.InstalledCapacityMWt,
			&row.TotalAggregates,
			&row.HasReservoir,
			&row.SortOrder,
		); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		if cascadeID.Valid {
			row.CascadeID = &cascadeID.Int64
		}
		if cascadeName.Valid {
			row.CascadeName = &cascadeName.String
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows: %w", op, err)
	}
	return result, nil
}

// GetGESProductionAggregations returns MTD, YTD, and prev-year equivalents
// for all stations in a single query.
func (r *Repo) GetGESProductionAggregations(ctx context.Context, date string) ([]gesreport.ProductionAggregation, error) {
	const op = "storage.repo.GESReport.GetGESProductionAggregations"

	const query = `
		SELECT
			organization_id,
			SUM(CASE WHEN date >= DATE_TRUNC('month', $1::date) AND date <= $1::date
			         THEN daily_production_mln_kwh ELSE 0 END) AS mtd,
			SUM(CASE WHEN date >= DATE_TRUNC('year', $1::date) AND date <= $1::date
			         THEN daily_production_mln_kwh ELSE 0 END) AS ytd,
			SUM(CASE WHEN date >= DATE_TRUNC('month', ($1::date - INTERVAL '1 year'))
			          AND date <= ($1::date - INTERVAL '1 year')
			         THEN daily_production_mln_kwh ELSE 0 END) AS prev_year_mtd,
			SUM(CASE WHEN date >= DATE_TRUNC('year', ($1::date - INTERVAL '1 year'))
			          AND date <= ($1::date - INTERVAL '1 year')
			         THEN daily_production_mln_kwh ELSE 0 END) AS prev_year_ytd
		FROM ges_daily_data
		WHERE date >= DATE_TRUNC('year', ($1::date - INTERVAL '1 year'))
		  AND date <= $1::date
		GROUP BY organization_id`

	rows, err := r.db.QueryContext(ctx, query, date)
	if err != nil {
		return nil, fmt.Errorf("%s: query: %w", op, err)
	}
	defer rows.Close()

	result := make([]gesreport.ProductionAggregation, 0)
	for rows.Next() {
		var agg gesreport.ProductionAggregation
		if err := rows.Scan(
			&agg.OrganizationID,
			&agg.MTD,
			&agg.YTD,
			&agg.PrevYearMTD,
			&agg.PrevYearYTD,
		); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		result = append(result, agg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows: %w", op, err)
	}
	return result, nil
}

// GetGESPlansForReport retrieves production plans for the given year and months
// (typically the 3 months of the current quarter).
func (r *Repo) GetGESPlansForReport(ctx context.Context, year int, months []int) ([]gesreport.PlanRow, error) {
	const op = "storage.repo.GESReport.GetGESPlansForReport"

	rows, err := r.db.QueryContext(ctx,
		`SELECT organization_id, year, month, plan_mln_kwh
		 FROM ges_production_plan
		 WHERE year = $1 AND month = ANY($2)`,
		year, pq.Array(months),
	)
	if err != nil {
		return nil, fmt.Errorf("%s: query: %w", op, err)
	}
	defer rows.Close()

	result := make([]gesreport.PlanRow, 0)
	for rows.Next() {
		var p gesreport.PlanRow
		if err := rows.Scan(&p.OrganizationID, &p.Year, &p.Month, &p.PlanMlnKWh); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		result = append(result, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows: %w", op, err)
	}
	return result, nil
}

// GetIdleDischargesForDate returns all idle water discharges active during the
// given operational day window [start, end).
func (r *Repo) GetIdleDischargesForDate(ctx context.Context, start, end time.Time) ([]gesreport.IdleDischargeRow, error) {
	const op = "storage.repo.GESReport.GetIdleDischargesForDate"

	const query = `
		SELECT organization_id, flow_rate_m3_s, total_volume_mln_m3, reason, is_ongoing
		FROM v_idle_water_discharges_with_volume
		WHERE start_time < $2 AND (end_time > $1 OR end_time IS NULL)`

	rows, err := r.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, fmt.Errorf("%s: query: %w", op, err)
	}
	defer rows.Close()

	result := make([]gesreport.IdleDischargeRow, 0)
	for rows.Next() {
		var row gesreport.IdleDischargeRow
		if err := rows.Scan(
			&row.OrganizationID,
			&row.FlowRateM3s,
			&row.VolumeMlnM3,
			&row.Reason,
			&row.IsOngoing,
		); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows: %w", op, err)
	}
	return result, nil
}

// UpsertCascadeDailyWeather inserts or updates weather data for a cascade organization.
// Keyed by (organization_id, date). Nil values are written as NULL.
func (r *Repo) UpsertCascadeDailyWeather(ctx context.Context, cascadeOrgID int64, date string, temperature *float64, weatherCondition *string) error {
	const op = "storage.repo.GESReport.UpsertCascadeDailyWeather"

	const query = `
		INSERT INTO cascade_daily_data (organization_id, date, temperature, weather_condition)
		VALUES ($1, $2::date, $3, $4)
		ON CONFLICT (organization_id, date) DO UPDATE SET
			temperature = EXCLUDED.temperature,
			weather_condition = EXCLUDED.weather_condition,
			updated_at = NOW()`

	if _, err := r.db.ExecContext(ctx, query, cascadeOrgID, date, temperature, weatherCondition); err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

// GetCascadeDailyWeatherBatch fetches weather rows for the given cascade org IDs and dates.
// Returns a map keyed by (OrgID, Date as "YYYY-MM-DD").
func (r *Repo) GetCascadeDailyWeatherBatch(ctx context.Context, orgIDs []int64, dates []string) (map[gesreport.CascadeWeatherKey]*gesreport.CascadeWeather, error) {
	const op = "storage.repo.GESReport.GetCascadeDailyWeatherBatch"

	result := make(map[gesreport.CascadeWeatherKey]*gesreport.CascadeWeather)
	if len(orgIDs) == 0 || len(dates) == 0 {
		return result, nil
	}

	const query = `
		SELECT organization_id, date::text, temperature, weather_condition
		FROM cascade_daily_data
		WHERE organization_id = ANY($1) AND date = ANY($2::date[])`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(orgIDs), pq.Array(dates))
	if err != nil {
		return nil, fmt.Errorf("%s: query: %w", op, err)
	}
	defer rows.Close()

	for rows.Next() {
		var orgID int64
		var date string
		var temp sql.NullFloat64
		var cond sql.NullString
		if err := rows.Scan(&orgID, &date, &temp, &cond); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		w := &gesreport.CascadeWeather{}
		if temp.Valid {
			v := temp.Float64
			w.Temperature = &v
		}
		if cond.Valid {
			v := cond.String
			w.Condition = &v
		}
		result[gesreport.CascadeWeatherKey{OrgID: orgID, Date: date}] = w
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows: %w", op, err)
	}
	return result, nil
}
