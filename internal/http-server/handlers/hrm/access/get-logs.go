package access

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/access"
	"strconv"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type LogGetter interface {
	GetLogs(ctx context.Context, filters dto.AccessLogFilters) ([]*access.AccessLog, error)
}

func GetLogs(log *slog.Logger, svc LogGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.access.GetLogs"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		q := r.URL.Query()
		var filters dto.AccessLogFilters

		if v := q.Get("employee_id"); v != "" {
			if id, err := strconv.ParseInt(v, 10, 64); err == nil {
				filters.EmployeeID = &id
			}
		}
		if v := q.Get("zone_id"); v != "" {
			if id, err := strconv.ParseInt(v, 10, 64); err == nil {
				filters.ZoneID = &id
			}
		}
		if v := q.Get("direction"); v != "" {
			filters.Direction = &v
		}
		if v := q.Get("status"); v != "" {
			filters.Status = &v
		}
		if v := q.Get("date_from"); v != "" {
			filters.DateFrom = &v
		}
		if v := q.Get("date_to"); v != "" {
			filters.DateTo = &v
		}

		logs, err := svc.GetLogs(r.Context(), filters)
		if err != nil {
			log.Error("failed to get access logs", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve access logs"))
			return
		}

		render.JSON(w, r, logs)
	}
}
