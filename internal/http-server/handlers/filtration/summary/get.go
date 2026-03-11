package summary

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/filtration"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type SummaryGetter interface {
	GetOrgFiltrationSummary(ctx context.Context, orgID int64, date string) (*filtration.OrgFiltrationSummary, error)
}

func Get(log *slog.Logger, getter SummaryGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.filtration.summary.Get"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		orgIDStr := r.URL.Query().Get("organization_id")
		if orgIDStr == "" {
			log.Warn("missing required 'organization_id' parameter")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Missing required 'organization_id' parameter"))
			return
		}

		orgID, err := strconv.ParseInt(orgIDStr, 10, 64)
		if err != nil {
			log.Warn("invalid organization_id", sl.Err(err), slog.String("organization_id", orgIDStr))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'organization_id' parameter"))
			return
		}

		date := r.URL.Query().Get("date")
		if date == "" {
			log.Warn("missing required 'date' parameter")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Missing required 'date' parameter (format: YYYY-MM-DD)"))
			return
		}

		result, err := getter.GetOrgFiltrationSummary(r.Context(), orgID, date)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("filtration summary not found",
					slog.Int64("organization_id", orgID),
					slog.String("date", date),
				)
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Filtration summary not found"))
				return
			}

			log.Error("failed to get filtration summary", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve filtration summary"))
			return
		}

		log.Info("successfully retrieved filtration summary",
			slog.Int64("organization_id", orgID),
			slog.String("date", date),
		)

		render.JSON(w, r, result)
	}
}
