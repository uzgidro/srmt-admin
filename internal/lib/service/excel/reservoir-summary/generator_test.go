package reservoirsummary

import (
	"strconv"
	"strings"
	"testing"

	reservoirsummarymodel "srmt-admin/internal/lib/model/reservoir-summary"
	"srmt-admin/internal/lib/service/excel/templates"
)

// fixtureData mirrors what the repo returns: 8 reservoirs in the NEW sort
// order (Чотқол at index 6, Пском at index 7) plus the ИТОГО row
// (OrganizationID == nil). The generator is expected to drop the ИТОГО row
// silently — Excel now owns summation via SUM formulas in rows 20-21 of both
// templates.
func fixtureData() []*reservoirsummarymodel.ResponseModel {
	id := func(n int64) *int64 { return &n }
	mk := func(orgID *int64, lvl, vol float64) *reservoirsummarymodel.ResponseModel {
		return &reservoirsummarymodel.ResponseModel{
			OrganizationID: orgID,
			Level:          reservoirsummarymodel.ValueResponse{Current: lvl, Previous: lvl - 5},
			Volume:         reservoirsummarymodel.ValueResponse{Current: vol},
			Modsnow:        reservoirsummarymodel.ValueResponse{Current: lvl * 0.1, YearAgo: lvl * 0.05},
		}
	}
	return []*reservoirsummarymodel.ResponseModel{
		mk(id(1), 100, 10),  // Андижон   → rows 6/7
		mk(id(2), 110, 11),  // Оҳангарон → 8/9
		mk(id(3), 120, 12),  // Сардоба   → 10/11
		mk(id(4), 130, 13),  // Ҳисорак   → 12/13
		mk(id(5), 140, 14),  // Тўполанг  → 14/15
		mk(id(6), 150, 15),  // Чорвоқ    → 16/17
		mk(id(7), 210, 1.5), // Чотқол    → 18/19 (NEW slot)
		mk(id(8), 300, 30),  // Пском     → 22/23 (NEW slot)
		mk(nil, 0, 999),     // ИТОГО     → must NOT appear in sheet
	}
}

// allModsnowEnabledConfig returns a config map where every org in the
// fixture has modsnow_enabled=true — the default behaviour for the
// existing layout/regression tests that don't care about modsnow gating.
func allModsnowEnabledConfig() map[int64]reservoirsummarymodel.ReservoirSummaryConfig {
	out := map[int64]reservoirsummarymodel.ReservoirSummaryConfig{}
	for _, orgID := range []int64{1, 2, 3, 4, 5, 6, 7, 8} {
		out[orgID] = reservoirsummarymodel.ReservoirSummaryConfig{
			OrganizationID: orgID,
			ModsnowEnabled: true,
		}
	}
	return out
}

// readNumeric pulls a cell value and parses it to a float, tolerating the
// locale-specific decimal separator excelize may emit ("," vs ".") and the
// signed-number format prefix ("+5,00"). Returns false if the cell is empty
// or unparseable so callers can fail with a useful message.
func readNumeric(t *testing.T, getCell func(string) (string, error), coord string) (float64, string, bool) {
	t.Helper()
	raw, err := getCell(coord)
	if err != nil {
		t.Fatalf("read %s: %v", coord, err)
	}
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "+")
	cleaned = strings.ReplaceAll(cleaned, ",", ".")
	v, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0, raw, false
	}
	return v, raw, true
}

// Чотқол must land in rows 18/19 (was 22/23 in the old layout). The old
// in-Go ИТОГО writer used to overwrite C18 with the total volume — the C18
// check catches that regression too. We parse the cell value numerically so
// the test is agnostic to the template's display format (0.00, signed, etc).
func TestGenerateExcel_ChotkolInRows18and19(t *testing.T) {
	g := New("", templates.ResSummary)
	f, err := g.GenerateExcel("2025-12-16", fixtureData(), allModsnowEnabledConfig(), "Test")
	if err != nil {
		t.Fatalf("GenerateExcel: %v", err)
	}
	defer f.Close()
	sheet := f.GetSheetName(0)
	get := func(c string) (string, error) { return f.GetCellValue(sheet, c) }

	if v, raw, ok := readNumeric(t, get, "B18"); !ok || v != 210 {
		t.Errorf("B18 (Чотқол level): want 210, got %q (parsed=%v)", raw, v)
	}
	if v, raw, ok := readNumeric(t, get, "B19"); !ok || v != 5 {
		t.Errorf("B19 (Чотқол level diff): want 5, got %q (parsed=%v)", raw, v)
	}
	if v, raw, ok := readNumeric(t, get, "C18"); !ok || v != 1.5 {
		t.Errorf("C18 (Чотқол volume): want 1.5, got %q (parsed=%v) — likely ИТОГО overwrote", raw, v)
	}
}

// Пском moved from rows 20/21 to 22/23.
func TestGenerateExcel_PskomInRows22and23(t *testing.T) {
	g := New("", templates.ResSummary)
	f, err := g.GenerateExcel("2025-12-16", fixtureData(), allModsnowEnabledConfig(), "Test")
	if err != nil {
		t.Fatalf("GenerateExcel: %v", err)
	}
	defer f.Close()
	sheet := f.GetSheetName(0)
	get := func(c string) (string, error) { return f.GetCellValue(sheet, c) }

	if v, raw, ok := readNumeric(t, get, "B22"); !ok || v != 300 {
		t.Errorf("B22 (Пском level): want 300, got %q (parsed=%v)", raw, v)
	}
}

