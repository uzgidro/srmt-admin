package export

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
	"srmt-admin/internal/lib/model/discharge"
	"srmt-admin/internal/lib/model/incident"
	"srmt-admin/internal/lib/model/shutdown"
	"srmt-admin/internal/lib/model/visit"
	scgen "srmt-admin/internal/lib/service/excel/sc"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/xuri/excelize/v2"
)

// DischargeGetter defines the interface for fetching discharge data
type DischargeGetter interface {
	GetAllDischarges(ctx context.Context, isOngoing *bool, startDate, endDate *time.Time) ([]discharge.Model, error)
}

// ShutdownGetter defines the interface for fetching shutdown data
type ShutdownGetter interface {
	GetShutdowns(ctx context.Context, day time.Time) ([]*shutdown.ResponseModel, error)
}

// OrgTypesGetter defines the interface for fetching organization types
type OrgTypesGetter interface {
	GetOrganizationTypesMap(ctx context.Context) (map[int64][]string, error)
}

// VisitGetter defines the interface for fetching visit data
type VisitGetter interface {
	GetVisits(ctx context.Context, day time.Time) ([]*visit.ResponseModel, error)
}

// IncidentGetter defines the interface for fetching incident data
type IncidentGetter interface {
	GetIncidents(ctx context.Context, day time.Time) ([]*incident.ResponseModel, error)
}

// New returns an HTTP handler for Excel/PDF export of SC reports
func New(
	log *slog.Logger,
	dischargeGetter DischargeGetter,
	shutdownGetter ShutdownGetter,
	orgTypesGetter OrgTypesGetter,
	visitGetter VisitGetter,
	incidentGetter IncidentGetter,
	generator *scgen.Generator,
	loc *time.Location,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.sc.export.New"
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

		// Validate date format and calculate period (7:00 to 7:00 next day in local time)
		parsedDate, err := time.ParseInLocation("2006-01-02", dateStr, loc)
		if err != nil {
			log.Warn("invalid date format", sl.Err(err), slog.String("date", dateStr))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid date format. Expected YYYY-MM-DD"))
			return
		}

		// Operational day starts at 7:00 local time and ends at 7:00 next day
		startDate := time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), 7, 0, 0, 0, loc)
		endDate := startDate.Add(24 * time.Hour)

		// Fetch discharge data for the operational day
		discharges, err := dischargeGetter.GetAllDischarges(r.Context(), nil, &startDate, &endDate)
		if err != nil {
			log.Error("failed to fetch discharge data", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to fetch discharge data"))
			return
		}

		// Fetch shutdown data for the operational day
		shutdowns, err := shutdownGetter.GetShutdowns(r.Context(), startDate)
		if err != nil {
			log.Error("failed to fetch shutdown data", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to fetch shutdown data"))
			return
		}

		// Fetch organization types map
		orgTypesMap, err := orgTypesGetter.GetOrganizationTypesMap(r.Context())
		if err != nil {
			log.Error("failed to fetch organization types", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to fetch organization types"))
			return
		}

		// Group shutdowns by organization type (ges/mini/micro)
		groupedShutdowns := groupShutdownsByType(shutdowns, orgTypesMap)

		// Fetch visit data for the operational day
		visits, err := visitGetter.GetVisits(r.Context(), startDate)
		if err != nil {
			log.Error("failed to fetch visit data", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to fetch visit data"))
			return
		}

		// Fetch incident data for the operational day
		incidents, err := incidentGetter.GetIncidents(r.Context(), startDate)
		if err != nil {
			log.Error("failed to fetch incident data", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to fetch incident data"))
			return
		}

		// Generate Excel file
		excelFile, err := generator.GenerateExcel(startDate, endDate, discharges, groupedShutdowns, visits, incidents, loc)
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
			filename := fmt.Sprintf("SC-%s.xlsx", parsedDate.Format("2006-01-02"))
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
				slog.Int("discharge_count", len(discharges)),
			)
		} else {
			// PDF export
			if err := exportPDF(w, excelFile, parsedDate, log); err != nil {
				log.Error("failed to export PDF", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to convert to PDF"))
				return
			}

			log.Info("successfully generated PDF export",
				slog.String("date", dateStr),
				slog.Int("discharge_count", len(discharges)),
			)
		}
	}
}

// exportPDF converts Excel to PDF using LibreOffice and sends to client
func exportPDF(w http.ResponseWriter, excelFile *excelize.File, parsedDate time.Time, log *slog.Logger) error {
	// Create temporary directory for conversion
	tempDir, err := os.MkdirTemp("", "sc-pdf-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Add page margins to Excel file for PDF conversion
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
		return fmt.Errorf("failed to set page margins: %w", err)
	}

	// Save Excel file temporarily with margins
	excelPath := filepath.Join(tempDir, "sc.xlsx")
	if err := excelFile.SaveAs(excelPath); err != nil {
		return fmt.Errorf("failed to save Excel file: %w", err)
	}

	// Convert Excel to PDF using LibreOffice
	pdfPath := filepath.Join(tempDir, "sc.pdf")

	cmd := exec.Command(
		"soffice",
		"--headless",
		"--convert-to", "pdf",
		"--outdir", tempDir,
		excelPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("LibreOffice conversion failed",
			slog.String("output", string(output)),
		)
		return fmt.Errorf("failed to convert Excel to PDF: %w", err)
	}

	// Read PDF file
	pdfData, err := os.ReadFile(pdfPath)
	if err != nil {
		return fmt.Errorf("failed to read PDF file: %w", err)
	}

	// Set headers for PDF download
	filename := fmt.Sprintf("SC-%s.pdf", parsedDate.Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfData)))

	// Stream file to client
	if _, err := w.Write(pdfData); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	return nil
}

// groupShutdownsByType groups shutdowns by organization type (ges, mini, micro)
func groupShutdownsByType(shutdowns []*shutdown.ResponseModel, orgTypesMap map[int64][]string) *scgen.GroupedShutdowns {
	result := &scgen.GroupedShutdowns{
		Ges:   make([]*shutdown.ResponseModel, 0),
		Mini:  make([]*shutdown.ResponseModel, 0),
		Micro: make([]*shutdown.ResponseModel, 0),
	}

	for _, s := range shutdowns {
		types, ok := orgTypesMap[s.OrganizationID]
		if !ok {
			continue
		}

		// Determine the type of organization
		orgType := determineOrgType(types)
		switch orgType {
		case "ges":
			result.Ges = append(result.Ges, s)
		case "mini":
			result.Mini = append(result.Mini, s)
		case "micro":
			result.Micro = append(result.Micro, s)
		}
	}

	return result
}

// determineOrgType determines the organization type from a list of types
// Priority: micro > mini > ges (more specific wins)
func determineOrgType(types []string) string {
	for _, t := range types {
		if t == "micro" {
			return "micro"
		}
	}
	for _, t := range types {
		if t == "mini" {
			return "mini"
		}
	}
	for _, t := range types {
		if t == "ges" {
			return "ges"
		}
	}
	return ""
}
