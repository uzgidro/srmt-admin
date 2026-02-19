package reservoirsummaryhourly

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	model "srmt-admin/internal/lib/model/reservoir-hourly"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/xuri/excelize/v2"
)

// ReportBuilder builds the hourly reservoir report
type ReportBuilder interface {
	BuildReport(ctx context.Context, date string) (*model.HourlyReport, error)
}

// ExcelGenerator generates an Excel file from the hourly report
type ExcelGenerator interface {
	GenerateExcel(report *model.HourlyReport) (*excelize.File, error)
}

// GetExport returns an HTTP handler that builds the hourly report and returns it as an Excel file
func GetExport(log *slog.Logger, builder ReportBuilder, generator ExcelGenerator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.reservoirsummaryhourly.GetExport"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// Parse date query parameter, default to today
		dateStr := r.URL.Query().Get("date")
		if dateStr == "" {
			dateStr = time.Now().Format("2006-01-02")
		}

		// Validate date format
		parsedDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			log.Warn("invalid date format", sl.Err(err), slog.String("date", dateStr))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid date format. Expected YYYY-MM-DD"))
			return
		}

		// Build report
		report, err := builder.BuildReport(r.Context(), dateStr)
		if err != nil {
			log.Error("failed to build hourly report", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to build hourly report"))
			return
		}

		// Generate Excel file
		excelFile, err := generator.GenerateExcel(report)
		if err != nil {
			log.Error("failed to generate Excel file", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to generate Excel file"))
			return
		}
		defer excelFile.Close()

		// Write Excel to buffer
		var buf bytes.Buffer
		if err := excelFile.Write(&buf); err != nil {
			log.Error("failed to write Excel to buffer", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to prepare Excel file"))
			return
		}

		// Set headers for Excel download
		filename := fmt.Sprintf("СВОД-ПОЧАСОВОЙ-%s.xlsx", parsedDate.Format("2006-01-02"))
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", buf.Len()))

		if _, err := w.Write(buf.Bytes()); err != nil {
			log.Error("failed to write response", sl.Err(err))
			return
		}

		log.Info("successfully generated hourly Excel export",
			slog.String("date", dateStr),
			slog.String("filename", filename),
			slog.Int("file_size", buf.Len()),
		)
	}
}
