// Package ownneeds renders the dedicated own-needs (СН/ХН) GES Excel report
// from a workbook template.
//
// The template (template/own-needs.xlsx) carries two body styles in adjacent
// rows: row 6 is the cascade-summary template (bold + tinted fills), row 7
// is the station template (regular weight + neutral fills). The grand-total
// row at row 8 mirrors the cascade style with a different fill.
//
// For a report with N cascades, the generator:
//
//  1. Duplicates the (cascade, station) 2-row block N-1 times so each cascade
//     has its own pair (rows 6+7, 8+9, 10+11, ...). This preserves the per-row
//     styling because DuplicateRowTo/DuplicateRow copy formatting verbatim.
//  2. Within each block, duplicates the station row for every station beyond
//     the first so the cascade sub-rows have the right station style.
//  3. Fills the cells in document order, then the (now-shifted) grand-total
//     row at the end.
package ownneeds

import (
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"

	model "srmt-admin/internal/lib/model/ges-report"
	"srmt-admin/internal/lib/service/excel/templates"
)

// Template layout constants — see package comment for context.
const (
	// templateHeaderDateRow holds the report date in column H (merged H3:I3).
	templateHeaderDateRow = 3
	// templateYearRow holds the year header (merged D4:P4) and column-A/B
	// vertical merges spanning into row 5.
	templateYearRow = 4
	// templateSubHeaderRow holds the per-column captions (C5..P5).
	templateSubHeaderRow = 5
	// templateCascadeRow is the source row that carries the cascade-summary
	// style (bold + tinted fills). Duplicated as the first row of each
	// per-cascade 2-row block.
	templateCascadeRow = 6
	// templateStationRow is the source row that carries the station style
	// (regular weight + neutral fills). Duplicated as the second row of each
	// per-cascade block, plus once more for every additional station.
	templateStationRow = 7
	// templateGrandTotalRow is the static grand-total row in the template
	// before any duplication.
	templateGrandTotalRow = 8
	// blockSize is the number of template rows that constitute one cascade
	// block (cascade summary + first station).
	blockSize = 2
	// numColumns is the count of data columns (A..P).
	numColumns = 16
)

// uzMonths maps month numbers to Uzbek month names used in the C5/D5 captions.
// The sample report uses these names verbatim ("Апрель", "Январь-Апрель").
var uzMonths = map[time.Month]string{
	time.January:   "Январь",
	time.February:  "Феврал",
	time.March:     "Март",
	time.April:     "Апрель",
	time.May:       "Май",
	time.June:      "Июнь",
	time.July:      "Июль",
	time.August:    "Август",
	time.September: "Сентябрь",
	time.October:   "Октябрь",
	time.November:  "Ноябрь",
	time.December:  "Декабрь",
}

// uzMonthName returns the Uzbek name for the given month, falling back to
// time.Month.String() (English) if absent — should never trigger in practice.
func uzMonthName(m time.Month) string {
	if n, ok := uzMonths[m]; ok {
		return n
	}
	return m.String()
}

// Generator produces an Excel workbook for the own-needs report from a template.
type Generator struct {
	overrideDir string
}

// New creates a Generator. overrideDir, when non-empty, is forwarded to
// templates.Open as the dev-time on-disk override directory; pass "" to
// always use the embedded template.
func New(overrideDir string) *Generator {
	return &Generator{overrideDir: overrideDir}
}

// Params bundles inputs to GenerateExcel.
type Params struct {
	Report *model.OwnNeedsReport
	Date   time.Time
}

