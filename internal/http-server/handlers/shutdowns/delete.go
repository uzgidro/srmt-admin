package shutdowns

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
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"
	"strconv"
)

type shutdownDeleter interface {
	DeleteShutdown(ctx context.Context, id int64) error
	GetShutdownOrganizationID(ctx context.Context, id int64) (int64, error)
	GetOrganizationParentID(ctx context.Context, orgID int64) (*int64, error)
}

func Delete(log *slog.Logger, deleter shutdownDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.shutdown.Delete"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		// Lookup current org → 404 if missing.
		curOrgID, err := deleter.GetShutdownOrganizationID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("shutdown not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Shutdown not found"))
				return
			}
			log.Error("failed to load shutdown org", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to load shutdown"))
			return
		}

		// Access check — foreign resource → 404 (enumeration defense).
		if err := auth.CheckCascadeStationAccess(r.Context(), curOrgID, deleter); err != nil {
			if errors.Is(err, auth.ErrForbidden) || errors.Is(err, auth.ErrNoOrganization) {
				log.Warn("cascade access denied on delete", slog.Int64("user_id", userID), slog.Int64("shutdown_id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Shutdown not found"))
				return
			}
			log.Error("cascade access check failed", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to verify access"))
			return
		}

		err = deleter.DeleteShutdown(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("shutdown not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Shutdown not found"))
				return
			}
			log.Error("failed to delete shutdown", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete shutdown"))
			return
		}

		log.Info("shutdown deleted",
			slog.Int64("id", id),
			slog.Int64("user_id", userID),
			slog.Int64("target_org_id", curOrgID),
		)
		w.WriteHeader(http.StatusOK)
	}
}
