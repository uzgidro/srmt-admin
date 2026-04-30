package ownneeds

import (
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	model "srmt-admin/internal/lib/model/ges-report"
)

const templatePath = "../../../../../template/own-needs.xlsx"

func ptr(v float64) *float64 { return &v }

func approxEqual(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

// isCellBold returns true if the cell's resolved style has a bold font.
// Used to assert that the cascade-summary row keeps the template's bold
// formatting and station rows stay non-bold.
func isCellBold(f *excelize.File, sheet, cell string) (bool, error) {
	styleID, err := f.GetCellStyle(sheet, cell)
	if err != nil {
		return false, err
	}
	style, err := f.GetStyle(styleID)
	if err != nil {
		return false, err
	}
	if style == nil || style.Font == nil {
		return false, nil
	}
	return style.Font.Bold, nil
}

func mustFloat(t *testing.T, f *excelize.File, sheet, cell string) float64 {
	t.Helper()
	s, err := f.GetCellValue(sheet, cell)
	if err != nil {
		t.Fatalf("GetCellValue %s: %v", cell, err)
	}
	if s == "" {
		t.Fatalf("cell %s is empty", cell)
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		t.Fatalf("parse %s=%q: %v", cell, s, err)
	}
	return v
}

// TestGenerator_OneCascadeTwoStations_ProducesCorrectCells verifies the
// generator inserts cascade + station rows in order, fills 16 columns per
// row, and shifts the grand total accordingly.
func TestGenerator_OneCascadeTwoStations_ProducesCorrectCells(t *testing.T) {
	rep := &model.OwnNeedsReport{
		Date: "2026-04-27",
		Cascades: []model.OwnNeedsCascade{
			{
				CascadeID:   1,
				CascadeName: "Cascade A",
				Stations: []model.OwnNeedsStation{
					{
						OrganizationID:        100, Name: "Station One",
						InstalledCapacityMWt:  10.0, MonthlyPlanMlnKWh: 6.5,
						CumulativePlanMlnKWh:  20.0, DailyProductionMlnKWh: 2.5,
						DailyProductionDelta:  ptr(0.5),
						MTDProductionMlnKWh:   50, YTDProductionMlnKWh: 200,
						OwnConsumptionKWh:     ptr(500.0), OwnConsumptionDelta: ptr(20.0),
						MTDOwnConsumptionKWh:  12000, YTDOwnConsumptionKWh: 60000,
					},
					{
						OrganizationID:        101, Name: "Station Two",
						InstalledCapacityMWt:  5.0, MonthlyPlanMlnKWh: 2.5,
						CumulativePlanMlnKWh:  10.0, DailyProductionMlnKWh: 1.0,
						DailyProductionDelta:  ptr(-0.1),
						MTDProductionMlnKWh:   20, YTDProductionMlnKWh: 80,
						OwnConsumptionKWh:     ptr(200.0), OwnConsumptionDelta: ptr(-5.0),
						MTDOwnConsumptionKWh:  5000, YTDOwnConsumptionKWh: 25000,
					},
				},
				Totals: model.OwnNeedsTotals{
					InstalledCapacityMWt: 15.0, MonthlyPlanMlnKWh: 9.0,
					CumulativePlanMlnKWh: 30.0, DailyProductionMlnKWh: 3.5,
					DailyProductionDelta: 0.4,
					MTDProductionMlnKWh:  70, YTDProductionMlnKWh: 280,
					OwnConsumptionKWh:    700.0, OwnConsumptionDelta: 15.0,
					MTDOwnConsumptionKWh: 17000, YTDOwnConsumptionKWh: 85000,
				},
			},
		},
		GrandTotal: model.OwnNeedsTotals{
			InstalledCapacityMWt: 15.0, MonthlyPlanMlnKWh: 9.0,
			CumulativePlanMlnKWh: 30.0, DailyProductionMlnKWh: 3.5,
			DailyProductionDelta: 0.4,
			MTDProductionMlnKWh:  70, YTDProductionMlnKWh: 280,
			OwnConsumptionKWh:    700.0, OwnConsumptionDelta: 15.0,
			MTDOwnConsumptionKWh: 17000, YTDOwnConsumptionKWh: 85000,
		},
	}

	g := New(templatePath)
	f, err := g.GenerateExcel(Params{Report: rep, Date: time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("GenerateExcel: %v", err)
	}
	defer f.Close()

	// Sheet rename to DD.MM.YY.
	sheets := f.GetSheetList()
	if len(sheets) != 1 || sheets[0] != "27.04.26" {
		t.Fatalf("sheet name: got %v, want [27.04.26]", sheets)
	}
	sheet := sheets[0]

	// Row 6 = cascade.
	if got, _ := f.GetCellValue(sheet, "A6"); got != "Cascade A" {
		t.Errorf("A6 cascade name: got %q, want Cascade A", got)
	}
	if v := mustFloat(t, f, sheet, "B6"); !approxEqual(v, 15.0) {
		t.Errorf("B6 cascade capacity: got %v, want 15", v)
	}
	if v := mustFloat(t, f, sheet, "I6"); !approxEqual(v, 700.0) {
		t.Errorf("I6 cascade own_consumption: got %v, want 700", v)
	}
	// M6 = 700/15 ≈ 46.6666... (template format may round display).
	if v := mustFloat(t, f, sheet, "M6"); math.Abs(v-700.0/15.0) > 0.01 {
		t.Errorf("M6 per-kW: got %v, want ≈ %v", v, 700.0/15.0)
	}

	// Row 7 = first station.
	if got, _ := f.GetCellValue(sheet, "A7"); got != "Station One" {
		t.Errorf("A7 station name: got %q, want Station One", got)
	}
	if v := mustFloat(t, f, sheet, "I7"); !approxEqual(v, 500.0) {
		t.Errorf("I7 station own_consumption: got %v, want 500", v)
	}
	// M7 = 500/10 = 50
	if v := mustFloat(t, f, sheet, "M7"); !approxEqual(v, 50.0) {
		t.Errorf("M7 per-kW: got %v, want 50", v)
	}

	// Row 8 = second station.
	if got, _ := f.GetCellValue(sheet, "A8"); got != "Station Two" {
		t.Errorf("A8 station name: got %q, want Station Two", got)
	}
	// M8 = 200/5 = 40
	if v := mustFloat(t, f, sheet, "M8"); !approxEqual(v, 40.0) {
		t.Errorf("M8 per-kW: got %v, want 40", v)
	}

	// Grand total shifted from row 8 to row 9: one cascade contributes no
	// extra block rows; one extra station (2 stations - 1) shifts grand by 1.
	if got, _ := f.GetCellValue(sheet, "A9"); !strings.Contains(got, "Ўзбекгидроэнерго") {
		t.Errorf("A9 grand-total caption: got %q, want contains 'Ўзбекгидроэнерго'", got)
	}
	if v := mustFloat(t, f, sheet, "I9"); !approxEqual(v, 700.0) {
		t.Errorf("I9 grand-total own_consumption: got %v, want 700", v)
	}
}

// TestGenerator_CascadeRowKeepsBoldStyle verifies that the cascade-summary
// row (row 6 template) preserves bold formatting from the template, and the
// station row (row 7 template) stays non-bold. This guards against the older
// bug where every body row was DuplicateRow'd from row 6, blanket-applying
// bold to all stations.
func TestGenerator_CascadeRowKeepsBoldStyle(t *testing.T) {
	rep := &model.OwnNeedsReport{
		Date: "2026-04-27",
		Cascades: []model.OwnNeedsCascade{{
			CascadeID: 1, CascadeName: "Cascade A",
			Stations: []model.OwnNeedsStation{{
				OrganizationID: 1, Name: "Station A",
				InstalledCapacityMWt: 5.0, OwnConsumptionKWh: ptr(50.0),
			}},
			Totals: model.OwnNeedsTotals{InstalledCapacityMWt: 5.0, OwnConsumptionKWh: 50.0},
		}},
	}
	g := New(templatePath)
	f, err := g.GenerateExcel(Params{Report: rep, Date: time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("GenerateExcel: %v", err)
	}
	defer f.Close()
	sheet := f.GetSheetList()[0]

	cascadeBold, err := isCellBold(f, sheet, "A6")
	if err != nil {
		t.Fatalf("isCellBold A6: %v", err)
	}
	if !cascadeBold {
		t.Errorf("A6 (cascade row) expected bold, got non-bold")
	}
	stationBold, err := isCellBold(f, sheet, "A7")
	if err != nil {
		t.Fatalf("isCellBold A7: %v", err)
	}
	if stationBold {
		t.Errorf("A7 (station row) expected non-bold, got bold")
	}
}

// TestGenerator_TwoCascadesEachWithStations verifies that the per-cascade
// (cascade, station) block is reproduced for every cascade so each cascade
// gets its own bold summary row followed by its station rows in the right
// order, and grand-total lands at the right shifted row.
func TestGenerator_TwoCascadesEachWithStations(t *testing.T) {
	rep := &model.OwnNeedsReport{
		Date: "2026-04-27",
		Cascades: []model.OwnNeedsCascade{
			{
				CascadeID: 1, CascadeName: "Cascade A",
				Stations: []model.OwnNeedsStation{
					{OrganizationID: 100, Name: "A1", InstalledCapacityMWt: 5, OwnConsumptionKWh: ptr(10.0)},
					{OrganizationID: 101, Name: "A2", InstalledCapacityMWt: 5, OwnConsumptionKWh: ptr(20.0)},
				},
				Totals: model.OwnNeedsTotals{InstalledCapacityMWt: 10, OwnConsumptionKWh: 30},
			},
			{
				CascadeID: 2, CascadeName: "Cascade B",
				Stations: []model.OwnNeedsStation{
					{OrganizationID: 200, Name: "B1", InstalledCapacityMWt: 7, OwnConsumptionKWh: ptr(40.0)},
				},
				Totals: model.OwnNeedsTotals{InstalledCapacityMWt: 7, OwnConsumptionKWh: 40},
			},
		},
		GrandTotal: model.OwnNeedsTotals{InstalledCapacityMWt: 17, OwnConsumptionKWh: 70},
	}
	g := New(templatePath)
	f, err := g.GenerateExcel(Params{Report: rep, Date: time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("GenerateExcel: %v", err)
	}
	defer f.Close()
	sheet := f.GetSheetList()[0]

	// Layout: 6=Cascade A, 7=A1, 8=A2, 9=Cascade B, 10=B1, 11=grand.
	// Inserted rows = (2-1)*2 + (1) extra A2 station + (0) for B = 3.
	// Grand = 8 + 3 = 11.
	cases := []struct {
		cell string
		want string
	}{
		{"A6", "Cascade A"},
		{"A7", "A1"},
		{"A8", "A2"},
		{"A9", "Cascade B"},
		{"A10", "B1"},
	}
	for _, c := range cases {
		if got, _ := f.GetCellValue(sheet, c.cell); got != c.want {
			t.Errorf("%s: got %q, want %q", c.cell, got, c.want)
		}
	}
	if got, _ := f.GetCellValue(sheet, "A11"); !strings.Contains(got, "Ўзбекгидроэнерго") {
		t.Errorf("A11 grand-total caption: got %q, want contains 'Ўзбекгидроэнерго'", got)
	}
	if v := mustFloat(t, f, sheet, "I11"); !approxEqual(v, 70.0) {
		t.Errorf("I11 grand-total own_consumption: got %v, want 70", v)
	}

	// Each cascade row stays bold; each station row stays non-bold.
	for _, cell := range []string{"A6", "A9", "A11"} {
		b, err := isCellBold(f, sheet, cell)
		if err != nil {
			t.Fatalf("isCellBold %s: %v", cell, err)
		}
		if !b {
			t.Errorf("%s expected bold", cell)
		}
	}
	for _, cell := range []string{"A7", "A8", "A10"} {
		b, err := isCellBold(f, sheet, cell)
		if err != nil {
			t.Fatalf("isCellBold %s: %v", cell, err)
		}
		if b {
			t.Errorf("%s expected non-bold", cell)
		}
	}
}

// TestGenerator_DivisionByZeroCapacity verifies that stations with zero
// installed capacity (e.g. dam-site reservoirs) leave per-kW cells empty.
func TestGenerator_DivisionByZeroCapacity(t *testing.T) {
	rep := &model.OwnNeedsReport{
		Date: "2026-04-27",
		Cascades: []model.OwnNeedsCascade{
			{
				CascadeID:   1, CascadeName: "C",
				Stations: []model.OwnNeedsStation{
					{
						OrganizationID: 1, Name: "ZeroCap",
						InstalledCapacityMWt: 0,
						OwnConsumptionKWh:    ptr(100.0),
						MTDOwnConsumptionKWh: 1000, YTDOwnConsumptionKWh: 5000,
					},
				},
				Totals: model.OwnNeedsTotals{InstalledCapacityMWt: 0, OwnConsumptionKWh: 100, MTDOwnConsumptionKWh: 1000, YTDOwnConsumptionKWh: 5000},
			},
		},
	}

	g := New(templatePath)
	f, err := g.GenerateExcel(Params{Report: rep, Date: time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("GenerateExcel: %v", err)
	}
	defer f.Close()
	sheet := f.GetSheetList()[0]

	for _, cell := range []string{"M7", "N7", "O7", "P7"} {
		if got, _ := f.GetCellValue(sheet, cell); got != "" {
			t.Errorf("%s expected empty (capacity=0), got %q", cell, got)
		}
	}
}

// TestGenerator_DateInHeader verifies that the date and Uzbek month name
// land in the right cells.
func TestGenerator_DateInHeader(t *testing.T) {
	rep := &model.OwnNeedsReport{Date: "2026-04-27"}
	g := New(templatePath)
	f, err := g.GenerateExcel(Params{Report: rep, Date: time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("GenerateExcel: %v", err)
	}
	defer f.Close()
	sheet := f.GetSheetList()[0]

	if got, _ := f.GetCellValue(sheet, "D4"); got != "2026 йил" {
		t.Errorf("D4 year header: got %q, want '2026 йил'", got)
	}
	if got, _ := f.GetCellValue(sheet, "C5"); !strings.Contains(got, "Апрель") {
		t.Errorf("C5 monthly caption: got %q, want contains 'Апрель'", got)
	}
	if got, _ := f.GetCellValue(sheet, "D5"); !strings.Contains(got, "Январь-Апрель") {
		t.Errorf("D5 cumulative caption: got %q, want contains 'Январь-Апрель'", got)
	}
}

// TestGenerator_NilOptionalsLeaveCellsEmpty verifies that a station with nil
// own_consumption and nil delta leaves I/J/M/N empty (rather than writing 0).
func TestGenerator_NilOptionalsLeaveCellsEmpty(t *testing.T) {
	rep := &model.OwnNeedsReport{
		Date: "2026-04-27",
		Cascades: []model.OwnNeedsCascade{{
			CascadeID: 1, CascadeName: "C",
			Stations: []model.OwnNeedsStation{{
				OrganizationID: 1, Name: "NoData",
				InstalledCapacityMWt: 5.0,
				// OwnConsumptionKWh, OwnConsumptionDelta, DailyProductionDelta nil
			}},
			Totals: model.OwnNeedsTotals{InstalledCapacityMWt: 5.0},
		}},
	}
	g := New(templatePath)
	f, err := g.GenerateExcel(Params{Report: rep, Date: time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("GenerateExcel: %v", err)
	}
	defer f.Close()
	sheet := f.GetSheetList()[0]

	// Row 7 is the station row (row 6 is cascade).
	for _, cell := range []string{"F7", "I7", "J7", "M7", "N7"} {
		if got, _ := f.GetCellValue(sheet, cell); got != "" {
			t.Errorf("%s expected empty (nil pointer), got %q", cell, got)
		}
	}
}