// GenerateExcel returns an excelize.File ready to be streamed to the client.
// The caller is responsible for closing the returned file.
func (g *Generator) GenerateExcel(p Params) (*excelize.File, error) {
	if p.Report == nil {
		return nil, fmt.Errorf("nil report")
	}
	f, err := templates.Open(templates.OwnNeeds, g.overrideDir)
	if err != nil {
		return nil, fmt.Errorf("open template: %w", err)
	}

	sheet := f.GetSheetList()[0]
	newSheet := p.Date.Format("02.01.06")
	if err := f.SetSheetName(sheet, newSheet); err != nil {
		return nil, fmt.Errorf("rename sheet: %w", err)
	}

	if err := f.SetCellValue(newSheet, "H3", p.Date); err != nil {
		return nil, fmt.Errorf("set H3 date: %w", err)
	}
	// Override the formula =H3 in D4 with an explicit "<year> йил" caption to
	// match the sample workbook's layout.
	if err := f.SetCellStr(newSheet, "D4", fmt.Sprintf("%d йил", p.Date.Year())); err != nil {
		return nil, fmt.Errorf("set D4 year: %w", err)
	}
	month := uzMonthName(p.Date.Month())
	if err := f.SetCellStr(newSheet, "C5", fmt.Sprintf("Ойлик режа (%s)", month)); err != nil {
		return nil, fmt.Errorf("set C5 monthly caption: %w", err)
	}
	if err := f.SetCellStr(newSheet, "D5", fmt.Sprintf("Режа  (Январь-%s)            ", month)); err != nil {
		return nil, fmt.Errorf("set D5 cumulative caption: %w", err)
	}

	cascades := p.Report.Cascades
	n := len(cascades)
	if n == 0 {
		// No cascades — leave the template's two body rows (6, 7) blank and
		// fill the grand total at its original row 8.
		fillGrandTotalRow(f, newSheet, templateGrandTotalRow, p.Report.GrandTotal)
		return f, nil
	}

	// Phase 1a: clone the (cascade, station) 2-row block for every cascade
	// past the first. DuplicateRowTo copies each source row to a target
	// position, preserving the per-row formatting (bold/fills/number formats).
	// Source rows 6 and 7 stay above every insertion point so they never shift.
	for i := 1; i < n; i++ {
		targetBase := templateCascadeRow + i*blockSize
		for j := 0; j < blockSize; j++ {
			if err := f.DuplicateRowTo(newSheet, templateCascadeRow+j, targetBase+j); err != nil {
				return nil, fmt.Errorf("duplicate block %d row %d: %w", i, j, err)
			}
		}
	}

	// Phase 1b: within each block, duplicate the station row for stations
	// beyond the first. Track the cumulative row offset introduced by the
	// inserted station rows so each next block's station-row index is right.
	offset := 0
	for i, c := range cascades {
		extra := len(c.Stations) - 1
		if extra <= 0 {
			continue
		}
		stationRow := templateCascadeRow + i*blockSize + 1 + offset
		for j := 0; j < extra; j++ {
			if err := f.DuplicateRow(newSheet, stationRow); err != nil {
				return nil, fmt.Errorf("duplicate station row cascade %d: %w", i, err)
			}
		}
		offset += extra
	}

	// Phase 2: fill cells in document order.
	row := templateCascadeRow
	for _, c := range cascades {
		fillCascadeRow(f, newSheet, row, c)
		row++
		if len(c.Stations) == 0 {
			// Block always reserves one station row; leave it blank but skip.
			row++
			continue
		}
		for _, st := range c.Stations {
			fillStationRow(f, newSheet, row, st, c.Totals.InstalledCapacityMWt)
			row++
		}
	}

	// The grand-total row has been pushed down by all the rows we inserted:
	// each cascade past the first added blockSize rows; each cascade also
	// added (len(stations)-1) station rows when present.
	grandRow := templateGrandTotalRow + insertedRows(cascades)
	fillGrandTotalRow(f, newSheet, grandRow, p.Report.GrandTotal)

	return f, nil
}

// insertedRows returns the total number of rows DuplicateRow* calls add to
// the sheet for the given cascades — used to locate the shifted grand-total
// row. Each cascade past the first contributes `blockSize` (2). Each cascade
// with K stations contributes max(K-1, 0) extra station rows.
func insertedRows(cascades []model.OwnNeedsCascade) int {
	if len(cascades) == 0 {
		return 0
	}
	extra := (len(cascades) - 1) * blockSize
	for _, c := range cascades {
		if len(c.Stations) > 1 {
			extra += len(c.Stations) - 1
		}
	}
	return extra
}

// fillCascadeRow writes a cascade-summary row. The cascade name goes in
// column A; columns B..P show summed values from cascade totals. Per-kW
// efficiency columns M..P are derived from totals + summed capacity.
func fillCascadeRow(f *excelize.File, sheet string, row int, c model.OwnNeedsCascade) {
	t := c.Totals
	setStr(f, sheet, row, "A", c.CascadeName)
	setNum(f, sheet, row, "B", t.InstalledCapacityMWt)
	setNum(f, sheet, row, "C", t.MonthlyPlanMlnKWh)
	setNum(f, sheet, row, "D", t.CumulativePlanMlnKWh)
	setNum(f, sheet, row, "E", t.DailyProductionMlnKWh)
	setNum(f, sheet, row, "F", t.DailyProductionDelta)
	setNum(f, sheet, row, "G", t.MTDProductionMlnKWh)
	setNum(f, sheet, row, "H", t.YTDProductionMlnKWh)
	setNum(f, sheet, row, "I", t.OwnConsumptionKWh)
	setNum(f, sheet, row, "J", t.OwnConsumptionDelta)
	setNum(f, sheet, row, "K", t.MTDOwnConsumptionKWh)
	setNum(f, sheet, row, "L", t.YTDOwnConsumptionKWh)
	setPerKW(f, sheet, row, "M", t.OwnConsumptionKWh, t.InstalledCapacityMWt)
	setPerKW(f, sheet, row, "N", t.OwnConsumptionDelta, t.InstalledCapacityMWt)
	setPerKW(f, sheet, row, "O", t.MTDOwnConsumptionKWh, t.InstalledCapacityMWt)
	setPerKW(f, sheet, row, "P", t.YTDOwnConsumptionKWh, t.InstalledCapacityMWt)
}

