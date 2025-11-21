package get

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/discharge"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type DischargeGetter interface {
	GetCurrentDischarges(ctx context.Context) ([]discharge.Model, error)
}

func New(log *slog.Logger, getter DischargeGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.discharge.get-current.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// Получаем текущие активные сбросы
		discharges, err := getter.GetCurrentDischarges(r.Context())
		if err != nil {
			log.Error("failed to get current discharges", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve current discharges"))
			return
		}

		log.Info("successfully retrieved current discharges", slog.Int("count", len(discharges)))
		render.JSON(w, r, discharges)
	}
}
