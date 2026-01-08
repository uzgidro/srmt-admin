package repo

import (
	"context"
	"database/sql"
	"fmt"
	gesproduction "srmt-admin/internal/lib/model/ges-production"
	"time"
)

// UpsertGesProduction inserts or updates GES production data
func (r *Repo) UpsertGesProduction(ctx context.Context, data gesproduction.Model) error {
	const op = "storage.repo.UpsertGesProduction"

	query := `
		INSERT INTO ges_production (date, total_energy_production, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT (date) DO UPDATE SET
			total_energy_production = EXCLUDED.total_energy_production,
			updated_at = NOW()
	`

	_, err := r.db.ExecContext(ctx, query, data.Date, data.TotalEnergyProduction)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// GetGesProductionDashboard returns the latest production data and percentage change
func (r *Repo) GetGesProductionDashboard(ctx context.Context) (*gesproduction.DashboardResponse, error) {
	const op = "storage.repo.GetGesProductionDashboard"

	// 1. Get the latest record
	var latestDate time.Time
	var latestVal float64

	err := r.db.QueryRowContext(ctx, `
		SELECT date, total_energy_production 
		FROM ges_production 
		ORDER BY date DESC 
		LIMIT 1
	`).Scan(&latestDate, &latestVal)

	if err != nil {
		if err == sql.ErrNoRows {
			// No data at all
			return nil, nil
		}
		return nil, fmt.Errorf("%s: failed to get latest record: %w", op, err)
	}

	// 2. Get the previous day's record (relative to the latest record)
	prevDate := latestDate.AddDate(0, 0, -1)
	var prevVal float64

	err = r.db.QueryRowContext(ctx, `
		SELECT total_energy_production 
		FROM ges_production 
		WHERE date = $1
	`, prevDate).Scan(&prevVal)

	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("%s: failed to get previous day record: %w", op, err)
	}
	// If ErrNoRows, prevVal remains 0, which is correct

	// 3. Calculate change
	percentChange := 0.0
	direction := "flat"

	if prevVal > 0 {
		diff := latestVal - prevVal
		percentChange = (diff / prevVal) * 100
	} else if prevVal == 0 && latestVal > 0 {
		percentChange = 100.0 // Treated as 100% growth from 0 (or just max indicator)
	}

	if percentChange > 0 {
		direction = "up"
	} else if percentChange < 0 {
		direction = "down"
	}

	return &gesproduction.DashboardResponse{
		Date:            latestDate.Format("2006-01-02"),
		Value:           latestVal,
		ChangePercent:   percentChange,
		ChangeDirection: direction,
	}, nil
}
