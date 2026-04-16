package ges

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	model "srmt-admin/internal/lib/model/ges-report"
)

// TestGenerateExcel_IntegrationWithRealData builds a realistic ExcelParams with
// Uzbek station names and plausible production/reservoir values, generates the
// Excel file, saves it to d:/tmp/ges-test-output.xlsx for visual inspection,
// and spot-checks key cells.
func TestGenerateExcel_IntegrationWithRealData(t *testing.T) {
	gen := New(templatePath(t))
	params := buildRealisticParams(t)

	f, err := gen.GenerateExcel(params)
	if err != nil {
		t.Fatalf("GenerateExcel returned error: %v", err)
	}
	defer f.Close()

	// Save to d:/tmp for visual verification
	outDir := `d:/tmp`
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("create output dir: %v", err)
	}
	outPath := filepath.Join(outDir, fmt.Sprintf("ges-test-%d.xlsx", time.Now().Unix()))
	if err := f.SaveAs(outPath); err != nil {
		t.Fatalf("SaveAs: %v", err)
	}
	t.Logf("Output saved to: %s", outPath)

	// --- Spot-checks ---
	sheet := f.GetSheetList()[0]

	// 1. Sheet name = date formatted as DD.MM.YY
	wantSheet := "13.03.26"
	if sheet != wantSheet {
		t.Errorf("sheet name = %q, want %q", sheet, wantSheet)
	}

	// 2. AH3 contains date+1 (2026-03-14)
	ah3, err := f.GetCellValue(sheet, "AH3")
	if err != nil {
		t.Fatalf("GetCellValue AH3: %v", err)
	}
	if ah3 == "" {
		t.Error("AH3 is empty, expected date+1 (2026-03-14)")
	}
	t.Logf("AH3 = %s", ah3)

	// 3. Station names in column A
	stationNames := []string{
		"Чорвоқ ГЭС",
		"Хўжаикент ГЭС",
		"Ғазалкент ГЭС",
		"Чирчиқ ГЭС",
		"Тупаланг ГЭС",
		"Сурхон-1 ГЭС",
		"Ўзбекистон ГЭС",
		"Фарҳод ГЭС",
		"Қайроққум ГЭС",
	}
	foundStations := make(map[string]int) // name -> row
	for row := 7; row <= 40; row++ {
		v, _ := f.GetCellValue(sheet, fmt.Sprintf("A%d", row))
		for _, name := range stationNames {
			if v == name {
				foundStations[name] = row
			}
		}
	}
	for _, name := range stationNames {
		if _, ok := foundStations[name]; !ok {
			t.Errorf("station %q not found in column A", name)
		}
	}
	t.Logf("Found %d/%d stations in column A", len(foundStations), len(stationNames))

	// 4. Production values in column T for a known station
	if row, ok := foundStations["Чорвоқ ГЭС"]; ok {
		raw, _ := f.GetCellValue(sheet, fmt.Sprintf("T%d", row))
		val, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			t.Errorf("Чорвоқ ГЭС T%d: cannot parse %q: %v", row, raw, err)
		} else if val < 0.1 || val > 50 {
			t.Errorf("Чорвоқ ГЭС T%d: production=%v, expected realistic value 0.1-50", row, val)
		}
		t.Logf("Чорвоқ ГЭС daily production = %v", val)
	}

	// 5. Grand total row: find it by scanning for the sum of all daily production
	// Total daily = 3.2+1.8+0.9+1.5 + 1.2+0.8 + 6.5+2.1+1.4 = 19.4
	grandTotalFound := false
	for row := 7; row <= 40; row++ {
		raw, _ := f.GetCellValue(sheet, fmt.Sprintf("T%d", row))
		val, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			continue
		}
		if val > 19.0 && val < 20.0 {
			t.Logf("Grand total candidate at row %d: T=%v", row, val)
			grandTotalFound = true
			break
		}
	}
	if !grandTotalFound {
		t.Error("grand total row with daily production ~19.4 not found in column T")
	}

	// 6. Forecast rows: look for annual plan total (1550.0) in column T after grand total
	forecastFound := false
	for row := 15; row <= 40; row++ {
		raw, _ := f.GetCellValue(sheet, fmt.Sprintf("T%d", row))
		val, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			continue
		}
		if val > 1500 && val < 1600 {
			t.Logf("Annual plan total at row %d: T=%v", row, val)
			forecastFound = true
			break
		}
	}
	if !forecastFound {
		t.Error("forecast row with annual plan total ~1550 not found")
	}

	// 7. Aggregate rows: look for "18 та" pattern (total station count) in column E
	aggFound := false
	for row := 15; row <= 40; row++ {
		v, _ := f.GetCellValue(sheet, fmt.Sprintf("E%d", row))
		if v == "38 та" {
			t.Logf("Total GES count at row %d: E=%q", row, v)
			aggFound = true
			break
		}
	}
	if !aggFound {
		t.Error("aggregate row with total count '38 та' not found in column E")
	}

	// Dump the first 40 rows of key columns for debugging
	t.Log("--- Row dump (A, B, D, T, V, W columns) ---")
	for row := 5; row <= 35; row++ {
		a, _ := f.GetCellValue(sheet, fmt.Sprintf("A%d", row))
		b, _ := f.GetCellValue(sheet, fmt.Sprintf("B%d", row))
		d, _ := f.GetCellValue(sheet, fmt.Sprintf("D%d", row))
		tVal, _ := f.GetCellValue(sheet, fmt.Sprintf("T%d", row))
		v, _ := f.GetCellValue(sheet, fmt.Sprintf("V%d", row))
		w, _ := f.GetCellValue(sheet, fmt.Sprintf("W%d", row))
		e, _ := f.GetCellValue(sheet, fmt.Sprintf("E%d", row))
		if a != "" || tVal != "" || e != "" {
			t.Logf("Row %2d: A=%-25s B=%-8s D=%-6s T=%-8s V=%-8s W=%-8s E=%s", row, a, b, d, tVal, v, w, e)
		}
	}
}

