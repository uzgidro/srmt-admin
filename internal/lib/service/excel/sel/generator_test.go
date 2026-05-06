package sel

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// templatePath returns the absolute path to template/sel.xlsx so tests can
// run regardless of the working directory.
func templatePath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// .../internal/lib/service/excel/sel/generator_test.go → repo root → template/sel.xlsx
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "..")
	return filepath.Join(repoRoot, "template", "sel.xlsx")
}

func ptr(v float64) *float64 { return &v }

func TestGenerator_FillsHeaderS2S3(t *testing.T) {
	cases := []struct {
		name     string
		hour     int
		wantS2   string // text-formatted value as Excel renders
	}{
		{"hour 0", 0, "00:00"},
		{"hour 15", 15, "15:00"},
		{"hour 23", 23, "23:00"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := New(templatePath(t))
			date := time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC)
			f, err := g.GenerateExcel(&Report{Date: date, Hour: tc.hour, AuthorShort: "И. Иванов"})
			if err != nil {
				t.Fatalf("GenerateExcel: %v", err)
			}
			defer f.Close()
			sheet := f.GetSheetList()[0]
			gotS2, err := f.GetCellValue(sheet, "S2")
			if err != nil {
				t.Fatalf("GetCellValue S2: %v", err)
			}
			if gotS2 != tc.wantS2 {
				t.Errorf("S2: want %q, got %q", tc.wantS2, gotS2)
			}
			gotS3, err := f.GetCellValue(sheet, "S3")
			if err != nil {
				t.Fatalf("GetCellValue S3: %v", err)
			}
			// S3 format is mm-dd-yy, so "05-04-26".
			if gotS3 != "05-04-26" {
				t.Errorf("S3: want %q, got %q", "05-04-26", gotS3)
			}
		})
	}
}

func TestGenerator_OneRow(t *testing.T) {
	g := New(templatePath(t))
	row := ReservoirRow{
		Name:        "Чотқол",
		LevelPrev:   ptr(929.57),
		LevelCurr:   ptr(929.64),
		VolumePrev:  ptr(6.091),
		VolumeCurr:  ptr(6.12),
		InflowPrev:  ptr(205),
		InflowCurr:  ptr(205),
		OutflowPrev: ptr(218),
		OutflowCurr: ptr(158),
		DutyName:    "Иванов И.И.",
	}
	f, err := g.GenerateExcel(&Report{
		Date: time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC),
		Hour: 0, AuthorShort: "И. Иванов",
		Reservoirs: []ReservoirRow{row},
	})
	if err != nil {
		t.Fatalf("GenerateExcel: %v", err)
	}
	defer f.Close()
	sheet := f.GetSheetList()[0]

	expectCell := func(cell, want string) {
		t.Helper()
		got, err := f.GetCellValue(sheet, cell)
		if err != nil {
			t.Fatalf("GetCellValue %s: %v", cell, err)
		}
		if got != want {
			t.Errorf("%s: want %q, got %q", cell, want, got)
		}
	}

	expectCell("A6", "1")
	expectCell("B6", "Чотқол")
	expectCell("C6", "929.57")
	expectCell("D6", "929.64")
	expectCell("S6", "Иванов И.И.")
	// Signer in M9.
	expectCell("M9", "И. Иванов")
}