// Generator must NOT touch the JAMI formula cells. GetCellFormula returns
// the formula string for a formula cell, empty string for a plain-value
// cell. If the generator wrote a number there via SetCellValue, the formula
// is destroyed. Sample the highest-risk overwrite targets (C/F/I/L/M in
// rows 20-21).
func TestGenerateExcel_JamiFormulasUntouched(t *testing.T) {
	g := New("", templates.ResSummary)
	f, err := g.GenerateExcel("2025-12-16", fixtureData(), allModsnowEnabledConfig(), "Test")
	if err != nil {
		t.Fatalf("GenerateExcel: %v", err)
	}
	defer f.Close()
	sheet := f.GetSheetName(0)

	for _, coord := range []string{"C20", "F20", "I20", "L20", "M20", "C21", "F21", "I21", "L21", "M21"} {
		formula, err := f.GetCellFormula(sheet, coord)
		if err != nil {
			t.Fatalf("GetCellFormula %s: %v", coord, err)
		}
		if formula == "" {
			t.Errorf("%s: formula lost — generator overwrote a JAMI formula cell", coord)
		}
	}
}

// TestGenerateExcel_ModsnowSkippedByConfig — when an org has
// modsnow_enabled=false in its config, the generator must leave the
// modsnow cell empty (NOT zero, NOT the value from the data row). Org 1
// occupies row 6 → cells N6 (current) and O6 (year-ago).
func TestGenerateExcel_ModsnowSkippedByConfig(t *testing.T) {
	cfg := allModsnowEnabledConfig()
	c := cfg[1]
	c.ModsnowEnabled = false
	cfg[1] = c

	g := New("", templates.ResSummary)
	f, err := g.GenerateExcel("2025-12-16", fixtureData(), cfg, "Test")
	if err != nil {
		t.Fatalf("GenerateExcel: %v", err)
	}
	defer f.Close()
	sheet := f.GetSheetName(0)

	for _, coord := range []string{"N6", "O6"} {
		raw, err := f.GetCellValue(sheet, coord)
		if err != nil {
			t.Fatalf("read %s: %v", coord, err)
		}
		if strings.TrimSpace(raw) != "" {
			t.Errorf("%s: want empty (modsnow_enabled=false), got %q", coord, raw)
		}
	}
}

// TestGenerateExcel_ModsnowWrittenWhenEnabled — the same slot must
// receive the Modsnow.Current value when the org has modsnow_enabled=true.
// Asserts the positive branch to prevent a future bug that empties every
// cell because the lookup is wrong.
func TestGenerateExcel_ModsnowWrittenWhenEnabled(t *testing.T) {
	cfg := allModsnowEnabledConfig() // org 1 has enabled=true

	g := New("", templates.ResSummary)
	data := fixtureData()
	f, err := g.GenerateExcel("2025-12-16", data, cfg, "Test")
	if err != nil {
		t.Fatalf("GenerateExcel: %v", err)
	}
	defer f.Close()
	sheet := f.GetSheetName(0)
	get := func(c string) (string, error) { return f.GetCellValue(sheet, c) }

	wantCurrent := data[0].Modsnow.Current
	if v, raw, ok := readNumeric(t, get, "N6"); !ok || v != wantCurrent {
		t.Errorf("N6 (Андижон modsnow current): want %v, got %q (parsed=%v)", wantCurrent, raw, v)
	}
}

// TestGenerateExcel_SardobaSlotNoLongerHardcoded — historical behaviour
// was the hardcoded `if i == 2 { continue }` (Сардоба skip). After the
// hardcode is removed, the gating must come from config. When config
// allows it, the value lands in N10/O10 — proves the hardcode is truly gone.
func TestGenerateExcel_SardobaSlotNoLongerHardcoded(t *testing.T) {
	cfg := allModsnowEnabledConfig() // org 3 (Сардоба slot) explicit true

	g := New("", templates.ResSummary)
	data := fixtureData()
	f, err := g.GenerateExcel("2025-12-16", data, cfg, "Test")
	if err != nil {
		t.Fatalf("GenerateExcel: %v", err)
	}
	defer f.Close()
	sheet := f.GetSheetName(0)
	get := func(c string) (string, error) { return f.GetCellValue(sheet, c) }

	wantCurrent := data[2].Modsnow.Current
	if v, raw, ok := readNumeric(t, get, "N10"); !ok || v != wantCurrent {
		t.Errorf("N10 (Сардоба slot modsnow current with enabled=true): want %v, got %q (parsed=%v) — hardcode likely still present", wantCurrent, raw, v)
	}
}

// res-summary-filter.xlsx now shares the same row layout as res-summary.xlsx.
// One generator, two templates, identical slot mapping.
func TestGenerateExcel_FilterTemplate_SameLayout(t *testing.T) {
	g := New("", templates.ResSummaryFilt)
	f, err := g.GenerateExcel("2025-12-16", fixtureData(), allModsnowEnabledConfig(), "Test")
	if err != nil {
		t.Fatalf("GenerateExcel: %v", err)
	}
	defer f.Close()
	sheet := f.GetSheetName(0)
	get := func(c string) (string, error) { return f.GetCellValue(sheet, c) }

	if v, raw, ok := readNumeric(t, get, "B18"); !ok || v != 210 {
		t.Errorf("filter B18 (Чотқол): want 210, got %q (parsed=%v)", raw, v)
	}
	if v, raw, ok := readNumeric(t, get, "C18"); !ok || v != 1.5 {
		t.Errorf("filter C18 (Чотқол volume): want 1.5, got %q (parsed=%v) — ИТОГО overwrote", raw, v)
	}
}
