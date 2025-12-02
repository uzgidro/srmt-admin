package repo

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	past_events "srmt-admin/internal/lib/dto/past-events"
	"srmt-admin/internal/lib/model/file"
	"time"
)

type intervalEntity struct {
	ID               int64
	StartTime        time.Time
	EndTime          sql.NullTime
	OrganizationID   int64
	OrganizationName string
	Reason           string
	IsContinuation   bool
	HasContinuation  bool
	EntityType       string
}

func (r *Repo) GetPastEvents(ctx context.Context, days int, timezone *time.Location) ([]past_events.DateGroup, error) {
	const op = "storage.repo.GetPastEvents"

	now := time.Now().In(timezone)
	startDate := now.AddDate(0, 0, -days)

	var allEvents []past_events.Event

	incidents, err := r.getIncidentsHelper(ctx, startDate, now)
	if err != nil {
		return nil, fmt.Errorf("%s: incidents: %w", op, err)
	}
	allEvents = append(allEvents, incidents...)

	shutdownsQuery := r.buildExactIntervalQuery("shutdowns")
	shutdowns, err := r.getIntervalEvents(ctx, shutdownsQuery, "shutdown", startDate, now)
	if err != nil {
		return nil, fmt.Errorf("%s: shutdowns: %w", op, err)
	}
	allEvents = append(allEvents, shutdowns...)

	dischargesQuery := r.buildExactIntervalQuery("idle_water_discharges")
	discharges, err := r.getIntervalEvents(ctx, dischargesQuery, "discharge", startDate, now)
	if err != nil {
		return nil, fmt.Errorf("%s: discharges: %w", op, err)
	}
	allEvents = append(allEvents, discharges...)

	r.enrichWithFiles(ctx, allEvents)

	return groupAndSortEvents(allEvents, timezone), nil
}

func (r *Repo) buildExactIntervalQuery(tableName string) string {
	return fmt.Sprintf(`SELECT
          t.id,
          t.start_time,
          t.end_time,
          t.organization_id,
          COALESCE(o.name, '') as org_name,
          COALESCE(t.reason, '') as reason,
          EXISTS(
             SELECT 1 FROM %s prev
             WHERE prev.organization_id = t.organization_id
             AND prev.end_time = t.start_time
          ) as is_continuation,
          EXISTS(
             SELECT 1 FROM %s next
             WHERE next.organization_id = t.organization_id
             AND next.start_time = t.end_time
          ) as has_continuation
       FROM
          %s t
       LEFT JOIN
          organizations o ON t.organization_id = o.id
       WHERE
          (t.start_time >= $1 AND t.start_time <= $2)
          OR (t.end_time >= $1 AND t.end_time <= $2)
       ORDER BY
          t.start_time DESC`, tableName, tableName, tableName)
}

func (r *Repo) getIntervalEvents(ctx context.Context, query string, entityType string, filterStart, filterEnd time.Time) ([]past_events.Event, error) {
	rows, err := r.db.QueryContext(ctx, query, filterStart, filterEnd)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []past_events.Event

	for rows.Next() {
		var e intervalEntity
		e.EntityType = entityType

		if err := rows.Scan(&e.ID, &e.StartTime, &e.EndTime, &e.OrganizationID, &e.OrganizationName, &e.Reason, &e.IsContinuation, &e.HasContinuation); err != nil {
			return nil, err
		}

		events = append(events, processEntityToEvents(e, filterStart, filterEnd)...)
	}
	return events, nil
}

func processEntityToEvents(e intervalEntity, filterStart, filterEnd time.Time) []past_events.Event {
	var result []past_events.Event

	orgID := e.OrganizationID
	orgName := e.OrganizationName

	if isDateInRange(e.StartTime, filterStart, filterEnd) {

		eventType := past_events.EventTypeWarning
		desc := e.Reason

		if e.IsContinuation {
			eventType = past_events.EventTypeInfo
			desc = e.Reason
		}

		result = append(result, past_events.Event{
			Type:             eventType,
			Date:             e.StartTime,
			OrganizationID:   &orgID,
			OrganizationName: &orgName,
			Description:      desc,
			EntityType:       e.EntityType,
			EntityID:         e.ID,
		})
	}

	if e.EndTime.Valid && !e.HasContinuation {
		if isDateInRange(e.EndTime.Time, filterStart, filterEnd) {

			endMsg := "Работы завершены"
			if e.EntityType == "discharge" {
				endMsg = "Водосброс остановлен"
			} else if e.EntityType == "shutdown" {
				endMsg = "Агрегат исправен"
			}

			result = append(result, past_events.Event{
				Type:             past_events.EventTypeSuccess,
				Date:             e.EndTime.Time,
				OrganizationID:   &orgID,
				OrganizationName: &orgName,
				Description:      endMsg,
				EntityType:       e.EntityType,
				EntityID:         e.ID,
			})
		}
	}

	return result
}

func isDateInRange(target, start, end time.Time) bool {
	return (target.After(start) || target.Equal(start)) && (target.Before(end) || target.Equal(end))
}

func groupAndSortEvents(events []past_events.Event, timezone *time.Location) []past_events.DateGroup {
	grouped := make(map[string][]past_events.Event)
	for _, e := range events {
		dateKey := e.Date.In(timezone).Format("2006-01-02")
		grouped[dateKey] = append(grouped[dateKey], e)
	}

	var result []past_events.DateGroup
	for date, evs := range grouped {
		sort.Slice(evs, func(i, j int) bool {
			return evs[i].Date.After(evs[j].Date)
		})
		result = append(result, past_events.DateGroup{Date: date, Events: evs})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Date > result[j].Date
	})

	return result
}

func (r *Repo) enrichWithFiles(ctx context.Context, events []past_events.Event) {
	for i := range events {
		if events[i].EntityID == 0 || events[i].EntityType == "" {
			continue
		}

		var files []file.Model
		var err error

		switch events[i].EntityType {
		case "incident":
			files, err = r.loadIncidentFiles(ctx, events[i].EntityID)
		case "shutdown":
			files, err = r.loadShutdownFiles(ctx, events[i].EntityID)
		case "discharge":
			files, err = r.loadDischargeFiles(ctx, events[i].EntityID)
		}

		if err == nil {
			events[i].Files = files
		}
	}
}

func (r *Repo) getIncidentsHelper(ctx context.Context, start, end time.Time) ([]past_events.Event, error) {
	const op = "storage.repo.getIncidentsHelper"

	var result []past_events.Event

	incidentsQuery := `
       SELECT
          i.id,
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

	rows, err := r.db.QueryContext(ctx, incidentsQuery, start, end)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query incidents: %w", op, err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			incidentID  int64
			date        time.Time
			orgID       sql.NullInt64
			orgName     string
			description string
		)

		if err := rows.Scan(&incidentID, &date, &orgID, &orgName, &description); err != nil {
			return nil, fmt.Errorf("%s: failed to scan incident row: %w", op, err)
		}

		var orgIDPtr *int64
		var orgNamePtr *string

		if orgID.Valid {
			orgIDPtr = &orgID.Int64
			orgNamePtr = &orgName
		} else {
			val := "Все предприятия"
			orgNamePtr = &val
		}

		result = append(result, past_events.Event{
			Type:             past_events.EventTypeDanger,
			Date:             date,
			OrganizationID:   orgIDPtr,
			OrganizationName: orgNamePtr,
			Description:      description,
			EntityType:       "incident",
			EntityID:         incidentID,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if result == nil {
		return []past_events.Event{}, nil
	}

	return result, nil
}
