package repo

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	reservoirsummary "srmt-admin/internal/lib/model/reservoir-summary"
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
			IncomingVolume:         summaryRaw.IncomingVolumeMlnM3,
			IncomingVolumePrevYear: summaryRaw.IncomingVolumeMlnM3PrevYear,
		}

		summaries = append(summaries, summary)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	// Return empty slice instead of nil for consistency
	if summaries == nil {
		summaries = make([]*reservoirsummary.ResponseModel, 0)
	}

	sortOrder := map[string]int{
		"Андижон сув омбори":   1,
		"Охангарон сув омбори": 2,
		"Сардоба сув омбори":   3,
		"Хисорак сув омбори":   4,
		"Топаланг сув омбори":  5,
		"Чорвок сув омбори":    6,
		"ИТОГО":                7,
	}

	sort.Slice(summaries, func(i, j int) bool {
		orderI, okI := sortOrder[summaries[i].OrganizationName]
		orderJ, okJ := sortOrder[summaries[j].OrganizationName]

		// If both have defined order, sort by order
		if okI && okJ {
			return orderI < orderJ
		}
		// If only i has defined order, i comes first
		if okI {
			return true
		}
		// If only j has defined order, j comes first
		if okJ {
			return false
		}
		// If neither has defined order, maintain original order
		return i < j
	})

	return summaries, nil
}

