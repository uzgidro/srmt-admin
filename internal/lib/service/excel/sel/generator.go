// Package sel renders the "Тезкор маълумот" (operational flood report) Excel
// workbook from a parameterized template.
//
// Data lives in A..S:
//   - A = №,  B = name,  C/D = level prev/curr,  E/F = volume,  G/H = inflow,
//     I/J = outflow, K/L = ges flow, M/N = capacity, O/P = idle discharge,
//     Q = weather, R = temperature, S = duty.
//
// Header time/date are in S2/S3; per-column subheaders in row 5 derive the
// prev/curr hour from S2 via =MOD($S$2-TIME(1,0,0),1) and =$S$2.
//
// One 2-row block (rows 6-7) per reservoir: row 6 holds values, row 7 holds
// delta formulas (=IFERROR(D6-C6,"-"), etc.). Row 9 carries the signer line
// (E9:J9 hardcoded, M9:Q9 holds the operator's short name).
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

	"srmt-admin/internal/lib/service/excel/templates"
)

// Template layout constants.
const (
	templateBlockStartRow = 6 // first row of the value block
	templateBlockSize     = 2 // value row + delta-formula row
	templateSignerRow     = 9 // E9:J9 + M9:Q9 in the original template
)

// Generator renders the report.
type Generator struct {
	overrideDir string
}

// New binds the generator to an optional override directory for the embedded
// template. When overrideDir is empty (or the file is missing in it), the
// embedded sel.xlsx is used.
func New(overrideDir string) *Generator {
	return &Generator{overrideDir: overrideDir}
}

// Report bundles everything needed to fill the template.
type Report struct {
	Date        time.Time      // → S3 (YYYY-MM-DD, текст)
	Hour        int            // → S2 (HH:00); also drives C5..O5 via =MOD($S$2-TIME(1,0,0),1)
	AuthorShort string         // → M9 (or row 9 + (N-1)*2 after cloning)
	Reservoirs  []ReservoirRow // one entry per reservoir, rendered in order
}

// ReservoirRow holds one reservoir's values for the prev/curr hour pair.
// All numeric fields are nullable; nil → cell becomes "-". Strings: ""
// → cell becomes "-".
type ReservoirRow struct {
	Name              string     // B
	PrevAt            *time.Time // local time of the prev record; nil = no prev
	LevelPrev         *float64   // C
	LevelCurr         *float64 // D
	VolumePrev        *float64 // E
	VolumeCurr        *float64 // F
	InflowPrev        *float64 // G
	InflowCurr        *float64 // H
	OutflowPrev       *float64 // I
	OutflowCurr       *float64 // J
	GESFlowPrev       *float64 // K
	GESFlowCurr       *float64 // L
	CapacityPrev      *float64 // M
	CapacityCurr      *float64 // N
	IdleDischargePrev *float64 // O
	IdleDischargeCurr *float64 // P
	WeatherCondition  string   // Q (current hour only)
	TemperatureC      *float64 // R (current hour only)
	DutyName          string   // S (current hour only)
}

const dash = "-"

