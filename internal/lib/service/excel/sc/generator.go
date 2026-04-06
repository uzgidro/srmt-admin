package sc

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"srmt-admin/internal/lib/model/discharge"
	"srmt-admin/internal/lib/model/incident"
	infraevent "srmt-admin/internal/lib/model/infra-event"
	infraeventcategory "srmt-admin/internal/lib/model/infra-event-category"
	reservoirdevicesummary "srmt-admin/internal/lib/model/reservoir-device-summary"
	"srmt-admin/internal/lib/model/shutdown"
	"srmt-admin/internal/lib/model/visit"

	"github.com/xuri/excelize/v2"
)

// SectionInfo holds information about a section in the template
type SectionInfo struct {
	Tag         string
	HeaderRow   int
	TemplateRow int // HeaderRow + 1
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
	shutdowns []*shutdown.ResponseModel,
	orgTypesMap map[int64][]string,
	orgParentMap map[int64]*int64,
	visits []*visit.ResponseModel,
	incidents []*incident.ResponseModel,
	resDevices []*reservoirdevicesummary.ResponseModel,
	infraEvents []*infraevent.ResponseModel,
	infraCategories []*infraeventcategory.Model,
	loc *time.Location,
	authorShortName string,
) (*excelize.File, error) {
	// Open template file
	f, err := excelize.OpenFile(g.templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open template: %w", err)
	}

	sheet := f.GetSheetName(0)

	// Replace DATE_START, DATE_END, and SHORT_NAME placeholders
	if err := g.replacePlaceholders(f, sheet, dateStart, dateEnd, authorShortName); err != nil {
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
		if err := g.processDischarges(f, sheet, dischargesSection, discharges, orgParentMap, loc, set); err != nil {
			f.Close()
			return nil, fmt.Errorf("failed to process discharges: %w", err)
		}
	}

	if writeErr != nil {
		f.Close()
		return nil, writeErr
	}

	// Group shutdowns by organization type
	shutdownsByType := map[string][]*shutdown.ResponseModel{"ges": {}, "mini": {}, "micro": {}}
	for _, s := range shutdowns {
		orgType := determineOrgType(orgTypesMap[s.OrganizationID])
		if orgType != "" {
			shutdownsByType[orgType] = append(shutdownsByType[orgType], s)
		}
	}

	// Process shutdown sections (ges, mini, micro)
	for _, sType := range []string{"ges", "mini", "micro"} {
		// Re-scan sections (rows may have shifted)
		sections, err = g.scanSections(f, sheet)
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("failed to re-scan sections: %w", err)
		}

		if section, ok := sections[sType]; ok {
			if err := g.processShutdowns(f, sheet, section, shutdownsByType[sType], orgParentMap, loc, set); err != nil {
				f.Close()
				return nil, fmt.Errorf("failed to process %s shutdowns: %w", sType, err)
			}
		}
	}

	if writeErr != nil {
		f.Close()
		return nil, writeErr
	}

	// Process visits section
	sections, err = g.scanSections(f, sheet)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to re-scan sections for visits: %w", err)
	}

	if visitsSection, ok := sections["visits"]; ok {
		if err := g.processVisits(f, sheet, visitsSection, visits, orgParentMap, loc, set); err != nil {
			f.Close()
			return nil, fmt.Errorf("failed to process visits: %w", err)
		}
	}

	if writeErr != nil {
		f.Close()
		return nil, writeErr
	}

	// Process incidents section
	sections, err = g.scanSections(f, sheet)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to re-scan sections for incidents: %w", err)
	}

	if incidentsSection, ok := sections["incidents"]; ok {
		if err := g.processIncidents(f, sheet, incidentsSection, incidents, orgParentMap, loc, set); err != nil {
			f.Close()
			return nil, fmt.Errorf("failed to process incidents: %w", err)
		}
	}

	if writeErr != nil {
		f.Close()
		return nil, writeErr
	}

	// Process reservoir devices
	sections, err = g.scanSections(f, sheet)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to re-scan sections for res: %w", err)
	}
	if sec, ok := sections["res"]; ok {
		if err := g.processReservoirDevices(f, sheet, sec, resDevices, orgParentMap, set); err != nil {
			f.Close()
			return nil, fmt.Errorf("failed to process reservoir devices: %w", err)
		}
	}
	if writeErr != nil {
		f.Close()
		return nil, writeErr
	}

	// Process infra events section
	sections, err = g.scanSections(f, sheet)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to re-scan sections for infra: %w", err)
	}
	if sec, ok := sections["infra"]; ok {
		if err := g.processInfraEvents(f, sheet, sec, infraEvents, infraCategories, loc, set); err != nil {
			f.Close()
			return nil, fmt.Errorf("failed to process infra events: %w", err)
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

	// Force formula recalculation when file is opened
	if err := f.UpdateLinkedValue(); err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to update linked values: %w", err)
	}

	return f, nil
}