// fillStationRow writes a single station row. cascadeCapacityMWt is unused
// here — kept as a parameter to signal that station-level per-kW values use
// the station's own capacity, not the cascade aggregate.
func fillStationRow(f *excelize.File, sheet string, row int, st model.OwnNeedsStation, _ float64) {
	setStr(f, sheet, row, "A", st.Name)
	setNum(f, sheet, row, "B", st.InstalledCapacityMWt)
	setNum(f, sheet, row, "C", st.MonthlyPlanMlnKWh)
	setNum(f, sheet, row, "D", st.CumulativePlanMlnKWh)
	setNum(f, sheet, row, "E", st.DailyProductionMlnKWh)
	setOptional(f, sheet, row, "F", st.DailyProductionDelta)
	setNum(f, sheet, row, "G", st.MTDProductionMlnKWh)
	setNum(f, sheet, row, "H", st.YTDProductionMlnKWh)
	setOptional(f, sheet, row, "I", st.OwnConsumptionKWh)
	setOptional(f, sheet, row, "J", st.OwnConsumptionDelta)
	setNum(f, sheet, row, "K", st.MTDOwnConsumptionKWh)
	setNum(f, sheet, row, "L", st.YTDOwnConsumptionKWh)
	setPerKWOptional(f, sheet, row, "M", st.OwnConsumptionKWh, st.InstalledCapacityMWt)
	setPerKWOptional(f, sheet, row, "N", st.OwnConsumptionDelta, st.InstalledCapacityMWt)
	setPerKW(f, sheet, row, "O", st.MTDOwnConsumptionKWh, st.InstalledCapacityMWt)
	setPerKW(f, sheet, row, "P", st.YTDOwnConsumptionKWh, st.InstalledCapacityMWt)
}

// fillGrandTotalRow writes the grand-total row. Row 8 in the template carries
// the "«Ўзбекгидроэнерго» АЖ бўйича:" caption in column A; we leave that
// caption alone and write the totals into B..P.
func fillGrandTotalRow(f *excelize.File, sheet string, row int, t model.OwnNeedsTotals) {
	setNum(f, sheet, row, "B", t.InstalledCapacityMWt)
	setNum(f, sheet, row, "C", t.MonthlyPlanMlnKWh)
	setNum(f, sheet, row, "D", t.CumulativePlanMlnKWh)
	setNum(f, sheet, row, "E", t.DailyProductionMlnKWh)
	setNum(f, sheet, row, "F", t.DailyProductionDelta)
	setNum(f, sheet, row, "G", t.MTDProductionMlnKWh)
	setNum(f, sheet, row, "H", t.YTDProductionMlnKWh)
	setNum(f, sheet, row, "I", t.OwnConsumptionKWh)
	setNum(f, sheet, row, "J", t.OwnConsumptionDelta)
	setNum(f, sheet, row, "K", t.MTDOwnConsumptionKWh)
	setNum(f, sheet, row, "L", t.YTDOwnConsumptionKWh)
	setPerKW(f, sheet, row, "M", t.OwnConsumptionKWh, t.InstalledCapacityMWt)
	setPerKW(f, sheet, row, "N", t.OwnConsumptionDelta, t.InstalledCapacityMWt)
	setPerKW(f, sheet, row, "O", t.MTDOwnConsumptionKWh, t.InstalledCapacityMWt)
	setPerKW(f, sheet, row, "P", t.YTDOwnConsumptionKWh, t.InstalledCapacityMWt)
}

// --- Cell-write helpers ---

func cellRef(col string, row int) string { return fmt.Sprintf("%s%d", col, row) }

func setStr(f *excelize.File, sheet string, row int, col, v string) {
	_ = f.SetCellStr(sheet, cellRef(col, row), v)
}

func setNum(f *excelize.File, sheet string, row int, col string, v float64) {
	_ = f.SetCellFloat(sheet, cellRef(col, row), v, -1, 64)
}

// setOptional writes the pointer's value if non-nil; otherwise leaves the
// cell empty so the template's default styling renders a blank cell.
func setOptional(f *excelize.File, sheet string, row int, col string, v *float64) {
	if v == nil {
		_ = f.SetCellStr(sheet, cellRef(col, row), "")
		return
	}
	setNum(f, sheet, row, col, *v)
}

// setPerKW writes "watt-hours per installed kW" = kWh / capacityMW.
// (kWh / (capacityMW * 1000)) * 1000 simplifies to kWh / capacityMW.
// If capacity is zero the cell is left blank — division would be undefined
// and the sample workbook leaves these cells empty for capacity-less rows
// like dam-site reservoirs.
func setPerKW(f *excelize.File, sheet string, row int, col string, kWh, capacityMW float64) {
	if capacityMW == 0 {
		_ = f.SetCellStr(sheet, cellRef(col, row), "")
		return
	}
	setNum(f, sheet, row, col, kWh/capacityMW)
}

// setPerKWOptional combines setOptional + setPerKW for nullable inputs.
func setPerKWOptional(f *excelize.File, sheet string, row int, col string, kWh *float64, capacityMW float64) {
	if kWh == nil {
		_ = f.SetCellStr(sheet, cellRef(col, row), "")
		return
	}
	setPerKW(f, sheet, row, col, *kWh, capacityMW)
}
