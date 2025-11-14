package get_short

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"strconv"
	"strings"
	"time"
)

// EventShortGetter defines repository interface for retrieving events in short format
type EventShortGetter interface {
	GetAllEventsShort(ctx context.Context, filters dto.GetAllEventsFilters) ([]dto.EventShort, error)
}

func New(log *slog.Logger, getter EventShortGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.event.get_short.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// Parse filters from query parameters
		var filters dto.GetAllEventsFilters
		q := r.URL.Query()

		// Parse event_status_id[] - multiple values allowed
		if statusIDStrs := q["event_status_id[]"]; len(statusIDStrs) > 0 {
			for _, statusIDStr := range statusIDStrs {
				statusID, err := strconv.Atoi(statusIDStr)
				if err != nil {
					log.Warn("invalid 'event_status_id' parameter", sl.Err(err), slog.String("value", statusIDStr))
					render.Status(r, http.StatusBadRequest)
					render.JSON(w, r, resp.BadRequest("Invalid 'event_status_id[]' parameter"))
					return
				}
				filters.EventStatusIDs = append(filters.EventStatusIDs, statusID)
			}
		}

		// Parse event_type_id[] - multiple values allowed
		if typeIDStrs := q["event_type_id[]"]; len(typeIDStrs) > 0 {
			for _, typeIDStr := range typeIDStrs {
				typeID, err := strconv.Atoi(typeIDStr)
				if err != nil {
					log.Warn("invalid 'event_type_id' parameter", sl.Err(err), slog.String("value", typeIDStr))
					render.Status(r, http.StatusBadRequest)
					render.JSON(w, r, resp.BadRequest("Invalid 'event_type_id[]' parameter"))
					return
				}
				filters.EventTypeIDs = append(filters.EventTypeIDs, typeID)
			}
		}

		// Parse start_date (format: YYYY-MM-DD or RFC3339)
		if startDateStr := q.Get("start_date"); startDateStr != "" {
			startDate, err := parseDate(startDateStr)
			if err != nil {
				log.Warn("invalid 'start_date' parameter", sl.Err(err), slog.String("value", startDateStr))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'start_date' format. Use YYYY-MM-DD or RFC3339"))
				return
			}
			filters.StartDate = &startDate
		}

		// Parse end_date (format: YYYY-MM-DD or RFC3339)
		if endDateStr := q.Get("end_date"); endDateStr != "" {
			endDate, err := parseDate(endDateStr)
			if err != nil {
				log.Warn("invalid 'end_date' parameter", sl.Err(err), slog.String("value", endDateStr))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'end_date' format. Use YYYY-MM-DD or RFC3339"))
				return
			}
			filters.EndDate = &endDate
		}

		// Parse organization_id
		if orgIDStr := q.Get("organization_id"); orgIDStr != "" {
			orgID, err := strconv.ParseInt(orgIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'organization_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'organization_id' parameter"))
				return
			}
			filters.OrganizationID = &orgID
		}

		// Get events with filters
		events, err := getter.GetAllEventsShort(r.Context(), filters)
		if err != nil {
			log.Error("failed to get events", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve events"))
			return
		}

		log.Info("successfully retrieved events in short format",
			slog.Int("count", len(events)),
			slog.Int("status_filters", len(filters.EventStatusIDs)),
			slog.Int("type_filters", len(filters.EventTypeIDs)),
		)

		render.JSON(w, r, events)
	}
}

// parseDate attempts to parse a date string in multiple formats
func parseDate(dateStr string) (time.Time, error) {
	// Try YYYY-MM-DD format first
	if t, err := time.Parse("2006-01-02", dateStr); err == nil {
		return t, nil
	}

	// Try RFC3339 format
	if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
		return t, nil
	}

	// Try date-only with time at start of day
	if !strings.Contains(dateStr, "T") {
		dateStr = dateStr + "T00:00:00Z"
		if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, ErrInvalidDateFormat
}

var ErrInvalidDateFormat = errors.New("invalid date format")
