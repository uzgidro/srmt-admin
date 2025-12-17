package reservoirsummary

import (
	"fmt"
	"time"

	reservoirsummarymodel "srmt-admin/internal/lib/model/reservoir-summary"

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
func (g *Generator) GenerateExcel(date string, data []*reservoirsummarymodel.ResponseModel) (*excelize.File, error) {
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

	// Filter out summary row (where OrganizationID == nil)
	var filteredData []*reservoirsummarymodel.ResponseModel
	for _, item := range data {
		if item.OrganizationID != nil {
			filteredData = append(filteredData, item)
		}
	}

	// Set date in cell L2
	sheet := f.GetSheetName(0) // Get first sheet name

	var writeErr error
	set := func(cell string, value interface{}) {
		if writeErr != nil {
			return // Если уже есть ошибка, пропускаем
		}
		if err := f.SetCellValue(sheet, cell, value); err != nil {
			writeErr = fmt.Errorf("failed to set cell %s: %w", cell, err)
		}
	}

	set("L2", parsedDate)

	// Populate level data in cells B6-B21 (skip last organization and skip B18-B19, B22-B23)
	currentLevelCells := []string{"B6", "B8", "B10", "B12", "B14", "B16", "B20"}
	differenceCells := []string{"B7", "B9", "B11", "B13", "B15", "B17", "B21"}

	// Calculate the number of organizations to display (all except the last one)
	maxIndex := len(filteredData)
	if maxIndex > 7 {
		maxIndex = 7 // Limit to 7 cells max
	}

	// Populate cells with level data
	for i := 0; i < maxIndex; i++ {
		org := filteredData[i]
		set(currentLevelCells[i], org.Level.Current)
		set(differenceCells[i], org.Level.Current-org.Level.Previous)
	}

	// Populate volume data in cells C6-E16 (skip last organization)
	currentVolumeCells := []string{"C6", "C8", "C10", "C12", "C14", "C16"}
	volumeDifferenceCells := []string{"C7", "C9", "C11", "C13", "C15", "C17"}
	pastYearVolumeCells := []string{"D6", "D8", "D10", "D12", "D14", "D16"}
	twoYearsAgoVolumeCells := []string{"E6", "E8", "E10", "E12", "E14", "E16"}

	// Calculate the number of organizations to display for volume (max 6 cells)
	maxVolumeIndex := len(filteredData) // Skip last organization
	if maxVolumeIndex > 6 {
		maxVolumeIndex = 6 // Limit to 6 cells max
	}

	// Populate cells with volume data
	for i := 0; i < maxVolumeIndex; i++ {
		org := filteredData[i]
		set(currentVolumeCells[i], org.Volume.Current)
		set(volumeDifferenceCells[i], org.Volume.Current-org.Volume.Previous)
		set(pastYearVolumeCells[i], org.Volume.YearAgo)
		set(twoYearsAgoVolumeCells[i], org.Volume.TwoYearsAgo)
	}

	// Populate income data in cells F6-H16
	currentIncomeCells := []string{"F6", "F8", "F10", "F12", "F14", "F16", "F20", "F22"}
	incomeDifferenceCells := []string{"F7", "F9", "F11", "F13", "F15", "F17", "F21", "F23"}
	pastYearIncomeCells := []string{"G6", "G8", "G10", "G12", "G14", "G16"}
	twoYearsAgoIncomeCells := []string{"H6", "H8", "H10", "H12", "H14", "H16"}

	// Calculate the number of organizations to display for income current/diff (max 8 cells)
	maxIncomeIndex := len(filteredData)
	if maxIncomeIndex > 8 {
		maxIncomeIndex = 8 // Limit to 8 cells max
	}

	// Populate cells with current income and income difference
	for i := 0; i < maxIncomeIndex; i++ {
		org := filteredData[i]
		set(currentIncomeCells[i], org.Income.Current)
		set(incomeDifferenceCells[i], org.Income.Current-org.Income.Previous)
	}

	// Calculate the number of organizations to display for income past year/two years ago (max 6 cells)
	maxIncomeHistoryIndex := len(filteredData)
	if maxIncomeHistoryIndex > 6 {
		maxIncomeHistoryIndex = 6 // Limit to 6 cells max
	}

	// Populate cells with past year and two years ago income
	for i := 0; i < maxIncomeHistoryIndex; i++ {
		org := filteredData[i]
		set(pastYearIncomeCells[i], org.Income.YearAgo)
		set(twoYearsAgoIncomeCells[i], org.Income.TwoYearsAgo)
	}

	// Populate release data in cells I6-K16
	currentReleaseCells := []string{"I6", "I8", "I10", "I12", "I14", "I16", "I20", "I22"}
	releaseDifferenceCells := []string{"I7", "I9", "I11", "I13", "I15", "I17", "I21", "I23"}
	pastYearReleaseCells := []string{"J6", "J8", "J10", "J12", "J14", "J16"}
	twoYearsAgoReleaseCells := []string{"K6", "K8", "K10", "K12", "K14", "K16"}

	// Calculate the number of organizations to display for release current/diff (max 8 cells)
	maxReleaseIndex := len(filteredData)
	if maxReleaseIndex > 8 {
		maxReleaseIndex = 8 // Limit to 8 cells max
	}

	// Populate cells with current release and release difference
	for i := 0; i < maxReleaseIndex; i++ {
		org := filteredData[i]
		set(currentReleaseCells[i], org.Release.Current)
		set(releaseDifferenceCells[i], org.Release.Current-org.Release.Previous)
	}

	// Calculate the number of organizations to display for release past year/two years ago (max 6 cells)
	maxReleaseHistoryIndex := len(filteredData)
	if maxReleaseHistoryIndex > 6 {
		maxReleaseHistoryIndex = 6 // Limit to 6 cells max
	}

	// Populate cells with past year and two years ago release
	for i := 0; i < maxReleaseHistoryIndex; i++ {
		org := filteredData[i]
		set(pastYearReleaseCells[i], org.Release.YearAgo)
		set(twoYearsAgoReleaseCells[i], org.Release.TwoYearsAgo)
	}

	// Populate incoming volume (total income) data in cells L6-M16
	currentYearIncomingVolumeCells := []string{"L6", "L8", "L10", "L12", "L14", "L16"}
	pastYearIncomingVolumeCells := []string{"M6", "M8", "M10", "M12", "M14", "M16"}

	// Calculate the number of organizations to display for incoming volume (max 6 cells)
	maxIncomingVolumeIndex := len(filteredData)
	if maxIncomingVolumeIndex > 6 {
		maxIncomingVolumeIndex = 6 // Limit to 6 cells max
	}

	// Populate cells with incoming volume data
	for i := 0; i < maxIncomingVolumeIndex; i++ {
		org := filteredData[i]
		set(currentYearIncomingVolumeCells[i], org.IncomingVolume)
		set(pastYearIncomingVolumeCells[i], org.IncomingVolumePrevYear)
	}

	// Populate modsnow data in cells N6-O22 (skip 3rd index element - index 2)
	currentYearModsnowCells := []string{"N6", "N8", "N12", "N14", "N16", "N20", "N22"}
	pastYearModsnowCells := []string{"O6", "O8", "O12", "O14", "O16", "O20", "O22"}

	// Calculate the number of organizations to display for modsnow (max 8, but skip index 2)
	maxModsnowIndex := len(filteredData)
	if maxModsnowIndex > 8 {
		maxModsnowIndex = 8 // Limit to 8 organizations max
	}

	// Populate cells with modsnow data, skipping index 2
	cellIndex := 0
	for i := 0; i < maxModsnowIndex; i++ {
		// Skip the 3rd index element (index 2)
		if i == 2 {
			continue
		}
		org := filteredData[i]
		set(currentYearModsnowCells[cellIndex], org.Modsnow.Current)
		set(pastYearModsnowCells[cellIndex], org.Modsnow.YearAgo)
		cellIndex++
	}

	if writeErr != nil {
		f.Close()
		return nil, writeErr
	}

	if err := f.UpdateLinkedValue(); err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to calculate formulas: %w", err)
	}

	return f, nil
}
