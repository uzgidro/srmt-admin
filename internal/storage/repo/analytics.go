package repo

import (
	"context"
	"fmt"
	complexValue "srmt-admin/internal/lib/model/dto/complex-value"
	"srmt-admin/internal/lib/model/dto/value"
	"srmt-admin/internal/storage"
)

func (s *Repo) GetSelectedYearDataIncome(ctx context.Context, id, year int) (complexValue.ComplexValue, error) {
	const op = "storage.repo.GetSelectedYearData"

	const query = `
			SELECT
				EXTRACT(MONTH FROM dv.date) AS month,
				-- 1. Умножаем суточный приток (м³/с) на кол-во секунд в сутках (86400) для получения суточного объема (м³).
				-- 2. Суммируем суточные объемы для получения месячного объема (м³).
				-- 3. Делим на 1,000,000 для перевода в млн. м³.
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

	rows, err := s.Driver.QueryContext(ctx, query, id, year)
	if err != nil {
		return complexValue.ComplexValue{}, fmt.Errorf("%s: failed to execute query: %w", op, err)
	}
	defer rows.Close()

	type rawData struct {
		Month     int
		Value     float64
		AvgIncome float64
		Reservoir string
	}

	var rawResults []rawData
	for rows.Next() {
		var item rawData
		if err := rows.Scan(&item.Month, &item.Value, &item.AvgIncome, &item.Reservoir); err != nil {
			return complexValue.ComplexValue{}, fmt.Errorf("%s: failed to scan row: %w", op, err)
		}
		rawResults = append(rawResults, item)
	}

	if err := rows.Err(); err != nil {
		return complexValue.ComplexValue{}, fmt.Errorf("%s: error during rows iteration: %w", op, err)
	}

	if len(rawResults) == 0 {
		return complexValue.ComplexValue{}, storage.ErrNotFound
	}

	result := complexValue.ComplexValue{
		ReservoirID: id,
		Reservoir:   rawResults[0].Reservoir,
		AvgIncome:   rawResults[0].AvgIncome,
		Data:        make([]value.Value, 0, len(rawResults)),
	}

	for _, item := range rawResults {
		result.Data = append(result.Data, value.Value{
			Date:  fmt.Sprintf("%d-%02d-01", year, item.Month),
			Value: item.Value,
		})
	}

	return result, nil
}
