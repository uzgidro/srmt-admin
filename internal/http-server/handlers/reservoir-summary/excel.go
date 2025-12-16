package reservoirsummary

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	excelgen "srmt-admin/internal/lib/service/excel/reservoir-summary"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

// GetExcel returns an HTTP handler for Excel export
func GetExcel(log *slog.Logger, generator *excelgen.Generator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.reservoirsummary.GetExcel"
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

		// Validate date format (YYYY-MM-DD)
		parsedDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			log.Warn("invalid date format", sl.Err(err), slog.String("date", dateStr))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid date format. Expected YYYY-MM-DD"))
			return
		}

		// Generate Excel file
		excelFile, err := generator.GenerateExcel(dateStr)
		if err != nil {
			log.Error("failed to generate Excel file", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to generate Excel file"))
			return
		}
		defer excelFile.Close()

		// Write to buffer
		var buf bytes.Buffer
		if err := excelFile.Write(&buf); err != nil {
			log.Error("failed to write Excel to buffer", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to prepare Excel file"))
			return
		}

		// Set headers for file download
		filename := fmt.Sprintf("reservoir-summary-%s.xlsx", parsedDate.Format("2006-01-02"))

		w.Header().Set("Content-Type",
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition",
			fmt.Sprintf("attachment; filename=\"%s\"", filename))
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
	}
}
