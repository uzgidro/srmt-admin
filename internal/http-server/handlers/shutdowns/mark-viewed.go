package shutdowns

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type shutdownViewedMarker interface {
	MarkShutdownAsViewed(ctx context.Context, id int64) error
	GetShutdownOrganizationID(ctx context.Context, id int64) (int64, error)
	GetOrganizationParentID(ctx context.Context, orgID int64) (*int64, error)
}

func MarkViewed(log *slog.Logger, marker shutdownViewedMarker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.shutdown.MarkViewed"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		curOrgID, err := marker.GetShutdownOrganizationID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("shutdown not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Shutdown not found"))
				return
			}
			log.Error("failed to load shutdown org", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to mark shutdown as viewed"))
			return
		}

		if err := auth.CheckCascadeStationAccess(r.Context(), curOrgID, marker); err != nil {
			if errors.Is(err, auth.ErrForbidden) || errors.Is(err, auth.ErrNoOrganization) {
				log.Warn("cascade access denied on mark-viewed", slog.Int64("shutdown_id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Shutdown not found"))
				return
			}
			log.Error("cascade access check failed", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to verify access"))
			return
		}

		err = marker.MarkShutdownAsViewed(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("shutdown not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Shutdown not found"))
				return
			}
			log.Error("failed to mark shutdown as viewed", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to mark shutdown as viewed"))
			return
		}

		log.Info("shutdown marked as viewed",
			slog.Int64("id", id),
			slog.Int64("target_org_id", curOrgID),
		)
		render.Status(r, http.StatusOK)
		render.JSON(w, r, struct{}{})
	}
}
