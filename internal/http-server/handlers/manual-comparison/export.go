package manualcomparison

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	resp "srmt-admin/internal/lib/api/response"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/lib/logger/sl"
	reservoirsummarymodel "srmt-admin/internal/lib/model/reservoir-summary"
	filtergen "srmt-admin/internal/lib/service/excel/filter"
	resSummaryGen "srmt-admin/internal/lib/service/excel/reservoir-summary"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/xuri/excelize/v2"
)

type ExportReservoirSummaryGetter interface {
	GetReservoirSummary(ctx context.Context, date string) ([]*reservoirsummarymodel.ResponseModel, error)
}

func Export(
	log *slog.Logger,
	summaryGetter ExportReservoirSummaryGetter,
	mcGetter ManualComparisonDataGetter,
	summaryGen *resSummaryGen.Generator,
	filterGen *filtergen.Generator,
	loc *time.Location,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.manual-comparison.Export"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

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

		var authorShort string
		if claims, ok := mwauth.ClaimsFromContext(r.Context()); ok {
			authorShort = shortenName(claims.Name)
		}

		// Section 1: reservoir summary
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

		// Section 2: manual comparison filtration blocks
		yesterday := parsedDate.AddDate(0, 0, -1).Format("2006-01-02")

		orgIDs, err := mcGetter.GetFiltrationOrgIDs(r.Context())
		if err != nil {
			log.Error("failed to get filtration org IDs", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve organizations"))
			return
		}

		comparisons, err := buildAllComparisons(r.Context(), mcGetter, orgIDs, yesterday)
		if err != nil {
			log.Error("failed to build manual comparisons", sl.Err(err))
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

func exportExcel(w http.ResponseWriter, f *excelize.File, date time.Time, log *slog.Logger) {
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		log.Error("failed to write Excel to buffer", sl.Err(err))
		return
	}

	filename := fmt.Sprintf("ManualComparison-%s.xlsx", date.Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", buf.Len()))
	_, _ = w.Write(buf.Bytes())
}

func exportPDF(w http.ResponseWriter, excelFile *excelize.File, date time.Time, log *slog.Logger) error {
	tempDir, err := os.MkdirTemp("", "mc-pdf-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	sheet := excelFile.GetSheetName(0)

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

	orientation := "portrait"
	fitToHeight := 1
	fitToWidth := 1
	_ = excelFile.SetPageLayout(sheet, &excelize.PageLayoutOptions{
		Orientation: &orientation,
		FitToHeight: &fitToHeight,
		FitToWidth:  &fitToWidth,
	})

	excelPath := filepath.Join(tempDir, "mc.xlsx")
	if err := excelFile.SaveAs(excelPath); err != nil {
		return fmt.Errorf("failed to save Excel file: %w", err)
	}

	pdfPath := filepath.Join(tempDir, "mc.pdf")
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

	filename := fmt.Sprintf("ManualComparison-%s.pdf", date.Format("2006-01-02"))
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
	return fmt.Sprintf("%c. %s", firstRune, parts[0])
}
