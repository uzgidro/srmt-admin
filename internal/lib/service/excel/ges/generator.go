package ges

import (
	"context"
	"fmt"
	_ "image/png"
	"io"
	"log/slog"
	"math"
	"net/http"
	"os"
	"time"

	model "srmt-admin/internal/lib/model/ges-report"
	"github.com/xuri/excelize/v2"
)

// Generator creates Excel reports from GES daily data using a template.
type Generator struct {
	templatePath string
}

// New creates a Generator with the given template path.
func New(templatePath string) *Generator {
	return &Generator{templatePath: templatePath}
}

// ExcelParams holds all data needed to generate the Excel report.
type ExcelParams struct {
	Report        *model.DailyReport
	YTDPlans      map[int64]float64
	AnnualPlans   map[int64]float64
	MonthlyPlans  map[int64]float64
	OrgTypeCounts OrgTypeCounts
	Date          time.Time
	Loc           *time.Location
	Log           *slog.Logger
}

// OrgTypeCounts holds station counts by type.
type OrgTypeCounts struct {
	GES, Mini, Micro, Total int
}

// Template layout constants.
const (
	templateCascadeRow = 7 // cascade total row in template
	templateStationRow = 8 // station row in template
	templateGrandRow   = 9 // grand total row in template
)

// iconFetcher downloads weather icon PNGs at runtime and caches the bytes
// for the lifetime of the fetcher (per-request scope). It is not safe for
// concurrent use; reports are generated sequentially in a single goroutine.
type iconFetcher struct {
	httpClient *http.Client
	cache      map[string][]byte
	urlFn      func(code string) string
}

// newIconFetcher constructs a fetcher with the given HTTP timeout and URL
// builder. The urlFn lets tests redirect requests to httptest.NewServer.
func newIconFetcher(timeout time.Duration, urlFn func(code string) string) *iconFetcher {
	return &iconFetcher{
		httpClient: &http.Client{Timeout: timeout},
		cache:      make(map[string][]byte, 4),
		urlFn:      urlFn,
	}
}

// newDefaultIconFetcher returns a fetcher pointed at openweathermap.org with
// a 5s per-request timeout. Used in production by GenerateExcel.
func newDefaultIconFetcher() *iconFetcher {
	return newIconFetcher(5*time.Second, func(code string) string {
		return "https://openweathermap.org/payload/api/media/file/" + code + ".png"
	})
}

// Get returns the icon bytes for the given OpenWeatherMap condition code.
// Subsequent calls for the same code are served from the in-memory cache.
func (f *iconFetcher) Get(ctx context.Context, code string) ([]byte, error) {
	if b, ok := f.cache[code]; ok {
		return b, nil
	}
	url := f.urlFn(code)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build icon request: %w", err)
	}
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch icon: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch icon %q: status %d", code, resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read icon body: %w", err)
	}
	f.cache[code] = b
	return b, nil
}

// weatherIconLog emits a warning about an icon problem. Falls back to stderr
// when no logger is wired (e.g. in unit tests).
func weatherIconLog(log *slog.Logger, msg string, kvs ...any) {
	if log != nil {
		log.Warn(msg, kvs...)
		return
	}
	fmt.Fprintf(os.Stderr, "%s %v\n", msg, kvs)
}

