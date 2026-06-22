package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	reservoirsummary "srmt-admin/internal/lib/model/reservoir-summary"
	"srmt-admin/internal/storage"
)

// GetReservoirSummary retrieves reservoir summary data for all organizations
// grouped by organization_id with a summary row (organization_id = NULL)
func (r *Repo) GetReservoirSummary(ctx context.Context, date string) ([]*reservoirsummary.ResponseModel, error) {
	const op = "storage.repo.GetReservoirSummary"

	query := getReservoirSummaryQuery()

	rows, err := r.db.QueryContext(ctx, query, date)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query reservoir summary: %w", op, err)
	}
	defer rows.Close()

	var summaries []*reservoirsummary.ResponseModel
	for rows.Next() {
		summaryRaw, err := scanReservoirSummaryRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan reservoir summary row: %w", op, err)
		}

		// Determine final incoming volume values and IsCalculated flags
		var incomingVolume float64
		var incomingVolumeIsCalculated bool
		var incomingVolumePrevYear float64
		var incomingVolumePrevYearIsCalculated bool

		// Current year logic: use stored value only if not nil AND not 0
		if summaryRaw.StoredIncomingVolume != nil && *summaryRaw.StoredIncomingVolume != 0 {
			incomingVolume = *summaryRaw.StoredIncomingVolume
			incomingVolumeIsCalculated = false
		} else {
			incomingVolume = summaryRaw.IncomingVolumeMlnM3
			incomingVolumeIsCalculated = true
		}

		// Previous year logic: use stored value only if not nil AND not 0
		if summaryRaw.StoredIncomingVolumePrevYear != nil && *summaryRaw.StoredIncomingVolumePrevYear != 0 {
			incomingVolumePrevYear = *summaryRaw.StoredIncomingVolumePrevYear
			incomingVolumePrevYearIsCalculated = false
		} else {
			incomingVolumePrevYear = summaryRaw.IncomingVolumeMlnM3PrevYear
			incomingVolumePrevYearIsCalculated = true
		}

		summary := &reservoirsummary.ResponseModel{
			OrganizationID:   summaryRaw.OrganizationID,
			OrganizationName: summaryRaw.OrganizationName,
			Level: reservoirsummary.ValueResponse{
				Current:     summaryRaw.LevelCurrent,
				Previous:    summaryRaw.LevelPrev,
				YearAgo:     summaryRaw.LevelYearAgo,
				TwoYearsAgo: summaryRaw.LevelTwoYearsAgo,
			},
			Volume: reservoirsummary.ValueResponse{
				Current:     summaryRaw.VolumeCurrent,
				Previous:    summaryRaw.VolumePrev,
				YearAgo:     summaryRaw.VolumeYearAgo,
				TwoYearsAgo: summaryRaw.VolumeTwoYearsAgo,
			},
			Income: reservoirsummary.ValueResponse{
				Current:     summaryRaw.IncomeCurrent,
				Previous:    summaryRaw.IncomePrev,
				YearAgo:     summaryRaw.IncomeYearAgo,
				TwoYearsAgo: summaryRaw.IncomeTwoYearsAgo,
			},
			Release: reservoirsummary.ValueResponse{
				Current:     summaryRaw.ReleaseCurrent,
				Previous:    summaryRaw.ReleasePrev,
				YearAgo:     summaryRaw.ReleaseYearAgo,
				TwoYearsAgo: summaryRaw.ReleaseTwoYearsAgo,
			},
			Modsnow: reservoirsummary.ValueResponse{
				Current:     summaryRaw.ModsnowCurrent,
				Previous:    0,
				YearAgo:     summaryRaw.ModsnowYearAgo,
				TwoYearsAgo: 0,
			},
			IncomingVolume:                     incomingVolume,
			IncomingVolumePrevYear:             incomingVolumePrevYear,
			IncomingVolumeIsCalculated:         incomingVolumeIsCalculated,
			IncomingVolumePrevYearIsCalculated: incomingVolumePrevYearIsCalculated,
			IncomingVolumeBaseDate:             summaryRaw.IncomingVolumeBaseDate,
			IncomingVolumeBaseValue:            summaryRaw.IncomingVolumeBaseValue,
			IncomingVolumePrevYearBaseDate:     summaryRaw.IncomingVolumePrevYearBaseDate,
			IncomingVolumePrevYearBaseValue:    summaryRaw.IncomingVolumePrevYearBaseValue,
		}

		summaries = append(summaries, summary)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	// Return empty slice instead of nil for consistency.
	// Order is already established by SQL ORDER BY sort_position (see
	// getReservoirSummaryQuery); no Go-side sort needed.
	if summaries == nil {
		summaries = make([]*reservoirsummary.ResponseModel, 0)
	}

	return summaries, nil
}

