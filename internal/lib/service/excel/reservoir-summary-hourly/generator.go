package reservoirsummaryhourly

import (
	"fmt"

	model "srmt-admin/internal/lib/model/reservoir-hourly"

	"github.com/xuri/excelize/v2"
)

// Generator handles Excel file generation for hourly reservoir summaries
type Generator struct {
	templatePath string
}

// New creates a new Generator with the template path
func New(templatePath string) *Generator {
	return &Generator{
		templatePath: templatePath,
	}
}

// GenerateExcel creates an Excel file from the template with report data
func (g *Generator) GenerateExcel(report *model.HourlyReport) (*excelize.File, error) {
	f, err := excelize.OpenFile(g.templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open template: %w", err)
	}

	sheet := f.GetSheetName(0)

	var writeErr error
	set := func(cell string, value interface{}) {
		if writeErr != nil {
			return
		}
		if err := f.SetCellValue(sheet, cell, value); err != nil {
			writeErr = fmt.Errorf("failed to set cell %s: %w", cell, err)
		}
	}

	// Header cells
	set("R2", report.LatestTime)
	set("S2", report.Period)

	// Rows 6–11: one row per reservoir (max 6)
	maxRows := len(report.Reservoirs)
	if maxRows > 6 {
		maxRows = 6
	}

	for i := 0; i < maxRows; i++ {
		res := report.Reservoirs[i]
		row := 6 + i
		rowStr := fmt.Sprintf("%d", row)

		set("B"+rowStr, res.Weather.DayBegin)
		set("C"+rowStr, res.Weather.Current)
		set("D"+rowStr, res.Level.DayBegin)
		set("E"+rowStr, res.Level.Current)
		set("G"+rowStr, res.Volume.DayBegin)
		set("H"+rowStr, res.Volume.Current)

		// Income[0..5] → columns J..O
		incomeCols := []string{"J", "K", "L", "M", "N", "O"}
		for j, col := range incomeCols {
			if j < len(res.Income) {
				set(col+rowStr, res.Income[j])
			}
		}

		set("Q"+rowStr, res.Release)
		set("R"+rowStr, res.OrganizationID)
		set("S"+rowStr, res.IncomeAtDayBegin)
	}

	if writeErr != nil {
		f.Close()
		return nil, writeErr
	}

	// Recalculate formulas
	if err := f.UpdateLinkedValue(); err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to calculate formulas: %w", err)
	}

	// Clear column R (rows 6–11) after formula recalculation
	for row := 6; row <= 11; row++ {
		cell := fmt.Sprintf("R%d", row)
		if err := f.SetCellValue(sheet, cell, ""); err != nil {
			f.Close()
			return nil, fmt.Errorf("failed to clear cell %s: %w", cell, err)
		}
	}

	return f, nil
}
