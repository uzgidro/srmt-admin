package get_by_id

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/event"
	"srmt-admin/internal/storage"
	"strconv"
)

// EventGetter defines repository interface for retrieving a single event
type EventGetter interface {
	GetEventByID(ctx context.Context, id int64) (*event.Model, error)
}

func New(log *slog.Logger, getter EventGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.event.get_by_id.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// Get event ID from URL parameter
		idStr := chi.URLParam(r, "id")
		eventID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Error("invalid event ID", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid event ID"))
			return
		}

		// Get event by ID
		event, err := getter.GetEventByID(r.Context(), eventID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Event not found"))
				return
			}

			log.Error("failed to get event", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve event"))
			return
		}

		log.Info("successfully retrieved event", slog.Int64("event_id", eventID))
		render.JSON(w, r, event)
	}
}
