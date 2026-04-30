package gesreport

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

		format := r.URL.Query().Get("format")
		if format == "" {
			format = "excel"
		}
		if format != "excel" && format != "pdf" {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid format, expected 'excel' or 'pdf'"))
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

		if format == "excel" {
			writeOwnNeedsExcel(w, excelFile, parsedDate, log)
			return
		}
		if err := exportOwnNeedsPDF(w, excelFile, parsedDate, log); err != nil {
			log.Error("failed to export PDF", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to convert to PDF"))
		}
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

// exportOwnNeedsPDF converts the workbook to PDF via headless LibreOffice
// and writes it to the response. The conversion path mirrors exportPDFGes
// from export.go — the two helpers are kept separate because they differ
// in print-titles range, page layout (own-needs forces landscape +
// fit-to-width), and filename. See plan §1.
func exportOwnNeedsPDF(w http.ResponseWriter, f *excelize.File, date time.Time, log *slog.Logger) error {
	tempDir, err := os.MkdirTemp("", "own-needs-pdf-*")
	if err != nil {
		return fmt.Errorf("create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	sheet := f.GetSheetName(0)

	marginTop := 0.3
	marginBottom := 0.3
	marginLeft := 0.3
	marginRight := 0.3
	marginHeader := 0.1
	marginFooter := 0.0
	if err := f.SetPageMargins(sheet, &excelize.PageLayoutMarginsOptions{
		Top:    &marginTop,
		Bottom: &marginBottom,
		Left:   &marginLeft,
		Right:  &marginRight,
		Header: &marginHeader,
		Footer: &marginFooter,
	}); err != nil {
		return fmt.Errorf("set page margins: %w", err)
	}

	// own-needs has 16 columns (A..P); portrait clips. Force landscape +
	// fit-to-width so LibreOffice scales the body to one printable width.
	// We do not rely on the template's own PageSetup because the user could
	// re-save the xlsx in Excel and silently flip orientation.
	orientation := "landscape"
	fit := 1
	if err := f.SetPageLayout(sheet, &excelize.PageLayoutOptions{
		Orientation: &orientation,
		FitToWidth:  &fit,
		FitToHeight: &fit,
	}); err != nil {
		return fmt.Errorf("set page layout: %w", err)
	}

	if err := setOwnNeedsPDFPrintTitles(f, sheet); err != nil {
		return fmt.Errorf("set print titles: %w", err)
	}

	excelPath := filepath.Join(tempDir, "own-needs.xlsx")
	if err := f.SaveAs(excelPath); err != nil {
		return fmt.Errorf("save Excel file: %w", err)
	}
	pdfPath := filepath.Join(tempDir, "own-needs.pdf")

	cmd := exec.Command(
		"soffice",
		"--headless",
		"--language=ru-RU",
		"--convert-to", "pdf",
		"--outdir", tempDir,
		excelPath,
	)
	cmd.Env = append(os.Environ(),
		"LANG=ru_RU.UTF-8",
		"LC_ALL=ru_RU.UTF-8",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("LibreOffice conversion failed", slog.String("output", string(output)))
		return fmt.Errorf("convert Excel to PDF: %w", err)
	}

	pdfData, err := os.ReadFile(pdfPath)
	if err != nil {
		return fmt.Errorf("read PDF file: %w", err)
	}

	filename := fmt.Sprintf("Own-Needs-%s.pdf", date.Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfData)))

	if _, err := w.Write(pdfData); err != nil {
		return fmt.Errorf("write response: %w", err)
	}

	log.Info("generated own-needs PDF export",
		slog.String("date", date.Format("2006-01-02")),
		slog.String("filename", filename),
		slog.Int("file_size", len(pdfData)),
	)
	return nil
}

// setOwnNeedsPDFPrintTitles installs Print_Titles for rows 1..5 (the
// own-needs header block) so they repeat on every printed page. The
// template carries Print_Area / _FilterDatabase / Print_Titles defined
// names scoped to the original sheet. After SetSheetName excelize
// rewrites each name's Scope but leaves RefersTo pointing at the old
// sheet — LibreOffice silently drops them and the narrow Print_Area
// clips the PDF to one page. This helper drops those stale entries
// before writing a fresh Print_Titles. Print_Area is intentionally
// not reinstalled so LibreOffice paginates the whole body.
//
// Mirrors setPDFPrintTitles in export.go; the only difference is the
// row range ($1:$5 vs $1:$6).
func setOwnNeedsPDFPrintTitles(f *excelize.File, sheet string) error {
	for _, name := range []string{"_xlnm._FilterDatabase", "_xlnm.Print_Titles", "_xlnm.Print_Area"} {
		err := f.DeleteDefinedName(&excelize.DefinedName{Name: name, Scope: sheet})
		if err != nil && !errors.Is(err, excelize.ErrDefinedNameScope) {
			return fmt.Errorf("delete %s: %w", name, err)
		}
	}
	return f.SetDefinedName(&excelize.DefinedName{
		Name:     "_xlnm.Print_Titles",
		RefersTo: fmt.Sprintf("'%s'!$1:$5", sheet),
		Scope:    sheet,
	})
}