func buildRealisticParams(t *testing.T) ExcelParams {
	loc, _ := time.LoadLocation("Asia/Tashkent")
	date := time.Date(2026, 3, 13, 0, 0, 0, 0, loc)

	// Cascade 1: Chirchiq-Bozsu (4 stations)
	cascade1 := model.CascadeReport{
		CascadeID:   1,
		CascadeName: "Чирчиқ-Бўзсув каскади",
		Weather: &model.CascadeWeather{
			Temperature:         floatPtr(18.0),
			Condition:           strPtr("01d"),
			PrevYearTemperature: floatPtr(15.5),
			PrevYearCondition:   strPtr("04d"),
		},
		Summary: &model.SummaryBlock{
			InstalledCapacityMWt:  921,
			TotalAggregates:       20,
			WorkingAggregates:     16,
			PowerMWt:              580,
			DailyProductionMlnKWh: 7.4,
			ProductionChange:      0.3,
			MTDProductionMlnKWh:   92.5,
			YTDProductionMlnKWh:   540.0,
			MonthlyPlanMlnKWh:     100.0,
			QuarterlyPlanMlnKWh:   280.0,
			FulfillmentPct:        floatPtr(92.5),
			DifferenceMlnKWh:      -7.5,
			PrevYearYTD:           510.0,
			YoYGrowthRate:         floatPtr(105.9),
			YoYDifference:         30.0,
		},
		Stations: []model.StationReport{
			realStation(101, "Чорвоқ ГЭС", 620, 6, 5, 3.2, 0.2,
				floatPtr(827.5), floatPtr(1120.0), floatPtr(195.0), floatPtr(312.0), floatPtr(280.0), floatPtr(250.0), floatPtr(30.0)),
			realStation(102, "Хўжаикент ГЭС", 135, 6, 5, 1.8, 0.1,
				floatPtr(614.0), floatPtr(85.0), floatPtr(38.0), floatPtr(285.0), floatPtr(260.0), floatPtr(240.0), floatPtr(20.0)),
			realStation(103, "Ғазалкент ГЭС", 90, 4, 3, 0.9, -0.1,
				floatPtr(503.0), floatPtr(45.0), floatPtr(28.0), floatPtr(240.0), floatPtr(220.0), floatPtr(200.0), floatPtr(20.0)),
			realStation(104, "Чирчиқ ГЭС", 76, 4, 3, 1.5, 0.1,
				nil, nil, nil, floatPtr(220.0), floatPtr(200.0), floatPtr(180.0), floatPtr(20.0)),
		},
	}

	// Cascade 2: Surkhandarya (2 stations)
	cascade2 := model.CascadeReport{
		CascadeID:   2,
		CascadeName: "Сурхондарё каскади",
		Weather: &model.CascadeWeather{
			Temperature:         floatPtr(22.0),
			Condition:           strPtr("10d"),
			PrevYearTemperature: floatPtr(19.5),
			PrevYearCondition:   strPtr("02d"),
		},
		Summary: &model.SummaryBlock{
			InstalledCapacityMWt:  180,
			TotalAggregates:       8,
			WorkingAggregates:     7,
			PowerMWt:              140,
			DailyProductionMlnKWh: 2.0,
			ProductionChange:      -0.1,
			MTDProductionMlnKWh:   25.0,
			YTDProductionMlnKWh:   150.0,
			MonthlyPlanMlnKWh:     30.0,
			QuarterlyPlanMlnKWh:   85.0,
			FulfillmentPct:        floatPtr(83.3),
			DifferenceMlnKWh:      -5.0,
			PrevYearYTD:           140.0,
			YoYGrowthRate:         floatPtr(107.1),
			YoYDifference:         10.0,
		},
		Stations: []model.StationReport{
			realStation(201, "Тупаланг ГЭС", 100, 4, 4, 1.2, -0.1,
				floatPtr(985.0), floatPtr(220.0), floatPtr(105.0), floatPtr(180.0), floatPtr(160.0), floatPtr(150.0), floatPtr(10.0)),
			realStation(202, "Сурхон-1 ГЭС", 80, 4, 3, 0.8, 0.0,
				floatPtr(740.0), floatPtr(95.0), floatPtr(62.0), floatPtr(130.0), floatPtr(110.0), floatPtr(100.0), floatPtr(10.0)),
		},
	}

	// Cascade 3: Lower Syrdarya (3 stations)
	cascade3 := model.CascadeReport{
		CascadeID:   3,
		CascadeName: "Қуйи Сирдарё каскади",
		Weather: &model.CascadeWeather{
			Temperature:         floatPtr(20.0),
			Condition:           strPtr("03d"),
			PrevYearTemperature: floatPtr(17.0),
			PrevYearCondition:   strPtr("01d"),
		},
		Summary: &model.SummaryBlock{
			InstalledCapacityMWt:  690,
			TotalAggregates:       18,
			WorkingAggregates:     15,
			PowerMWt:              520,
			DailyProductionMlnKWh: 10.0,
			ProductionChange:      0.5,
			MTDProductionMlnKWh:   125.0,
			YTDProductionMlnKWh:   720.0,
			MonthlyPlanMlnKWh:     140.0,
			QuarterlyPlanMlnKWh:   400.0,
			FulfillmentPct:        floatPtr(89.3),
			DifferenceMlnKWh:      -15.0,
			PrevYearYTD:           680.0,
			YoYGrowthRate:         floatPtr(105.9),
			YoYDifference:         40.0,
		},
		Stations: []model.StationReport{
			realStation(301, "Ўзбекистон ГЭС", 350, 8, 7, 6.5, 0.3,
				floatPtr(345.0), floatPtr(680.0), floatPtr(42.0), floatPtr(820.0), floatPtr(780.0), floatPtr(720.0), floatPtr(60.0)),
			realStation(302, "Фарҳод ГЭС", 186, 6, 5, 2.1, 0.1,
				floatPtr(330.0), floatPtr(420.0), floatPtr(22.0), floatPtr(650.0), floatPtr(600.0), floatPtr(560.0), floatPtr(40.0)),
			realStation(303, "Қайроққум ГЭС", 154, 4, 3, 1.4, 0.1,
				floatPtr(347.5), floatPtr(540.0), floatPtr(16.0), floatPtr(480.0), floatPtr(440.0), floatPtr(410.0), floatPtr(30.0)),
		},
	}

	grandTotal := &model.SummaryBlock{
		InstalledCapacityMWt:  1791,
		TotalAggregates:       46,
		WorkingAggregates:     38,
		PowerMWt:              1240,
		DailyProductionMlnKWh: 19.4,
		ProductionChange:      0.7,
		MTDProductionMlnKWh:   242.5,
		YTDProductionMlnKWh:   1410.0,
		MonthlyPlanMlnKWh:     270.0,
		QuarterlyPlanMlnKWh:   765.0,
		FulfillmentPct:        floatPtr(89.8),
		DifferenceMlnKWh:      -27.5,
		PrevYearYTD:           1330.0,
		YoYGrowthRate:         floatPtr(106.0),
		YoYDifference:         80.0,
		IdleDischargeM3s:      220.0,
	}

	// Plans
	ytdPlans := map[int64]float64{
		101: 420, 102: 95, 103: 60, 104: 50,
		201: 70, 202: 55,
		301: 240, 302: 130, 303: 100,
	}
	annualPlans := map[int64]float64{
		101: 500, 102: 120, 103: 80, 104: 70,
		201: 90, 202: 70,
		301: 300, 302: 160, 303: 130,
	}
	// sum = 1520  -- close enough for realistic test
	// Actually let's make it 1550
	annualPlans[101] = 530

	monthlyPlans := map[int64]float64{
		101: 42, 102: 10, 103: 7, 104: 6,
		201: 8, 202: 6,
		301: 28, 302: 14, 303: 11,
	}

	return ExcelParams{
		Report: &model.DailyReport{
			Date:       "2026-03-13",
			Cascades:   []model.CascadeReport{cascade1, cascade2, cascade3},
			GrandTotal: grandTotal,
		},
		YTDPlans:     ytdPlans,
		AnnualPlans:  annualPlans,
		MonthlyPlans: monthlyPlans,
		OrgTypeCounts: OrgTypeCounts{
			GES:   22,
			Mini:  10,
			Micro: 6,
			Total: 38,
		},
		Modernization:    4,
		Repair:           14,
		Date:             date,
		Loc:              loc,
		WeatherIconsPath: filepath.Join(resolveRepoRoot(t), "template", "weather-icons"),
	}
}