func TestGenerator_NineRows(t *testing.T) {
	g := New(templatePath(t))
	names := []string{"Чотқол", "Чорвоқ", "Пском", "Андижон", "Норин", "Оҳангарон", "Сардоба", "Ҳисорак", "Тўполанг"}
	rows := make([]ReservoirRow, 0, len(names))
	for _, n := range names {
		rows = append(rows, ReservoirRow{Name: n, LevelPrev: ptr(100), LevelCurr: ptr(101)})
	}
	f, err := g.GenerateExcel(&Report{
		Date: time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC),
		Hour: 0, AuthorShort: "И. Иванов",
		Reservoirs: rows,
	})
	if err != nil {
		t.Fatalf("GenerateExcel: %v", err)
	}
	defer f.Close()
	sheet := f.GetSheetList()[0]

	// First block at rows 6-7; ninth block at rows 22-23.
	v, _ := f.GetCellValue(sheet, "B6")
	if v != "Чотқол" {
		t.Errorf("B6: want Чотқол, got %q", v)
	}
	v, _ = f.GetCellValue(sheet, "B22")
	if v != "Тўполанг" {
		t.Errorf("B22: want Тўполанг, got %q", v)
	}
	v, _ = f.GetCellValue(sheet, "A22")
	if v != "9" {
		t.Errorf("A22: want 9, got %q", v)
	}
	// Signer row shifted from 9 to 9 + 8*2 = 25.
	v, _ = f.GetCellValue(sheet, "M25")
	if v != "И. Иванов" {
		t.Errorf("M25 (shifted signer): want И. Иванов, got %q", v)
	}
}

func TestGenerator_NilFieldsAreDash(t *testing.T) {
	g := New(templatePath(t))
	row := ReservoirRow{
		Name: "X",
		// All numeric and string fields left zero/nil.
	}
	f, err := g.GenerateExcel(&Report{
		Date: time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC),
		Hour: 0, AuthorShort: "И. Иванов",
		Reservoirs: []ReservoirRow{row},
	})
	if err != nil {
		t.Fatalf("GenerateExcel: %v", err)
	}
	defer f.Close()
	sheet := f.GetSheetList()[0]

	for _, cell := range []string{"C6", "D6", "E6", "F6", "G6", "H6", "I6", "J6", "K6", "L6", "M6", "N6", "O6", "P6", "Q6", "R6", "S6"} {
		got, err := f.GetCellValue(sheet, cell)
		if err != nil {
			t.Fatalf("GetCellValue %s: %v", cell, err)
		}
		if got != "-" {
			t.Errorf("%s: want %q (dash for missing data), got %q", cell, "-", got)
		}
	}
}

// excelize does not evaluate formulas on Save() — Excel/LibreOffice does that
// at open time. So instead of reading computed cell values for delta cells, we
// inspect the formula text itself: it must remain in place AND must reference
// the value-row of THIS block (not the original template row 6) after cloning.

func TestGenerator_DeltaFormulaPresent(t *testing.T) {
	g := New(templatePath(t))
	row := ReservoirRow{Name: "X", LevelPrev: nil, LevelCurr: ptr(929.64)}
	f, err := g.GenerateExcel(&Report{
		Date: time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC),
		Hour: 0, AuthorShort: "И. Иванов",
		Reservoirs: []ReservoirRow{row},
	})
	if err != nil {
		t.Fatalf("GenerateExcel: %v", err)
	}
	defer f.Close()
	sheet := f.GetSheetList()[0]

	formula, err := f.GetCellFormula(sheet, "C7")
	if err != nil {
		t.Fatalf("GetCellFormula C7: %v", err)
	}
	// Whitespace inside the formula varies between template-preserved and
	// generator-rewritten cells (excelize strips spaces on rewrite). Assert
	// the load-bearing parts: the IFERROR wrapper, the right cell refs, and
	// the dash literal.
	if !strings.Contains(formula, "IFERROR") ||
		!strings.Contains(formula, "D6-C6") ||
		!strings.Contains(formula, `"-"`) {
		t.Errorf("C7 formula: want IFERROR(D6-C6,...,\"-\"), got %q", formula)
	}
}

