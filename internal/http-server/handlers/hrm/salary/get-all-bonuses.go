package salary

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	salary "srmt-admin/internal/lib/model/hrm/salary"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type AllBonusesGetter interface {
	GetAllBonuses(ctx context.Context) ([]*salary.Bonus, error)
}

func GetAllBonuses(log *slog.Logger, svc AllBonusesGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.salary.GetAllBonuses"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		bonuses, err := svc.GetAllBonuses(r.Context())
		if err != nil {
			log.Error("failed to get bonuses", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve bonuses"))
			return
		}

		render.JSON(w, r, bonuses)
	}
}
