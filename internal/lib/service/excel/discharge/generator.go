package discharge

import (
	"fmt"
	"strconv"
	"time"

	"srmt-admin/internal/lib/model/discharge"

	"github.com/xuri/excelize/v2"
)

// Generator handles Excel file generation for discharge reports
type Generator struct {
	templatePath string
}

// New creates a new Generator with the template path
func New(templatePath string) *Generator {
	return &Generator{
		templatePath: templatePath,
	}
}

// GenerateExcel creates an Excel file from the template with discharge data
func (g *Generator) GenerateExcel(date string, data []discharge.Model, loc *time.Location) (*excelize.File, error) {
	// Open template file
	f, err := excelize.OpenFile(g.templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open template: %w", err)
	}

	sheet := f.GetSheetName(0)

	// Read organization IDs from column A
	orgRowMap, err := g.readTemplateOrganizations(f, sheet)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to read template organizations: %w", err)
	}

	// Aggregate data by organization
	aggregated := g.aggregateByOrganization(data, loc)

	var writeErr error
	set := func(cell string, value interface{}) {
		if writeErr != nil {
			return
		}
		if err := f.SetCellValue(sheet, cell, value); err != nil {
			writeErr = fmt.Errorf("failed to set cell %s: %w", cell, err)
		}
	}

	// Collect rows to delete (organizations without data) and their orgIDs
	var rowsToDelete []int
	var orgsToDelete []int64
	for orgID, rowNum := range orgRowMap {
		if _, hasData := aggregated[orgID]; !hasData {
			rowsToDelete = append(rowsToDelete, rowNum)
			orgsToDelete = append(orgsToDelete, orgID)
		}
	}

	// Delete rows in reverse order to maintain correct row indices
	sortReverse(rowsToDelete)
	for _, rowNum := range rowsToDelete {
		if err := f.RemoveRow(sheet, rowNum); err != nil {
			f.Close()
			return nil, fmt.Errorf("failed to remove row %d: %w", rowNum, err)
		}
		// Update orgRowMap for remaining rows
		for oid, r := range orgRowMap {
			if r > rowNum {
				orgRowMap[oid] = r - 1
			}
		}
	}

	// Remove deleted organizations from map
	for _, orgID := range orgsToDelete {
		delete(orgRowMap, orgID)
	}

	// Fill data for organizations that have discharges
	for orgID, row := range aggregated {
		rowNum, exists := orgRowMap[orgID]
		if !exists {
			continue
		}

		// D: Start date (dd.MM.yyyy)
		set(fmt.Sprintf("D%d", rowNum), row.StartDate.Format("02.01.2006"))

		// E: Start time (HH:mm)
		set(fmt.Sprintf("E%d", rowNum), row.StartTime)

		// G: End date (dd.MM.yyyy)
		if row.EndDate != nil {
			set(fmt.Sprintf("G%d", rowNum), row.EndDate.Format("02.01.2006"))
		}

		// H: End time (HH:mm)
		if row.EndTime != nil {
			set(fmt.Sprintf("H%d", rowNum), *row.EndTime)
		}

		// I: Duration ("X кун, Y соат, Z минут")
		set(fmt.Sprintf("I%d", rowNum), row.Duration)

		// F: Calculated value (J / 0.0864)
		set(fmt.Sprintf("F%d", rowNum), row.TotalVolume/0.0864)

		// J: Total volume
		set(fmt.Sprintf("J%d", rowNum), row.TotalVolume)

		// K: Reason
		if row.Reason != nil {
			set(fmt.Sprintf("K%d", rowNum), *row.Reason)
		}
	}

	// Collect remaining row numbers and sort them
	var remainingRows []int
	for _, rowNum := range orgRowMap {
		remainingRows = append(remainingRows, rowNum)
	}
	sortAsc(remainingRows)

	// Set sequential numbering in column B for data rows
	for i, rowNum := range remainingRows {
		set(fmt.Sprintf("B%d", rowNum), i+1) // 1-based numbering
	}

	// Clear entire column A
	rows, _ := f.GetRows(sheet)
	for rowIdx := range rows {
		set(fmt.Sprintf("A%d", rowIdx+1), "")
	}

	if writeErr != nil {
		f.Close()
		return nil, writeErr
	}

	// Set print area (A1 to L{lastDataRow})
	lastDataRow := 1
	if len(remainingRows) > 0 {
		lastDataRow = remainingRows[len(remainingRows)-1]
	}
	printArea := fmt.Sprintf("$A$1:$L$%d", lastDataRow)
	if err := f.SetDefinedName(&excelize.DefinedName{
		Name:     "_xlnm.Print_Area",
		RefersTo: fmt.Sprintf("'%s'!%s", sheet, printArea),
		Scope:    sheet,
	}); err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to set print area: %w", err)
	}

	return f, nil
}

