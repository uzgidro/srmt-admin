package competency

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/competency"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type ReportGetter interface {
	GetCompetencyReport(ctx context.Context) (*competency.CompetencyReport, error)
}

func GetReports(log *slog.Logger, svc ReportGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.GetReports"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		result, err := svc.GetCompetencyReport(r.Context())
		if err != nil {
			log.Error("failed to get competency report", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve competency report"))
			return
		}

		render.JSON(w, r, result)
	}
}
