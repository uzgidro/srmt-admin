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

type pastEventsGetter interface {
	GetPastEvents(ctx context.Context, days int, timezone *time.Location) ([]past_events.DateGroup, error)
	GetPastEventsByDate(ctx context.Context, date time.Time, timezone *time.Location) ([]past_events.DateGroup, error)
}

const defaultDays = 7

func Get(log *slog.Logger, getter pastEventsGetter, minioRepo helpers.MinioURLGenerator, loc *time.Location) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.past_events.get.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// Check for 'date' parameter first (takes precedence over 'days')
		var eventsByDate []past_events.DateGroup
		var dateStr string
		var err error
		dateVal, err := formparser.GetFormDateInLocation(r, "date", loc)
		if err != nil {
			log.Warn("invalid 'date' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'date' parameter, must be in format YYYY-MM-DD"))
			return
		}

		if dateVal != nil {
			// Convert parsed date to the specified timezone
			parsedDate := *dateVal
			dateInTimezone := time.Date(
				parsedDate.Year(),
				parsedDate.Month(),
				parsedDate.Day(),
				0, 0, 0, 0,
				loc,
			)

			log.Info("using provided 'date' parameter", "date", dateInTimezone.Format("2006-01-02"))
			eventsByDate, err = getter.GetPastEventsByDate(r.Context(), dateInTimezone, loc)
			// Need to maintain dateStr for logging later? Reconstruct it.
			dateStr = dateInTimezone.Format("2006-01-02")
		} else {
			// Get 'days' parameter from query (default: 7)
			days := defaultDays

			daysVal, err := formparser.GetFormInt64(r, "days")
			if err != nil {
				log.Warn("invalid 'days' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'days' parameter"))
				return
			}

			if daysVal != nil {
				parsedDays := int(*daysVal)
				if parsedDays < 1 || parsedDays > 365 {
					log.Warn("invalid 'days' parameter range", slog.Int("days", parsedDays))
					render.Status(r, http.StatusBadRequest)
					render.JSON(w, r, resp.BadRequest("Invalid 'days' parameter, must be between 1 and 365"))
					return
				}
				days = parsedDays
				log.Info("using provided 'days' parameter", "days", days)
			} else {
				log.Info("no 'days' parameter provided, using default", "days", defaultDays)
			}

			eventsByDate, err = getter.GetPastEvents(r.Context(), days, loc)
		}
		if err != nil {
			log.Error("failed to get past events", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve past events"))
			return
		}

		// Transform to response model with presigned URLs
		dateGroupsWithURLs := make([]past_events.DateGroupWithURLs, len(eventsByDate))
		totalEvents := 0

		for i, dateGroup := range eventsByDate {
			eventsWithURLs := make([]past_events.EventWithURLs, len(dateGroup.Events))

			for j, event := range dateGroup.Events {
				eventsWithURLs[j] = past_events.EventWithURLs{
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

			dateGroupsWithURLs[i] = past_events.DateGroupWithURLs{
				Date:   dateGroup.Date,
				Events: eventsWithURLs,
			}
			totalEvents += len(eventsWithURLs)
		}

		logFields := []any{
			slog.Int("dates", len(dateGroupsWithURLs)),
			slog.Int("total_events", totalEvents),
		}
		if dateStr != "" {
			logFields = append(logFields, slog.String("date", dateStr))
		}
		log.Info("successfully retrieved past events", logFields...)

		render.JSON(w, r, dateGroupsWithURLs)
	}
}
