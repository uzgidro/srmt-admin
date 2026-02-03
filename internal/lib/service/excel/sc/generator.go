package sc

import (
	"fmt"
	"strconv"
	"time"

	"srmt-admin/internal/lib/model/discharge"
	"srmt-admin/internal/lib/model/shutdown"

	"github.com/xuri/excelize/v2"
)

// SectionInfo holds information about a section in the template
type SectionInfo struct {
	Tag       string        // "discharges", "ges", "mini", "micro", "visits"
	HeaderRow int           // row number of section header
	OrgRows   map[int64]int // organization_id -> row_number
}

// GroupedShutdowns holds shutdowns grouped by organization type
type GroupedShutdowns struct {
	Ges   []*shutdown.ResponseModel
	Mini  []*shutdown.ResponseModel
	Micro []*shutdown.ResponseModel
}

// Generator handles Excel file generation for SC reports
type Generator struct {
	templatePath string
}

// New creates a new Generator with the template path
func New(templatePath string) *Generator {
	return &Generator{
		templatePath: templatePath,
	}
}

// GenerateExcel creates an Excel file from the template with all SC data
func (g *Generator) GenerateExcel(
	dateStart, dateEnd time.Time,
	discharges []discharge.Model,
	shutdowns *GroupedShutdowns,
	loc *time.Location,
) (*excelize.File, error) {
	// Open template file
	f, err := excelize.OpenFile(g.templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open template: %w", err)
	}

	sheet := f.GetSheetName(0)

	// Replace DATE_START and DATE_END placeholders
	if err := g.replaceDatePlaceholders(f, sheet, dateStart, dateEnd); err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to replace date placeholders: %w", err)
	}

	// Scan column P to build section map
	sections, err := g.scanSections(f, sheet)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to scan sections: %w", err)
	}

	var writeErr error
	set := func(cell string, value interface{}) {
		if writeErr != nil {
			return
		}
		if err := f.SetCellValue(sheet, cell, value); err != nil {
			writeErr = fmt.Errorf("failed to set cell %s: %w", cell, err)
		}
	}

	// Process discharges section
	if dischargesSection, ok := sections["discharges"]; ok {
		if err := g.processDischarges(f, sheet, dischargesSection, discharges, loc, set); err != nil {
			f.Close()
			return nil, fmt.Errorf("failed to process discharges: %w", err)
		}
	}

	if writeErr != nil {
		f.Close()
		return nil, writeErr
	}

	// Process shutdown sections (ges, mini, micro)
	if shutdowns != nil {
		// Re-scan sections after discharges processing (rows may have shifted)
		sections, err = g.scanSections(f, sheet)
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("failed to re-scan sections: %w", err)
		}

		if gesSection, ok := sections["ges"]; ok {
			if err := g.processShutdowns(f, sheet, gesSection, shutdowns.Ges, loc, set); err != nil {
				f.Close()
				return nil, fmt.Errorf("failed to process ges shutdowns: %w", err)
			}
		}

		// Re-scan sections after ges processing
		sections, err = g.scanSections(f, sheet)
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("failed to re-scan sections: %w", err)
		}

		if miniSection, ok := sections["mini"]; ok {
			if err := g.processShutdowns(f, sheet, miniSection, shutdowns.Mini, loc, set); err != nil {
				f.Close()
				return nil, fmt.Errorf("failed to process mini shutdowns: %w", err)
			}
		}

		// Re-scan sections after mini processing
		sections, err = g.scanSections(f, sheet)
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("failed to re-scan sections: %w", err)
		}

		if microSection, ok := sections["micro"]; ok {
			if err := g.processShutdowns(f, sheet, microSection, shutdowns.Micro, loc, set); err != nil {
				f.Close()
				return nil, fmt.Errorf("failed to process micro shutdowns: %w", err)
			}
		}
	}

	if writeErr != nil {
		f.Close()
		return nil, writeErr
	}

	// Clear column P (tags column)
	if err := g.clearColumn(f, sheet, "P"); err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to clear column P: %w", err)
	}

	// Set print area (delete existing first to avoid "name already exists" error)
	lastRow := g.findLastDataRow(f, sheet)
	printArea := fmt.Sprintf("$A$1:$O$%d", lastRow)

	// Delete existing print area if it exists
	_ = f.DeleteDefinedName(&excelize.DefinedName{
		Name:  "_xlnm.Print_Area",
		Scope: sheet,
	})

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

