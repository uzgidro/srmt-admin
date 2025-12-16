package reservoirsummary

import (
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"
)

// Generator handles Excel file generation for reservoir summaries
type Generator struct {
	templatePath string
}

// New creates a new Generator with the template path
func New(templatePath string) *Generator {
	return &Generator{
		templatePath: templatePath,
	}
}

// GenerateExcel creates an Excel file from the template with the specified date
func (g *Generator) GenerateExcel(date string) (*excelize.File, error) {
	// Open template file
	f, err := excelize.OpenFile(g.templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open template: %w", err)
	}

	// Parse date string (format: YYYY-MM-DD)
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to parse date: %w", err)
	}

	// Set date in cell L2
	sheet := f.GetSheetName(0) // Get first sheet name
	if err := f.SetCellValue(sheet, "L2", parsedDate); err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to set date in cell L2: %w", err)
	}

	// Calculate and set year values in cells that contain formulas based on L2
	// The formulas extract year from the date, so we calculate those values here
	currentYear := parsedDate.Year()
	previousYear := currentYear - 1
	twoYearsAgo := currentYear - 2

	// D5, G5, J5, M5, O5: Previous year (formulas calculate year-1 from L2)
	prevYearCells := []string{"D5", "G5", "J5", "M5", "O5"}
	for _, cell := range prevYearCells {
		if err := f.SetCellValue(sheet, cell, previousYear); err != nil {
			f.Close()
			return nil, fmt.Errorf("failed to set previous year in cell %s: %w", cell, err)
		}
	}

	// E5, H5, K5: Two years ago (formulas calculate year-2 from L2)
	twoYearsAgoCells := []string{"E5", "H5", "K5"}
	for _, cell := range twoYearsAgoCells {
		if err := f.SetCellValue(sheet, cell, twoYearsAgo); err != nil {
			f.Close()
			return nil, fmt.Errorf("failed to set two years ago in cell %s: %w", cell, err)
		}
	}

	// L5, N5: Current year (formulas extract year from L2)
	currentYearCells := []string{"L5", "N5"}
	for _, cell := range currentYearCells {
		if err := f.SetCellValue(sheet, cell, currentYear); err != nil {
			f.Close()
			return nil, fmt.Errorf("failed to set current year in cell %s: %w", cell, err)
		}
	}

	if err := f.UpdateLinkedValue(); err != nil {
		return nil, fmt.Errorf("failed to calculate formulas: %w", err)
	}

	return f, nil
}
