package reservoirsummaryhourly

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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

// GetExport returns an HTTP handler that builds the hourly report and returns it as Excel or PDF
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

		// Parse and validate format query parameter
		format := r.URL.Query().Get("format")
		if format == "" {
			format = "excel"
		}
		if format != "excel" && format != "pdf" {
			log.Warn("invalid format parameter", slog.String("format", format))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid format parameter. Expected 'excel' or 'pdf'"))
			return
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

		if format == "excel" {
			var buf bytes.Buffer
			if err := excelFile.Write(&buf); err != nil {
				log.Error("failed to write Excel to buffer", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to prepare Excel file"))
				return
			}

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
		} else {
			// PDF export via LibreOffice
			tempDir, err := os.MkdirTemp("", "reservoir-hourly-pdf-*")
			if err != nil {
				log.Error("failed to create temp directory", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to prepare PDF conversion"))
				return
			}
			defer os.RemoveAll(tempDir)

			// Set page margins for PDF
			sheet := excelFile.GetSheetName(0)
			marginTop := 0.75
			marginBottom := 0.3
			marginLeft := 0.7
			marginRight := 0.7
			marginHeader := 0.3
			marginFooter := 0.0

			if err := excelFile.SetPageMargins(sheet,
				&excelize.PageLayoutMarginsOptions{
					Top:    &marginTop,
					Bottom: &marginBottom,
					Left:   &marginLeft,
					Right:  &marginRight,
					Header: &marginHeader,
					Footer: &marginFooter,
				},
			); err != nil {
				log.Error("failed to set page margins", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to prepare PDF conversion"))
				return
			}

			excelPath := filepath.Join(tempDir, "reservoir-hourly.xlsx")
			if err := excelFile.SaveAs(excelPath); err != nil {
				log.Error("failed to save Excel file", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to prepare PDF conversion"))
				return
			}

			cmd := exec.Command(
				"soffice",
				"--headless",
				"--convert-to", "pdf",
				"--outdir", tempDir,
				excelPath,
			)

			output, err := cmd.CombinedOutput()
			if err != nil {
				log.Error("failed to convert Excel to PDF",
					sl.Err(err),
					slog.String("output", string(output)),
				)
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to convert to PDF"))
				return
			}

			pdfPath := filepath.Join(tempDir, "reservoir-hourly.pdf")
			pdfData, err := os.ReadFile(pdfPath)
			if err != nil {
				log.Error("failed to read PDF file", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to read PDF file"))
				return
			}

			filename := fmt.Sprintf("СВОД-ПОЧАСОВОЙ-%s.pdf", parsedDate.Format("2006-01-02"))
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfData)))

			if _, err := w.Write(pdfData); err != nil {
				log.Error("failed to write response", sl.Err(err))
				return
			}

			log.Info("successfully generated hourly PDF export",
				slog.String("date", dateStr),
				slog.String("filename", filename),
				slog.Int("file_size", len(pdfData)),
			)
		}
	}
}
