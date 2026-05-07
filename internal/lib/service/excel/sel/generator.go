// Package sel renders the "Тезкор маълумот" (operational flood report) Excel
// workbook from a parameterized template.
//
// The template (template/sel.xlsx) keeps two empty padding columns — A on the
// left and U on the right — so the printable area is wider than the data. This
// gives soffice's fit-to-page enough breathing room on Linux to reliably shrink
// the 19-column table onto a single landscape page (without padding, some
// soffice versions spill the rightmost data column onto a second page).
//
// Data lives in B..T:
//   - B = №,  C = name,  D/E = level prev/curr,  F/G = volume,  H/I = inflow,
//     J/K = outflow, L/M = ges flow, N/O = capacity, P/Q = idle discharge,
//     R = weather, S = temperature, T = duty.
//
// Header time/date are in T2/T3; per-column subheaders in row 5 derive the
// prev/curr hour from T2 via =MOD($T$2-TIME(1,0,0),1) and =$T$2.
//
// One 2-row block (rows 6-7) per reservoir: row 6 holds values, row 7 holds
// delta formulas (=IFERROR(E6-D6,"-"), etc.). Row 9 carries the signer line
// (F9:K9 hardcoded, N9:R9 holds the operator's short name).
//
// For N reservoirs, the generator clones the 2-row block N-1 times via
// DuplicateRowTo (rows 6+7, 8+9, 10+11, …); the signer row shifts down
// automatically. excelize copies formulas verbatim, so the generator
// rewrites each cloned block's delta formulas to reference its own value row.
package sel

import (
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"
)

// Template layout constants.
const (
	templateBlockStartRow = 6 // first row of the value block
	templateBlockSize     = 2 // value row + delta-formula row
	templateSignerRow     = 9 // F9:K9 + N9:R9 in the original template
)

// Generator renders the report.
type Generator struct {
	templatePath string
}

// New binds the generator to a template path.
func New(templatePath string) *Generator {
	return &Generator{templatePath: templatePath}
}

// Report bundles everything needed to fill the template.
type Report struct {
	Date        time.Time      // → T3 (mm-dd-yy)
	Hour        int            // → T2 (HH:00); also drives D5..Q5 via =MOD($T$2-TIME(1,0,0),1)
	AuthorShort string         // → N9 (or row 9 + (N-1)*2 after cloning)
	Reservoirs  []ReservoirRow // one entry per reservoir, rendered in order
}

// ReservoirRow holds one reservoir's values for the prev/curr hour pair.
// All numeric fields are nullable; nil → cell becomes "-". Strings: ""
// → cell becomes "-".
type ReservoirRow struct {
	Name              string   // C
	LevelPrev         *float64 // D
	LevelCurr         *float64 // E
	VolumePrev        *float64 // F
	VolumeCurr        *float64 // G
	InflowPrev        *float64 // H
	InflowCurr        *float64 // I
	OutflowPrev       *float64 // J
	OutflowCurr       *float64 // K
	GESFlowPrev       *float64 // L
	GESFlowCurr       *float64 // M
	CapacityPrev      *float64 // N
	CapacityCurr      *float64 // O
	IdleDischargePrev *float64 // P
	IdleDischargeCurr *float64 // Q
	WeatherCondition  string   // R (current hour only)
	TemperatureC      *float64 // S (current hour only)
	DutyName          string   // T (current hour only)
}

const dash = "-"

