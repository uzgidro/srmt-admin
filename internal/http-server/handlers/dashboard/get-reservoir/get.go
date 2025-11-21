package get

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

type ReservoirGetter interface {
	GetOrganizationsWithReservoir(ctx context.Context, orgIDs []int64, reservoirFetcher dto.ReservoirFetcher) ([]*dto.OrganizationWithReservoir, error)
}

func New(log *slog.Logger, getter ReservoirGetter, orgIDs []int64, reservoirFetcher dto.ReservoirFetcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.dashboard.get-reservoir.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// Get organizations with reservoir metrics
		organizations, err := getter.GetOrganizationsWithReservoir(r.Context(), orgIDs, reservoirFetcher)
		if err != nil {
			log.Error("failed to get organizations with reservoir data", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve reservoir data"))
			return
		}

		log.Info("successfully retrieved organizations with reservoir data", slog.Int("count", len(organizations)))
		render.JSON(w, r, organizations)
	}
}