// GenerateExcel produces an Excel file from the template and params.
func (g *Generator) GenerateExcel(params ExcelParams) (*excelize.File, error) {
	f, err := excelize.OpenFile(g.templatePath)
	if err != nil {
		return nil, fmt.Errorf("open template: %w", err)
	}

	fetcher := newDefaultIconFetcher()

	oldSheet := f.GetSheetList()[0]

	// Phase 1: structural preparation
	newSheet := params.Date.Format("02.01.06")
	if err := f.SetSheetName(oldSheet, newSheet); err != nil {
		return nil, fmt.Errorf("rename sheet: %w", err)
	}

	// Set AH3 = date + 1 day
	nextDay := params.Date.AddDate(0, 0, 1)
	if err := f.SetCellValue(newSheet, "AH3", nextDay); err != nil {
		return nil, fmt.Errorf("set AH3: %w", err)
	}

	cascades := params.Report.Cascades
	n := len(cascades)
	if n == 0 {
		return f, nil
	}

	// Phase 1a: duplicate the 2-row block (cascade+station) for each cascade.
	// Template rows 7-8 form one block. DuplicateRowTo copies each row of
	// the source block to contiguous target positions, preserving formulas.
	// Source rows are always above insertion points, so they never shift.
	blockSize := 2
	for i := 1; i < n; i++ {
		targetBase := templateCascadeRow + i*blockSize
		for j := 0; j < blockSize; j++ {
			if err := f.DuplicateRowTo(newSheet, templateCascadeRow+j, targetBase+j); err != nil {
				return nil, fmt.Errorf("duplicate block %d row %d: %w", i, j, err)
			}
		}
	}

	// Phase 1b: within each block, duplicate the station row for extra stations.
	// Process forward, tracking cumulative offset from inserted rows.
	offset := 0
	for i, cascade := range cascades {
		stationCount := len(cascade.Stations)
		if stationCount <= 1 {
			continue
		}
		// Current station template row for this cascade's block
		stationRow := templateCascadeRow + i*blockSize + 1 + offset
		for j := 1; j < stationCount; j++ {
			if err := f.DuplicateRow(newSheet, stationRow); err != nil {
				return nil, fmt.Errorf("duplicate station row cascade %d: %w", i, err)
			}
		}
		offset += stationCount - 1
	}

	// Phase 2: fill data
	row := templateCascadeRow
	for _, cascade := range cascades {
		// Cascade total row — set formulas for X, Y, AJ, AK
		// (DuplicateRowTo does not copy shared formulas)
		setCascadeFormulas(f, newSheet, row)
		fillCascadeRow(f, newSheet, row, cascade, params)
		row++

		// Station rows
		stationStart := row
		for _, station := range cascade.Stations {
			fillStationRow(f, newSheet, row, station, params)
			row++
		}

		// Weather: merge station cells in D and Z, split into temp + icon
		if w := cascade.Weather; w != nil && len(cascade.Stations) > 0 {
			fillWeatherCells(f, newSheet, stationStart, len(cascade.Stations),
				w.Temperature, w.Condition, "D",
				fetcher, params.Log, cascade.CascadeID)
			fillWeatherCells(f, newSheet, stationStart, len(cascade.Stations),
				w.PrevYearTemperature, w.PrevYearCondition, "Z",
				fetcher, params.Log, cascade.CascadeID)
		}
	}

	// Grand total row
	grandRow := row
	setCascadeFormulas(f, newSheet, grandRow)
	fillGrandTotalRow(f, newSheet, grandRow, params.Report.GrandTotal, params.Report, params)

	// Forecast rows (originally rows 10-13, now shifted)
	forecastRow := grandRow + 1
	fillForecasts(f, newSheet, forecastRow, grandRow, params)

	// Aggregate rows start at forecastRow+4 (same row as Бажарилди in T column)
	// In template: row 14 has both Бажарилди (T14) and Умумий ГЭСлар (A14:C14, E14)
	aggRow := forecastRow + 4
	fillAggregates(f, newSheet, aggRow, grandRow, params)

	_ = f.UpdateLinkedValue()

	return f, nil
}

