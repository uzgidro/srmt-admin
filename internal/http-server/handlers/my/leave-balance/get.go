package leavebalance

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/profile"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type BalanceGetter interface {
	GetMyLeaveBalance(ctx context.Context, employeeID int64, year int) (*profile.LeaveBalance, error)
}

func Get(log *slog.Logger, repo BalanceGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.my.leave-balance.Get"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("unauthorized"))
			return
		}

		year := time.Now().Year()
		if v := r.URL.Query().Get("year"); v != "" {
			if parsed, err := strconv.Atoi(v); err == nil {
				year = parsed
			}
		}

		balance, err := repo.GetMyLeaveBalance(r.Context(), claims.ContactID, year)
		if err != nil {
			log.Error("failed to get leave balance", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to get leave balance"))
			return
		}

		render.JSON(w, r, balance)
	}
}
