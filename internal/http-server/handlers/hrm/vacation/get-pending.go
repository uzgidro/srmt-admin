package vacation

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	vacationmodel "srmt-admin/internal/lib/model/hrm/vacation"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type PendingGetter interface {
	GetPending(ctx context.Context) ([]*vacationmodel.Vacation, error)
}

func GetPending(log *slog.Logger, svc PendingGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.vacation.GetPending"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		vacations, err := svc.GetPending(r.Context())
		if err != nil {
			log.Error("failed to get pending vacations", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to get pending vacations"))
			return
		}

		if vacations == nil {
			vacations = []*vacationmodel.Vacation{}
		}
		render.JSON(w, r, vacations)
	}
}