// fillWeatherCells merges station rows in a column and splits them into
// temperature (upper half) and embedded PNG icon (lower half).
// If odd station count, the smaller half goes to temperature (top).
// The icon bytes are fetched at runtime via the supplied fetcher; failures
// are logged as warnings and the icon is skipped without aborting export.
func fillWeatherCells(f *excelize.File, sheet string, startRow, stationCount int,
	temperature *float64, conditionCode *string, col string,
	fetcher *iconFetcher, log *slog.Logger, cascadeID int64) {
	if temperature == nil && conditionCode == nil {
		return
	}
	if stationCount <= 0 {
		return
	}

	if stationCount == 1 {
		// Single station: temperature only, no icon
		if temperature != nil {
			_ = f.SetCellValue(sheet, cell(col, startRow), fmt.Sprintf("%.0f°С", math.Round(*temperature)))
		}
		return
	}

	// Split: smaller half on top (temperature), larger on bottom (icon)
	tempRows := stationCount / 2
	iconRows := stationCount - tempRows

	// Upper block: temperature
	if temperature != nil {
		topStart := cell(col, startRow)
		topEnd := cell(col, startRow+tempRows-1)
		if tempRows > 1 {
			_ = f.MergeCell(sheet, topStart, topEnd)
		}
		_ = f.SetCellValue(sheet, topStart, fmt.Sprintf("%.0f°С", math.Round(*temperature)))
	}

	// Lower block: embedded PNG icon, centered in merge area
	if conditionCode != nil && iconRows > 0 && fetcher != nil {
		iconStart := startRow + tempRows
		botStart := cell(col, iconStart)
		botEnd := cell(col, iconStart+iconRows-1)
		if iconRows > 1 {
			_ = f.MergeCell(sheet, botStart, botEnd)
		}

		// Dashed top border on icon block = separator between temp and icon
		if temperature != nil {
			styleID, _ := f.GetCellStyle(sheet, botStart)
			existing, _ := f.GetStyle(styleID)
			dashedTop := excelize.Border{Type: "top", Color: "000000", Style: 3}
			newStyle := &excelize.Style{
				Border: []excelize.Border{dashedTop},
			}
			if existing != nil {
				// Preserve existing borders (right, left, bottom), add dashed top
				for _, b := range existing.Border {
					if b.Type != "top" {
						newStyle.Border = append(newStyle.Border, b)
					}
				}
				newStyle.Fill = existing.Fill
				newStyle.Font = existing.Font
				newStyle.Alignment = existing.Alignment
				newStyle.NumFmt = existing.NumFmt
			}
			if sid, err := f.NewStyle(newStyle); err == nil {
				_ = f.SetCellStyle(sheet, botStart, botStart, sid)
			}
		}

		// Calculate offsets to center the icon.
		// OWM icons are 100x100px, scaled to 50x50px (scale 0.5).
		const iconPx = 50.0

		// Column width in characters → pixels (~7px per character unit)
		colWidth, _ := f.GetColWidth(sheet, col)
		colPx := colWidth * 7.0
		offsetX := int((colPx - iconPx) / 2)
		if offsetX < 0 {
			offsetX = 0
		}

		// Row heights in points → pixels (~1.33px per point)
		var totalHeightPx float64
		for r := iconStart; r < iconStart+iconRows; r++ {
			h, _ := f.GetRowHeight(sheet, r)
			totalHeightPx += h * 1.33
		}
		offsetY := int((totalHeightPx - iconPx) / 2)
		if offsetY < 0 {
			offsetY = 0
		}

		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()
		iconBytes, err := fetcher.Get(ctx, *conditionCode)
		if err != nil {
			weatherIconLog(log, "weather icon fetch failed",
				"code", *conditionCode, "cascade_id", cascadeID, "cell", botStart, "err", err)
		} else if err := f.AddPictureFromBytes(sheet, botStart, &excelize.Picture{
			Extension: ".png",
			File:      iconBytes,
			Format: &excelize.GraphicOptions{
				ScaleX:      0.5,
				ScaleY:      0.5,
				OffsetX:     offsetX,
				OffsetY:     offsetY,
				Positioning: "oneCell",
			},
		}); err != nil {
			weatherIconLog(log, "weather icon AddPictureFromBytes failed",
				"code", *conditionCode, "cascade_id", cascadeID, "cell", botStart, "err", err)
		}
	}
}

// setCascadeFormulas writes the template formulas for X, Y, AJ, AK into a
// cascade-total or grand-total row. DuplicateRowTo does not copy shared
// formulas, so we re-create them explicitly.
func setCascadeFormulas(f *excelize.File, sheet string, row int) {
	r := fmt.Sprintf("%d", row)
	_ = f.SetCellFormula(sheet, "X"+r, fmt.Sprintf("IFERROR(W%[1]s/C%[1]s,0)", r))
	_ = f.SetCellFormula(sheet, "Y"+r, fmt.Sprintf("W%[1]s-C%[1]s", r))
	_ = f.SetCellFormula(sheet, "AJ"+r, fmt.Sprintf(`IFERROR(W%[1]s/AI%[1]s-1,0)`, r))
	_ = f.SetCellFormula(sheet, "AK"+r, fmt.Sprintf("W%[1]s-AI%[1]s", r))
}

