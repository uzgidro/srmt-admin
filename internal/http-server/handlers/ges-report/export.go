package gesreport

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
	model "srmt-admin/internal/lib/model/ges-report"
	gesgen "srmt-admin/internal/lib/service/excel/ges"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/xuri/excelize/v2"
)

// ExportPlanGetter fetches production plans for a given year and set of months.
type ExportPlanGetter interface {
	GetGESPlansForReport(ctx context.Context, year int, months []int) ([]model.PlanRow, error)
}

// ExportOrgTypesGetter fetches the organization-type mapping for all orgs.
type ExportOrgTypesGetter interface {
	GetOrganizationTypesMap(ctx context.Context) (map[int64][]string, error)
}

// Export returns an HTTP handler that generates and downloads a GES daily
// report in Excel or PDF format.
func Export(
	log *slog.Logger,
	reportSvc ReportBuilder,
	planGetter ExportPlanGetter,
	orgTypesGetter ExportOrgTypesGetter,
	generator *gesgen.Generator,
	loc *time.Location,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.ges-report.Export"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// --- parse query params ---

		dateStr := r.URL.Query().Get("date")
		if dateStr == "" {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("date is required (YYYY-MM-DD)"))
			return
		}
		parsedDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid date format, expected YYYY-MM-DD"))
			return
		}

		format := r.URL.Query().Get("format")
		if format == "" {
			format = "excel"
		}
		if format != "excel" && format != "pdf" {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid format, expected 'excel' or 'pdf'"))
			return
		}

		// --- build report ---

		// Export is restricted to sc/rais by route middleware, so no cascade filter is applied.
		// Repair / Modernization / Reserve aggregate counts come from the report itself
		// (report.GrandTotal.*Aggregates), populated by the service from ges_daily_data
		// and clamped to ≥0. The DB CHECK constraint guarantees working+repair+mod ≤ total,
		// so no separate "reserve >= 0" handler-level validation is required.
		report, err := reportSvc.BuildDailyReport(r.Context(), dateStr, nil)
		if err != nil {
			log.Error("failed to build daily report", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to build report"))
			return
		}

		// --- fetch plans ---

		year := parsedDate.Year()
		currentMonth := int(parsedDate.Month())
		allMonths := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

		planRows, err := planGetter.GetGESPlansForReport(r.Context(), year, allMonths)
		if err != nil {
			log.Error("failed to fetch plans", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to fetch production plans"))
			return
		}

		ytdPlans := make(map[int64]float64)
		annualPlans := make(map[int64]float64)
		monthlyPlans := make(map[int64]float64)

		for _, row := range planRows {
			annualPlans[row.OrganizationID] += row.PlanMlnKWh
			if row.Month <= currentMonth {
				ytdPlans[row.OrganizationID] += row.PlanMlnKWh
			}
			if row.Month == currentMonth {
				monthlyPlans[row.OrganizationID] = row.PlanMlnKWh
			}
		}

		// --- fetch org types ---

		typesMap, err := orgTypesGetter.GetOrganizationTypesMap(r.Context())
		if err != nil {
			log.Error("failed to fetch organization types", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to fetch organization types"))
			return
		}

		var orgTypes gesgen.OrgTypeCounts
		for _, cascade := range report.Cascades {
			for _, station := range cascade.Stations {
				types := typesMap[station.OrganizationID]
				for _, t := range types {
					switch t {
					case "ges":
						orgTypes.GES++
					case "mini":
						orgTypes.Mini++
					case "micro":
						orgTypes.Micro++
					}
				}
			}
		}
		orgTypes.Total = orgTypes.GES + orgTypes.Mini + orgTypes.Micro

		// --- generate Excel ---

		excelFile, err := generator.GenerateExcel(gesgen.ExcelParams{
			Report:        report,
			Date:          parsedDate,
			Loc:           loc,
			YTDPlans:      ytdPlans,
			AnnualPlans:   annualPlans,
			MonthlyPlans:  monthlyPlans,
			OrgTypeCounts: orgTypes,
			Log:           log,
		})
		if err != nil {
			log.Error("failed to generate Excel file", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to generate Excel file"))
			return
		}
		defer excelFile.Close()

		// --- respond ---

		if format == "excel" {
			exportExcel(w, excelFile, parsedDate, log)
		} else {
			if err := exportPDFGes(w, excelFile, parsedDate, log); err != nil {
				log.Error("failed to export PDF", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("failed to convert to PDF"))
			}
		}
	}
}

// exportExcel writes the Excel file to the HTTP response.
func exportExcel(w http.ResponseWriter, f *excelize.File, date time.Time, log *slog.Logger) {
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		log.Error("failed to write Excel to buffer", sl.Err(err))
		http.Error(w, "failed to prepare Excel file", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("GES-%s.xlsx", date.Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", buf.Len()))

	if _, err := w.Write(buf.Bytes()); err != nil {
		log.Error("failed to write response", sl.Err(err))
	}

	log.Info("generated GES Excel export",
		slog.String("date", date.Format("2006-01-02")),
		slog.String("filename", filename),
		slog.Int("file_size", buf.Len()),
	)
}

// exportPDFGes converts the Excel file to PDF using LibreOffice and sends it.
func exportPDFGes(w http.ResponseWriter, f *excelize.File, date time.Time, log *slog.Logger) error {
	tempDir, err := os.MkdirTemp("", "ges-pdf-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	sheet := f.GetSheetName(0)
	marginTop := 0.3
	marginBottom := 0.3
	marginLeft := 0.3
	marginRight := 0.3
	marginHeader := 0.1
	marginFooter := 0.0

	if err := f.SetPageMargins(sheet,
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

	excelPath := filepath.Join(tempDir, "ges.xlsx")
	if err := f.SaveAs(excelPath); err != nil {
		return fmt.Errorf("failed to save Excel file: %w", err)
	}

	pdfPath := filepath.Join(tempDir, "ges.pdf")

	cmd := exec.Command(
		"soffice",
		"--headless",
		"--convert-to", "pdf",
		"--outdir", tempDir,
		excelPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("LibreOffice conversion failed", slog.String("output", string(output)))
		return fmt.Errorf("failed to convert Excel to PDF: %w", err)
	}

	pdfData, err := os.ReadFile(pdfPath)
	if err != nil {
		return fmt.Errorf("failed to read PDF file: %w", err)
	}

	filename := fmt.Sprintf("GES-%s.pdf", date.Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfData)))

	if _, err := w.Write(pdfData); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	log.Info("generated GES PDF export",
		slog.String("date", date.Format("2006-01-02")),
		slog.String("filename", filename),
	)

	return nil
}
