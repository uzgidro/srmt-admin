package performance

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/performance"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type KPIGetter interface {
	GetKPIs(ctx context.Context) ([]*performance.KPI, error)
}

func GetKPIs(log *slog.Logger, svc KPIGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.performance.GetKPIs"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		result, err := svc.GetKPIs(r.Context())
		if err != nil {
			log.Error("failed to get KPIs", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve KPIs"))
			return
		}

		render.JSON(w, r, result)
	}
}
