package piezometers

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/filtration"
	"srmt-admin/internal/lib/service/auth"
	"strconv"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type PiezometerCounter interface {
	GetPiezometerCountsByOrg(ctx context.Context, orgID int64) (filtration.PiezometerCounts, error)
}

func Counts(log *slog.Logger, counter PiezometerCounter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.filtration.piezometers.Counts"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		orgIDStr := r.URL.Query().Get("organization_id")
		if orgIDStr == "" {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("organization_id is required"))
			return
		}

		orgID, err := strconv.ParseInt(orgIDStr, 10, 64)
		if err != nil {
			log.Warn("invalid organization_id", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid organization_id"))
			return
		}

		if err := auth.CheckOrgAccess(r.Context(), orgID); err != nil {
			log.Warn("access denied to organization", slog.Int64("org_id", orgID))
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, resp.Forbidden("Access denied"))
			return
		}

		counts, err := counter.GetPiezometerCountsByOrg(r.Context(), orgID)
		if err != nil {
			log.Error("failed to get piezometer counts", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve piezometer counts"))
			return
		}

		log.Info("piezometer counts retrieved", slog.Int64("organization_id", orgID))
		render.JSON(w, r, counts)
	}
}