func setCellFloat(f *excelize.File, sheet, cell string, v *float64) {
	if v != nil {
		_ = f.SetCellValue(sheet, cell, *v)
	}
}

func setCellFloatVal(f *excelize.File, sheet, cell string, v float64) {
	_ = f.SetCellValue(sheet, cell, v)
}

func setCellInt(f *excelize.File, sheet, cell string, v int) {
	_ = f.SetCellValue(sheet, cell, v)
}

func cell(col string, row int) string {
	return fmt.Sprintf("%s%d", col, row)
}

func fillStationRow(f *excelize.File, sheet string, row int, s model.StationReport, params ExcelParams) {
	c := s.Current
	d := s.Diffs
	agg := s.Aggregations
	cfg := s.Config

	_ = f.SetCellValue(sheet, cell("A", row), s.Name)
	setCellFloatVal(f, sheet, cell("B", row), cfg.InstalledCapacityMWt)

	// C = YTD plan
	if ytd, ok := params.YTDPlans[s.OrganizationID]; ok {
		setCellFloatVal(f, sheet, cell("C", row), ytd)
	}

	// D = weather handled by fillWeatherCells (merged across station rows)

	// E-O: current water/flow data
	setCellFloat(f, sheet, cell("E", row), c.WaterLevelM)
	setCellFloat(f, sheet, cell("F", row), d.LevelChangeCm)
	setCellFloat(f, sheet, cell("G", row), c.WaterVolumeMlnM3)
	setCellFloat(f, sheet, cell("H", row), d.VolumeChangeMlnM3)
	setCellFloat(f, sheet, cell("I", row), c.WaterHeadM)
	setCellFloat(f, sheet, cell("J", row), c.ReservoirIncomeM3s)
	setCellFloat(f, sheet, cell("K", row), d.IncomeChangeM3s)
	setCellFloat(f, sheet, cell("L", row), c.TotalOutflowM3s)
	setCellFloat(f, sheet, cell("M", row), c.GESFlowM3s)
	setCellFloat(f, sheet, cell("N", row), d.GESFlowChangeM3s)
	setCellFloat(f, sheet, cell("O", row), c.IdleDischargeM3s)

	// P-Q: aggregates count
	setCellInt(f, sheet, cell("P", row), cfg.TotalAggregates)
	setCellInt(f, sheet, cell("Q", row), c.WorkingAggregates)

	// R-S: power
	setCellFloatVal(f, sheet, cell("R", row), c.PowerMWt)
	setCellFloat(f, sheet, cell("S", row), d.PowerChangeMWt)

	// T-U: daily production
	setCellFloatVal(f, sheet, cell("T", row), c.DailyProductionMlnKWh)
	setCellFloat(f, sheet, cell("U", row), d.ProductionChange)

	// V-W: MTD/YTD production
	setCellFloatVal(f, sheet, cell("V", row), agg.MTDProductionMlnKWh)
	setCellFloatVal(f, sheet, cell("W", row), agg.YTDProductionMlnKWh)

	// X-Y: formulas in template, skip
	// AJ-AK: formulas in template, skip

	// Previous year columns AA-AI (Z = weather handled by fillWeatherCells)
	if py := s.PreviousYear; py != nil {
		setCellFloat(f, sheet, cell("AA", row), py.WaterLevelM)
		setCellFloat(f, sheet, cell("AB", row), py.WaterVolumeMlnM3)
		setCellFloat(f, sheet, cell("AC", row), py.WaterHeadM)
		setCellFloat(f, sheet, cell("AD", row), py.ReservoirIncomeM3s)
		setCellFloat(f, sheet, cell("AE", row), py.GESFlowM3s)
		setCellFloat(f, sheet, cell("AF", row), py.PowerMWt)
		setCellFloat(f, sheet, cell("AG", row), py.DailyProduction)
		setCellFloatVal(f, sheet, cell("AH", row), py.MTDProduction)
		setCellFloatVal(f, sheet, cell("AI", row), py.YTDProduction)
	}
}