// scanReservoirSummaryRow scans a single row from the query result
func scanReservoirSummaryRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*reservoirsummary.ResponseModelRaw, error) {
	var m reservoirsummary.ResponseModelRaw
	var orgID sql.NullInt64
	var storedIncomingVolume sql.NullFloat64
	var storedIncomingVolumePrevYear sql.NullFloat64

	// sortPosition is read off the wire but not exposed on the model —
	// it exists only so the SQL ORDER BY can interleave ИТОГО with the
	// per-org rows (see itog_position CTE).
	var sortPosition int
	err := scanner.Scan(
		&orgID,
		&m.OrganizationName,
		&m.LevelCurrent,
		&m.LevelPrev,
		&m.LevelYearAgo,
		&m.LevelTwoYearsAgo,
		&m.VolumeCurrent,
		&m.VolumePrev,
		&m.VolumeYearAgo,
		&m.VolumeTwoYearsAgo,
		&m.IncomeCurrent,
		&m.IncomePrev,
		&m.IncomeYearAgo,
		&m.IncomeTwoYearsAgo,
		&m.ReleaseCurrent,
		&m.ReleasePrev,
		&m.ReleaseYearAgo,
		&m.ReleaseTwoYearsAgo,
		&m.ModsnowCurrent,
		&m.ModsnowYearAgo,
		&storedIncomingVolume,
		&storedIncomingVolumePrevYear,
		&m.IncomingVolumeMlnM3,
		&m.IncomingVolumeMlnM3PrevYear,
		&m.IncomingVolumeBaseDate,
		&m.IncomingVolumeBaseValue,
		&m.IncomingVolumePrevYearBaseDate,
		&m.IncomingVolumePrevYearBaseValue,
		&sortPosition,
	)
	if err != nil {
		return nil, err
	}

	// Handle nullable organization_id (NULL for summary row)
	if orgID.Valid {
		m.OrganizationID = &orgID.Int64
	}

	// Handle stored values (convert sql.NullFloat64 to *float64)
	if storedIncomingVolume.Valid {
		m.StoredIncomingVolume = &storedIncomingVolume.Float64
	}
	if storedIncomingVolumePrevYear.Valid {
		m.StoredIncomingVolumePrevYear = &storedIncomingVolumePrevYear.Float64
	}

	return &m, nil
}

