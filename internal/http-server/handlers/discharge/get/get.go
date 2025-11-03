package get

import (
	"context"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/discharge"
)

type DischargeGetter interface {
	GetAllDischarges(ctx context.Context) ([]discharge.Model, error)
}

func New(log *slog.Logger, getter DischargeGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.discharge.get.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		discharges, err := getter.GetAllDischarges(r.Context())
		if err != nil {
			log.Error("failed to get all discharges", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve discharges"))
			return
		}

		log.Info("successfully retrieved discharges", slog.Int("count", len(discharges)))
		render.JSON(w, r, discharges)
	}
}
