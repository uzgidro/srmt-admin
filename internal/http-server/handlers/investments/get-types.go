package investments

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	investment_type "srmt-admin/internal/lib/model/investment-type"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type investmentTypeGetter interface {
	GetAllInvestmentTypes(ctx context.Context) ([]investment_type.Model, error)
}

func GetTypes(log *slog.Logger, getter investmentTypeGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.investment.get-types"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		types, err := getter.GetAllInvestmentTypes(r.Context())
		if err != nil {
			log.Error("failed to get investment types", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve investment types"))
			return
		}

		log.Info("successfully retrieved investment types", slog.Int("count", len(types)))
		render.JSON(w, r, types)
	}
}