// getReservoirSummaryQuery returns the complete SQL query for reservoir summary data
func getReservoirSummaryQuery() string {
	return `
WITH date_params AS (
    SELECT
        $1::date AS target_date,
        ($1::date - INTERVAL '1 day')::date AS prev_date,
        ($1::date - INTERVAL '1 year')::date AS year_ago_date,
        ($1::date - INTERVAL '2 years')::date AS two_years_ago_date,
        DATE_TRUNC('year', $1::date)::date AS year_start,
        (DATE_TRUNC('year', $1::date) - INTERVAL '1 year')::date AS prev_year_start
),
-- Whitelist: only organizations explicitly configured for this report.
-- Anything in reservoir_data but missing from reservoir_summary_config is
-- silently excluded — manage membership via /reservoir-summary/config.
org_data AS (
    SELECT organization_id
    FROM reservoir_summary_config
),
-- Pre-compute the position the ИТОГО row should occupy: right after the
-- last "summed" reservoir (max sort_order among include_in_total = TRUE)
-- but before any "below-total" rows (include_in_total = FALSE, e.g. Пском).
-- *10 + 5 keeps an integer sort_position that interleaves cleanly with the
-- per-org *10 values below.
itog_position AS (
    SELECT COALESCE(MAX(sort_order), 0) * 10 + 5 AS pos
    FROM reservoir_summary_config
    WHERE include_in_total = TRUE
),
level_data AS (
    SELECT
        rd.organization_id,
        COALESCE(MAX(rd.level_m) FILTER (WHERE rd.date = dp.target_date), 0) AS level_current,
        COALESCE(MAX(rd.level_m) FILTER (WHERE rd.date = dp.prev_date), 0) AS level_prev,
        COALESCE(MAX(rd.level_m) FILTER (WHERE rd.date = dp.year_ago_date), 0) AS level_year_ago,
        COALESCE(MAX(rd.level_m) FILTER (WHERE rd.date = dp.two_years_ago_date), 0) AS level_two_years_ago
    FROM reservoir_data rd
    CROSS JOIN date_params dp
    WHERE rd.date IN (dp.target_date, dp.prev_date, dp.year_ago_date, dp.two_years_ago_date)
    GROUP BY rd.organization_id
),
volume_data AS (
    SELECT
        rd.organization_id,
        COALESCE(MAX(rd.volume_mln_m3) FILTER (WHERE rd.date = dp.target_date), 0) AS volume_current,
        COALESCE(MAX(rd.volume_mln_m3) FILTER (WHERE rd.date = dp.prev_date), 0) AS volume_prev,
        COALESCE(MAX(rd.volume_mln_m3) FILTER (WHERE rd.date = dp.year_ago_date), 0) AS volume_year_ago,
        COALESCE(MAX(rd.volume_mln_m3) FILTER (WHERE rd.date = dp.two_years_ago_date), 0) AS volume_two_years_ago
    FROM reservoir_data rd
    CROSS JOIN date_params dp
    WHERE rd.date IN (dp.target_date, dp.prev_date, dp.year_ago_date, dp.two_years_ago_date)
    GROUP BY rd.organization_id
),
income_data AS (
    SELECT
        rd.organization_id,
        COALESCE(MAX(rd.income_m3_s) FILTER (WHERE rd.date = dp.target_date), 0) AS income_current,
        COALESCE(MAX(rd.income_m3_s) FILTER (WHERE rd.date = dp.prev_date), 0) AS income_prev,
        COALESCE(MAX(rd.income_m3_s) FILTER (WHERE rd.date = dp.year_ago_date), 0) AS income_year_ago,
        COALESCE(MAX(rd.income_m3_s) FILTER (WHERE rd.date = dp.two_years_ago_date), 0) AS income_two_years_ago
    FROM reservoir_data rd
    CROSS JOIN date_params dp
    WHERE rd.date IN (dp.target_date, dp.prev_date, dp.year_ago_date, dp.two_years_ago_date)
    GROUP BY rd.organization_id
),
release_data AS (
    SELECT
        rd.organization_id,
        COALESCE(MAX(rd.release_m3_s) FILTER (WHERE rd.date = dp.target_date), 0) AS release_current,
        COALESCE(MAX(rd.release_m3_s) FILTER (WHERE rd.date = dp.prev_date), 0) AS release_prev,
        COALESCE(MAX(rd.release_m3_s) FILTER (WHERE rd.date = dp.year_ago_date), 0) AS release_year_ago,
        COALESCE(MAX(rd.release_m3_s) FILTER (WHERE rd.date = dp.two_years_ago_date), 0) AS release_two_years_ago
    FROM reservoir_data rd
    CROSS JOIN date_params dp
    WHERE rd.date IN (dp.target_date, dp.prev_date, dp.year_ago_date, dp.two_years_ago_date)
    GROUP BY rd.organization_id
),
modsnow_data AS (
    SELECT
        m.organization_id,
        COALESCE(MAX(m.cover) FILTER (WHERE m.date = dp.target_date), 0) AS modsnow_current,
        COALESCE(MAX(m.cover) FILTER (WHERE m.date = dp.year_ago_date), 0) AS modsnow_year_ago
    FROM modsnow m
    CROSS JOIN date_params dp
    WHERE m.date IN (dp.target_date, dp.year_ago_date)
    GROUP BY m.organization_id
),
incoming_volume AS (
    SELECT
        rd.organization_id,

        -- Current Year Calculation
        base_curr.date::text AS base_date_curr,
        base_curr.total_income_volume_mln_m3 AS base_val_curr,
        ROUND(
            CASE
                WHEN base_curr.date IS NOT NULL THEN
                    base_curr.total_income_volume_mln_m3 +
                    (SELECT COALESCE(SUM(inc.income_m3_s), 0) * 0.0864
                     FROM reservoir_data inc
                     WHERE inc.organization_id = rd.organization_id
                       AND inc.date > base_curr.date
                       AND inc.date <= dp.target_date)
                ELSE
                    (SELECT COALESCE(SUM(inc.income_m3_s), 0) * 0.0864
                     FROM reservoir_data inc
                     WHERE inc.organization_id = rd.organization_id
                       AND inc.date >= dp.year_start
                       AND inc.date <= dp.target_date)
            END,
        2) AS incoming_volume_mln_m3_current_year,

        -- Previous Year Calculation
        base_prev.date::text AS base_date_prev,
        base_prev.total_income_volume_mln_m3 AS base_val_prev,
        ROUND(
            CASE
                WHEN base_prev.date IS NOT NULL THEN
                    base_prev.total_income_volume_mln_m3 +
                    (SELECT COALESCE(SUM(inc.income_m3_s), 0) * 0.0864
                     FROM reservoir_data inc
                     WHERE inc.organization_id = rd.organization_id
                       AND inc.date > base_prev.date
                       AND inc.date <= dp.year_ago_date)
                ELSE
                    (SELECT COALESCE(SUM(inc.income_m3_s), 0) * 0.0864
                     FROM reservoir_data inc
                     WHERE inc.organization_id = rd.organization_id
                       AND inc.date >= dp.prev_year_start
                       AND inc.date <= dp.year_ago_date)
            END,
        2) AS incoming_volume_mln_m3_prev_year

    FROM org_data rd
    CROSS JOIN date_params dp
    -- Find latest stored value for current year
    LEFT JOIN LATERAL (
        SELECT date, total_income_volume_mln_m3
        FROM reservoir_data b
        WHERE b.organization_id = rd.organization_id
          AND b.date >= dp.year_start
          AND b.date <= dp.target_date
          AND b.total_income_volume_mln_m3 IS NOT NULL
          AND b.total_income_volume_mln_m3 > 0
        ORDER BY b.date DESC
        LIMIT 1
    ) base_curr ON TRUE
    -- Find latest stored value for previous year
    LEFT JOIN LATERAL (
        SELECT date, total_income_volume_mln_m3
        FROM reservoir_data b
        WHERE b.organization_id = rd.organization_id
          AND b.date >= dp.prev_year_start
          AND b.date <= dp.year_ago_date
          AND b.total_income_volume_mln_m3 IS NOT NULL
          AND b.total_income_volume_mln_m3 > 0
        ORDER BY b.date DESC
        LIMIT 1
    ) base_prev ON TRUE
),
stored_income_volume AS (
    SELECT
        rd.organization_id,
        MAX(rd.total_income_volume_mln_m3) FILTER (WHERE rd.date = dp.target_date) AS stored_total_income_volume,
        MAX(rd.total_income_volume_prev_year_mln_m3) FILTER (WHERE rd.date = dp.year_ago_date) AS stored_total_income_volume_prev_year
    FROM reservoir_data rd
    CROSS JOIN date_params dp
    WHERE rd.date IN (dp.target_date, dp.year_ago_date)
    GROUP BY rd.organization_id
)
-- Per-organization rows. sort_position = rsc.sort_order * 10 so the ИТОГО
-- row can interleave at *10+5 (see itog_position CTE above).
SELECT
    od.organization_id,
    COALESCE(o.name, '') AS organization_name,
    COALESCE(ld.level_current, 0) AS level_current,
    COALESCE(ld.level_prev, 0) AS level_prev,
    COALESCE(ld.level_year_ago, 0) AS level_year_ago,
    COALESCE(ld.level_two_years_ago, 0) AS level_two_years_ago,
    COALESCE(vd.volume_current, 0) AS volume_current,
    COALESCE(vd.volume_prev, 0) AS volume_prev,
    COALESCE(vd.volume_year_ago, 0) AS volume_year_ago,
    COALESCE(vd.volume_two_years_ago, 0) AS volume_two_years_ago,
    COALESCE(id.income_current, 0) AS income_current,
    COALESCE(id.income_prev, 0) AS income_prev,
    COALESCE(id.income_year_ago, 0) AS income_year_ago,
    COALESCE(id.income_two_years_ago, 0) AS income_two_years_ago,
    COALESCE(reld.release_current, 0) AS release_current,
    COALESCE(reld.release_prev, 0) AS release_prev,
    COALESCE(reld.release_year_ago, 0) AS release_year_ago,
    COALESCE(reld.release_two_years_ago, 0) AS release_two_years_ago,
    COALESCE(md.modsnow_current, 0) AS modsnow_current,
    COALESCE(md.modsnow_year_ago, 0) AS modsnow_year_ago,
    -- Stored values (NULL if not set)
    siv.stored_total_income_volume AS stored_incoming_volume,
    siv.stored_total_income_volume_prev_year AS stored_incoming_volume_prev_year,
    -- Incoming volume for all configured organizations
    COALESCE(iv.incoming_volume_mln_m3_current_year, 0) AS incoming_volume_mln_m3,
    COALESCE(iv.incoming_volume_mln_m3_prev_year, 0) AS incoming_volume_mln_m3_prev_year,
    -- Calculation Base Details
    iv.base_date_curr AS base_date_curr,
    iv.base_val_curr AS base_val_curr,
    iv.base_date_prev AS base_date_prev,
    iv.base_val_prev AS base_val_prev,
    rsc.sort_order * 10 AS sort_position

FROM org_data od
JOIN reservoir_summary_config rsc ON rsc.organization_id = od.organization_id
LEFT JOIN organizations o ON od.organization_id = o.id
LEFT JOIN level_data ld ON od.organization_id = ld.organization_id
LEFT JOIN volume_data vd ON od.organization_id = vd.organization_id
LEFT JOIN income_data id ON od.organization_id = id.organization_id
LEFT JOIN release_data reld ON od.organization_id = reld.organization_id
LEFT JOIN modsnow_data md ON od.organization_id = md.organization_id
LEFT JOIN incoming_volume iv ON od.organization_id = iv.organization_id
LEFT JOIN stored_income_volume siv ON od.organization_id = siv.organization_id

UNION ALL

-- ИТОГО row: sum only values from organizations with include_in_total = TRUE
-- in reservoir_summary_config. Single JOIN replaces 16 EXISTS clauses.
SELECT
    NULL AS organization_id,
    'ИТОГО' AS organization_name,
    0 AS level_current,
    0 AS level_prev,
    0 AS level_year_ago,
    0 AS level_two_years_ago,
    COALESCE(SUM(CASE WHEN rsc.include_in_total THEN COALESCE(vd.volume_current, 0) ELSE 0 END), 0) AS volume_current,
    COALESCE(SUM(CASE WHEN rsc.include_in_total THEN COALESCE(vd.volume_prev, 0) ELSE 0 END), 0) AS volume_prev,
    COALESCE(SUM(CASE WHEN rsc.include_in_total THEN COALESCE(vd.volume_year_ago, 0) ELSE 0 END), 0) AS volume_year_ago,
    COALESCE(SUM(CASE WHEN rsc.include_in_total THEN COALESCE(vd.volume_two_years_ago, 0) ELSE 0 END), 0) AS volume_two_years_ago,
    COALESCE(SUM(CASE WHEN rsc.include_in_total THEN COALESCE(id.income_current, 0) ELSE 0 END), 0) AS income_current,
    COALESCE(SUM(CASE WHEN rsc.include_in_total THEN COALESCE(id.income_prev, 0) ELSE 0 END), 0) AS income_prev,
    COALESCE(SUM(CASE WHEN rsc.include_in_total THEN COALESCE(id.income_year_ago, 0) ELSE 0 END), 0) AS income_year_ago,
    COALESCE(SUM(CASE WHEN rsc.include_in_total THEN COALESCE(id.income_two_years_ago, 0) ELSE 0 END), 0) AS income_two_years_ago,
    COALESCE(SUM(CASE WHEN rsc.include_in_total THEN COALESCE(reld.release_current, 0) ELSE 0 END), 0) AS release_current,
    COALESCE(SUM(CASE WHEN rsc.include_in_total THEN COALESCE(reld.release_prev, 0) ELSE 0 END), 0) AS release_prev,
    COALESCE(SUM(CASE WHEN rsc.include_in_total THEN COALESCE(reld.release_year_ago, 0) ELSE 0 END), 0) AS release_year_ago,
    COALESCE(SUM(CASE WHEN rsc.include_in_total THEN COALESCE(reld.release_two_years_ago, 0) ELSE 0 END), 0) AS release_two_years_ago,
    COALESCE(SUM(CASE WHEN rsc.include_in_total THEN COALESCE(md.modsnow_current, 0) ELSE 0 END), 0) AS modsnow_current,
    COALESCE(SUM(CASE WHEN rsc.include_in_total THEN COALESCE(md.modsnow_year_ago, 0) ELSE 0 END), 0) AS modsnow_year_ago,
    NULL AS stored_incoming_volume,
    NULL AS stored_incoming_volume_prev_year,
    -- incoming_volume historically summed across ALL orgs without the
    -- include filter; the config now governs it uniformly with the rest.
    COALESCE(SUM(CASE WHEN rsc.include_in_total THEN COALESCE(iv.incoming_volume_mln_m3_current_year, 0) ELSE 0 END), 0) AS incoming_volume_mln_m3,
    COALESCE(SUM(CASE WHEN rsc.include_in_total THEN COALESCE(iv.incoming_volume_mln_m3_prev_year, 0) ELSE 0 END), 0) AS incoming_volume_mln_m3_prev_year,
    NULL AS base_date_curr,
    NULL AS base_val_curr,
    NULL AS base_date_prev,
    NULL AS base_val_prev,
    (SELECT pos FROM itog_position) AS sort_position
FROM org_data od
JOIN reservoir_summary_config rsc ON rsc.organization_id = od.organization_id
LEFT JOIN volume_data vd ON od.organization_id = vd.organization_id
LEFT JOIN income_data id ON od.organization_id = id.organization_id
LEFT JOIN release_data reld ON od.organization_id = reld.organization_id
LEFT JOIN modsnow_data md ON od.organization_id = md.organization_id
LEFT JOIN incoming_volume iv ON od.organization_id = iv.organization_id

ORDER BY sort_position
`
}

