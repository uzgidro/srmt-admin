package vacation

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	vacationmodel "srmt-admin/internal/lib/model/hrm/vacation"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type AllBalancesGetter interface {
	GetAllBalances(ctx context.Context, year int) ([]*vacationmodel.Balance, error)
}

func GetBalances(log *slog.Logger, svc AllBalancesGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.vacation.GetBalances"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		year := time.Now().Year()
		if v := r.URL.Query().Get("year"); v != "" {
			if y, err := strconv.Atoi(v); err == nil {
				year = y
			}
		}

		balances, err := svc.GetAllBalances(r.Context(), year)
		if err != nil {
			log.Error("failed to get all balances", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to get balances"))
			return
		}

		if balances == nil {
			balances = []*vacationmodel.Balance{}
		}
		render.JSON(w, r, balances)
	}
}