// replacePlaceholders replaces DATE_START, DATE_END, and SHORT_NAME in the template
func (g *Generator) replacePlaceholders(f *excelize.File, sheet string, dateStart, dateEnd time.Time, authorShortName string) error {
	rows, err := f.GetRows(sheet)
	if err != nil {
		return fmt.Errorf("failed to get rows: %w", err)
	}

	// Format dates as "3 февраль" (day month in Cyrillic)
	dateStartStr := fmt.Sprintf("%d %s", dateStart.Day(), monthNameCyrillic(dateStart.Month()))
	dateEndStr := fmt.Sprintf("%d %s", dateEnd.Day(), monthNameCyrillic(dateEnd.Month()))

	for rowIdx, row := range rows {
		for colIdx, cellValue := range row {
			newValue := cellValue
			replaced := false

			if strings.Contains(newValue, "DATE_START") {
				newValue = strings.Replace(newValue, "DATE_START", dateStartStr, -1)
				replaced = true
			}
			if strings.Contains(newValue, "DATE_END") {
				newValue = strings.Replace(newValue, "DATE_END", dateEndStr, -1)
				replaced = true
			}
			if strings.Contains(newValue, "SHORT_NAME") {
				newValue = strings.Replace(newValue, "SHORT_NAME", authorShortName, -1)
				replaced = true
			}

			if replaced {
				cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
				if err := f.SetCellValue(sheet, cell, newValue); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// monthNameCyrillic returns month name in Cyrillic
func monthNameCyrillic(m time.Month) string {
	months := []string{
		"январь",
		"февраль",
		"март",
		"апрель",
		"май",
		"июнь",
		"июль",
		"август",
		"сентябрь",
		"октябрь",
		"ноябрь",
		"декабрь",
	}
	return months[m-1]
}

// scanSections reads column P to identify sections and their template rows
func (g *Generator) scanSections(f *excelize.File, sheet string) (map[string]*SectionInfo, error) {
	sections := make(map[string]*SectionInfo)
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows: %w", err)
	}
	for rowIdx, row := range rows {
		rowNum := rowIdx + 1
		if len(row) <= 15 {
			continue
		}
		cellValue := row[15]
		if cellValue == "" {
			continue
		}
		switch cellValue {
		case "ges", "mini", "micro", "discharges", "visits", "incidents", "res", "infra":
			sections[cellValue] = &SectionInfo{
				Tag:         cellValue,
				HeaderRow:   rowNum,
				TemplateRow: rowNum + 1,
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
	orgParentMap map[int64]*int64,
	loc *time.Location,
	set func(cell string, value interface{}),
) error {
	// If no data, delete template row
	if len(data) == 0 {
		if err := f.RemoveRow(sheet, section.TemplateRow); err != nil {
			return fmt.Errorf("failed to remove template row %d: %w", section.TemplateRow, err)
		}
		return nil
	}

	// Aggregate data by organization
	aggregated := g.aggregateDischargesByOrganization(data, loc)

	// All records may have nil Organization — aggregated can be empty
	if len(aggregated) == 0 {
		if err := f.RemoveRow(sheet, section.TemplateRow); err != nil {
			return fmt.Errorf("failed to remove template row %d: %w", section.TemplateRow, err)
		}
		return nil
	}

	// Collect org IDs and sort by parent hierarchy
	orgIDs := make([]int64, 0, len(aggregated))
	for orgID := range aggregated {
		orgIDs = append(orgIDs, orgID)
	}
	orgIDs = sortOrgIDs(orgIDs, orgParentMap)

	// Duplicate template row N-1 times
	for i := 1; i < len(orgIDs); i++ {
		if err := f.DuplicateRow(sheet, section.TemplateRow); err != nil {
			return fmt.Errorf("failed to duplicate row %d: %w", section.TemplateRow, err)
		}
	}

	// Fill data for each organization
	for i, orgID := range orgIDs {
		rowNum := section.TemplateRow + i
		row := aggregated[orgID]

		// A: № (numbering)
		set(fmt.Sprintf("A%d", rowNum), i+1)

		// B: Organization name
		set(fmt.Sprintf("B%d", rowNum), row.OrganizationName)

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
			_ = f.SetRowHeight(sheet, rowNum, calcRowHeight(*row.Reason, 45))
		}
	}

	// Restore bottom border for the last data row
	lastRow := section.TemplateRow + len(orgIDs) - 1
	if err := g.applyBottomBorder(f, sheet, lastRow, "A", "O"); err != nil {
		return fmt.Errorf("failed to apply bottom border: %w", err)
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
			// Preserve existing borders (top, left, right), replace any existing bottom
			filtered := make([]excelize.Border, 0, len(existingStyle.Border))
			for _, b := range existingStyle.Border {
				if b.Type != "bottom" {
					filtered = append(filtered, b)
				}
			}
			newStyle.Border = append(filtered, borderStyle)
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
	orgParentMap map[int64]*int64,
	loc *time.Location,
	set func(cell string, value interface{}),
) error {
	// If no data, delete template row
	if len(shutdowns) == 0 {
		if err := f.RemoveRow(sheet, section.TemplateRow); err != nil {
			return fmt.Errorf("failed to remove template row %d: %w", section.TemplateRow, err)
		}
		return nil
	}

	// Sort shutdowns by start time (earliest first)
	slices.SortFunc(shutdowns, func(a, b *shutdown.ResponseModel) int {
		return a.StartedAt.Compare(b.StartedAt)
	})

	// Duplicate template row len(shutdowns)-1 times
	for i := 1; i < len(shutdowns); i++ {
		if err := f.DuplicateRow(sheet, section.TemplateRow); err != nil {
			return fmt.Errorf("failed to duplicate row %d: %w", section.TemplateRow, err)
		}
	}

	// Fill data sequentially
	var allDataRows []int
	var totalGenerationLoss float64
	var totalIdleDischargeVolume float64
	currentRow := section.TemplateRow

	for _, s := range shutdowns {
		allDataRows = append(allDataRows, currentRow)

		// B: Organization name
		set(fmt.Sprintf("B%d", currentRow), s.OrganizationName)

		// C: StartedAt (dd.MM.yyyy HH:mm)
		set(fmt.Sprintf("C%d", currentRow), s.StartedAt.In(loc).Format("02.01.2006 15:04"))

		// D: EndedAt (dd.MM.yyyy HH:mm) or empty
		if s.EndedAt != nil {
			set(fmt.Sprintf("D%d", currentRow), s.EndedAt.In(loc).Format("02.01.2006 15:04"))
		}

		// E: Reason (merged cells E-I)
		if s.Reason != nil {
			set(fmt.Sprintf("E%d", currentRow), *s.Reason)
			_ = f.SetRowHeight(sheet, currentRow, calcRowHeight(*s.Reason, 45))
		}

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

		currentRow++
	}

	// Recalculate numbering in column A
	for i, rowNum := range allDataRows {
		set(fmt.Sprintf("A%d", rowNum), i+1) // 1-based numbering
	}

	// Restore bottom border for the last data row
	if len(allDataRows) > 0 {
		lastDataRow := allDataRows[len(allDataRows)-1]
		if err := g.applyBottomBorder(f, sheet, lastDataRow, "A", "O"); err != nil {
			return fmt.Errorf("failed to apply bottom border: %w", err)
		}

		// Find and update "Жами" row (totals)
		rows, err := f.GetRows(sheet)
		if err == nil {
			for rowIdx := lastDataRow; rowIdx < lastDataRow+5 && rowIdx <= len(rows); rowIdx++ {
				if rowIdx-1 < len(rows) {
					row := rows[rowIdx-1]
					for _, cellValue := range row {
						if cellValue == "Жами" || cellValue == "Жами:" {
							if totalGenerationLoss > 0 {
								set(fmt.Sprintf("N%d", rowIdx), totalGenerationLoss)
							}
							if totalIdleDischargeVolume > 0 {
								set(fmt.Sprintf("O%d", rowIdx), totalIdleDischargeVolume)
							}
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

// processReservoirDevices fills the reservoir devices section with data
func (g *Generator) processReservoirDevices(
	f *excelize.File,
	sheet string,
	section *SectionInfo,
	devices []*reservoirdevicesummary.ResponseModel,
	orgParentMap map[int64]*int64,
	set func(cell string, value interface{}),
) error {
	if len(devices) == 0 {
		if err := f.RemoveRow(sheet, section.TemplateRow); err != nil {
			return fmt.Errorf("failed to remove template row: %w", err)
		}
		return nil
	}

	// Sort by parent_id → org_id
	orgIDs := make([]int64, 0, len(devices))
	deviceByOrg := make(map[int64]*reservoirdevicesummary.ResponseModel, len(devices))
	for _, d := range devices {
		orgIDs = append(orgIDs, d.OrganizationID)
		deviceByOrg[d.OrganizationID] = d
	}
	orgIDs = sortOrgIDs(orgIDs, orgParentMap)

	// Duplicate template row N-1 times
	for i := 1; i < len(orgIDs); i++ {
		if err := f.DuplicateRow(sheet, section.TemplateRow); err != nil {
			return fmt.Errorf("failed to duplicate row %d: %w", section.TemplateRow, err)
		}
	}

	// Totals for "ЖАМИ" row
	var totalTotal, totalInstalled, totalOperational, totalFaulty, totalActive, totalAutomation int

	for i, orgID := range orgIDs {
		rowNum := section.TemplateRow + i
		d := deviceByOrg[orgID]

		set(fmt.Sprintf("A%d", rowNum), i+1)
		set(fmt.Sprintf("B%d", rowNum), d.OrganizationName)
		set(fmt.Sprintf("D%d", rowNum), d.CountTotal)
		set(fmt.Sprintf("E%d", rowNum), d.CountActive)
		set(fmt.Sprintf("F%d", rowNum), d.CountAutomationScope)

		// G: percentage of automation scope vs total
		if d.CountTotal > 0 {
			set(fmt.Sprintf("G%d", rowNum), fmt.Sprintf("(%d%%)", d.CountAutomationScope*100/d.CountTotal))
		}

		set(fmt.Sprintf("H%d", rowNum), d.CountInstalled)
		if d.CountTotal > 0 {
			set(fmt.Sprintf("I%d", rowNum), fmt.Sprintf("(%d%%)", d.CountInstalled*100/d.CountTotal))
		}

		set(fmt.Sprintf("J%d", rowNum), d.CountOperational)
		if d.CountTotal > 0 {
			set(fmt.Sprintf("K%d", rowNum), fmt.Sprintf("(%d%%)", d.CountOperational*100/d.CountTotal))
		}

		set(fmt.Sprintf("L%d", rowNum), d.CountFaulty)
		if d.CountTotal > 0 {
			set(fmt.Sprintf("M%d", rowNum), fmt.Sprintf("(%d%%)", d.CountFaulty*100/d.CountTotal))
		}

		if d.Criterion1 != nil {
			set(fmt.Sprintf("N%d", rowNum), *d.Criterion1)
		}
		if d.Criterion2 != nil {
			set(fmt.Sprintf("O%d", rowNum), *d.Criterion2)
		}

		totalTotal += d.CountTotal
		totalInstalled += d.CountInstalled
		totalOperational += d.CountOperational
		totalFaulty += d.CountFaulty
		totalActive += d.CountActive
		totalAutomation += d.CountAutomationScope
	}

	// Bottom border
	lastRow := section.TemplateRow + len(orgIDs) - 1
	if err := g.applyBottomBorder(f, sheet, lastRow, "A", "O"); err != nil {
		return fmt.Errorf("failed to apply bottom border: %w", err)
	}

	// Update "ЖАМИ" totals row
	rows, err := f.GetRows(sheet)
	if err == nil {
		for rowIdx := lastRow + 1; rowIdx < lastRow+5 && rowIdx <= len(rows); rowIdx++ {
			if rowIdx-1 < len(rows) {
				row := rows[rowIdx-1]
				for _, cellValue := range row {
					if cellValue == "ЖАМИ:" || cellValue == "ЖАМИ" {
						set(fmt.Sprintf("D%d", rowIdx), totalTotal)
						set(fmt.Sprintf("E%d", rowIdx), totalActive)
						set(fmt.Sprintf("F%d", rowIdx), totalAutomation)
						if totalTotal > 0 {
							set(fmt.Sprintf("G%d", rowIdx), fmt.Sprintf("(%d%%)", totalAutomation*100/totalTotal))
						}
						set(fmt.Sprintf("H%d", rowIdx), totalInstalled)
						if totalTotal > 0 {
							set(fmt.Sprintf("I%d", rowIdx), fmt.Sprintf("(%d%%)", totalInstalled*100/totalTotal))
						}
						set(fmt.Sprintf("J%d", rowIdx), totalOperational)
						if totalTotal > 0 {
							set(fmt.Sprintf("K%d", rowIdx), fmt.Sprintf("(%d%%)", totalOperational*100/totalTotal))
						}
						set(fmt.Sprintf("L%d", rowIdx), totalFaulty)
						if totalTotal > 0 {
							set(fmt.Sprintf("M%d", rowIdx), fmt.Sprintf("(%d%%)", totalFaulty*100/totalTotal))
						}
						break
					}
				}
			}
		}
	}

	return nil
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

// calcRowHeight returns the row height based on text length and characters per line.
// Base height is 35 (covers 1–2 lines). Each additional line adds 15.
func calcRowHeight(text string, charsPerLine int) float64 {
	const baseHeight = 35.0
	const extraPerLine = 15.0
	const baseLines = 2

	if charsPerLine <= 0 {
		return baseHeight
	}

	lines := 0
	for _, part := range strings.Split(text, "\n") {
		runeCount := len([]rune(strings.TrimSpace(part)))
		if runeCount == 0 {
			lines++
			continue
		}
		lines += (runeCount-1)/charsPerLine + 1
	}
	if lines <= baseLines {
		return baseHeight
	}
	return baseHeight + float64(lines-baseLines)*extraPerLine
}

// processVisits fills the visits section with data
func (g *Generator) processVisits(
	f *excelize.File,
	sheet string,
	section *SectionInfo,
	visits []*visit.ResponseModel,
	orgParentMap map[int64]*int64,
	loc *time.Location,
	set func(cell string, value interface{}),
) error {
	// If no visits, remove entire section (title row + header row + template row)
	if len(visits) == 0 {
		titleRow := section.HeaderRow - 1
		for i := 0; i < 3; i++ {
			if err := f.RemoveRow(sheet, titleRow); err != nil {
				return fmt.Errorf("failed to remove visits section row: %w", err)
			}
		}
		return nil
	}

	// Sort visits by org hierarchy
	sortVisitsByOrg(visits, orgParentMap)

	// Duplicate the template row for additional visits (len(visits) - 1) times
	for i := 1; i < len(visits); i++ {
		if err := f.DuplicateRow(sheet, section.TemplateRow); err != nil {
			return fmt.Errorf("failed to duplicate row %d: %w", section.TemplateRow, err)
		}
	}

	// Fill data for each visit
	for i, v := range visits {
		row := section.TemplateRow + i

		// A: № (numbering)
		set(fmt.Sprintf("A%d", row), i+1)

		// B: Organization name (B-E merged cells, write to first cell)
		set(fmt.Sprintf("B%d", row), v.OrganizationName)

		// F: Description - event name (F-L merged cells, write to first cell)
		set(fmt.Sprintf("F%d", row), v.Description)

		// M: Responsible name (M-O merged cells, write to first cell)
		set(fmt.Sprintf("M%d", row), v.ResponsibleName)

		_ = f.SetRowHeight(sheet, row, calcRowHeight(v.Description, 65))
	}

	// Restore bottom border for the last data row
	lastRow := section.TemplateRow + len(visits) - 1
	if err := g.applyBottomBorder(f, sheet, lastRow, "A", "O"); err != nil {
		return fmt.Errorf("failed to apply bottom border: %w", err)
	}

	return nil
}

// processIncidents fills the incidents section with data
func (g *Generator) processIncidents(
	f *excelize.File,
	sheet string,
	section *SectionInfo,
	incidents []*incident.ResponseModel,
	orgParentMap map[int64]*int64,
	loc *time.Location,
	set func(cell string, value interface{}),
) error {
	// If no incidents, delete template row
	if len(incidents) == 0 {
		if err := f.RemoveRow(sheet, section.TemplateRow); err != nil {
			return fmt.Errorf("failed to remove template row %d: %w", section.TemplateRow, err)
		}
		return nil
	}

	// Sort incidents by org hierarchy (NULL org_id first)
	sortIncidentsByOrg(incidents, orgParentMap)

	// Duplicate the template row for additional incidents (len(incidents) - 1) times
	for i := 1; i < len(incidents); i++ {
		if err := f.DuplicateRow(sheet, section.TemplateRow); err != nil {
			return fmt.Errorf("failed to duplicate row %d: %w", section.TemplateRow, err)
		}
	}

	// Fill data for each incident
	for i, inc := range incidents {
		row := section.TemplateRow + i

		// A: № (numbering)
		set(fmt.Sprintf("A%d", row), i+1)

		// B: Incident time (dd.MM.yyyy HH:mm)
		set(fmt.Sprintf("B%d", row), inc.IncidentTime.In(loc).Format("02.01.2006 15:04"))

		// C: Organization name (C-E merged cells, write to first cell)
		// Use default text if organization is NULL
		orgName := "Энергия хосил қилувчи корхона ва сув омборлар"
		if inc.OrganizationName != nil && *inc.OrganizationName != "" {
			orgName = *inc.OrganizationName
		}
		set(fmt.Sprintf("C%d", row), orgName)

		// F: Description (F-O merged cells, write to first cell)
		set(fmt.Sprintf("F%d", row), inc.Description)

		_ = f.SetRowHeight(sheet, row, calcRowHeight(inc.Description, 80))
	}

	// Restore bottom border for the last data row
	lastRow := section.TemplateRow + len(incidents) - 1
	if err := g.applyBottomBorder(f, sheet, lastRow, "A", "O"); err != nil {
		return fmt.Errorf("failed to apply bottom border: %w", err)
	}

	return nil
}

// processInfraEvents fills the infra events section with data grouped by category.
// The template has a 3-row block: label row (merged A:O) + header row + data template row.
// This block is duplicated for each category.
func (g *Generator) processInfraEvents(
	f *excelize.File,
	sheet string,
	section *SectionInfo,
	events []*infraevent.ResponseModel,
	categories []*infraeventcategory.Model,
	loc *time.Location,
	set func(cell string, value interface{}),
) error {
	// section.HeaderRow = label row (with P tag "infra")
	// section.TemplateRow = column headers row
	// section.TemplateRow + 1 = data template row
	labelRow := section.HeaderRow
	dataTemplateRow := section.TemplateRow + 1

	// Group events by category ID
	eventsByCategory := make(map[int64][]*infraevent.ResponseModel)
	for _, e := range events {
		eventsByCategory[e.CategoryID] = append(eventsByCategory[e.CategoryID], e)
	}

	// Filter categories to only those with events
	var activeCategories []*infraeventcategory.Model
	for _, cat := range categories {
		if len(eventsByCategory[cat.ID]) > 0 {
			activeCategories = append(activeCategories, cat)
		}
	}

	// If no active categories (no events at all), remove the entire 3-row block
	if len(activeCategories) == 0 {
		for i := 0; i < 3; i++ {
			if err := f.RemoveRow(sheet, labelRow); err != nil {
				return fmt.Errorf("failed to remove infra block row: %w", err)
			}
		}
		return nil
	}

	// Duplicate the 3-row block for additional active categories (all except the first).
	for i := len(activeCategories) - 1; i >= 1; i-- {
		for row := dataTemplateRow; row >= labelRow; row-- {
			if err := f.DuplicateRow(sheet, row); err != nil {
				return fmt.Errorf("failed to duplicate infra block row %d: %w", row, err)
			}
		}
	}

	// Now we have N blocks of 3 rows each, starting from labelRow.
	// Fill each block (only active categories — all have events).
	currentRow := labelRow
	for catIdx, cat := range activeCategories {
		catLabelRow := currentRow
		catHeaderRow := currentRow + 1
		catDataRow := currentRow + 2

		// Set category label in the merged label row
		set(fmt.Sprintf("A%d", catLabelRow), cat.Label)

		catEvents := eventsByCategory[cat.ID]

		// Sort events by occurred_at
		slices.SortFunc(catEvents, func(a, b *infraevent.ResponseModel) int {
			return a.OccurredAt.Compare(b.OccurredAt)
		})

		// Duplicate data template row for additional events
		for i := 1; i < len(catEvents); i++ {
			if err := f.DuplicateRow(sheet, catDataRow); err != nil {
				return fmt.Errorf("failed to duplicate data row for category %s: %w", cat.Slug, err)
			}
		}

		// Fill data rows
		now := time.Now().In(loc)
		for i, ev := range catEvents {
			row := catDataRow + i

			// A: sequential number
			set(fmt.Sprintf("A%d", row), i+1)

			// B: organization name
			set(fmt.Sprintf("B%d", row), ev.OrganizationName)

			// C: occurred_at (dd.MM.yyyy HH:mm)
			set(fmt.Sprintf("C%d", row), ev.OccurredAt.In(loc).Format("02.01.2006 15:04"))

			// D: duration
			var duration time.Duration
			if ev.RestoredAt != nil {
				duration = ev.RestoredAt.Sub(ev.OccurredAt)
			} else {
				duration = now.Sub(ev.OccurredAt)
			}
			set(fmt.Sprintf("D%d", row), formatDuration(duration))

			// E: restored_at (dd.MM.yyyy HH:mm) or empty
			if ev.RestoredAt != nil {
				set(fmt.Sprintf("E%d", row), ev.RestoredAt.In(loc).Format("02.01.2006 15:04"))
			}

			// F: description (merged F-I)
			set(fmt.Sprintf("F%d", row), ev.Description)

			// J: remediation (merged J-M)
			if ev.Remediation != nil {
				set(fmt.Sprintf("J%d", row), *ev.Remediation)
			}

			// N: notes (merged N-O)
			if ev.Notes != nil {
				set(fmt.Sprintf("N%d", row), *ev.Notes)
			}

			// Adjust row height for long text
			maxLen := len([]rune(ev.Description))
			if ev.Remediation != nil && len([]rune(*ev.Remediation)) > maxLen {
				maxLen = len([]rune(*ev.Remediation))
			}
			if maxLen > 40 {
				_ = f.SetRowHeight(sheet, row, calcRowHeight(ev.Description, 40))
			}
		}

		// Apply bottom border on last data row
		lastDataRow := catDataRow + len(catEvents) - 1
		if err := g.applyBottomBorder(f, sheet, lastDataRow, "A", "O"); err != nil {
			return fmt.Errorf("failed to apply bottom border for category %s: %w", cat.Slug, err)
		}

		// Next block: label + header + data rows
		_ = catIdx
		_ = catHeaderRow
		currentRow = lastDataRow + 1
	}

	return nil
}
