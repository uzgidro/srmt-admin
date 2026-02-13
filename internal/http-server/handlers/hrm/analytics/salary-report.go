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

type SalaryReportGetter interface {
	GetSalaryReport(ctx context.Context, filter dto.ReportFilter) (*analytics.SalaryReport, error)
}

type SalaryTrendGetter interface {
	GetSalaryTrend(ctx context.Context, filter dto.ReportFilter) (*analytics.SalaryTrend, error)
}

func GetSalaryReport(log *slog.Logger, svc SalaryReportGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.analytics.GetSalaryReport"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		filter := parseReportFilter(r)

		result, err := svc.GetSalaryReport(r.Context(), filter)
		if err != nil {
			log.Error("failed to get salary report", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve salary report"))
			return
		}

		render.JSON(w, r, result)
	}
}

func GetSalaryTrend(log *slog.Logger, svc SalaryTrendGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.analytics.GetSalaryTrend"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		filter := parseReportFilter(r)

		result, err := svc.GetSalaryTrend(r.Context(), filter)
		if err != nil {
			log.Error("failed to get salary trend", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve salary trend"))
			return
		}

		render.JSON(w, r, result)
	}
}
