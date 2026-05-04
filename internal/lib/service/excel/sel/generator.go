// Package sel renders the "Тезкор маълумот" (operational flood report) Excel
// workbook from a parameterized template.
//
// The template (template/sel.xlsx) carries one 2-row block (rows 6-7) for a
// single reservoir: row 6 holds the values, row 7 holds the delta formulas
// (=IFERROR(D6-C6,"-"), etc.). Rows 1-5 form the header (title, date, time,
// column captions, and per-column subheaders that derive the prev/curr hour
// from S2 via =MOD($S$2-1/24,1) and =$S$2). Row 9 carries the signer line
// (E9:J9 hardcoded, M9:Q9 holds the operator's short name).
//
// For a report with N reservoirs, the generator clones the 2-row block N-1
// times via DuplicateRowTo so each reservoir has its own pair (rows 6+7,
// 8+9, 10+11, ...). The hardcoded signer text and M9:Q9 merge shift down
// automatically; excelize rewrites the IFERROR formulas to reference the new
// block's value row.
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
	templateSignerRow     = 9 // E9:J9 + M9:Q9 in the original template
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
	Date        time.Time      // → S3 (mm-dd-yy)
	Hour        int            // → S2 (HH:00); also drives C5..O5 via =MOD($S$2-1/24,1)
	AuthorShort string         // → M9 (or row 9 + (N-1)*2 after cloning)
	Reservoirs  []ReservoirRow // one entry per reservoir, rendered in order
}

// ReservoirRow holds one reservoir's values for the prev/curr hour pair.
// All numeric fields are nullable; nil → cell becomes "-". Strings: ""
// → cell becomes "-".
type ReservoirRow struct {
	Name              string   // B
	LevelPrev         *float64 // C
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

	// Header: S2 (time-of-day) and S3 (date).
	// excelize writes time.Time under the existing [$-10819]hh:mm;@ format
	// correctly; the year/month/day are irrelevant — only the hour matters.
	set("S2", time.Date(2000, 1, 1, rep.Hour, 0, 0, 0, time.UTC))
	set("S3", rep.Date)

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

	// Phase 3: signer name in the (shifted) M9 merge.
	signerRow := templateSignerRow
	if n > 1 {
		signerRow += (n - 1) * templateBlockSize
	}
	set(fmt.Sprintf("M%d", signerRow), rep.AuthorShort)

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
