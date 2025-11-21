package repo

import (
	"context"
	"database/sql"
	"fmt"
	past_events "srmt-admin/internal/lib/dto/past-events"
	"time"
)

// GetPastEvents retrieves events from incidents, shutdowns, and idle_water_discharges
// for the specified number of days and returns them as a unified list sorted by date (newest first)
func (r *Repo) GetPastEvents(ctx context.Context, days int, timezone *time.Location) ([]past_events.Event, error) {
	const op = "storage.repo.GetPastEvents"

	// Calculate date range
	now := time.Now().In(timezone)
	startDate := now.AddDate(0, 0, -days)

	var events []past_events.Event

	// 1. Get incidents (type: warning)
	incidentsQuery := `
		SELECT
			i.incident_time,
			i.organization_id,
			COALESCE(o.name, '') as org_name,
			COALESCE(i.description, '') as description
		FROM
			incidents i
		LEFT JOIN
			organizations o ON i.organization_id = o.id
		WHERE
			i.incident_time >= $1 AND i.incident_time <= $2
		ORDER BY
			i.incident_time DESC
	`

	rows, err := r.db.QueryContext(ctx, incidentsQuery, startDate, now)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query incidents: %w", op, err)
	}

	for rows.Next() {
		var (
			date        time.Time
			orgID       sql.NullInt64
			orgName     string
			description string
		)

		if err := rows.Scan(&date, &orgID, &orgName, &description); err != nil {
			rows.Close()
			return nil, fmt.Errorf("%s: failed to scan incident row: %w", op, err)
		}

		var orgIDPtr *int64
		var orgNamePtr *string
		if orgID.Valid {
			orgIDPtr = &orgID.Int64
			orgNamePtr = &orgName
		}

		events = append(events, past_events.Event{
			Type:             past_events.EventTypeDanger,
			Date:             date,
			OrganizationID:   orgIDPtr,
			OrganizationName: orgNamePtr,
			Description:      description,
		})
	}
	rows.Close()

	// 2. Get shutdowns (create 2 events per shutdown)
	shutdownsQuery := `
		SELECT
			s.start_time,
			s.end_time,
			s.organization_id,
			COALESCE(o.name, '') as org_name,
			COALESCE(s.reason, '') as reason
		FROM
			shutdowns s
		LEFT JOIN
			organizations o ON s.organization_id = o.id
		WHERE
			(s.start_time >= $1 AND s.start_time <= $2)
			OR (s.end_time >= $1 AND s.end_time <= $2)
		ORDER BY
			s.start_time DESC
	`

	rows, err = r.db.QueryContext(ctx, shutdownsQuery, startDate, now)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query shutdowns: %w", op, err)
	}

	for rows.Next() {
		var (
			startTime time.Time
			endTime   sql.NullTime
			orgID     int64
			orgName   string
			reason    string
		)

		if err := rows.Scan(&startTime, &endTime, &orgID, &orgName, &reason); err != nil {
			rows.Close()
			return nil, fmt.Errorf("%s: failed to scan shutdown row: %w", op, err)
		}

		orgNamePtr := &orgName

		// Event 1: Start event (warning)
		if startTime.After(startDate) && startTime.Before(now) || startTime.Equal(startDate) || startTime.Equal(now) {
			events = append(events, past_events.Event{
				Type:             past_events.EventTypeWarning,
				Date:             startTime,
				OrganizationID:   &orgID,
				OrganizationName: orgNamePtr,
				Description:      reason,
			})
		}

		// Event 2: End event (info) - "аппарат исправен"
		if endTime.Valid {
			if endTime.Time.After(startDate) && endTime.Time.Before(now) || endTime.Time.Equal(startDate) || endTime.Time.Equal(now) {
				events = append(events, past_events.Event{
					Type:             past_events.EventTypeSuccess,
					Date:             endTime.Time,
					OrganizationID:   &orgID,
					OrganizationName: orgNamePtr,
					Description:      "аппарат исправен",
				})
			}
		}
	}
	rows.Close()

	// 3. Get idle_water_discharges (create 2 events per discharge, similar to shutdowns)
	dischargesQuery := `
		SELECT
			d.start_time,
			d.end_time,
			d.organization_id,
			COALESCE(o.name, '') as org_name,
			COALESCE(d.reason, '') as reason
		FROM
			idle_water_discharges d
		LEFT JOIN
			organizations o ON d.organization_id = o.id
		WHERE
			(d.start_time >= $1 AND d.start_time <= $2)
			OR (d.end_time >= $1 AND d.end_time <= $2)
		ORDER BY
			d.start_time DESC
	`

	rows, err = r.db.QueryContext(ctx, dischargesQuery, startDate, now)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query discharges: %w", op, err)
	}

	for rows.Next() {
		var (
			startTime time.Time
			endTime   sql.NullTime
			orgID     int64
			orgName   string
			reason    string
		)

		if err := rows.Scan(&startTime, &endTime, &orgID, &orgName, &reason); err != nil {
			rows.Close()
			return nil, fmt.Errorf("%s: failed to scan discharge row: %w", op, err)
		}

		orgNamePtr := &orgName

		// Event 1: Start event (warning)
		if startTime.After(startDate) && startTime.Before(now) || startTime.Equal(startDate) || startTime.Equal(now) {
			events = append(events, past_events.Event{
				Type:             past_events.EventTypeWarning,
				Date:             startTime,
				OrganizationID:   &orgID,
				OrganizationName: orgNamePtr,
				Description:      reason,
			})
		}

		// Event 2: End event (info) - "Водосброс остановлен"
		if endTime.Valid {
			if endTime.Time.After(startDate) && endTime.Time.Before(now) || endTime.Time.Equal(startDate) || endTime.Time.Equal(now) {
				events = append(events, past_events.Event{
					Type:             past_events.EventTypeInfo,
					Date:             endTime.Time,
					OrganizationID:   &orgID,
					OrganizationName: orgNamePtr,
					Description:      "Водосброс остановлен",
				})
			}
		}
	}
	rows.Close()

	// Sort events by date (newest first)
	// Using a simple bubble sort for simplicity, could use sort.Slice for better performance
	for i := 0; i < len(events)-1; i++ {
		for j := 0; j < len(events)-i-1; j++ {
			if events[j].Date.Before(events[j+1].Date) {
				events[j], events[j+1] = events[j+1], events[j]
			}
		}
	}

	if events == nil {
		events = make([]past_events.Event, 0)
	}

	return events, nil
}
