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

type DemographicsGetter interface {
	GetDemographicsReport(ctx context.Context, filter dto.ReportFilter) (*analytics.DemographicsReport, error)
}

type DiversityGetter interface {
	GetDiversityReport(ctx context.Context, filter dto.ReportFilter) (*analytics.DiversityReport, error)
}

func GetDemographics(log *slog.Logger, svc DemographicsGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.analytics.GetDemographics"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		filter := parseReportFilter(r)

		result, err := svc.GetDemographicsReport(r.Context(), filter)
		if err != nil {
			log.Error("failed to get demographics report", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve demographics report"))
			return
		}

		render.JSON(w, r, result)
	}
}

func GetDiversity(log *slog.Logger, svc DiversityGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.analytics.GetDiversity"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		filter := parseReportFilter(r)

		result, err := svc.GetDiversityReport(r.Context(), filter)
		if err != nil {
			log.Error("failed to get diversity report", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve diversity report"))
			return
		}

		render.JSON(w, r, result)
	}
}
