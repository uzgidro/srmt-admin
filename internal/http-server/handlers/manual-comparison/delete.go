package manualcomparison

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/auth"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type ManualComparisonDeleter interface {
	DeleteManualComparison(ctx context.Context, orgID int64, date string) error
}

func Delete(log *slog.Logger, deleter ManualComparisonDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.manual-comparison.Delete"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		orgIDStr := r.URL.Query().Get("organization_id")
		if orgIDStr == "" {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Missing required 'organization_id' parameter"))
			return
		}
		orgID, err := strconv.ParseInt(orgIDStr, 10, 64)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'organization_id' parameter"))
			return
		}

		date := r.URL.Query().Get("date")
		if date == "" {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Missing required 'date' parameter (format: YYYY-MM-DD)"))
			return
		}
		if _, err := time.Parse("2006-01-02", date); err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'date' parameter (format: YYYY-MM-DD)"))
			return
		}

		if err := auth.CheckOrgAccess(r.Context(), orgID); err != nil {
			log.Warn("access denied to organization", slog.Int64("org_id", orgID))
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, resp.Forbidden("Access denied"))
			return
		}

		if err := deleter.DeleteManualComparison(r.Context(), orgID, date); err != nil {
			log.Error("failed to delete manual comparison", sl.Err(err), slog.Int64("org_id", orgID))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete manual comparison"))
			return
		}

		log.Info("manual comparison deleted",
			slog.String("date", date),
			slog.Int64("org_id", orgID),
		)

		render.Status(r, http.StatusOK)
		render.JSON(w, r, resp.Delete())
	}
}
