package piezometercounts

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/filtration"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type PiezometerCountsGetter interface {
	GetPiezometerCounts(ctx context.Context, orgID int64) (*filtration.PiezometerCountsRecord, error)
}

func Get(log *slog.Logger, getter PiezometerCountsGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.filtration.piezometer-counts.Get"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		orgIDStr := r.URL.Query().Get("organization_id")
		if orgIDStr == "" {
			log.Warn("missing required 'organization_id' parameter")
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

		record, err := getter.GetPiezometerCounts(r.Context(), orgID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Info("piezometer counts not found, returning defaults", slog.Int64("organization_id", orgID))
				render.JSON(w, r, &filtration.PiezometerCountsRecord{
					OrganizationID: orgID,
				})
				return
			}
			log.Error("failed to get piezometer counts", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve piezometer counts"))
			return
		}

		log.Info("piezometer counts retrieved", slog.Int64("organization_id", orgID))
		render.JSON(w, r, record)
	}
}
