package calendar

import (
	"context"
	"log/slog"
	"net/http"
	"sort"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type calendarEventsGetter interface {
	GetCalendarEventsCounts(ctx context.Context, year, month int, timezone *time.Location) (map[string]*dto.DayCounts, error)
}

// Get returns a handler for retrieving calendar event counts
func Get(log *slog.Logger, getter calendarEventsGetter, loc *time.Location) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.calendar.get.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// Get current date in the specified timezone
		now := time.Now().In(loc)
		year := now.Year()
		month := int(now.Month())

		// Parse 'year' parameter
		yearStr := r.URL.Query().Get("year")
		if yearStr != "" {
			parsedYear, parseErr := strconv.Atoi(yearStr)
			if parseErr != nil || parsedYear < 1900 || parsedYear > 2100 {
				log.Warn("invalid 'year' parameter", sl.Err(parseErr))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'year' parameter, must be between 1900 and 2100"))
				return
			}
			year = parsedYear
			log.Info("using provided 'year' parameter", "year", year)
		}

		// Parse 'month' parameter
		monthStr := r.URL.Query().Get("month")
		if monthStr != "" {
			parsedMonth, parseErr := strconv.Atoi(monthStr)
			if parseErr != nil || parsedMonth < 1 || parsedMonth > 12 {
				log.Warn("invalid 'month' parameter", sl.Err(parseErr))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'month' parameter, must be between 1 and 12"))
				return
			}
			month = parsedMonth
			log.Info("using provided 'month' parameter", "month", month)
		}

		// Get calendar event counts from repository
		dayCounts, err := getter.GetCalendarEventsCounts(r.Context(), year, month, loc)
		if err != nil {
			log.Error("failed to get calendar events", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve calendar events"))
			return
		}

		// Transform map[string]*DayCounts to []DayCounts for JSON serialization
		days := make([]dto.DayCounts, 0, len(dayCounts))
		for date, counts := range dayCounts {
			days = append(days, dto.DayCounts{
				Date:       date,
				Incidents:  counts.Incidents,
				Shutdowns:  counts.Shutdowns,
				Discharges: counts.Discharges,
				Visits:     counts.Visits,
			})
		}

		// Sort days by date
		sort.Slice(days, func(i, j int) bool {
			return days[i].Date < days[j].Date
		})

		// Build response
		response := dto.CalendarResponse{
			Year:  year,
			Month: month,
			Days:  days,
		}

		log.Info("successfully retrieved calendar events",
			slog.Int("year", year),
			slog.Int("month", month),
			slog.Int("days_with_events", len(days)),
		)

		render.JSON(w, r, response)
	}
}