func TestGenerator_DuplicateRowPreservesDeltaFormula(t *testing.T) {
	g := New(templatePath(t))
	rows := []ReservoirRow{
		{Name: "First", LevelPrev: ptr(100), LevelCurr: ptr(105)},
		{Name: "Second", LevelPrev: ptr(200), LevelCurr: ptr(208)},
	}
	f, err := g.GenerateExcel(&Report{
		Date: time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC),
		Hour: 0, AuthorShort: "И. Иванов",
		Reservoirs: rows,
	})
	if err != nil {
		t.Fatalf("GenerateExcel: %v", err)
	}
	defer f.Close()
	sheet := f.GetSheetList()[0]

	// First block: delta formula in C7 must reference D6-C6 (template default).
	first, err := f.GetCellFormula(sheet, "C7")
	if err != nil {
		t.Fatalf("GetCellFormula C7: %v", err)
	}
	if !strings.Contains(first, "D6-C6") || !strings.Contains(first, "IFERROR") {
		t.Errorf("C7 formula (block 1): want IFERROR with D6-C6, got %q", first)
	}
	// Second block: delta formula in C9 must reference D8-C8 (rewritten by
	// generator after DuplicateRowTo, which copies references verbatim).
	second, err := f.GetCellFormula(sheet, "C9")
	if err != nil {
		t.Fatalf("GetCellFormula C9: %v", err)
	}
	if !strings.Contains(second, "D8-C8") {
		t.Errorf("C9 formula (block 2, generator-rewritten): want IFERROR with D8-C8, got %q", second)
	}
	if strings.Contains(second, "D6-C6") {
		t.Errorf("C9 formula MUST NOT still reference D6-C6 (would mean generator forgot to rewrite): got %q", second)
	}
}

func TestGenerator_PrintAreaTracksRowCount(t *testing.T) {
	cases := []struct {
		name     string
		count    int
		wantLast int // signer row = 9 + (count-1)*2
	}{
		{"one reservoir", 1, 9},
		{"three reservoirs", 3, 13},
		{"nine reservoirs", 9, 25},
		{"twelve reservoirs", 12, 31}, // larger than template's static A1:S25
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := New(templatePath(t))
			rows := make([]ReservoirRow, tc.count)
			for i := range rows {
				rows[i] = ReservoirRow{Name: fmt.Sprintf("R%d", i+1)}
			}
			f, err := g.GenerateExcel(&Report{
				Date: time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC),
				Hour: 0, AuthorShort: "И. Иванов",
				Reservoirs: rows,
			})
			if err != nil {
				t.Fatalf("GenerateExcel: %v", err)
			}
			defer f.Close()

			defined := f.GetDefinedName()
			var got string
			for _, dn := range defined {
				if dn.Name == "_xlnm.Print_Area" {
					got = dn.RefersTo
					break
				}
			}
			if got == "" {
				t.Fatalf("print_area not set; defined names = %+v", defined)
			}
			want := fmt.Sprintf("$A$1:$S$%d", tc.wantLast)
			if !strings.Contains(got, want) {
				t.Errorf("print_area: want suffix %q, got %q", want, got)
			}
		})
	}
}

func TestGenerator_NewFields(t *testing.T) {
	g := New(templatePath(t))
	row := ReservoirRow{
		Name:             "X",
		CapacityPrev:     ptr(100.5),
		CapacityCurr:     ptr(98.2),
		WeatherCondition: "облачно",
		TemperatureC:     ptr(-3.5),
	}
	f, err := g.GenerateExcel(&Report{
		Date: time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC),
		Hour: 0, AuthorShort: "И. Иванов",
		Reservoirs: []ReservoirRow{row},
	})
	if err != nil {
		t.Fatalf("GenerateExcel: %v", err)
	}
	defer f.Close()
	sheet := f.GetSheetList()[0]

	v, _ := f.GetCellValue(sheet, "M6")
	if v != "100.5" {
		t.Errorf("M6 capacity prev: want 100.5, got %q", v)
	}
	v, _ = f.GetCellValue(sheet, "N6")
	if v != "98.2" {
		t.Errorf("N6 capacity curr: want 98.2, got %q", v)
	}
	v, _ = f.GetCellValue(sheet, "Q6")
	if v != "облачно" {
		t.Errorf("Q6 weather: want облачно, got %q", v)
	}
	v, _ = f.GetCellValue(sheet, "R6")
	if v != "-3.5" {
		t.Errorf("R6 temperature (negative valid): want -3.5, got %q", v)
	}
}
