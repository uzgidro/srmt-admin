package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	complexValue "srmt-admin/internal/lib/model/dto/complex-value"
	"srmt-admin/internal/lib/model/dto/value"
	"srmt-admin/internal/storage"
	"time"
)

func (r *Repo) GetSelectedYearDataIncome(ctx context.Context, id, year int) (complexValue.Model, error) {
	const op = "storage.repo.analytics.GetSelectedYearData"

	const query = `
			SELECT
				EXTRACT(MONTH FROM dv.date) AS month,
				ROUND(SUM(dv.income * 86400) / 1000000) AS value,
				AVG(dv.income) AS avg_income,
				r.name AS reservoir
			FROM
				data dv
			INNER JOIN
				reservoirs r ON dv.res_id = r.id
			WHERE
				dv.res_id = $1
				AND EXTRACT(YEAR FROM dv.date) = $2
			GROUP BY
				month, r.name
			ORDER BY
				month
		`

	rows, err := r.db.QueryContext(ctx, query, id, year)
	if err != nil {
		return complexValue.Model{}, fmt.Errorf("%s: failed to execute query: %w", op, err)
	}
	defer rows.Close()

	type rawData struct {
		Month     int
		Value     float64
		AvgRate   float64
		Reservoir string
	}

	var rawResults []rawData
	for rows.Next() {
		var item rawData
		if err := rows.Scan(&item.Month, &item.Value, &item.AvgRate, &item.Reservoir); err != nil {
			return complexValue.Model{}, fmt.Errorf("%s: failed to scan row: %w", op, err)
		}
		rawResults = append(rawResults, item)
	}

	if err := rows.Err(); err != nil {
		return complexValue.Model{}, fmt.Errorf("%s: error during rows iteration: %w", op, err)
	}

	if len(rawResults) == 0 {
		return complexValue.Model{}, storage.ErrNotFound
	}

	result := complexValue.Model{
		ReservoirID: id,
		Reservoir:   rawResults[0].Reservoir,
		Data:        make([]value.Model, 0, len(rawResults)),
	}

	for _, item := range rawResults {
		result.Data = append(result.Data, value.Model{
			Date:      fmt.Sprintf("%d-%02d-01", year, item.Month),
			Value:     item.Value,
			AvgIncome: item.AvgRate,
		})
	}

	return result, nil
}

func (r *Repo) GetDataByYears(ctx context.Context, id int) (complexValue.Model, error) {
	const op = "storage.repo.analytics.GetDataByYears"

	const query = `
			SELECT
				EXTRACT(YEAR FROM dv.date) AS year,
				ROUND(SUM(dv.income * 86400) / 1000000) AS value,
				r.name AS reservoir
			FROM
				data dv
			INNER JOIN
				reservoirs r ON dv.res_id = r.id
			WHERE
				dv.res_id = $1
			GROUP BY
				year, r.name
			ORDER BY
				year
		`

	rows, err := r.db.QueryContext(ctx, query, id)
	if err != nil {
		return complexValue.Model{}, fmt.Errorf("%s: failed to execute query: %w", op, err)
	}
	defer rows.Close()

	type rawData struct {
		Year      int
		Value     float64
		Reservoir string
	}

	var rawResults []rawData
	for rows.Next() {
		var item rawData
		if err := rows.Scan(&item.Year, &item.Value, &item.Reservoir); err != nil {
			return complexValue.Model{}, fmt.Errorf("%s: failed to scan row: %w", op, err)
		}
		rawResults = append(rawResults, item)
	}

	if err := rows.Err(); err != nil {
		return complexValue.Model{}, fmt.Errorf("%s: error during rows iteration: %w", op, err)
	}

	if len(rawResults) == 0 {
		return complexValue.Model{}, storage.ErrNotFound
	}

	result := complexValue.Model{
		ReservoirID: id,
		Reservoir:   rawResults[0].Reservoir,
		Data:        make([]value.Model, 0, len(rawResults)),
	}

	for _, item := range rawResults {
		result.Data = append(result.Data, value.Model{
			Date:  fmt.Sprintf("%d-01-01", item.Year),
			Value: item.Value,
		})
	}

	return result, nil
}

