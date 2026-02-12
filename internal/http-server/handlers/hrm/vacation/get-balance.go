package vacation

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	vacationmodel "srmt-admin/internal/lib/model/hrm/vacation"
	"srmt-admin/internal/storage"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type BalanceGetter interface {
	GetBalance(ctx context.Context, employeeID int64, year int) (*vacationmodel.Balance, error)
}

func GetBalance(log *slog.Logger, svc BalanceGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.vacation.GetBalance"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		employeeID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid employee ID"))
			return
		}

		year := time.Now().Year()
		if v := r.URL.Query().Get("year"); v != "" {
			if y, err := strconv.Atoi(v); err == nil {
				year = y
			}
		}

		balance, err := svc.GetBalance(r.Context(), employeeID, year)
		if err != nil {
			if errors.Is(err, storage.ErrBalanceNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Vacation balance not found"))
				return
			}
			log.Error("failed to get vacation balance", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to get balance"))
			return
		}

		render.JSON(w, r, balance)
	}
}