// scanReservoirSummaryRow scans a single row from the query result
func scanReservoirSummaryRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*reservoirsummary.ResponseModelRaw, error) {
	var m reservoirsummary.ResponseModelRaw
	var orgID sql.NullInt64

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
		&m.IncomingVolumeMlnM3,
		&m.IncomingVolumeMlnM3PrevYear,
	)
	if err != nil {
		return nil, err
	}

	// Handle nullable organization_id (NULL for summary row)
	if orgID.Valid {
		m.OrganizationID = &orgID.Int64
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
org_data AS (
    SELECT DISTINCT organization_id
    FROM reservoir_data
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
        ROUND(
            COALESCE(
                (SELECT SUM(rd2.income_m3_s)
                 FROM reservoir_data rd2
                 WHERE rd2.organization_id = rd.organization_id
                   AND rd2.date >= dp.year_start
                   AND rd2.date < dp.target_date)
                * 0.0864,
                0
            ),
            2
        ) AS incoming_volume_mln_m3_current_year,

        ROUND(
            COALESCE(
                (SELECT SUM(rd2.income_m3_s)
                 FROM reservoir_data rd2
                 WHERE rd2.organization_id = rd.organization_id
                   AND rd2.date >= dp.prev_year_start
                   AND rd2.date < dp.year_ago_date)
                * 0.0864,
                0
            ),
            2
        ) AS incoming_volume_mln_m3_prev_year
    FROM org_data rd
    CROSS JOIN date_params dp
)
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
    -- Set incoming_volume to 0 for non-reservoir organizations (not linked to organization_type 8)
    CASE
        WHEN EXISTS (
            SELECT 1 FROM organization_type_links otl
            WHERE otl.organization_id = od.organization_id
            AND otl.type_id = 8
        ) THEN COALESCE(iv.incoming_volume_mln_m3_current_year, 0)
        ELSE 0
    END AS incoming_volume_mln_m3,
    CASE
        WHEN EXISTS (
            SELECT 1 FROM organization_type_links otl
            WHERE otl.organization_id = od.organization_id
            AND otl.type_id = 8
        ) THEN COALESCE(iv.incoming_volume_mln_m3_prev_year, 0)
        ELSE 0
    END AS incoming_volume_mln_m3_prev_year
FROM org_data od
LEFT JOIN organizations o ON od.organization_id = o.id
LEFT JOIN level_data ld ON od.organization_id = ld.organization_id
LEFT JOIN volume_data vd ON od.organization_id = vd.organization_id
LEFT JOIN income_data id ON od.organization_id = id.organization_id
LEFT JOIN release_data reld ON od.organization_id = reld.organization_id
LEFT JOIN modsnow_data md ON od.organization_id = md.organization_id
LEFT JOIN incoming_volume iv ON od.organization_id = iv.organization_id

UNION ALL

-- ИТОГО row: only sum values from reservoir organizations (linked to organization_type 8)
SELECT
    NULL AS organization_id,
    'ИТОГО' AS organization_name,
    0 AS level_current,
    0 AS level_prev,
    0 AS level_year_ago,
    0 AS level_two_years_ago,
    COALESCE(SUM(CASE WHEN EXISTS (
        SELECT 1 FROM organization_type_links otl
        WHERE otl.organization_id = od.organization_id
        AND otl.type_id = 8
    ) THEN COALESCE(vd.volume_current, 0) ELSE 0 END), 0) AS volume_current,
    COALESCE(SUM(CASE WHEN EXISTS (
        SELECT 1 FROM organization_type_links otl
        WHERE otl.organization_id = od.organization_id
        AND otl.type_id = 8
    ) THEN COALESCE(vd.volume_prev, 0) ELSE 0 END), 0) AS volume_prev,
    COALESCE(SUM(CASE WHEN EXISTS (
        SELECT 1 FROM organization_type_links otl
        WHERE otl.organization_id = od.organization_id
        AND otl.type_id = 8
    ) THEN COALESCE(vd.volume_year_ago, 0) ELSE 0 END), 0) AS volume_year_ago,
    COALESCE(SUM(CASE WHEN EXISTS (
        SELECT 1 FROM organization_type_links otl
        WHERE otl.organization_id = od.organization_id
        AND otl.type_id = 8
    ) THEN COALESCE(vd.volume_two_years_ago, 0) ELSE 0 END), 0) AS volume_two_years_ago,
    COALESCE(SUM(CASE WHEN EXISTS (
        SELECT 1 FROM organization_type_links otl
        WHERE otl.organization_id = od.organization_id
        AND otl.type_id = 8
    ) THEN COALESCE(id.income_current, 0) ELSE 0 END), 0) AS income_current,
    COALESCE(SUM(CASE WHEN EXISTS (
        SELECT 1 FROM organization_type_links otl
        WHERE otl.organization_id = od.organization_id
        AND otl.type_id = 8
    ) THEN COALESCE(id.income_prev, 0) ELSE 0 END), 0) AS income_prev,
    COALESCE(SUM(CASE WHEN EXISTS (
        SELECT 1 FROM organization_type_links otl
        WHERE otl.organization_id = od.organization_id
        AND otl.type_id = 8
    ) THEN COALESCE(id.income_year_ago, 0) ELSE 0 END), 0) AS income_year_ago,
    COALESCE(SUM(CASE WHEN EXISTS (
        SELECT 1 FROM organization_type_links otl
        WHERE otl.organization_id = od.organization_id
        AND otl.type_id = 8
    ) THEN COALESCE(id.income_two_years_ago, 0) ELSE 0 END), 0) AS income_two_years_ago,
    COALESCE(SUM(CASE WHEN EXISTS (
        SELECT 1 FROM organization_type_links otl
        WHERE otl.organization_id = od.organization_id
        AND otl.type_id = 8
    ) THEN COALESCE(reld.release_current, 0) ELSE 0 END), 0) AS release_current,
    COALESCE(SUM(CASE WHEN EXISTS (
        SELECT 1 FROM organization_type_links otl
        WHERE otl.organization_id = od.organization_id
        AND otl.type_id = 8
    ) THEN COALESCE(reld.release_prev, 0) ELSE 0 END), 0) AS release_prev,
    COALESCE(SUM(CASE WHEN EXISTS (
        SELECT 1 FROM organization_type_links otl
        WHERE otl.organization_id = od.organization_id
        AND otl.type_id = 8
    ) THEN COALESCE(reld.release_year_ago, 0) ELSE 0 END), 0) AS release_year_ago,
    COALESCE(SUM(CASE WHEN EXISTS (
        SELECT 1 FROM organization_type_links otl
        WHERE otl.organization_id = od.organization_id
        AND otl.type_id = 8
    ) THEN COALESCE(reld.release_two_years_ago, 0) ELSE 0 END), 0) AS release_two_years_ago,
    COALESCE(SUM(CASE WHEN EXISTS (
        SELECT 1 FROM organization_type_links otl
        WHERE otl.organization_id = od.organization_id
        AND otl.type_id = 8
    ) THEN COALESCE(md.modsnow_current, 0) ELSE 0 END), 0) AS modsnow_current,
    COALESCE(SUM(CASE WHEN EXISTS (
        SELECT 1 FROM organization_type_links otl
        WHERE otl.organization_id = od.organization_id
        AND otl.type_id = 8
    ) THEN COALESCE(md.modsnow_year_ago, 0) ELSE 0 END), 0) AS modsnow_year_ago,
    COALESCE(SUM(CASE WHEN EXISTS (
        SELECT 1 FROM organization_type_links otl
        WHERE otl.organization_id = od.organization_id
        AND otl.type_id = 8
    ) THEN COALESCE(iv.incoming_volume_mln_m3_current_year, 0) ELSE 0 END), 0) AS incoming_volume_mln_m3,
    COALESCE(SUM(CASE WHEN EXISTS (
        SELECT 1 FROM organization_type_links otl
        WHERE otl.organization_id = od.organization_id
        AND otl.type_id = 8
    ) THEN COALESCE(iv.incoming_volume_mln_m3_prev_year, 0) ELSE 0 END), 0) AS incoming_volume_mln_m3_prev_year
FROM org_data od
LEFT JOIN organizations o ON od.organization_id = o.id
LEFT JOIN level_data ld ON od.organization_id = ld.organization_id
LEFT JOIN volume_data vd ON od.organization_id = vd.organization_id
LEFT JOIN income_data id ON od.organization_id = id.organization_id
LEFT JOIN release_data reld ON od.organization_id = reld.organization_id
LEFT JOIN modsnow_data md ON od.organization_id = md.organization_id
LEFT JOIN incoming_volume iv ON od.organization_id = iv.organization_id

ORDER BY organization_id NULLS LAST
`
}
