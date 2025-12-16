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

	// Format date as dd.MM.yyyy (e.g., "16.12.2025")
	formattedDate := parsedDate.Format("02.01.2006")

	// Set date in cell L2
	sheet := f.GetSheetName(0) // Get first sheet name
	if err := f.SetCellValue(sheet, "L2", formattedDate); err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to set date in cell L2: %w", err)
	}

	return f, nil
}