// GenerateExcel returns a populated workbook. Caller must Close() it.
func (g *Generator) GenerateExcel(rep *Report) (*excelize.File, error) {
	if rep == nil {
		return nil, fmt.Errorf("nil report")
	}
	f, err := templates.Open(templates.Sel, g.overrideDir)
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

	// Header: S2 (time-of-day) and S3 (date).
	// excelize writes time.Time under the existing [$-10819]hh:mm;@ format
	// correctly; the year/month/day are irrelevant — only the hour matters.
	set("S2", time.Date(2000, 1, 1, rep.Hour, 0, 0, 0, time.UTC))
	// Write S3 as a string instead of time.Time so the cell's number format
	// (mmm.yy on prod, mm-dd-yy on CI) does not transform the displayed value.
	// Number formats only apply to numeric cells; plain text is rendered verbatim.
	set("S3", rep.Date.Format("2006-01-02"))

	// Row-5 prev/curr-hour subheaders. The template originally carries
	// =MOD($S$2-TIME(1,0,0),1) for prev and =$S$2 for curr. Excel renders
	// those as 16:00/17:00 for hour=17, but soffice headless (PDF path)
	// truncates 16/24 = 0.6666... down to 15:59 instead of rounding to 16:00.
	// Writing the value directly bypasses the formula-engine difference, and
	// we already have the hour at hand. Style cell (format hh:mm) is
	// preserved by SetCellValue.
	// SetCellValue alone leaves the original <f> formula in place, and the
	// trailing UpdateLinkedValue() then wipes the cached <v> we just wrote.
	// Clear the formula explicitly first.
	clearFormula := func(cell string) {
		if writeErr != nil {
			return
		}
		if err := f.SetCellFormula(sheet, cell, ""); err != nil {
			writeErr = fmt.Errorf("clear formula %s: %w", cell, err)
		}
	}
	// Row-5 prev cells: write the range as plain text. The template carries
	// the right style (text format @, wrap text, centered borders) for these
	// cells, so we just clear the formula and call set() — SetCellValue
	// preserves the existing cell style. Do NOT call SetCellStyle here:
	// constructing a fresh style would wipe alignment/borders/font from the
	// template.
	prevLabel := formatPrevRow5(rep.Reservoirs, rep.Date)
	for _, cell := range []string{"C5", "E5", "G5", "I5", "K5", "M5", "O5"} {
		clearFormula(cell)
		set(cell, prevLabel)
	}

	currHour := time.Date(2000, 1, 1, rep.Hour, 0, 0, 0, time.UTC)
	for _, cell := range []string{"D5", "F5", "H5", "J5", "L5", "N5", "P5"} {
		clearFormula(cell)
		set(cell, currHour)
	}

	// Phase 1: clone the 2-row block for every reservoir past the first.
	// DuplicateRowTo copies cell values, formulas, AND styling — but it does
	// NOT rewrite formula cell references (C9 ends up holding the literal
	// "IFERROR(D6-C6,\"-\")" instead of the expected "IFERROR(D8-C8,\"-\")").
	// We fix that in Phase 1b by rewriting each cloned block's delta formulas
	// with the correct row reference. The signer row 9 (E9:J9 + M9:Q9) does
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
	// in row 7 reference adjacent paired columns: C7=D6-C6, E7=F6-E6, etc.
	// Use literal column letters to keep the rewrite explicit.
	deltaPairs := [][2]string{{"C", "D"}, {"E", "F"}, {"G", "H"}, {"I", "J"}, {"K", "L"}, {"M", "N"}, {"O", "P"}}
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
	// replicates horizontal merges (the C7:D7 delta-formula cells) but drops
	// the value-row + delta-row vertical pairs (A6:A7, B6:B7, Q6:Q7, R6:R7,
	// S6:S7). Without this every name cell in cloned blocks would render as
	// one row tall while block 1's name spans two rows — visibly broken.
	verticalMergeCols := []string{"A", "B", "Q", "R", "S"}
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
		set("A"+rs, i+1)
		setStr("B"+rs, res.Name)

		setNum("C"+rs, res.LevelPrev)
		setNum("D"+rs, res.LevelCurr)
		setNum("E"+rs, res.VolumePrev)
		setNum("F"+rs, res.VolumeCurr)
		setNum("G"+rs, res.InflowPrev)
		setNum("H"+rs, res.InflowCurr)
		setNum("I"+rs, res.OutflowPrev)
		setNum("J"+rs, res.OutflowCurr)
		setNum("K"+rs, res.GESFlowPrev)
		setNum("L"+rs, res.GESFlowCurr)
		setNum("M"+rs, res.CapacityPrev)
		setNum("N"+rs, res.CapacityCurr)
		setNum("O"+rs, res.IdleDischargePrev)
		setNum("P"+rs, res.IdleDischargeCurr)

		setStr("Q"+rs, res.WeatherCondition)
		setNum("R"+rs, res.TemperatureC)
		setStr("S"+rs, res.DutyName)
	}

	// Phase 3: signer name in the (shifted) M9 merge (top-left of M9:Q9).
	signerRow := templateSignerRow
	if n > 1 {
		signerRow += (n - 1) * templateBlockSize
	}
	set(fmt.Sprintf("M%d", signerRow), rep.AuthorShort)

	// Phase 4: rebind print_area to the actual last content row. The template
	// ships with a static A1:S25 (sized for 9 reservoirs) so soffice would
	// otherwise either clip taller reports or leave a tail of empty rows
	// when N < 9. SetDefinedName refuses duplicates, so delete first.
	printAreaRef := fmt.Sprintf("'%s'!$A$1:$S$%d", sheet, signerRow)
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

// formatPrevRow5 builds the row-5 prev-hour label from per-row PrevAt values.
//
// Rules (see docs/plans/feature-reservoir-flood-prev-flex.md):
//   - If no row has PrevAt → "-".
//   - min..max time across all non-nil PrevAt.
//   - If min == max (single distinct time across all rows) → render only one side.
//   - Side's date is prepended ("02.01 ") only when that side's date differs
//     from reportDate. So all-yesterday at 23:00 + reportDate=today → "12.05 23:00";
//     mixed yesterday/today → "12.05 23:00–13.05 14:00"; same-day-only → "11:00–14:00".
//
// PrevAt values are expected to already be in the operator-facing location
// (set by the builder via .In(loc)). reportDate's Y/M/D are interpreted in
// PrevAt's location for the comparison; the location of reportDate's tzinfo
// is irrelevant.
func formatPrevRow5(rows []ReservoirRow, reportDate time.Time) string {
	var minT, maxT *time.Time
	for i := range rows {
		t := rows[i].PrevAt
		if t == nil {
			continue
		}
		if minT == nil || t.Before(*minT) {
			minT = t
		}
		if maxT == nil || t.After(*maxT) {
			maxT = t
		}
	}
	if minT == nil {
		return dash
	}

	// Use minT's location as the reference frame so the Y/M/D comparison
	// happens in the operator-facing timezone. Decompose reportDate via the
	// same loc so a non-UTC reportDate (defensive — callers currently pass
	// UTC midnight) wouldn't shift the calendar boundary.
	loc := minT.Location()
	rd := reportDate.In(loc)
	reportYMD := time.Date(rd.Year(), rd.Month(), rd.Day(), 0, 0, 0, 0, loc)

	sameAsReport := func(t time.Time) bool {
		t = t.In(loc)
		return t.Year() == reportYMD.Year() && t.Month() == reportYMD.Month() && t.Day() == reportYMD.Day()
	}

	// If EITHER side of the range is off-report-date, render both sides with
	// the date prefix so the cell is unambiguous. Otherwise both sides are
	// time-only.
	showDate := !sameAsReport(*minT) || !sameAsReport(*maxT)
	layout := "15:04"
	if showDate {
		layout = "02.01 15:04"
	}

	minStr := minT.In(loc).Format(layout)
	maxStr := maxT.In(loc).Format(layout)
	if minStr == maxStr {
		return minStr
	}
	return minStr + "–" + maxStr // en dash U+2013
}
