package get_statuses

import (
	"context"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/event_status"
)

// EventStatusGetter defines repository interface for retrieving event statuses
type EventStatusGetter interface {
	GetEventStatuses(ctx context.Context) ([]event_status.Model, error)
}

func New(log *slog.Logger, getter EventStatusGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.event.get_statuses.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// Get all event statuses
		statuses, err := getter.GetEventStatuses(r.Context())
		if err != nil {
			log.Error("failed to get event statuses", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve event statuses"))
			return
		}

		log.Info("successfully retrieved event statuses", slog.Int("count", len(statuses)))
		render.JSON(w, r, statuses)
	}
}
