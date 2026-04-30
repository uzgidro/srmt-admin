// Package ownneeds renders the dedicated own-needs (СН/ХН) GES Excel report
// from a workbook template.
//
// The template (template/own-needs.xlsx) contains:
//   - rows 2..5: report header (title, date, year, column captions)
//   - row 6: an empty body row used as the duplication template for cascade
//     and station rows
//   - row 8: the grand-total row labelled "«Ўзбекгидроэнерго» АЖ бўйича:"
//
// The generator inserts one row per cascade plus one row per station between
// the body template (row 6) and the grand total (row 8), then fills each row
// with the data from model.OwnNeedsReport.
package ownneeds

import (
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"

	model "srmt-admin/internal/lib/model/ges-report"
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
	// templateBodyRow is the source row that gets duplicated for each cascade
	// and station to create the body of the report.
	templateBodyRow = 6
	// templateGrandTotalRow is the static grand-total row in the template
	// before any duplication.
	templateGrandTotalRow = 8
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
	templatePath string
}

// New creates a Generator bound to the given template file path.
func New(templatePath string) *Generator {
	return &Generator{templatePath: templatePath}
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
	f, err := excelize.OpenFile(g.templatePath)
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

	bodyRowsNeeded := totalBodyRows(p.Report)
	// The template already has 1 body row (row 6). Duplicate it (bodyRowsNeeded-1)
	// extra times to get the required count. DuplicateRow inserts the new row
	// just below the source, pushing the grand-total down each time.
	for i := 1; i < bodyRowsNeeded; i++ {
		if err := f.DuplicateRow(newSheet, templateBodyRow); err != nil {
			return nil, fmt.Errorf("duplicate body row %d: %w", i, err)
		}
	}

	row := templateBodyRow
	for _, c := range p.Report.Cascades {
		fillCascadeRow(f, newSheet, row, c)
		row++
		for _, st := range c.Stations {
			fillStationRow(f, newSheet, row, st, c.Totals.InstalledCapacityMWt)
			row++
		}
	}

	// After body insertion, the grand-total row has shifted by (bodyRowsNeeded - 1)
	// because the original row 6 is now the first body row (no shift) and each
	// subsequent DuplicateRow on row 6 pushed the grand total down by 1.
	grandRow := templateGrandTotalRow + bodyRowsNeeded - 1
	fillGrandTotalRow(f, newSheet, grandRow, p.Report.GrandTotal)

	return f, nil
}

// totalBodyRows is the count of rows the body needs: one per cascade + one per
// station across all cascades. A report with no cascades still produces one
// (empty) body row so the template layout remains valid.
func totalBodyRows(rep *model.OwnNeedsReport) int {
	if len(rep.Cascades) == 0 {
		return 1
	}
	n := 0
	for _, c := range rep.Cascades {
		n += 1 + len(c.Stations)
	}
	return n
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
