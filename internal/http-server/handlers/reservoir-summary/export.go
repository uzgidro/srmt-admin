package reservoirsummary

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	excelgen "srmt-admin/internal/lib/service/excel/reservoir-summary"
	"srmt-admin/internal/storage/repo"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/user0608/excel2pdf"
)

// GetExport returns an HTTP handler for Excel/PDF export
func GetExport(log *slog.Logger, pgRepo *repo.Repo, generator *excelgen.Generator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.reservoirsummary.GetExport"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// Parse and validate date query parameter
		dateStr := r.URL.Query().Get("date")
		if dateStr == "" {
			log.Warn("missing required 'date' parameter")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Missing required 'date' parameter (format: YYYY-MM-DD)"))
			return
		}

		// Parse and validate format query parameter
		format := r.URL.Query().Get("format")
		if format == "" {
			format = "excel" // Default to excel
		}
		if format != "excel" && format != "pdf" {
			log.Warn("invalid format parameter", slog.String("format", format))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid format parameter. Expected 'excel' or 'pdf'"))
			return
		}

		// Validate date format (YYYY-MM-DD)
		parsedDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			log.Warn("invalid date format", sl.Err(err), slog.String("date", dateStr))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid date format. Expected YYYY-MM-DD"))
			return
		}

		// Fetch reservoir summary data
		data, err := pgRepo.GetReservoirSummary(r.Context(), dateStr)
		if err != nil {
			log.Error("failed to fetch reservoir summary data", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to fetch reservoir data"))
			return
		}

		// Generate Excel file
		excelFile, err := generator.GenerateExcel(dateStr, data)
		if err != nil {
			log.Error("failed to generate Excel file", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to generate Excel file"))
			return
		}
		defer excelFile.Close()

		// Handle format-specific export
		if format == "excel" {
			// Excel export
			var buf bytes.Buffer
			if err := excelFile.Write(&buf); err != nil {
				log.Error("failed to write Excel to buffer", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to prepare Excel file"))
				return
			}

			// Set headers for Excel download
			filename := fmt.Sprintf("СВОД-%s.xlsx", parsedDate.Format("2006-01-02"))
			w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
			w.Header().Set("Content-Length", fmt.Sprintf("%d", buf.Len()))

			// Stream file to client
			if _, err := w.Write(buf.Bytes()); err != nil {
				log.Error("failed to write response", sl.Err(err))
				return
			}

			log.Info("successfully generated Excel export",
				slog.String("date", dateStr),
				slog.String("filename", filename),
				slog.Int("file_size", buf.Len()),
			)
		} else {
			// PDF export
			// Create temporary directory for conversion
			tempDir, err := os.MkdirTemp("", "reservoir-summary-pdf-*")
			if err != nil {
				log.Error("failed to create temp directory", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to prepare PDF conversion"))
				return
			}
			defer os.RemoveAll(tempDir)

			// Save Excel file temporarily
			excelPath := filepath.Join(tempDir, "reservoir-summary.xlsx")
			if err := excelFile.SaveAs(excelPath); err != nil {
				log.Error("failed to save Excel file", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to prepare PDF conversion"))
				return
			}

			// Convert Excel to PDF
			pdfPath, err := excel2pdf.ConvertExcelToPdf(excelPath)
			if err != nil {
				log.Error("failed to convert Excel to PDF", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to convert to PDF"))
				return
			}

			// Read PDF file
			pdfData, err := os.ReadFile(pdfPath)
			if err != nil {
				log.Error("failed to read PDF file", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to read PDF file"))
				return
			}

			// Set headers for PDF download
			filename := fmt.Sprintf("СВОД-%s.pdf", parsedDate.Format("2006-01-02"))
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfData)))

			// Stream file to client
			if _, err := w.Write(pdfData); err != nil {
				log.Error("failed to write response", sl.Err(err))
				return
			}

			log.Info("successfully generated PDF export",
				slog.String("date", dateStr),
				slog.String("filename", filename),
				slog.Int("file_size", len(pdfData)),
			)
		}
	}
}