// GenerateExcel returns a populated workbook. Caller must Close() it.
func (g *Generator) GenerateExcel(rep *Report) (*excelize.File, error) {
	if rep == nil {
		return nil, fmt.Errorf("nil report")
	}
	f, err := excelize.OpenFile(g.templatePath)
	if err != nil {
		return nil, fmt.Errorf("open template: %w", err)
	}
	sheet := f.GetSheetList()[0]

	var writeErr error
	set := func(cell string, value any) {
		if writeErr != nil {
			return
		}
		if err := f.SetCellValue(sheet, cell, value); err != nil {
			writeErr = fmt.Errorf("set %s: %w", cell, err)
		}
	}
	setNum := func(cell string, v *float64) {
		if v == nil {
			set(cell, dash)
			return
		}
		set(cell, *v)
	}
	setStr := func(cell string, v string) {
		if v == "" {
			set(cell, dash)
			return
		}
		set(cell, v)
	}

	// Header: T2 (time-of-day) and T3 (date).
	// excelize writes time.Time under the existing [$-10819]hh:mm;@ format
	// correctly; the year/month/day are irrelevant — only the hour matters.
	set("T2", time.Date(2000, 1, 1, rep.Hour, 0, 0, 0, time.UTC))
	set("T3", rep.Date)

	// Phase 1: clone the 2-row block for every reservoir past the first.
	// DuplicateRowTo copies cell values, formulas, AND styling — but it does
	// NOT rewrite formula cell references (D9 ends up holding the literal
	// "IFERROR(E6-D6,\"-\")" instead of the expected "IFERROR(E8-D8,\"-\")").
	// We fix that in Phase 1b by rewriting each cloned block's delta formulas
	// with the correct row reference. The signer row 9 (F9:K9 + N9:R9) does
	// shift down automatically because its merge range moves with the rows.
	n := len(rep.Reservoirs)
	for i := 1; i < n; i++ {
		targetBase := templateBlockStartRow + i*templateBlockSize
		for j := 0; j < templateBlockSize; j++ {
			if err := f.DuplicateRowTo(sheet, templateBlockStartRow+j, targetBase+j); err != nil {
				_ = f.Close()
				return nil, fmt.Errorf("duplicate block %d row %d: %w", i, j, err)
			}
		}
	}

	// Phase 1b: rewrite delta formulas in the cloned blocks. The 7 delta cells
	// in row 7 reference adjacent paired columns: D7=E6-D6, F7=G6-F6, etc.
	// Use literal column letters to keep the rewrite explicit.
	deltaPairs := [][2]string{{"D", "E"}, {"F", "G"}, {"H", "I"}, {"J", "K"}, {"L", "M"}, {"N", "O"}, {"P", "Q"}}
	for i := 1; i < n; i++ {
		valueRow := templateBlockStartRow + i*templateBlockSize
		deltaRow := valueRow + 1
		for _, p := range deltaPairs {
			cell := fmt.Sprintf("%s%d", p[0], deltaRow)
			// Match the template's existing formula style (space after the
			// comma) so the rendered file looks uniform across blocks.
			formula := fmt.Sprintf(`IFERROR(%s%d-%s%d, "-")`, p[1], valueRow, p[0], valueRow)
			if err := f.SetCellFormula(sheet, cell, formula); err != nil {
				_ = f.Close()
				return nil, fmt.Errorf("set delta formula %s: %w", cell, err)
			}
		}
	}

	// Phase 1c: re-create vertical merges for cloned blocks. DuplicateRowTo
	// replicates horizontal merges (the D7:E7 delta-formula cells) but drops
	// the value-row + delta-row vertical pairs (B6:B7, C6:C7, R6:R7, S6:S7,
	// T6:T7). Without this every name cell in cloned blocks would render as
	// one row tall while block 1's name spans two rows — visibly broken.
	verticalMergeCols := []string{"B", "C", "R", "S", "T"}
	for i := 1; i < n; i++ {
		valueRow := templateBlockStartRow + i*templateBlockSize
		deltaRow := valueRow + 1
		for _, col := range verticalMergeCols {
			start := fmt.Sprintf("%s%d", col, valueRow)
			end := fmt.Sprintf("%s%d", col, deltaRow)
			if err := f.MergeCell(sheet, start, end); err != nil {
				_ = f.Close()
				return nil, fmt.Errorf("merge %s:%s: %w", start, end, err)
			}
		}
	}

	// Phase 2: fill each block's value row.
	for i, res := range rep.Reservoirs {
		row := templateBlockStartRow + i*templateBlockSize
		rs := fmt.Sprintf("%d", row)
		set("B"+rs, i+1)
		setStr("C"+rs, res.Name)

		setNum("D"+rs, res.LevelPrev)
		setNum("E"+rs, res.LevelCurr)
		setNum("F"+rs, res.VolumePrev)
		setNum("G"+rs, res.VolumeCurr)
		setNum("H"+rs, res.InflowPrev)
		setNum("I"+rs, res.InflowCurr)
		setNum("J"+rs, res.OutflowPrev)
		setNum("K"+rs, res.OutflowCurr)
		setNum("L"+rs, res.GESFlowPrev)
		setNum("M"+rs, res.GESFlowCurr)
		setNum("N"+rs, res.CapacityPrev)
		setNum("O"+rs, res.CapacityCurr)
		setNum("P"+rs, res.IdleDischargePrev)
		setNum("Q"+rs, res.IdleDischargeCurr)

		setStr("R"+rs, res.WeatherCondition)
		setNum("S"+rs, res.TemperatureC)
		setStr("T"+rs, res.DutyName)
	}

	// Phase 3: signer name in the (shifted) N9 merge (top-left of N9:R9).
	signerRow := templateSignerRow
	if n > 1 {
		signerRow += (n - 1) * templateBlockSize
	}
	set(fmt.Sprintf("N%d", signerRow), rep.AuthorShort)

	// Phase 4: rebind print_area to the actual last content row. The template
	// ships with a static $B$1:$U$25 (sized for 9 reservoirs) so soffice would
	// otherwise either clip taller reports or leave a tail of empty rows
	// when N < 9. We extend left to A as well — both empty padding columns
	// (A and U) live in the print area to give soffice's fit-to-page reliable
	// breathing room on both sides. SetDefinedName refuses duplicates, so
	// delete first.
	printAreaRef := fmt.Sprintf("'%s'!$A$1:$U$%d", sheet, signerRow)
	_ = f.DeleteDefinedName(&excelize.DefinedName{Name: "_xlnm.Print_Area", Scope: sheet})
	if err := f.SetDefinedName(&excelize.DefinedName{
		Name:     "_xlnm.Print_Area",
		RefersTo: printAreaRef,
		Scope:    sheet,
	}); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("set print_area: %w", err)
	}

	if writeErr != nil {
		_ = f.Close()
		return nil, writeErr
	}

	if err := f.UpdateLinkedValue(); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("recalculate formulas: %w", err)
	}
	return f, nil
}