// --- Reservoir Summary Config CRUD ---

// upsertReservoirSummaryConfigQuery returns the SQL for upserting a
// reservoir-summary config row. Extracted into a function so structural
// tests can pin the column list — see reservoir_summary_test.go.
func upsertReservoirSummaryConfigQuery() string {
	return `
		INSERT INTO reservoir_summary_config (organization_id, sort_order, include_in_total, modsnow_enabled)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (organization_id) DO UPDATE SET
			sort_order = EXCLUDED.sort_order,
			include_in_total = EXCLUDED.include_in_total,
			modsnow_enabled = EXCLUDED.modsnow_enabled,
			updated_at = NOW()`
}

// UpsertReservoirSummaryConfig inserts or updates a config row. Upsert key
// is organization_id (UNIQUE in DB).
func (r *Repo) UpsertReservoirSummaryConfig(ctx context.Context, req reservoirsummary.UpsertReservoirSummaryConfigRequest) error {
	const op = "storage.repo.UpsertReservoirSummaryConfig"

	_, err := r.db.ExecContext(ctx, upsertReservoirSummaryConfigQuery(),
		req.OrganizationID, req.SortOrder, req.IncludeInTotal, req.ModsnowEnabled,
	)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

// getAllReservoirSummaryConfigsQuery returns the SQL for the full config
// listing. Function form so structural tests can grep the column list.
func getAllReservoirSummaryConfigsQuery() string {
	return `
		SELECT
			rsc.id,
			rsc.organization_id,
			o.name AS organization_name,
			rsc.sort_order,
			rsc.include_in_total,
			rsc.modsnow_enabled
		FROM reservoir_summary_config rsc
		JOIN organizations o ON o.id = rsc.organization_id
		ORDER BY rsc.sort_order, o.name`
}

// GetAllReservoirSummaryConfigs returns all config rows joined with
// organization names, ordered by sort_order so the result is render-ready.
func (r *Repo) GetAllReservoirSummaryConfigs(ctx context.Context) ([]reservoirsummary.ReservoirSummaryConfig, error) {
	const op = "storage.repo.GetAllReservoirSummaryConfigs"

	rows, err := r.db.QueryContext(ctx, getAllReservoirSummaryConfigsQuery())
	if err != nil {
		return nil, fmt.Errorf("%s: query: %w", op, err)
	}
	defer rows.Close()

	result := make([]reservoirsummary.ReservoirSummaryConfig, 0)
	for rows.Next() {
		var cfg reservoirsummary.ReservoirSummaryConfig
		if err := rows.Scan(
			&cfg.ID,
			&cfg.OrganizationID,
			&cfg.OrganizationName,
			&cfg.SortOrder,
			&cfg.IncludeInTotal,
			&cfg.ModsnowEnabled,
		); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		result = append(result, cfg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows: %w", op, err)
	}
	return result, nil
}

// DeleteReservoirSummaryConfig removes one config row by organization_id.
// Returns storage.ErrNotFound when no row matched.
func (r *Repo) DeleteReservoirSummaryConfig(ctx context.Context, organizationID int64) error {
	const op = "storage.repo.DeleteReservoirSummaryConfig"

	res, err := r.db.ExecContext(ctx, "DELETE FROM reservoir_summary_config WHERE organization_id = $1", organizationID)
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

// getReservoirSummaryConfigByOrgIDQuery returns the SQL for the single-row
// lookup. Function form so structural tests can grep the column list.
func getReservoirSummaryConfigByOrgIDQuery() string {
	return `
		SELECT
			rsc.id,
			rsc.organization_id,
			o.name AS organization_name,
			rsc.sort_order,
			rsc.include_in_total,
			rsc.modsnow_enabled
		FROM reservoir_summary_config rsc
		JOIN organizations o ON o.id = rsc.organization_id
		WHERE rsc.organization_id = $1`
}

// GetReservoirSummaryConfigByOrgID returns one config row by organization_id,
// or storage.ErrNotFound when nothing matched. Useful for validating that an
// org is part of the report before writing related data.
func (r *Repo) GetReservoirSummaryConfigByOrgID(ctx context.Context, orgID int64) (*reservoirsummary.ReservoirSummaryConfig, error) {
	const op = "storage.repo.GetReservoirSummaryConfigByOrgID"

	var cfg reservoirsummary.ReservoirSummaryConfig
	err := r.db.QueryRowContext(ctx, getReservoirSummaryConfigByOrgIDQuery(), orgID).Scan(
		&cfg.ID,
		&cfg.OrganizationID,
		&cfg.OrganizationName,
		&cfg.SortOrder,
		&cfg.IncludeInTotal,
		&cfg.ModsnowEnabled,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &cfg, nil
}
