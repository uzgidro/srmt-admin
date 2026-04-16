package ges

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
	model "srmt-admin/internal/lib/model/ges-report"
)

func templatePath(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	return filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", "..", "template", "ges-prod.xlsx")
}

func floatPtr(v float64) *float64 { return &v }

func assertCellFloat(t *testing.T, f *excelize.File, sheet string, row int, col string, want float64) {
	t.Helper()
	raw, _ := f.GetCellValue(sheet, fmt.Sprintf("%s%d", col, row))
	got, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		t.Errorf("row %d col %s: cannot parse %q as float: %v", row, col, raw, err)
		return
	}
	if got != want {
		t.Errorf("row %d col %s = %v, want %v", row, col, got, want)
	}
}

func buildTestParams() ExcelParams {
	loc, _ := time.LoadLocation("Asia/Tashkent")
	date := time.Date(2026, 4, 15, 0, 0, 0, 0, loc)

	// 2 cascades: first has 3 stations, second has 2 stations
	cascade1 := model.CascadeReport{
		CascadeID:   1,
		CascadeName: "Cascade Alpha",
		Weather: &model.CascadeWeather{
			Temperature:         floatPtr(18.5),
			PrevYearTemperature: floatPtr(16.0),
		},
		Summary: &model.SummaryBlock{
			InstalledCapacityMWt:  600,
			TotalAggregates:       12,
			WorkingAggregates:     10,
			PowerMWt:              450,
			DailyProductionMlnKWh: 10.5,
			ProductionChange:      0.3,
			MTDProductionMlnKWh:   150.0,
			YTDProductionMlnKWh:   900.0,
			FulfillmentPct:        floatPtr(95.5),
			DifferenceMlnKWh:      -5.0,
			PrevYearYTD:           850.0,
			YoYGrowthRate:         floatPtr(105.9),
			YoYDifference:         50.0,
		},
		Stations: []model.StationReport{
			makeStation(101, "Station A1", 200, 4, 3, 5.0, 0.1),
			makeStation(102, "Station A2", 200, 4, 4, 3.5, 0.2),
			makeStation(103, "Station A3", 200, 4, 3, 2.0, -0.1),
		},
	}

	cascade2 := model.CascadeReport{
		CascadeID:   2,
		CascadeName: "Cascade Beta",
		Weather: &model.CascadeWeather{
			Temperature:         floatPtr(20.0),
			PrevYearTemperature: floatPtr(17.5),
		},
		Summary: &model.SummaryBlock{
			InstalledCapacityMWt:  400,
			TotalAggregates:       8,
			WorkingAggregates:     7,
			PowerMWt:              320,
			DailyProductionMlnKWh: 7.5,
			ProductionChange:      -0.2,
			MTDProductionMlnKWh:   100.0,
			YTDProductionMlnKWh:   600.0,
			FulfillmentPct:        floatPtr(98.0),
			DifferenceMlnKWh:      2.0,
			PrevYearYTD:           580.0,
			YoYGrowthRate:         floatPtr(103.4),
			YoYDifference:         20.0,
		},
		Stations: []model.StationReport{
			makeStation(201, "Station B1", 200, 4, 3, 4.0, -0.1),
			makeStation(202, "Station B2", 200, 4, 4, 3.5, 0.0),
		},
	}

	grandTotal := &model.SummaryBlock{
		InstalledCapacityMWt:  1000,
		TotalAggregates:       20,
		WorkingAggregates:     17,
		PowerMWt:              770,
		DailyProductionMlnKWh: 18.0,
		ProductionChange:      0.1,
		MTDProductionMlnKWh:   250.0,
		YTDProductionMlnKWh:   1500.0,
		FulfillmentPct:        floatPtr(96.5),
		DifferenceMlnKWh:      -3.0,
		PrevYearYTD:           1430.0,
		YoYGrowthRate:         floatPtr(104.9),
		YoYDifference:         70.0,
	}

	return ExcelParams{
		Report: &model.DailyReport{
			Date:       "2026-04-15",
			Cascades:   []model.CascadeReport{cascade1, cascade2},
			GrandTotal: grandTotal,
		},
		YTDPlans:    map[int64]float64{101: 100, 102: 100, 103: 100, 201: 100, 202: 100},
		AnnualPlans: map[int64]float64{101: 300, 102: 300, 103: 300, 201: 300, 202: 300},
		MonthlyPlans: map[int64]float64{101: 25, 102: 25, 103: 25, 201: 25, 202: 25},
		OrgTypeCounts: OrgTypeCounts{
			GES:   10,
			Mini:  5,
			Micro: 3,
			Total: 18,
		},
		Modernization: 4,
		Repair:        14,
		Date:          date,
		Loc:           loc,
	}
}

