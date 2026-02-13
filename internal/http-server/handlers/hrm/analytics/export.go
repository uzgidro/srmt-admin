package analytics

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/xuri/excelize/v2"
)

type CustomReportGetter interface {
	GetCustomReport(ctx context.Context, filter dto.ReportFilter) (interface{}, error)
}

type ExcelExporter interface {
	ExportExcel(ctx context.Context, filter dto.ReportFilter) (*excelize.File, error)
}

func ExportGeneric(log *slog.Logger, svc CustomReportGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.analytics.ExportGeneric"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		filter := parseReportFilter(r)

		result, err := svc.GetCustomReport(r.Context(), filter)
		if err != nil {
			log.Error("failed to get custom report", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve report"))
			return
		}

		render.JSON(w, r, result)
	}
}

func ExportExcel(log *slog.Logger, svc ExcelExporter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.analytics.ExportExcel"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		filter := parseReportFilter(r)

		f, err := svc.ExportExcel(r.Context(), filter)
		if err != nil {
			log.Error("failed to export excel", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to export report"))
			return
		}
		defer f.Close()

		reportType := "analytics"
		if filter.ReportType != nil {
			reportType = *filter.ReportType
		}

		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="hrm_%s_report.xlsx"`, reportType))

		if err := f.Write(w); err != nil {
			log.Error("failed to write excel to response", sl.Err(err))
		}
	}
}

func ExportPDF(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.analytics.ExportPDF"
		_ = log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		render.Status(r, http.StatusNotImplemented)
		render.JSON(w, r, map[string]string{
			"error":   "PDF export is not implemented",
			"message": "PDF export requires LibreOffice which may not be available on the server",
		})
	}
}
