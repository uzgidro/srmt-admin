package piezometers

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/filtration"
	"strconv"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type PiezometerGetter interface {
	GetPiezometersByOrg(ctx context.Context, orgID int64) ([]filtration.Piezometer, error)
}

func Get(log *slog.Logger, getter PiezometerGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.filtration.piezometers.Get"
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

		piezometers, err := getter.GetPiezometersByOrg(r.Context(), orgID)
		if err != nil {
			log.Error("failed to get piezometers", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve piezometers"))
			return
		}

		log.Info("piezometers retrieved", slog.Int("count", len(piezometers)))
		render.JSON(w, r, piezometers)
	}
}