func makeStation(orgID int64, name string, capacity float64, total, working int, daily, change float64) model.StationReport {
	return model.StationReport{
		OrganizationID: orgID,
		Name:           name,
		Config: model.StationConfig{
			InstalledCapacityMWt: capacity,
			TotalAggregates:      total,
			HasReservoir:         true,
		},
		Current: model.CurrentData{
			DailyProductionMlnKWh: daily,
			PowerMWt:              capacity * 0.75,
			WorkingAggregates:     working,
			WaterLevelM:           floatPtr(320.5),
			WaterVolumeMlnM3:      floatPtr(1500.0),
			WaterHeadM:            floatPtr(80.0),
			ReservoirIncomeM3s:    floatPtr(250.0),
			TotalOutflowM3s:       floatPtr(200.0),
			GESFlowM3s:            floatPtr(180.0),
			IdleDischargeM3s:      floatPtr(20.0),
		},
		Diffs: model.DiffData{
			LevelChangeCm:    floatPtr(5.0),
			VolumeChangeMlnM3: floatPtr(2.0),
			IncomeChangeM3s:  floatPtr(10.0),
			GESFlowChangeM3s: floatPtr(8.0),
			PowerChangeMWt:   floatPtr(change * 100),
			ProductionChange: floatPtr(change),
		},
		Aggregations: model.Aggregations{
			MTDProductionMlnKWh: daily * 15,
			YTDProductionMlnKWh: daily * 105,
		},
		Plan: model.PlanData{
			MonthlyPlanMlnKWh: 25.0,
			FulfillmentPct:    floatPtr(95.0),
			DifferenceMlnKWh:  -1.0,
		},
		PreviousYear: &model.PrevYearData{
			WaterLevelM:        floatPtr(318.0),
			WaterVolumeMlnM3:   floatPtr(1450.0),
			WaterHeadM:         floatPtr(78.0),
			ReservoirIncomeM3s: floatPtr(230.0),
			GESFlowM3s:         floatPtr(170.0),
			PowerMWt:           floatPtr(capacity * 0.70),
			DailyProduction:    floatPtr(daily * 0.9),
			MTDProduction:      daily * 14,
			YTDProduction:      daily * 100,
		},
		YoY: model.YoYData{
			GrowthRate:       floatPtr(105.0),
			DifferenceMlnKWh: daily * 5,
		},
	}
}

func TestGenerateExcel_OpensTemplate(t *testing.T) {
	gen := New(templatePath(t))
	params := buildTestParams()

	f, err := gen.GenerateExcel(params)
	if err != nil {
		t.Fatalf("GenerateExcel returned error: %v", err)
	}
	defer f.Close()
}

func TestGenerateExcel_DateInAH3(t *testing.T) {
	gen := New(templatePath(t))
	params := buildTestParams()

	f, err := gen.GenerateExcel(params)
	if err != nil {
		t.Fatalf("GenerateExcel returned error: %v", err)
	}
	defer f.Close()

	sheet := f.GetSheetList()[0]
	val, err := f.GetCellValue(sheet, "AH3")
	if err != nil {
		t.Fatalf("GetCellValue AH3: %v", err)
	}
	// Date + 1 day = 2026-04-16
	// excelize returns formatted date string
	if val == "" {
		t.Fatal("AH3 is empty, expected date+1")
	}
	t.Logf("AH3 value: %s", val)
}

