package gesreport

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/xuri/excelize/v2"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	model "srmt-admin/internal/lib/model/ges-report"
	ownneedsgen "srmt-admin/internal/lib/service/excel/ownneeds"
)

// OwnNeedsReportBuilder is the local interface the own-needs export handler
// depends on — narrower than ReportBuilder because this endpoint does not
// support cascade filtering.
type OwnNeedsReportBuilder interface {
	BuildOwnNeedsReport(ctx context.Context, date string) (*model.OwnNeedsReport, error)
}

// ExportOwnNeeds returns an HTTP handler that builds the own-needs (СН/ХН)
// daily report and streams it as an Excel file. Access is gated to sc/rais
// roles by the route middleware (defence in depth).
func ExportOwnNeeds(
	log *slog.Logger,
	reportSvc OwnNeedsReportBuilder,
	generator *ownneedsgen.Generator,
	loc *time.Location,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.ges-report.ExportOwnNeeds"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		dateStr := r.URL.Query().Get("date")
		if dateStr == "" {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("date is required (YYYY-MM-DD)"))
			return
		}
		parsedDate, err := time.ParseInLocation("2006-01-02", dateStr, loc)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid date format, expected YYYY-MM-DD"))
			return
		}

		report, err := reportSvc.BuildOwnNeedsReport(r.Context(), dateStr)
		if err != nil {
			log.Error("failed to build own-needs report", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to build report"))
			return
		}

		excelFile, err := generator.GenerateExcel(ownneedsgen.Params{
			Report: report,
			Date:   parsedDate,
		})
		if err != nil {
			log.Error("failed to generate Excel file", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to generate Excel file"))
			return
		}
		defer excelFile.Close()

		writeOwnNeedsExcel(w, excelFile, parsedDate, log)
	}
}

// writeOwnNeedsExcel serializes the workbook and writes it to the response
// with the expected headers. Errors during serialization use http.Error so
// the response body remains diagnostic-shaped (we cannot retroactively swap
// to JSON once headers may have been flushed).
func writeOwnNeedsExcel(w http.ResponseWriter, f *excelize.File, date time.Time, log *slog.Logger) {
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		log.Error("failed to write Excel to buffer", sl.Err(err))
		http.Error(w, "failed to prepare Excel file", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("Own-Needs-%s.xlsx", date.Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", buf.Len()))

	if _, err := w.Write(buf.Bytes()); err != nil {
		log.Error("failed to write response", sl.Err(err))
	}

	log.Info("generated own-needs Excel export",
		slog.String("date", date.Format("2006-01-02")),
		slog.String("filename", filename),
		slog.Int("file_size", buf.Len()),
	)
}