// replaceDatePlaceholders replaces DATE_START and DATE_END in the template
func (g *Generator) replaceDatePlaceholders(f *excelize.File, sheet string, dateStart, dateEnd time.Time) error {
	rows, err := f.GetRows(sheet)
	if err != nil {
		return fmt.Errorf("failed to get rows: %w", err)
	}

	for rowIdx, row := range rows {
		for colIdx, cellValue := range row {
			if cellValue == "DATE_START" {
				cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
				if err := f.SetCellValue(sheet, cell, dateStart.Format("02.01.2006 15:04")); err != nil {
					return err
				}
			} else if cellValue == "DATE_END" {
				cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
				if err := f.SetCellValue(sheet, cell, dateEnd.Format("02.01.2006 15:04")); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// scanSections reads column P to identify sections and their organization rows
func (g *Generator) scanSections(f *excelize.File, sheet string) (map[string]*SectionInfo, error) {
	sections := make(map[string]*SectionInfo)

	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows: %w", err)
	}

	var currentSection *SectionInfo

	for rowIdx, row := range rows {
		rowNum := rowIdx + 1

		// Column P is index 15 (0-based)
		if len(row) <= 15 {
			continue
		}

		cellValue := row[15]
		if cellValue == "" {
			continue
		}

		// Check if it's a section tag
		switch cellValue {
		case "ges", "mini", "micro", "discharges", "visits", "res":
			currentSection = &SectionInfo{
				Tag:       cellValue,
				HeaderRow: rowNum,
				OrgRows:   make(map[int64]int),
			}
			sections[cellValue] = currentSection
		default:
			// Try to parse as organization ID
			if orgID, err := strconv.ParseInt(cellValue, 10, 64); err == nil {
				if currentSection != nil {
					currentSection.OrgRows[orgID] = rowNum
				}
			}
		}
	}

	return sections, nil
}

// processDischarges fills the discharges section with data
func (g *Generator) processDischarges(
	f *excelize.File,
	sheet string,
	section *SectionInfo,
	data []discharge.Model,
	loc *time.Location,
	set func(cell string, value interface{}),
) error {
	// Aggregate data by organization
	aggregated := g.aggregateDischargesByOrganization(data, loc)

	// Collect rows to delete (organizations without data)
	var rowsToDelete []int
	var orgsToDelete []int64
	for orgID, rowNum := range section.OrgRows {
		if _, hasData := aggregated[orgID]; !hasData {
			rowsToDelete = append(rowsToDelete, rowNum)
			orgsToDelete = append(orgsToDelete, orgID)
		}
	}

	// Delete rows in reverse order
	sortReverse(rowsToDelete)
	for _, rowNum := range rowsToDelete {
		if err := f.RemoveRow(sheet, rowNum); err != nil {
			return fmt.Errorf("failed to remove row %d: %w", rowNum, err)
		}
		// Update orgRowMap for remaining rows
		for oid, r := range section.OrgRows {
			if r > rowNum {
				section.OrgRows[oid] = r - 1
			}
		}
	}

	// Remove deleted organizations from map
	for _, orgID := range orgsToDelete {
		delete(section.OrgRows, orgID)
	}

	// Fill data for organizations that have discharges
	for orgID, row := range aggregated {
		rowNum, exists := section.OrgRows[orgID]
		if !exists {
			continue
		}

		// C: Start date (dd.MM.yyyy)
		set(fmt.Sprintf("C%d", rowNum), row.StartDate.Format("02.01.2006"))

		// D: Start time (HH:mm)
		set(fmt.Sprintf("D%d", rowNum), row.StartTime)

		// E: Flow rate (м3/сек) - calculated as TotalVolume / 0.0864
		set(fmt.Sprintf("E%d", rowNum), row.TotalVolume/0.0864)

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

		// K: Total volume (млн.м3)
		set(fmt.Sprintf("K%d", rowNum), row.TotalVolume)

		// M: Reason
		if row.Reason != nil {
			set(fmt.Sprintf("M%d", rowNum), *row.Reason)
		}
	}

	// Recalculate numbering in column A
	var remainingRows []int
	for _, rowNum := range section.OrgRows {
		remainingRows = append(remainingRows, rowNum)
	}
	sortAsc(remainingRows)

	for i, rowNum := range remainingRows {
		set(fmt.Sprintf("A%d", rowNum), i+1) // 1-based numbering
	}

	// Restore bottom border for the last data row (lost after row deletion)
	if len(remainingRows) > 0 {
		lastRow := remainingRows[len(remainingRows)-1]
		if err := g.applyBottomBorder(f, sheet, lastRow, "A", "O"); err != nil {
			return fmt.Errorf("failed to apply bottom border: %w", err)
		}
	}

	return nil
}

// applyBottomBorder applies a bottom border to cells in a row from startCol to endCol
func (g *Generator) applyBottomBorder(f *excelize.File, sheet string, row int, startCol, endCol string) error {
	// Create style with bottom border
	borderStyle := excelize.Border{
		Type:  "bottom",
		Color: "000000",
		Style: 1, // thin border
	}

	// Get column indices
	startColIdx, _ := excelize.ColumnNameToNumber(startCol)
	endColIdx, _ := excelize.ColumnNameToNumber(endCol)

	for colIdx := startColIdx; colIdx <= endColIdx; colIdx++ {
		colName, _ := excelize.ColumnNumberToName(colIdx)
		cell := fmt.Sprintf("%s%d", colName, row)

		// Get existing style
		existingStyleID, _ := f.GetCellStyle(sheet, cell)

		// Get existing style details
		existingStyle, _ := f.GetStyle(existingStyleID)

		// Build new style preserving existing properties
		newStyle := &excelize.Style{
			Border: []excelize.Border{borderStyle},
		}

		if existingStyle != nil {
			// Preserve existing borders (top, left, right) and add bottom
			newStyle.Border = append(existingStyle.Border, borderStyle)
			newStyle.Fill = existingStyle.Fill
			newStyle.Font = existingStyle.Font
			newStyle.Alignment = existingStyle.Alignment
			newStyle.NumFmt = existingStyle.NumFmt
		}

		styleID, err := f.NewStyle(newStyle)
		if err != nil {
			return fmt.Errorf("failed to create style for cell %s: %w", cell, err)
		}

		if err := f.SetCellStyle(sheet, cell, cell, styleID); err != nil {
			return fmt.Errorf("failed to set style for cell %s: %w", cell, err)
		}
	}

	return nil
}

// processShutdowns fills a shutdown section (ges/mini/micro) with data
func (g *Generator) processShutdowns(
	f *excelize.File,
	sheet string,
	section *SectionInfo,
	shutdowns []*shutdown.ResponseModel,
	loc *time.Location,
	set func(cell string, value interface{}),
) error {
	// Group shutdowns by organization ID
	shutdownsByOrg := make(map[int64][]*shutdown.ResponseModel)
	for _, s := range shutdowns {
		shutdownsByOrg[s.OrganizationID] = append(shutdownsByOrg[s.OrganizationID], s)
	}

	// Collect rows to delete (organizations without data)
	var rowsToDelete []int
	var orgsToDelete []int64
	for orgID, rowNum := range section.OrgRows {
		if _, hasData := shutdownsByOrg[orgID]; !hasData {
			rowsToDelete = append(rowsToDelete, rowNum)
			orgsToDelete = append(orgsToDelete, orgID)
		}
	}

	// Delete rows in reverse order to preserve row numbers
	sortReverse(rowsToDelete)
	for _, rowNum := range rowsToDelete {
		if err := f.RemoveRow(sheet, rowNum); err != nil {
			return fmt.Errorf("failed to remove row %d: %w", rowNum, err)
		}
		// Update orgRowMap for remaining rows
		for oid, r := range section.OrgRows {
			if r > rowNum {
				section.OrgRows[oid] = r - 1
			}
		}
	}

	// Remove deleted organizations from map
	for _, orgID := range orgsToDelete {
		delete(section.OrgRows, orgID)
	}

	// Track all data rows (for numbering and totals)
	var allDataRows []int
	var totalGenerationLoss float64
	var totalIdleDischargeVolume float64

	// Process organizations with shutdowns
	// We need to handle row duplication for multiple shutdowns per org
	// Collect orgs sorted by row number to process in order
	type orgRow struct {
		orgID  int64
		rowNum int
	}
	var orgRows []orgRow
	for orgID, rowNum := range section.OrgRows {
		orgRows = append(orgRows, orgRow{orgID: orgID, rowNum: rowNum})
	}
	// Sort by row number ascending
	for i := 0; i < len(orgRows)-1; i++ {
		for j := i + 1; j < len(orgRows); j++ {
			if orgRows[i].rowNum > orgRows[j].rowNum {
				orgRows[i], orgRows[j] = orgRows[j], orgRows[i]
			}
		}
	}

	// Track row offset due to insertions
	rowOffset := 0

	for _, or := range orgRows {
		orgID := or.orgID
		baseRowNum := or.rowNum + rowOffset
		shutdownList := shutdownsByOrg[orgID]

		if len(shutdownList) == 0 {
			continue
		}

		// For multiple shutdowns, we need to duplicate the row
		for i := 1; i < len(shutdownList); i++ {
			// Duplicate the row (insert after baseRowNum)
			if err := f.DuplicateRow(sheet, baseRowNum); err != nil {
				return fmt.Errorf("failed to duplicate row %d: %w", baseRowNum, err)
			}
			rowOffset++
		}

		// Fill data for each shutdown
		for i, s := range shutdownList {
			currentRow := baseRowNum + i
			allDataRows = append(allDataRows, currentRow)

			// C: StartedAt (dd.MM.yyyy HH:mm)
			set(fmt.Sprintf("C%d", currentRow), s.StartedAt.In(loc).Format("02.01.2006 15:04"))

			// D: EndedAt (dd.MM.yyyy HH:mm) or empty
			if s.EndedAt != nil {
				set(fmt.Sprintf("D%d", currentRow), s.EndedAt.In(loc).Format("02.01.2006 15:04"))
			}

			// E: Reason (merged cells E-I)
			if s.Reason != nil {
				set(fmt.Sprintf("E%d", currentRow), *s.Reason)
			}

			// J-M: leave empty (already empty in template)

			// N: GenerationLossMwh (convert from kWh to thousand kWh)
			if s.GenerationLossMwh != nil {
				valueInThousands := *s.GenerationLossMwh / 1000
				set(fmt.Sprintf("N%d", currentRow), valueInThousands)
				totalGenerationLoss += valueInThousands
			}

			// O: IdleDischargeVolumeThousandM3
			if s.IdleDischargeVolumeThousandM3 != nil {
				set(fmt.Sprintf("O%d", currentRow), *s.IdleDischargeVolumeThousandM3)
				totalIdleDischargeVolume += *s.IdleDischargeVolumeThousandM3
			}
		}
	}

	// Sort allDataRows for numbering
	sortAsc(allDataRows)

	// Recalculate numbering in column A
	for i, rowNum := range allDataRows {
		set(fmt.Sprintf("A%d", rowNum), i+1) // 1-based numbering
	}

	// Restore bottom border for the last data row (lost after row deletion)
	if len(allDataRows) > 0 {
		lastDataRow := allDataRows[len(allDataRows)-1]
		if err := g.applyBottomBorder(f, sheet, lastDataRow, "A", "O"); err != nil {
			return fmt.Errorf("failed to apply bottom border: %w", err)
		}

		// Find and update "Жами" row (totals)
		// "Жами" should be somewhere after the last data row
		rows, err := f.GetRows(sheet)
		if err == nil {
			// Look for "Жами" in columns B or C within the next 5 rows
			for rowIdx := lastDataRow; rowIdx < lastDataRow+5 && rowIdx <= len(rows); rowIdx++ {
				if rowIdx-1 < len(rows) {
					row := rows[rowIdx-1]
					for colIdx, cellValue := range row {
						if cellValue == "Жами" || cellValue == "Жами:" {
							// Found "Жами" row - update totals in columns N and O
							if totalGenerationLoss > 0 {
								set(fmt.Sprintf("N%d", rowIdx), totalGenerationLoss)
							}
							if totalIdleDischargeVolume > 0 {
								set(fmt.Sprintf("O%d", rowIdx), totalIdleDischargeVolume)
							}
							_ = colIdx // suppress unused warning
							break
						}
					}
				}
			}
		}
	}

	return nil
}

// aggregateDischargesByOrganization aggregates discharge data by organization_id
func (g *Generator) aggregateDischargesByOrganization(data []discharge.Model, loc *time.Location) map[int64]*discharge.ReportRow {
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

// clearColumn clears all values in a column
func (g *Generator) clearColumn(f *excelize.File, sheet, col string) error {
	rows, err := f.GetRows(sheet)
	if err != nil {
		return fmt.Errorf("failed to get rows: %w", err)
	}

	for rowIdx := range rows {
		cell := fmt.Sprintf("%s%d", col, rowIdx+1)
		if err := f.SetCellValue(sheet, cell, ""); err != nil {
			return fmt.Errorf("failed to clear cell %s: %w", cell, err)
		}
	}

	return nil
}

// findLastDataRow finds the last row with data
func (g *Generator) findLastDataRow(f *excelize.File, sheet string) int {
	rows, _ := f.GetRows(sheet)
	return len(rows)
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
