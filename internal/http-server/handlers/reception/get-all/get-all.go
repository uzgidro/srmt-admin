package get_all

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"srmt-admin/internal/lib/api/formparser"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/reception"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type receptionGetter interface {
	GetAllReceptions(ctx context.Context, filters dto.GetAllReceptionsFilters) ([]*reception.Model, error)
}

const layout = "2006-01-02"

func New(log *slog.Logger, getter receptionGetter, loc *time.Location) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.reception.get_all.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// Parse query parameters for filtering
		filters := dto.GetAllReceptionsFilters{}

		// Parse date filter (start_date) - using local timezone
		if startDateVal, err := formparser.GetFormDateInLocation(r, "start_date", loc); err != nil {
			log.Warn("invalid start_date format", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest(fmt.Sprintf("Invalid start_date format, use YYYY-MM-DD: %v", err)))
			return
		} else {
			filters.StartDate = startDateVal
			if startDateVal != nil {
				log.Info("parsed start_date", "date", startDateVal.Format(layout))
			}
		}

		// Parse date filter (end_date) - using local timezone
		if endDateVal, err := formparser.GetFormDateInLocation(r, "end_date", loc); err != nil {
			log.Warn("invalid end_date format", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest(fmt.Sprintf("Invalid end_date format, use YYYY-MM-DD: %v", err)))
			return
		} else if endDateVal != nil {
			// Set to end of day in local timezone
			endDate := endDateVal.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			filters.EndDate = &endDate
			log.Info("parsed end_date", "date", endDateVal.Format(layout))
		}

		// Parse status filter
		if statusVal := formparser.GetFormString(r, "status"); statusVal != nil && *statusVal != "" {
			statusStr := *statusVal
			// Validate status value
			if statusStr != "default" && statusStr != "true" && statusStr != "false" {
				log.Warn("invalid status value", slog.String("status", statusStr))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid status, must be 'default', 'true', or 'false'"))
				return
			}
			filters.Status = &statusStr
		}

		receptions, err := getter.GetAllReceptions(r.Context(), filters)
		if err != nil {
			log.Error("failed to get receptions", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve receptions"))
			return
		}

		log.Info("successfully retrieved receptions",
			slog.Int("count", len(receptions)),
			slog.Bool("has_filters", filters.StartDate != nil || filters.EndDate != nil || filters.Status != nil),
		)
		render.JSON(w, r, receptions)
	}
}