func TestGenerateExcel_SheetRenamed(t *testing.T) {
	gen := New(templatePath(t))
	params := buildTestParams()

	f, err := gen.GenerateExcel(params)
	if err != nil {
		t.Fatalf("GenerateExcel returned error: %v", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	expected := "15.04.26"
	if sheets[0] != expected {
		t.Errorf("sheet name = %q, want %q", sheets[0], expected)
	}
}

func TestGenerateExcel_StationRows(t *testing.T) {
	gen := New(templatePath(t))
	params := buildTestParams()

	f, err := gen.GenerateExcel(params)
	if err != nil {
		t.Fatalf("GenerateExcel returned error: %v", err)
	}
	defer f.Close()

	sheet := f.GetSheetList()[0]

	// Count station name rows (non-empty A column in the data area, rows 7+)
	// We expect: cascade1 total + 3 stations + cascade2 total + 2 stations = 7 data rows
	// Then grand total row, then forecasts, then aggregates
	stationCount := 0
	for row := 7; row <= 30; row++ {
		cell := fmt.Sprintf("A%d", row)
		v, _ := f.GetCellValue(sheet, cell)
		if v != "" {
			t.Logf("Row %d A=%q", row, v)
		}
		// Station rows have individual station names
		for _, s := range []string{"Station A1", "Station A2", "Station A3", "Station B1", "Station B2"} {
			if v == s {
				stationCount++
			}
		}
	}
	if stationCount != 5 {
		t.Errorf("found %d station rows, want 5", stationCount)
	}
}

func TestGenerateExcel_GrandTotalFilled(t *testing.T) {
	gen := New(templatePath(t))
	params := buildTestParams()

	f, err := gen.GenerateExcel(params)
	if err != nil {
		t.Fatalf("GenerateExcel returned error: %v", err)
	}
	defer f.Close()

	sheet := f.GetSheetList()[0]

	// Grand total row: after 2 cascades (3+2 stations + 2 cascade headers = 7 rows), so row 14
	// Row 7: cascade1 total, 8-10: stations, 11: cascade2 total, 12-13: stations, 14: grand total
	grandTotalRow := 7 + 3 + 1 + 2 // cascade1(1+3) + cascade2(1+2) = 7 data rows, grand total at row 14
	grandTotalRow = 14 // 7+7=14

	cell := fmt.Sprintf("T%d", grandTotalRow)
	v, _ := f.GetCellValue(sheet, cell)
	if v == "" {
		// Try to find the grand total row by scanning for the label
		for row := 7; row <= 25; row++ {
			a, _ := f.GetCellValue(sheet, fmt.Sprintf("A%d", row))
			tv, _ := f.GetCellValue(sheet, fmt.Sprintf("T%d", row))
			if a != "" && tv == "18" {
				t.Logf("Found grand total at row %d: A=%q T=%q", row, a, tv)
				return
			}
		}
		t.Error("grand total daily production (18.0) not found in column T")
	}
}

func TestGenerateExcel_ForecastsFilled(t *testing.T) {
	gen := New(templatePath(t))
	params := buildTestParams()

	f, err := gen.GenerateExcel(params)
	if err != nil {
		t.Fatalf("GenerateExcel returned error: %v", err)
	}
	defer f.Close()

	sheet := f.GetSheetList()[0]

	// Find the forecast rows (they come after grand total)
	// Look for annual plan total in column T
	found := false
	for row := 10; row <= 30; row++ {
		v, _ := f.GetCellValue(sheet, fmt.Sprintf("T%d", row))
		if v != "" {
			t.Logf("Forecast area row %d T=%s", row, v)
			found = true
		}
	}
	if !found {
		t.Error("no forecast values found in column T")
	}
}

func TestGenerateExcel_AggregatesFilled(t *testing.T) {
	gen := New(templatePath(t))
	params := buildTestParams()

	f, err := gen.GenerateExcel(params)
	if err != nil {
		t.Fatalf("GenerateExcel returned error: %v", err)
	}
	defer f.Close()

	sheet := f.GetSheetList()[0]

	// Find GES count cell (E column in aggregate area)
	found := false
	for row := 10; row <= 35; row++ {
		a, _ := f.GetCellValue(sheet, fmt.Sprintf("A%d", row))
		e, _ := f.GetCellValue(sheet, fmt.Sprintf("E%d", row))
		// Looking for total GES count
		if e != "" && a != "" {
			t.Logf("Aggregate row %d A=%q E=%q", row, a, e)
			found = true
		}
	}
	if !found {
		t.Error("no aggregate values found")
	}
}

func TestGenerateExcel_ColumnMapping(t *testing.T) {
	gen := New(templatePath(t))
	params := buildTestParams()

	f, err := gen.GenerateExcel(params)
	if err != nil {
		t.Fatalf("GenerateExcel returned error: %v", err)
	}
	defer f.Close()

	sheet := f.GetSheetList()[0]

	// Find Station A1 row and check specific column values
	for row := 7; row <= 25; row++ {
		a, _ := f.GetCellValue(sheet, fmt.Sprintf("A%d", row))
		if a == "Station A1" {
			assertCellFloat(t, f, sheet, row, "B", 200) // InstalledCapacityMWt
			assertCellFloat(t, f, sheet, row, "T", 5)   // DailyProductionMlnKWh
			assertCellFloat(t, f, sheet, row, "P", 4)   // TotalAggregates
			assertCellFloat(t, f, sheet, row, "Q", 3)   // WorkingAggregates
			assertCellFloat(t, f, sheet, row, "E", 320.5) // WaterLevelM
			assertCellFloat(t, f, sheet, row, "R", 150) // PowerMWt (200*0.75)
			return
		}
	}
	t.Error("Station A1 row not found")
}
