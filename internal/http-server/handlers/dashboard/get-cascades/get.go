package get_cascades

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type CascadeGetter interface {
	GetCascadesWithDetails(ctx context.Context, ascueFetcher dto.ASCUEFetcher) ([]*dto.CascadeWithDetails, error)
}

func New(log *slog.Logger, getter CascadeGetter, ascueFetcher dto.ASCUEFetcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.organizations.get-cascades.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// Get cascades with details (contacts, discharges, and ASCUE metrics)
		cascades, err := getter.GetCascadesWithDetails(r.Context(), ascueFetcher)
		if err != nil {
			log.Error("failed to get cascades with details", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve cascades"))
			return
		}

		log.Info("successfully retrieved cascades with details", slog.Int("count", len(cascades)))
		render.JSON(w, r, cascades)
	}
}
