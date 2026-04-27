package reservoirflood

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type ConfigDeleter interface {
	DeleteReservoirFloodConfig(ctx context.Context, organizationID int64) error
}

func DeleteConfig(log *slog.Logger, repo ConfigDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.reservoir-flood.DeleteConfig"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		if !callerIsAdmin(r.Context()) {
			// Audit trail for the most destructive endpoint: log the user id
			// (extracted best-effort) so a privilege-escalation attempt is
			// visible in incident analysis.
			userID, _ := auth.GetUserID(r.Context())
			log.Warn("non-admin attempted config delete",
				slog.Int64("user_id", userID),
			)
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, resp.Forbidden("only sc/rais may modify config"))
			return
		}

		orgIDStr := r.URL.Query().Get("organization_id")
		orgID, err := strconv.ParseInt(orgIDStr, 10, 64)
		if err != nil || orgID <= 0 {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("organization_id is required"))
			return
		}

		if err := repo.DeleteReservoirFloodConfig(r.Context(), orgID); err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("config not found"))
				return
			}
			log.Error("failed to delete config", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to delete config"))
			return
		}

		log.Info("config deleted", slog.Int64("organization_id", orgID))
		render.Status(r, http.StatusNoContent)
		render.JSON(w, r, resp.Delete())
	}
}