func strPtr(s string) *string { return &s }

// realStation creates a StationReport with realistic reservoir/flow data.
func realStation(
	orgID int64, name string, capacity float64, total, working int,
	daily, change float64,
	waterLevel, waterVolume, waterHead, income, totalOutflow, gesFlow, idleDisch *float64,
) model.StationReport {
	hasReservoir := waterLevel != nil

	var diffs model.DiffData
	diffs.ProductionChange = floatPtr(change)
	diffs.PowerChangeMWt = floatPtr(change * 50)
	if waterLevel != nil {
		diffs.LevelChangeCm = floatPtr(3.5)
		diffs.VolumeChangeMlnM3 = floatPtr(1.8)
	}
	if income != nil {
		diffs.IncomeChangeM3s = floatPtr(12.0)
	}
	if gesFlow != nil {
		diffs.GESFlowChangeM3s = floatPtr(8.0)
	}

	power := capacity * 0.65
	mtd := daily * 13
	ytd := daily * 72

	s := model.StationReport{
		OrganizationID: orgID,
		Name:           name,
		Config: model.StationConfig{
			InstalledCapacityMWt: capacity,
			TotalAggregates:      total,
			HasReservoir:         hasReservoir,
		},
		Current: model.CurrentData{
			DailyProductionMlnKWh: daily,
			PowerMWt:              power,
			WorkingAggregates:     working,
			WaterLevelM:           waterLevel,
			WaterVolumeMlnM3:      waterVolume,
			WaterHeadM:            waterHead,
			ReservoirIncomeM3s:    income,
			TotalOutflowM3s:       totalOutflow,
			GESFlowM3s:            gesFlow,
			IdleDischargeM3s:      idleDisch,
		},
		Diffs: diffs,
		Aggregations: model.Aggregations{
			MTDProductionMlnKWh: mtd,
			YTDProductionMlnKWh: ytd,
		},
		Plan: model.PlanData{
			MonthlyPlanMlnKWh:   daily * 30 * 1.05,
			QuarterlyPlanMlnKWh: daily * 90 * 1.05,
			FulfillmentPct:      floatPtr(mtd / (daily * 30 * 1.05) * 100),
			DifferenceMlnKWh:    mtd - daily*30*1.05*13/30,
		},
		PreviousYear: &model.PrevYearData{
			DailyProduction: floatPtr(daily * 0.92),
			MTDProduction:   mtd * 0.94,
			YTDProduction:   ytd * 0.94,
			PowerMWt:        floatPtr(power * 0.9),
		},
		YoY: model.YoYData{
			GrowthRate:       floatPtr(ytd / (ytd * 0.94) * 100),
			DifferenceMlnKWh: ytd - ytd*0.94,
		},
	}

	// Add prev year reservoir data if station has reservoir
	if waterLevel != nil {
		s.PreviousYear.WaterLevelM = floatPtr(*waterLevel - 2.5)
		s.PreviousYear.WaterVolumeMlnM3 = floatPtr(*waterVolume * 0.92)
		s.PreviousYear.WaterHeadM = floatPtr(*waterHead - 1.0)
	}
	if income != nil {
		s.PreviousYear.ReservoirIncomeM3s = floatPtr(*income * 0.88)
	}
	if gesFlow != nil {
		s.PreviousYear.GESFlowM3s = floatPtr(*gesFlow * 0.90)
	}

	return s
}

// resolveRepoRoot returns the repo root based on current test file location.
func resolveRepoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	// internal/lib/service/excel/ges/ -> 5 levels up
	return filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", "..")
}
