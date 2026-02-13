package timesheet

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	ts "srmt-admin/internal/lib/model/hrm/timesheet"
	"strconv"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type CorrectionGetter interface {
	GetCorrections(ctx context.Context, filters dto.CorrectionFilters) ([]*ts.Correction, error)
}

func GetCorrections(log *slog.Logger, svc CorrectionGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.timesheet.GetCorrections"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filters dto.CorrectionFilters
		q := r.URL.Query()

		if v := q.Get("employee_id"); v != "" {
			val, _ := strconv.ParseInt(v, 10, 64)
			filters.EmployeeID = &val
		}
		if v := q.Get("status"); v != "" {
			filters.Status = &v
		}

		corrections, err := svc.GetCorrections(r.Context(), filters)
		if err != nil {
			log.Error("failed to get corrections", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve corrections"))
			return
		}

		render.JSON(w, r, corrections)
	}
}
