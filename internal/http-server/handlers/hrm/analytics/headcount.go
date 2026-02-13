package analytics

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/analytics"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type HeadcountGetter interface {
	GetHeadcountReport(ctx context.Context, filter dto.ReportFilter) (*analytics.HeadcountReport, error)
}

func GetHeadcount(log *slog.Logger, svc HeadcountGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.analytics.GetHeadcount"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		filter := parseReportFilter(r)

		result, err := svc.GetHeadcountReport(r.Context(), filter)
		if err != nil {
			log.Error("failed to get headcount report", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve headcount report"))
			return
		}

		render.JSON(w, r, result)
	}
}
