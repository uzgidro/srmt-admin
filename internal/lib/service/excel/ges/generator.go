package ges

import (
	"fmt"
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
	Modernization int
	Repair        int
	Date          time.Time
	Loc           *time.Location
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

// GenerateExcel produces an Excel file from the template and params.
func (g *Generator) GenerateExcel(params ExcelParams) (*excelize.File, error) {
	f, err := excelize.OpenFile(g.templatePath)
	if err != nil {
		return nil, fmt.Errorf("open template: %w", err)
	}

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

	// Calculate total station count per cascade and total rows needed
	stationsPerCascade := make([]int, n)
	totalDataRows := 0
	for i, c := range cascades {
		sc := len(c.Stations)
		if sc == 0 {
			sc = 1
		}
		stationsPerCascade[i] = sc
		totalDataRows += 1 + sc // cascade header + stations
	}

	// Template has 2 rows (7=cascade, 8=station). We need totalDataRows.
	// We'll duplicate row 8 (station template) to create all needed rows.
	// Strategy: insert rows after row 8 to make space, then fill.
	extraRows := totalDataRows - 2 // already have cascade+station=2
	if extraRows > 0 {
		if err := f.DuplicateRowTo(newSheet, templateStationRow, templateStationRow+1); err != nil {
			return nil, fmt.Errorf("initial duplicate: %w", err)
		}
		// Now we have 3 rows at 7,8,9. We need totalDataRows total.
		// Insert remaining rows after current position.
		for i := 1; i < extraRows; i++ {
			if err := f.DuplicateRowTo(newSheet, templateStationRow, templateStationRow+1+i); err != nil {
				return nil, fmt.Errorf("duplicate row %d: %w", i, err)
			}
		}
	}

	// Phase 2: fill data
	row := templateCascadeRow
	for i, cascade := range cascades {
		// Cascade total row
		fillCascadeRow(f, newSheet, row, cascade, params)
		row++

		// Station rows
		for _, station := range cascade.Stations {
			fillStationRow(f, newSheet, row, station, cascade.Weather, params)
			row++
		}
		_ = stationsPerCascade[i]
	}

	// Grand total row
	grandRow := row
	fillGrandTotalRow(f, newSheet, grandRow, params.Report.GrandTotal)

	// Forecast rows (originally rows 10-13, now shifted)
	forecastRow := grandRow + 1
	fillForecasts(f, newSheet, forecastRow, params)

	// Aggregate rows (originally rows 14-19, now shifted)
	aggRow := forecastRow + 4
	fillAggregates(f, newSheet, aggRow, params)

	if err := f.UpdateLinkedValue(); err != nil {
		// Non-fatal — some templates may not have linked values
		_ = err
	}

	return f, nil
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

func fillStationRow(f *excelize.File, sheet string, row int, s model.StationReport, weather *model.CascadeWeather, params ExcelParams) {
	c := s.Current
	d := s.Diffs
	agg := s.Aggregations
	plan := s.Plan
	cfg := s.Config

	_ = f.SetCellValue(sheet, cell("A", row), s.Name)
	setCellFloatVal(f, sheet, cell("B", row), cfg.InstalledCapacityMWt)

	// C = YTD plan
	if ytd, ok := params.YTDPlans[s.OrganizationID]; ok {
		setCellFloatVal(f, sheet, cell("C", row), ytd)
	}

	// D = temperature (cascade-level)
	if weather != nil {
		setCellFloat(f, sheet, cell("D", row), weather.Temperature)
	}

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

	// X-Y: plan fulfillment
	setCellFloat(f, sheet, cell("X", row), plan.FulfillmentPct)
	setCellFloatVal(f, sheet, cell("Y", row), plan.DifferenceMlnKWh)

	// Previous year columns Z-AI
	if weather != nil {
		setCellFloat(f, sheet, cell("Z", row), weather.PrevYearTemperature)
	}
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

	// YoY columns AJ-AK
	setCellFloat(f, sheet, cell("AJ", row), s.YoY.GrowthRate)
	setCellFloatVal(f, sheet, cell("AK", row), s.YoY.DifferenceMlnKWh)
}

func fillCascadeRow(f *excelize.File, sheet string, row int, c model.CascadeReport, params ExcelParams) {
	_ = f.SetCellValue(sheet, cell("A", row), c.CascadeName)

	s := c.Summary
	if s == nil {
		return
	}

	setCellFloatVal(f, sheet, cell("B", row), s.InstalledCapacityMWt)
	setCellInt(f, sheet, cell("P", row), s.TotalAggregates)
	setCellInt(f, sheet, cell("Q", row), s.WorkingAggregates)
	setCellFloatVal(f, sheet, cell("R", row), s.PowerMWt)
	setCellFloatVal(f, sheet, cell("T", row), s.DailyProductionMlnKWh)
	setCellFloatVal(f, sheet, cell("U", row), s.ProductionChange)
	setCellFloatVal(f, sheet, cell("V", row), s.MTDProductionMlnKWh)
	setCellFloatVal(f, sheet, cell("W", row), s.YTDProductionMlnKWh)
	setCellFloat(f, sheet, cell("X", row), s.FulfillmentPct)
	setCellFloatVal(f, sheet, cell("Y", row), s.DifferenceMlnKWh)
	setCellFloatVal(f, sheet, cell("AI", row), s.PrevYearYTD)
	setCellFloat(f, sheet, cell("AJ", row), s.YoYGrowthRate)
	setCellFloatVal(f, sheet, cell("AK", row), s.YoYDifference)
}

func fillGrandTotalRow(f *excelize.File, sheet string, row int, gt *model.SummaryBlock) {
	if gt == nil {
		return
	}

	setCellFloatVal(f, sheet, cell("B", row), gt.InstalledCapacityMWt)
	setCellInt(f, sheet, cell("P", row), gt.TotalAggregates)
	setCellInt(f, sheet, cell("Q", row), gt.WorkingAggregates)
	setCellFloatVal(f, sheet, cell("R", row), gt.PowerMWt)
	setCellFloatVal(f, sheet, cell("T", row), gt.DailyProductionMlnKWh)
	setCellFloatVal(f, sheet, cell("U", row), gt.ProductionChange)
	setCellFloatVal(f, sheet, cell("V", row), gt.MTDProductionMlnKWh)
	setCellFloatVal(f, sheet, cell("W", row), gt.YTDProductionMlnKWh)
	setCellFloat(f, sheet, cell("X", row), gt.FulfillmentPct)
	setCellFloatVal(f, sheet, cell("Y", row), gt.DifferenceMlnKWh)
	setCellFloatVal(f, sheet, cell("AI", row), gt.PrevYearYTD)
	setCellFloat(f, sheet, cell("AJ", row), gt.YoYGrowthRate)
	setCellFloatVal(f, sheet, cell("AK", row), gt.YoYDifference)
}

func fillForecasts(f *excelize.File, sheet string, row int, params ExcelParams) {
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

	// Row 2: daily production (from grand total)
	if params.Report.GrandTotal != nil {
		setCellFloatVal(f, sheet, cell("T", row+2), params.Report.GrandTotal.DailyProductionMlnKWh)
	}

	// Row 3: actual (same as daily for now)
	if params.Report.GrandTotal != nil {
		setCellFloatVal(f, sheet, cell("T", row+3), params.Report.GrandTotal.DailyProductionMlnKWh)
	}
}

func fillAggregates(f *excelize.File, sheet string, row int, params ExcelParams) {
	gt := params.Report.GrandTotal
	counts := params.OrgTypeCounts

	// Row 0: total GES count
	_ = f.SetCellValue(sheet, cell("E", row), fmt.Sprintf("%d та", counts.Total))

	// Row 1: total aggregates
	if gt != nil {
		_ = f.SetCellValue(sheet, cell("E", row+1), fmt.Sprintf("%d та", gt.TotalAggregates))
	}

	// Row 2: working aggregates
	if gt != nil {
		_ = f.SetCellValue(sheet, cell("E", row+2), fmt.Sprintf("%d та", gt.WorkingAggregates))
	}

	// Row 3: reserve aggregates (total - working - repair - modernization)
	if gt != nil {
		reserve := gt.TotalAggregates - gt.WorkingAggregates - params.Repair - params.Modernization
		_ = f.SetCellValue(sheet, cell("E", row+3), fmt.Sprintf("%d та", reserve))
	}

	// Row 4: repair
	_ = f.SetCellValue(sheet, cell("E", row+4), fmt.Sprintf("%d та", params.Repair))

	// Row 5: modernization
	_ = f.SetCellValue(sheet, cell("E", row+5), fmt.Sprintf("%d та", params.Modernization))
}

// fillForecastFulfillment writes the fulfillment percentage row
// (row after forecast actual).
func fillForecastFulfillment(f *excelize.File, sheet string, row int, params ExcelParams) {
	if params.Report.GrandTotal != nil {
		gt := params.Report.GrandTotal
		setCellFloat(f, sheet, cell("T", row), gt.FulfillmentPct)
	}
}