// readTemplateOrganizations reads organization IDs from column A
// Returns a map of organization_id -> row_number
func (g *Generator) readTemplateOrganizations(f *excelize.File, sheet string) (map[int64]int, error) {
	result := make(map[int64]int)

	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows: %w", err)
	}

	for rowIdx, row := range rows {
		rowNum := rowIdx + 1 // Excel rows are 1-based
		if len(row) == 0 {
			continue
		}

		// Try to parse organization ID from column A
		cellValue := row[0]
		if cellValue == "" {
			continue
		}

		orgID, err := strconv.ParseInt(cellValue, 10, 64)
		if err != nil {
			// Not a valid integer, skip this row
			continue
		}

		result[orgID] = rowNum
	}

	return result, nil
}

// aggregateByOrganization aggregates discharge data by organization_id
func (g *Generator) aggregateByOrganization(data []discharge.Model, loc *time.Location) map[int64]*discharge.ReportRow {
	result := make(map[int64]*discharge.ReportRow)

	for _, d := range data {
		if d.Organization == nil {
			continue
		}

		orgID := d.Organization.ID

		if _, exists := result[orgID]; !exists {
			// Initialize with first record
			result[orgID] = &discharge.ReportRow{
				OrganizationID:   orgID,
				OrganizationName: d.Organization.Name,
				StartDate:        d.StartedAt.In(loc),
				StartTime:        d.StartedAt.In(loc).Format("15:04"),
				EndDate:          nil,
				EndTime:          nil,
				Duration:         "",
				TotalVolume:      d.TotalVolume,
				Reason:           d.Reason,
			}
			if d.EndedAt != nil {
				endInLoc := d.EndedAt.In(loc)
				result[orgID].EndDate = &endInLoc
				endTime := endInLoc.Format("15:04")
				result[orgID].EndTime = &endTime
			}
		} else {
			row := result[orgID]

			// Update min start date/time
			if d.StartedAt.Before(row.StartDate) {
				row.StartDate = d.StartedAt.In(loc)
				row.StartTime = d.StartedAt.In(loc).Format("15:04")
			}

			// Update max end date/time
			if d.EndedAt != nil {
				endInLoc := d.EndedAt.In(loc)
				if row.EndDate == nil || endInLoc.After(*row.EndDate) {
					row.EndDate = &endInLoc
					endTime := endInLoc.Format("15:04")
					row.EndTime = &endTime
				}
			}

			// Sum total volume
			row.TotalVolume += d.TotalVolume
		}
	}

	// Calculate duration for each organization
	for _, row := range result {
		if row.EndDate != nil {
			duration := row.EndDate.Sub(row.StartDate)
			row.Duration = formatDuration(duration)
		}
	}

	return result
}

// formatDuration formats duration as "X кун, Y соат, Z минут" (skipping zero values)
func formatDuration(d time.Duration) string {
	totalMinutes := int(d.Minutes())
	days := totalMinutes / (24 * 60)
	hours := (totalMinutes % (24 * 60)) / 60
	minutes := totalMinutes % 60

	var parts []string
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%d кун", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%d соат", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%d минут", minutes))
	}

	if len(parts) == 0 {
		return "0 минут"
	}

	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += ", " + parts[i]
	}
	return result
}

// sortReverse sorts integers in descending order
func sortReverse(arr []int) {
	for i := 0; i < len(arr)-1; i++ {
		for j := i + 1; j < len(arr); j++ {
			if arr[i] < arr[j] {
				arr[i], arr[j] = arr[j], arr[i]
			}
		}
	}
}

// sortAsc sorts integers in ascending order
func sortAsc(arr []int) {
	for i := 0; i < len(arr)-1; i++ {
		for j := i + 1; j < len(arr); j++ {
			if arr[i] > arr[j] {
				arr[i], arr[j] = arr[j], arr[i]
			}
		}
	}
}
