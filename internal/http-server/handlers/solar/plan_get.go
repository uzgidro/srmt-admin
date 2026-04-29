package solar

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	model "srmt-admin/internal/lib/model/solar"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type PlanGetter interface {
	GetSolarPlans(ctx context.Context, year int) ([]model.ProductionPlan, error)
}

func GetPlans(log *slog.Logger, repo PlanGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.solar.GetPlans"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		yearStr := r.URL.Query().Get("year")
		if yearStr == "" {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("year query parameter required"))
			return
		}
		year, err := strconv.Atoi(yearStr)
		if err != nil || year <= 0 {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid year"))
			return
		}

		plans, err := repo.GetSolarPlans(r.Context(), year)
		if err != nil {
			log.Error("failed to get solar plans", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to retrieve plans"))
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, plans)
	}
}
