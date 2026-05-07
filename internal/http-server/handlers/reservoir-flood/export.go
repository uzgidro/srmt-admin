package reservoirflood

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/xuri/excelize/v2"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/auth"
	selgen "srmt-admin/internal/lib/service/excel/sel"
)

// SelReportBuilder converts repo data into a generator-ready Report.
type SelReportBuilder interface {
	BuildReport(ctx context.Context, date time.Time, hour int, authorShort string) (*selgen.Report, error)
}

// SelExcelGenerator renders the Report into an excelize workbook.
type SelExcelGenerator interface {
	GenerateExcel(report *selgen.Report) (*excelize.File, error)
}

// reportingWindowHours marks the hours during which this report is normally
// produced (vечер–утро). hour values outside this set are not rejected; the
// handler emits a single WARN log line so operators can spot accidental
// daytime runs without breaking ad-hoc/debug usage.
var reportingWindowHours = map[int]struct{}{
	21: {}, 22: {}, 23: {},
	0: {}, 1: {}, 2: {}, 3: {}, 4: {}, 5: {}, 6: {}, 7: {}, 8: {},
}

// GetExport returns an HTTP handler that produces the tezkor-maumolot report
// in Excel or PDF form.
//
// Query params:
//   - date (required, YYYY-MM-DD): report date in `loc`.
//   - hour (optional, 0..23, default 0): hour-of-day for the "current" snapshot.
//     Hours outside 21..08 are accepted but logged as a warning.
//   - format (optional, "excel"|"pdf", default "excel").
func GetExport(log *slog.Logger, builder SelReportBuilder, generator SelExcelGenerator, loc *time.Location) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.reservoirflood.GetExport"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// Parse date (default = today in loc).
		dateStr := r.URL.Query().Get("date")
		if dateStr == "" {
			dateStr = time.Now().In(loc).Format("2006-01-02")
		}
		parsedDate, err := time.ParseInLocation("2006-01-02", dateStr, loc)
		if err != nil {
			log.Warn("invalid date", sl.Err(err), slog.String("date", dateStr))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid date, expected YYYY-MM-DD"))
			return
		}

		// Parse hour (default 0; range 0..23).
		hour := 0
		if s := r.URL.Query().Get("hour"); s != "" {
			h, err := strconv.Atoi(s)
			if err != nil || h < 0 || h > 23 {
				log.Warn("invalid hour", slog.String("hour", s))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("invalid hour, expected integer in [0,23]"))
				return
			}
			hour = h
		}
		if _, ok := reportingWindowHours[hour]; !ok {
			log.Warn("hour outside reporting window 21..08", slog.Int("hour", hour))
		}

		// Parse format.
		format := r.URL.Query().Get("format")
		if format == "" {
			format = "excel"
		}
		if format != "excel" && format != "pdf" {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid format, expected 'excel' or 'pdf'"))
			return
		}

		// Author short name from JWT claims.
		var authorShort string
		if claims, ok := mwauth.ClaimsFromContext(r.Context()); ok && claims != nil {
			authorShort = auth.ShortenName(claims.Name)
		}

		report, err := builder.BuildReport(r.Context(), parsedDate, hour, authorShort)
		if err != nil {
			log.Error("build report", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to build report"))
			return
		}

		excelFile, err := generator.GenerateExcel(report)
		if err != nil {
			log.Error("generate excel", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to render workbook"))
			return
		}
		defer excelFile.Close()

		baseName := fmt.Sprintf("ТЕЗКОР-МАЪЛУМОТ-%s-%02d", parsedDate.Format("2006-01-02"), hour)

		if format == "excel" {
			var buf bytes.Buffer
			if err := excelFile.Write(&buf); err != nil {
				log.Error("write excel", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("failed to serialize workbook"))
				return
			}
			filename := baseName + ".xlsx"
			w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
			w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
			w.Header().Set("Content-Length", fmt.Sprintf("%d", buf.Len()))
			if _, err := w.Write(buf.Bytes()); err != nil {
				log.Error("response write", sl.Err(err))
			}
			return
		}

		// PDF path: convert via headless soffice.
		tempDir, err := os.MkdirTemp("", "sel-pdf-*")
		if err != nil {
			log.Error("temp dir", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to prepare PDF conversion"))
			return
		}
		defer os.RemoveAll(tempDir)

		sheet := excelFile.GetSheetName(0)
		// The template now carries empty A/U padding columns inside the
		// print_area, so we no longer need the L/R=0 hack: standard 0.3"
		// margins are fine.
		mt, mb, ml, mr, mh, mf := 0.3, 0.3, 0.3, 0.3, 0.2, 0.0
		if err := excelFile.SetPageMargins(sheet, &excelize.PageLayoutMarginsOptions{
			Top: &mt, Bottom: &mb, Left: &ml, Right: &mr, Header: &mh, Footer: &mf,
		}); err != nil {
			log.Error("page margins", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to prepare PDF conversion"))
			return
		}
		// Force fit-to-page so the wide table (19 columns) lands on a single
		// page even if the template is replaced with one that drops the flag.
		fitW, fitH := 1, 1
		if err := excelFile.SetPageLayout(sheet, &excelize.PageLayoutOptions{
			FitToWidth: &fitW, FitToHeight: &fitH,
		}); err != nil {
			log.Error("page layout", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to prepare PDF conversion"))
			return
		}

		excelPath := filepath.Join(tempDir, "sel.xlsx")
		if err := excelFile.SaveAs(excelPath); err != nil {
			log.Error("save excel", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to prepare PDF conversion"))
			return
		}

		cmd := exec.Command("soffice", "--headless", "--convert-to", "pdf", "--outdir", tempDir, excelPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Error("soffice convert", sl.Err(err), slog.String("output", string(output)))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to convert to PDF"))
			return
		}

		pdfPath := filepath.Join(tempDir, "sel.pdf")
		pdfData, err := os.ReadFile(pdfPath)
		if err != nil {
			log.Error("read pdf", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to read PDF"))
			return
		}

		filename := baseName + ".pdf"
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfData)))
		if _, err := w.Write(pdfData); err != nil {
			log.Error("response write", sl.Err(err))
		}
	}
}