func fillCascadeRow(f *excelize.File, sheet string, row int, c model.CascadeReport, params ExcelParams) {
	_ = f.SetCellValue(sheet, cell("A", row), c.CascadeName)

	s := c.Summary
	if s == nil {
		return
	}

	// B: installed capacity
	setCellFloatVal(f, sheet, cell("B", row), s.InstalledCapacityMWt)

	// C: YTD plan (sum of station plans in this cascade)
	var cascadeYTDPlan float64
	for _, st := range c.Stations {
		if ytd, ok := params.YTDPlans[st.OrganizationID]; ok {
			cascadeYTDPlan += ytd
		}
	}
	setCellFloatVal(f, sheet, cell("C", row), cascadeYTDPlan)

	// P-W: aggregates through YTD production (all summed)
	setCellInt(f, sheet, cell("P", row), s.TotalAggregates)
	setCellInt(f, sheet, cell("Q", row), s.WorkingAggregates)
	setCellFloatVal(f, sheet, cell("R", row), s.PowerMWt)

	// S: power change (sum from stations, not in SummaryBlock)
	var powerChangeSum float64
	for _, st := range c.Stations {
		if st.Diffs.PowerChangeMWt != nil {
			powerChangeSum += *st.Diffs.PowerChangeMWt
		}
	}
	setCellFloatVal(f, sheet, cell("S", row), powerChangeSum)

	setCellFloatVal(f, sheet, cell("T", row), s.DailyProductionMlnKWh)
	setCellFloatVal(f, sheet, cell("U", row), s.ProductionChange)
	setCellFloatVal(f, sheet, cell("V", row), s.MTDProductionMlnKWh)
	setCellFloatVal(f, sheet, cell("W", row), s.YTDProductionMlnKWh)

	// X-Y, AJ-AK: formulas in template, skip

	// AF-AI: previous year sums (AF, AG, AH not in SummaryBlock — sum from stations)
	var prevPowerSum, prevProductionSum, prevMTDSum float64
	for _, st := range c.Stations {
		if py := st.PreviousYear; py != nil {
			if py.PowerMWt != nil {
				prevPowerSum += *py.PowerMWt
			}
			if py.DailyProduction != nil {
				prevProductionSum += *py.DailyProduction
			}
			prevMTDSum += py.MTDProduction
		}
	}
	setCellFloatVal(f, sheet, cell("AF", row), prevPowerSum)
	setCellFloatVal(f, sheet, cell("AG", row), prevProductionSum)
	setCellFloatVal(f, sheet, cell("AH", row), prevMTDSum)
	setCellFloatVal(f, sheet, cell("AI", row), s.PrevYearYTD)
}

