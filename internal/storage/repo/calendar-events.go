package repo

import (
	"context"
	"fmt"
	"srmt-admin/internal/lib/dto"
	"time"
)

// GetCalendarEventsCounts retrieves event counts grouped by date for a given month
func (r *Repo) GetCalendarEventsCounts(ctx context.Context, year, month int, timezone *time.Location) (map[string]*dto.DayCounts, error) {
	const op = "storage.repo.GetCalendarEventsCounts"

	// Define month boundaries in the given timezone
	startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, timezone)
	endOfMonth := startOfMonth.AddDate(0, 1, 0) // First day of next month

	// SQL query to aggregate counts by date and event type
	query := `
		SELECT
			DATE_TRUNC('day', incident_time AT TIME ZONE $3)::date as event_date,
			'incident' as event_type,
			COUNT(*) as count
		FROM incidents
		WHERE incident_time >= $1 AND incident_time < $2
		GROUP BY event_date

		UNION ALL

		SELECT
			DATE_TRUNC('day', start_time AT TIME ZONE $3)::date as event_date,
			'shutdown',
			COUNT(*)
		FROM shutdowns
		WHERE start_time >= $1 AND start_time < $2
		GROUP BY event_date

		UNION ALL

		SELECT
			DATE_TRUNC('day', start_time AT TIME ZONE $3)::date as event_date,
			'discharge',
			COUNT(*)
		FROM idle_water_discharges
		WHERE start_time >= $1 AND start_time < $2
		GROUP BY event_date

		UNION ALL

		SELECT
			DATE_TRUNC('day', visit_date AT TIME ZONE $3)::date as event_date,
			'visit',
			COUNT(*)
		FROM visits
		WHERE visit_date >= $1 AND visit_date < $2
		GROUP BY event_date
	`

	rows, err := r.db.QueryContext(ctx, query, startOfMonth, endOfMonth, timezone.String())
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query calendar events: %w", op, err)
	}
	defer rows.Close()

	// Initialize result map
	result := make(map[string]*dto.DayCounts)

	// Process rows and populate the map
	for rows.Next() {
		var (
			eventDate time.Time
			eventType string
			count     int
		)

		if err := rows.Scan(&eventDate, &eventType, &count); err != nil {
			return nil, fmt.Errorf("%s: failed to scan row: %w", op, err)
		}

		// Format date as YYYY-MM-DD
		dateKey := eventDate.Format("2006-01-02")

		// Initialize DayCounts if not exists
		if result[dateKey] == nil {
			result[dateKey] = &dto.DayCounts{}
		}

		// Populate the appropriate counter based on event type
		switch eventType {
		case "incident":
			result[dateKey].Incidents = count
		case "shutdown":
			result[dateKey].Shutdowns = count
		case "discharge":
			result[dateKey].Discharges = count
		case "visit":
			result[dateKey].Visits = count
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	return result, nil
}
