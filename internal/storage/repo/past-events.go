package repo

import (
	"context"
	"database/sql"
	"fmt"
	past_events "srmt-admin/internal/lib/dto/past-events"
	"time"
)

// GetPastEvents retrieves events from incidents, shutdowns, and idle_water_discharges
// for the specified number of days and returns them grouped by date (dates: newest first, events within date: oldest first)
func (r *Repo) GetPastEvents(ctx context.Context, days int, timezone *time.Location) ([]past_events.DateGroup, error) {
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
		} else {
			// When organization_id is null, set organization_name to "все предприятия"
			allEnterprises := "Все предприятия"
			orgNamePtr = &allEnterprises
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

	// Group events by date (YYYY-MM-DD format)
	eventsByDate := make(map[string][]past_events.Event)

	for _, event := range events {
		// Format date as YYYY-MM-DD in the configured timezone
		dateKey := event.Date.In(timezone).Format("2006-01-02")
		eventsByDate[dateKey] = append(eventsByDate[dateKey], event)
	}

	// Sort events within each date group (oldest first)
	for dateKey := range eventsByDate {
		eventsForDate := eventsByDate[dateKey]
		for i := 0; i < len(eventsForDate)-1; i++ {
			for j := 0; j < len(eventsForDate)-i-1; j++ {
				if eventsForDate[j].Date.Before(eventsForDate[j+1].Date) {
					eventsForDate[j], eventsForDate[j+1] = eventsForDate[j+1], eventsForDate[j]
				}
			}
		}
		eventsByDate[dateKey] = eventsForDate
	}

	// Extract date keys and sort them in descending order (newest first)
	dateKeys := make([]string, 0, len(eventsByDate))
	for dateKey := range eventsByDate {
		dateKeys = append(dateKeys, dateKey)
	}

	// Sort date keys in descending order (2025-11-21, 2025-11-20, etc.)
	for i := 0; i < len(dateKeys)-1; i++ {
		for j := 0; j < len(dateKeys)-i-1; j++ {
			if dateKeys[j] < dateKeys[j+1] {
				dateKeys[j], dateKeys[j+1] = dateKeys[j+1], dateKeys[j]
			}
		}
	}

	// Build the result as a slice of DateGroup
	result := make([]past_events.DateGroup, 0, len(dateKeys))
	for _, dateKey := range dateKeys {
		result = append(result, past_events.DateGroup{
			Date:   dateKey,
			Events: eventsByDate[dateKey],
		})
	}

	return result, nil
}