func fillGrandTotalRow(f *excelize.File, sheet string, row int, gt *model.SummaryBlock, report *model.DailyReport, params ExcelParams) {
	if gt == nil {
		return
	}

	// B: installed capacity
	setCellFloatVal(f, sheet, cell("B", row), gt.InstalledCapacityMWt)

	// C: YTD plan total (sum across all stations)
	var ytdPlanTotal float64
	for _, v := range params.YTDPlans {
		ytdPlanTotal += v
	}
	setCellFloatVal(f, sheet, cell("C", row), ytdPlanTotal)

	// P-W: aggregates through YTD production
	setCellInt(f, sheet, cell("P", row), gt.TotalAggregates)
	setCellInt(f, sheet, cell("Q", row), gt.WorkingAggregates)
	setCellFloatVal(f, sheet, cell("R", row), gt.PowerMWt)

	// S: power change total (sum from all stations)
	var powerChangeTotal float64
	for _, cascade := range report.Cascades {
		for _, st := range cascade.Stations {
			if st.Diffs.PowerChangeMWt != nil {
				powerChangeTotal += *st.Diffs.PowerChangeMWt
			}
		}
	}
	setCellFloatVal(f, sheet, cell("S", row), powerChangeTotal)

	setCellFloatVal(f, sheet, cell("T", row), gt.DailyProductionMlnKWh)
	setCellFloatVal(f, sheet, cell("U", row), gt.ProductionChange)
	setCellFloatVal(f, sheet, cell("V", row), gt.MTDProductionMlnKWh)
	setCellFloatVal(f, sheet, cell("W", row), gt.YTDProductionMlnKWh)

	// X-Y, AJ-AK: formulas in template, skip

	// AF-AI: previous year sums
	var prevPowerTotal, prevProductionTotal, prevMTDTotal float64
	for _, cascade := range report.Cascades {
		for _, st := range cascade.Stations {
			if py := st.PreviousYear; py != nil {
				if py.PowerMWt != nil {
					prevPowerTotal += *py.PowerMWt
				}
				if py.DailyProduction != nil {
					prevProductionTotal += *py.DailyProduction
				}
				prevMTDTotal += py.MTDProduction
			}
		}
	}
	setCellFloatVal(f, sheet, cell("AF", row), prevPowerTotal)
	setCellFloatVal(f, sheet, cell("AG", row), prevProductionTotal)
	setCellFloatVal(f, sheet, cell("AH", row), prevMTDTotal)
	setCellFloatVal(f, sheet, cell("AI", row), gt.PrevYearYTD)
}

func fillForecasts(f *excelize.File, sheet string, row int, grandRow int, params ExcelParams) {
	// Row 0: annual plan total
	var annualTotal float64
	for _, v := range params.AnnualPlans {
		annualTotal += v
	}
	setCellFloatVal(f, sheet, cell("T", row), annualTotal)

	// Row 1: monthly plan total
	var monthlyTotal float64
	for _, v := range params.MonthlyPlans {
		monthlyTotal += v
	}
	setCellFloatVal(f, sheet, cell("T", row+1), monthlyTotal)

	// Row 2: daily forecast
	if params.Report.GrandTotal != nil {
		setCellFloatVal(f, sheet, cell("T", row+2), params.Report.GrandTotal.DailyProductionMlnKWh)
	}

	// Row 3: Амалда (факт) = MTD production from grand total (formula: +V{grandRow})
	_ = f.SetCellFormula(sheet, cell("T", row+3), fmt.Sprintf("+V%d", grandRow))

	// Row 4: Бажарилди (выполнение %) = IFERROR(T{row+3}/T{row+1}*100, 0)
	_ = f.SetCellFormula(sheet, cell("T", row+4),
		fmt.Sprintf("IFERROR(T%d/T%d*100,0)", row+3, row+1))
}

func fillAggregates(f *excelize.File, sheet string, row int, grandRow int, params ExcelParams) {
	counts := params.OrgTypeCounts

	// Row 0: Умумий ГЭСлар сони (total GES count — value)
	setCellInt(f, sheet, cell("E", row), counts.Total)

	// Row 1: Умумий агрегатлар сони = +P{grandRow} (formula)
	_ = f.SetCellFormula(sheet, cell("E", row+1), fmt.Sprintf("+P%d", grandRow))

	// Row 2: Ишлаётган агрегатлар сони = +Q{grandRow} (formula)
	_ = f.SetCellFormula(sheet, cell("E", row+2), fmt.Sprintf("+Q%d", grandRow))

	// Rows 3-5 read from report.GrandTotal aggregates (clamped in service).
	// DB trigger guarantees working+repair+mod ≤ total, so reserve ≥ 0.
	var repair, modernization, reserve int
	if gt := params.Report.GrandTotal; gt != nil {
		repair = gt.RepairAggregates
		modernization = gt.ModernizationAggregates
		reserve = gt.ReserveAggregates
	}

	// Row 3: Заҳирадаги (reserve — from grand total)
	setCellInt(f, sheet, cell("E", row+3), reserve)

	// Row 4: Таъмирдаги (repair — from grand total)
	setCellInt(f, sheet, cell("E", row+4), repair)

	// Row 5: Модернизацияда (modernization — from grand total)
	setCellInt(f, sheet, cell("E", row+5), modernization)
}

