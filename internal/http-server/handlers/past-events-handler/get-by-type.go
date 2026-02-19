package past_events_handler

import (
	"context"
	"log/slog"
	"net/http"
	"srmt-admin/internal/lib/api/formparser"
	resp "srmt-admin/internal/lib/api/response"
	past_events "srmt-admin/internal/lib/dto/past-events"
	"srmt-admin/internal/lib/helpers"
	"srmt-admin/internal/lib/logger/sl"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type pastEventsByTypeGetter interface {
	GetPastEventsByDateAndType(ctx context.Context, date time.Time, eventType string, timezone *time.Location) ([]past_events.Event, error)
}

type byTypeResponse struct {
	Date   string                      `json:"date"`
	Type   string                      `json:"type"`
	Events []past_events.EventWithURLs `json:"events"`
}

// GetByType returns a handler for retrieving past events by date and type
func GetByType(log *slog.Logger, getter pastEventsByTypeGetter, minioRepo helpers.MinioURLGenerator, loc *time.Location) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.past_events.get_by_type.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// Get 'date' parameter (required)

		// Get 'date' parameter (required)
		dateVal, err := formparser.GetFormDateInLocation(r, "date", loc)
		if err != nil {
			log.Warn("invalid 'date' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'date' parameter, must be in format YYYY-MM-DD"))
			return
		}
		if dateVal == nil {
			log.Warn("missing 'date' parameter")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Missing required parameter 'date' (format: YYYY-MM-DD)"))
			return
		}

		// Convert parsed date to the specified timezone
		parsedDate := *dateVal
		dateInTimezone := time.Date(
			parsedDate.Year(),
			parsedDate.Month(),
			parsedDate.Day(),
			0, 0, 0, 0,
			loc,
		)

		// Get 'type' parameter (required)
		eventType := formparser.GetFormString(r, "type")
		if eventType == nil || *eventType == "" {
			log.Warn("missing 'type' parameter")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Missing required parameter 'type' (incident|shutdown|discharge|visit)"))
			return
		}

		// Validate event type
		// Validate event type
		validTypes := map[string]bool{
			"incident":  true,
			"shutdown":  true,
			"discharge": true,
			"visit":     true,
		}
		if !validTypes[*eventType] {
			log.Warn("invalid 'type' parameter", slog.String("type", *eventType))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'type' parameter, must be one of: incident, shutdown, discharge, visit"))
			return
		}

		log.Info("fetching past events by type", "date", dateInTimezone.Format("2006-01-02"), "type", *eventType)

		// Get events from repository
		events, err := getter.GetPastEventsByDateAndType(r.Context(), dateInTimezone, *eventType, loc)
		if err != nil {
			log.Error("failed to get past events by type", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve past events"))
			return
		}

		// Transform to response model with presigned URLs
		eventsWithURLs := make([]past_events.EventWithURLs, len(events))
		for i, event := range events {
			eventsWithURLs[i] = past_events.EventWithURLs{
				Type:             event.Type,
				Date:             event.Date,
				OrganizationID:   event.OrganizationID,
				OrganizationName: event.OrganizationName,
				Description:      event.Description,
				EntityType:       event.EntityType,
				EntityID:         event.EntityID,
				Files:            helpers.TransformFilesWithURLs(r.Context(), event.Files, minioRepo, log),
			}
		}

		response := byTypeResponse{
			Date:   dateInTimezone.Format("2006-01-02"),
			Type:   *eventType,
			Events: eventsWithURLs,
		}

		log.Info("successfully retrieved past events by type",
			slog.String("date", dateInTimezone.Format("2006-01-02")),
			slog.String("type", *eventType),
			slog.Int("count", len(eventsWithURLs)),
		)

		render.JSON(w, r, response)
	}
}
