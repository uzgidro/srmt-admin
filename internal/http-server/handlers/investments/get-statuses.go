package investments

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	investment_status "srmt-admin/internal/lib/model/investment-status"
	"strconv"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type investmentStatusGetter interface {
	GetAllInvestmentStatuses(ctx context.Context) ([]investment_status.Model, error)
	GetInvestmentStatusesByType(ctx context.Context, typeID int) ([]investment_status.Model, error)
}

func GetStatuses(log *slog.Logger, getter investmentStatusGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.investment.get-statuses"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// Check if type_id filter is provided
		typeIDStr := r.URL.Query().Get("type_id")
		var statuses []investment_status.Model
		var err error

		if typeIDStr != "" {
			typeID, parseErr := strconv.Atoi(typeIDStr)
			if parseErr != nil || typeID < 1 {
				log.Warn("invalid type_id parameter", slog.String("type_id", typeIDStr))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid type_id parameter"))
				return
			}
			statuses, err = getter.GetInvestmentStatusesByType(r.Context(), typeID)
		} else {
			statuses, err = getter.GetAllInvestmentStatuses(r.Context())
		}

		if err != nil {
			log.Error("failed to get investment statuses", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve investment statuses"))
			return
		}

		log.Info("successfully retrieved investment statuses",
			slog.Int("count", len(statuses)),
			slog.String("type_filter", typeIDStr))
		render.JSON(w, r, statuses)
	}
}
