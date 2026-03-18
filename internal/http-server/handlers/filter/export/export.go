package export

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/filtration"
	reservoirsummarymodel "srmt-admin/internal/lib/model/reservoir-summary"
	filtergen "srmt-admin/internal/lib/service/excel/filter"
	resSummaryGen "srmt-admin/internal/lib/service/excel/reservoir-summary"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/xuri/excelize/v2"
)

type ReservoirSummaryGetter interface {
	GetReservoirSummary(ctx context.Context, date string) ([]*reservoirsummarymodel.ResponseModel, error)
}

type FiltrationComparisonGetter interface {
	GetFiltrationOrgIDs(ctx context.Context) ([]int64, error)
	GetOrgFiltrationSummary(ctx context.Context, orgID int64, date string) (*filtration.OrgFiltrationSummary, error)
	GetReservoirLevelVolume(ctx context.Context, orgID int64, date string) (*float64, *float64, error)
	GetClosestLevelDate(ctx context.Context, orgID int64, level float64, excludeDate string) (string, error)
}

func New(
	log *slog.Logger,
	summaryGetter ReservoirSummaryGetter,
	filtrationGetter FiltrationComparisonGetter,
	summaryGen *resSummaryGen.Generator,
	filterGen *filtergen.Generator,
	loc *time.Location,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.filter.export.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// Parse date
		dateStr := r.URL.Query().Get("date")
		if dateStr == "" {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Missing required 'date' parameter (format: YYYY-MM-DD)"))
			return
		}
		parsedDate, err := time.ParseInLocation("2006-01-02", dateStr, loc)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid date format. Expected YYYY-MM-DD"))
			return
		}

		format := r.URL.Query().Get("format")
		if format == "" {
			format = "excel"
		}
		if format != "excel" && format != "pdf" {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid format parameter. Expected 'excel' or 'pdf'"))
			return
		}

		// Get author short name
		var authorShort string
		if claims, ok := mwauth.ClaimsFromContext(r.Context()); ok {
			authorShort = shortenName(claims.Name)
		}

		// Section 1: reservoir summary via existing generator
		summaries, err := summaryGetter.GetReservoirSummary(r.Context(), dateStr)
		if err != nil {
			log.Error("failed to fetch reservoir summary", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to fetch reservoir summary"))
			return
		}

		excelFile, err := summaryGen.GenerateExcel(dateStr, summaries, "")
		if err != nil {
			log.Error("failed to generate summary", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to generate summary"))
			return
		}
		defer excelFile.Close()

		// Section 2: filtration/piezometer blocks
		// Filtration data uses previous day
		yesterday := parsedDate.AddDate(0, 0, -1).Format("2006-01-02")
		comparisons, err := buildComparisons(r.Context(), filtrationGetter, yesterday)
		if err != nil {
			log.Error("failed to build filtration comparisons", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to build filtration data"))
			return
		}

		sheet := excelFile.GetSheetName(0)
		if err := filterGen.FillFiltrationBlocks(excelFile, sheet, comparisons, authorShort); err != nil {
			log.Error("failed to fill filtration blocks", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to fill filtration blocks"))
			return
		}

		// Export
		if format == "excel" {
			exportExcel(w, excelFile, parsedDate, log)
		} else {
			if err := exportPDF(w, excelFile, parsedDate, log); err != nil {
				log.Error("failed to export PDF", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to convert to PDF"))
			}
		}
	}
}

func buildComparisons(ctx context.Context, getter FiltrationComparisonGetter, date string) ([]filtration.OrgComparison, error) {
	orgIDs, err := getter.GetFiltrationOrgIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("get filtration org IDs: %w", err)
	}

	result := make([]filtration.OrgComparison, 0, len(orgIDs))
	for _, orgID := range orgIDs {
		comp, err := buildOrgComparison(ctx, getter, orgID, date)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				continue
			}
			return nil, fmt.Errorf("build comparison for org %d: %w", orgID, err)
		}
		result = append(result, *comp)
	}
	return result, nil
}