func (r *Repo) GetAvgData(ctx context.Context, id int) (complexValue.Model, error) {
	const op = "storage.repo.analytics.GetAverageMonthlyData"

	const query = `
		WITH monthly_totals AS (
			SELECT
				EXTRACT(MONTH FROM dv.date) AS month,
				SUM(dv.income * 86400) / 1000000 AS monthly_volume,
				AVG(dv.income) as monthly_avg_rate,
				r.name AS reservoir
			FROM
				data dv
			INNER JOIN
				reservoirs r ON dv.res_id = r.id
			WHERE
				dv.res_id = $1
			GROUP BY
				EXTRACT(YEAR FROM dv.date), month, r.name
		)
		SELECT
			mt.month,
			ROUND(AVG(mt.monthly_volume)) AS avg_monthly_volume,
			AVG(mt.monthly_avg_rate) AS overall_avg_daily_rate,
			mt.reservoir
		FROM
			monthly_totals mt
		GROUP BY
			mt.month, mt.reservoir
		ORDER BY
			mt.month;
	`

	rows, err := r.db.QueryContext(ctx, query, id)
	if err != nil {
		return complexValue.Model{}, fmt.Errorf("%s: failed to execute query: %w", op, err)
	}
	defer rows.Close()

	type rawData struct {
		Month     int
		Value     float64
		AvgRate   float64
		Reservoir string
	}

	var rawResults []rawData
	for rows.Next() {
		var item rawData
		if err := rows.Scan(&item.Month, &item.Value, &item.AvgRate, &item.Reservoir); err != nil {
			return complexValue.Model{}, fmt.Errorf("%s: failed to scan row: %w", op, err)
		}
		rawResults = append(rawResults, item)
	}

	if err := rows.Err(); err != nil {
		return complexValue.Model{}, fmt.Errorf("%s: error during rows iteration: %w", op, err)
	}

	if len(rawResults) == 0 {
		return complexValue.Model{}, storage.ErrNotFound
	}

	result := complexValue.Model{
		ReservoirID: id,
		Reservoir:   rawResults[0].Reservoir,
		Data:        make([]value.Model, 0, len(rawResults)),
	}

	for _, item := range rawResults {
		result.Data = append(result.Data, value.Model{
			Date:      fmt.Sprintf("2020-%02d-01", item.Month),
			Value:     item.Value,
			AvgIncome: item.AvgRate,
		})
	}

	return result, nil
}

func (r *Repo) GetTenYearsAvgData(ctx context.Context, id int) (complexValue.Model, error) {
	const op = "storage.repo.analytics.GetTenYearsAvgData"

	// 1. Определяем диапазон дат: 10 полных лет до текущего года.
	upperYear := time.Now().Year() - 1
	lowerYear := upperYear - 10
	lowerYearStart := fmt.Sprintf("%d-01-01", lowerYear)
	upperYearEnd := fmt.Sprintf("%d-12-31", upperYear)

	// 2. Запрос аналогичен GetAvgData, но с фильтром по дате.
	const query = `
		WITH monthly_totals AS (
			SELECT
				EXTRACT(MONTH FROM dv.date) AS month,
				SUM(dv.income * 86400) / 1000000 AS monthly_volume,
				AVG(dv.income) as monthly_avg_rate,
				r.name AS reservoir
			FROM
				data dv
			INNER JOIN
				reservoirs r ON dv.res_id = r.id
			WHERE
				dv.res_id = $1
				AND dv.date BETWEEN $2 AND $3
			GROUP BY
				EXTRACT(YEAR FROM dv.date), month, r.name
		)
		SELECT
			mt.month,
			ROUND(AVG(mt.monthly_volume)) AS avg_monthly_volume,
			AVG(mt.monthly_avg_rate) AS overall_avg_daily_rate,
			mt.reservoir
		FROM
			monthly_totals mt
		GROUP BY
			mt.month, mt.reservoir
		ORDER BY
			mt.month;
	`

	rows, err := r.db.QueryContext(ctx, query, id, lowerYearStart, upperYearEnd)
	if err != nil {
		return complexValue.Model{}, fmt.Errorf("%s: failed to execute query: %w", op, err)
	}
	defer rows.Close()

	type rawData struct {
		Month     int
		Value     float64
		AvgRate   float64
		Reservoir string
	}

	var rawResults []rawData
	for rows.Next() {
		var item rawData
		if err := rows.Scan(&item.Month, &item.Value, &item.AvgRate, &item.Reservoir); err != nil {
			return complexValue.Model{}, fmt.Errorf("%s: failed to scan row: %w", op, err)
		}
		rawResults = append(rawResults, item)
	}

	if err := rows.Err(); err != nil {
		return complexValue.Model{}, fmt.Errorf("%s: error during rows iteration: %w", op, err)
	}

	if len(rawResults) == 0 {
		return complexValue.Model{}, storage.ErrNotFound
	}

	result := complexValue.Model{
		ReservoirID: id,
		Reservoir:   rawResults[0].Reservoir,
		Data:        make([]value.Model, 0, len(rawResults)),
	}

	for _, item := range rawResults {
		result.Data = append(result.Data, value.Model{
			Date:      fmt.Sprintf("2020-%02d-01", item.Month),
			Value:     item.Value,
			AvgIncome: item.AvgRate,
		})
	}

	return result, nil
}

func (r *Repo) GetExtremumYear(ctx context.Context, id int, extremumType string) (int, error) {
	const op = "storage.repo.analytics.GetExtremumYear"

	var sortOrder string
	switch extremumType {
	case "max":
		sortOrder = "DESC"
	case "min":
		sortOrder = "ASC"
	default:
		return 0, fmt.Errorf("%s: invalid extremum type: %s", op, extremumType)
	}

	query := fmt.Sprintf(`
		SELECT
			EXTRACT(YEAR FROM dv.date)::int AS year
		FROM
			data dv
		WHERE
			dv.res_id = $1
			AND EXTRACT(YEAR FROM dv.date) != $2
		GROUP BY
			year
		ORDER BY
			SUM(dv.income * 86400) %s
		LIMIT 1
	`, sortOrder)

	currentYear := time.Now().Year()
	var resultYear int

	err := r.db.QueryRowContext(ctx, query, id, currentYear).Scan(&resultYear)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, storage.ErrNotFound
		}
		return 0, fmt.Errorf("%s: failed to execute query: %w", op, err)
	}

	return resultYear, nil
}