func buildOrgComparison(ctx context.Context, getter FiltrationComparisonGetter, orgID int64, date string) (*filtration.OrgComparison, error) {
	summary, err := getter.GetOrgFiltrationSummary(ctx, orgID, date)
	if err != nil {
		return nil, err
	}

	level, volume, err := getter.GetReservoirLevelVolume(ctx, orgID, date)
	if err != nil {
		return nil, err
	}

	comparison := &filtration.OrgComparison{
		OrganizationID:   summary.OrganizationID,
		OrganizationName: summary.OrganizationName,
		Current: filtration.ComparisonSnapshot{
			Date:        date,
			Level:       level,
			Volume:      volume,
			Locations:   summary.Locations,
			Piezometers: summary.Piezometers,
			PiezoCounts: summary.PiezoCounts,
		},
	}

	if level != nil {
		histDate, err := getter.GetClosestLevelDate(ctx, orgID, *level, date)
		if err != nil {
			if !errors.Is(err, storage.ErrNotFound) {
				return nil, err
			}
			return comparison, nil
		}

		histSummary, err := getter.GetOrgFiltrationSummary(ctx, orgID, histDate)
		if err != nil {
			if !errors.Is(err, storage.ErrNotFound) {
				return nil, err
			}
			return comparison, nil
		}

		histLevel, histVolume, err := getter.GetReservoirLevelVolume(ctx, orgID, histDate)
		if err != nil {
			return nil, err
		}

		comparison.Historical = &filtration.ComparisonSnapshot{
			Date:        histDate,
			Level:       histLevel,
			Volume:      histVolume,
			Locations:   histSummary.Locations,
			Piezometers: histSummary.Piezometers,
			PiezoCounts: summary.PiezoCounts,
		}
	}

	return comparison, nil
}

func exportExcel(w http.ResponseWriter, f *excelize.File, date time.Time, log *slog.Logger) {
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		log.Error("failed to write Excel to buffer", sl.Err(err))
		render.Status(nil, http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("Filter-%s.xlsx", date.Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", buf.Len()))
	_, _ = w.Write(buf.Bytes())
}

func exportPDF(w http.ResponseWriter, excelFile *excelize.File, date time.Time, log *slog.Logger) error {
	tempDir, err := os.MkdirTemp("", "filter-pdf-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	sheet := excelFile.GetSheetName(0)

	// Minimal margins
	marginTop := 0.2
	marginBottom := 0.2
	marginLeft := 0.1
	marginRight := 0.1
	marginHeader := 0.0
	marginFooter := 0.0
	_ = excelFile.SetPageMargins(sheet, &excelize.PageLayoutMarginsOptions{
		Top:    &marginTop,
		Bottom: &marginBottom,
		Left:   &marginLeft,
		Right:  &marginRight,
		Header: &marginHeader,
		Footer: &marginFooter,
	})

	// Portrait orientation, fit everything on one page
	orientation := "portrait"
	fitToHeight := 1
	fitToWidth := 1
	_ = excelFile.SetPageLayout(sheet, &excelize.PageLayoutOptions{
		Orientation:  &orientation,
		FitToHeight:  &fitToHeight,
		FitToWidth:   &fitToWidth,
	})

	excelPath := filepath.Join(tempDir, "filter.xlsx")
	if err := excelFile.SaveAs(excelPath); err != nil {
		return fmt.Errorf("failed to save Excel file: %w", err)
	}

	pdfPath := filepath.Join(tempDir, "filter.pdf")
	cmd := exec.Command("soffice", "--headless", "--convert-to", "pdf", "--outdir", tempDir, excelPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("LibreOffice conversion failed", slog.String("output", string(output)))
		return fmt.Errorf("failed to convert Excel to PDF: %w", err)
	}

	pdfData, err := os.ReadFile(pdfPath)
	if err != nil {
		return fmt.Errorf("failed to read PDF file: %w", err)
	}

	filename := fmt.Sprintf("Filter-%s.pdf", date.Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfData)))
	_, _ = w.Write(pdfData)
	return nil
}

func shortenName(fullName string) string {
	parts := strings.Fields(fullName)
	if len(parts) < 2 {
		return fullName
	}
	firstRune := []rune(parts[1])[0]
	return fmt.Sprintf("%s %c.", parts[0], firstRune)
}
