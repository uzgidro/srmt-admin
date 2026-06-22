package reservoirsummary

import (
	"fmt"
	"time"

	reservoirsummarymodel "srmt-admin/internal/lib/model/reservoir-summary"
	"srmt-admin/internal/lib/service/excel/templates"

	"github.com/xuri/excelize/v2"
)

// Generator handles Excel file generation for reservoir summaries.
// The same generator code drives two distinct templates (res-summary.xlsx and
// res-summary-filter.xlsx), so the template name is part of the generator's
// identity rather than a per-call argument.
type Generator struct {
	overrideDir string
	template    string // templates.ResSummary or templates.ResSummaryFilt
}

// New creates a new Generator bound to a specific embedded template.
// It panics on an empty template name — wrong wiring is a programmer error,
// not runtime data we want to surface as an HTTP 500 on every report request.
func New(overrideDir, template string) *Generator {
	if template == "" {
		panic("reservoir-summary: template name must not be empty")
	}
	return &Generator{
		overrideDir: overrideDir,
		template:    template,
	}
}

// GenerateExcel creates an Excel file from the template with the specified date.
//
// configByOrgID is the per-org reservoir_summary_config map used to gate
// optional per-org behaviour. Today it only governs ModsnowEnabled (false →
// leave the modsnow cell empty), but the map is the long-term seam for
// any future per-org Excel toggle so we don't need to grow the signature
// every release.
//
// Semantics of configByOrgID:
//   - nil          → legacy behaviour: render modsnow for every org
//     (used by filter/manual-comparison exports that haven't been wired
//     to load the config yet).
//   - non-nil      → strict config-driven mode: only orgs whose config
//     has ModsnowEnabled=true get modsnow rendered. Missing key or
//     ModsnowEnabled=false → cell stays empty. Used by the
//     /reservoir-summary/export handler.
func (g *Generator) GenerateExcel(
	date string,
	data []*reservoirsummarymodel.ResponseModel,
	configByOrgID map[int64]reservoirsummarymodel.ReservoirSummaryConfig,
	authorShortName string,
) (*excelize.File, error) {
	// Open template (embedded, with optional override directory)
	f, err := templates.Open(g.template, g.overrideDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open template: %w", err)
	}

	// Parse date string (format: YYYY-MM-DD)
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to parse date: %w", err)
	}

	// The ИТОГО row (OrganizationID == nil) is silently dropped — Excel
	// computes totals via SUM formulas in rows 20-21 of both templates.
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

	// Populate level data in cells B6-B23 (skip B18-B19)
	currentLevelCells := []string{"B6", "B8", "B10", "B12", "B14", "B16", "B18", "B22"}
	differenceCells := []string{"B7", "B9", "B11", "B13", "B15", "B17", "B19", "B23"}

	// Calculate the number of organizations to display
	maxIndex := len(filteredData)
	if maxIndex > len(currentLevelCells) {
		maxIndex = len(currentLevelCells)
	}

	// Populate cells with level data
	for i := 0; i < maxIndex; i++ {
		org := filteredData[i]
		set(currentLevelCells[i], org.Level.Current)
		set(differenceCells[i], org.Level.Current-org.Level.Previous)
	}

	// Populate volume data in cells C6-E22 (skip C18-C19)
	currentVolumeCells := []string{"C6", "C8", "C10", "C12", "C14", "C16", "C18", "C22"}
	volumeDifferenceCells := []string{"C7", "C9", "C11", "C13", "C15", "C17", "C19", "C23"}
	pastYearVolumeCells := []string{"D6", "D8", "D10", "D12", "D14", "D16", "D18", "D22"}
	twoYearsAgoVolumeCells := []string{"E6", "E8", "E10", "E12", "E14", "E16", "E18", "E22"}

	// Calculate the number of organizations to display for volume
	maxVolumeIndex := len(filteredData)
	if maxVolumeIndex > len(currentVolumeCells) {
		maxVolumeIndex = len(currentVolumeCells)
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
	currentIncomeCells := []string{"F6", "F8", "F10", "F12", "F14", "F16", "F18", "F22"}
	incomeDifferenceCells := []string{"F7", "F9", "F11", "F13", "F15", "F17", "F19", "F23"}
	pastYearIncomeCells := []string{"G6", "G8", "G10", "G12", "G14", "G16", "G18", "G22"}
	twoYearsAgoIncomeCells := []string{"H6", "H8", "H10", "H12", "H14", "H16", "H18", "H22"}

	// Calculate the number of organizations to display for income current/diff
	maxIncomeIndex := len(filteredData)
	if maxIncomeIndex > len(currentIncomeCells) {
		maxIncomeIndex = len(currentIncomeCells)
	}

	// Populate cells with current income and income difference
	for i := 0; i < maxIncomeIndex; i++ {
		org := filteredData[i]
		set(currentIncomeCells[i], org.Income.Current)
		set(incomeDifferenceCells[i], org.Income.Current-org.Income.Previous)
		set(pastYearIncomeCells[i], org.Income.YearAgo)
		set(twoYearsAgoIncomeCells[i], org.Income.TwoYearsAgo)
	}

	// Populate release data in cells I6-K16
	currentReleaseCells := []string{"I6", "I8", "I10", "I12", "I14", "I16", "I18", "I22"}
	releaseDifferenceCells := []string{"I7", "I9", "I11", "I13", "I15", "I17", "I19", "I23"}
	pastYearReleaseCells := []string{"J6", "J8", "J10", "J12", "J14", "J16", "J18", "J22"}
	twoYearsAgoReleaseCells := []string{"K6", "K8", "K10", "K12", "K14", "K16", "K18", "K22"}

	// Calculate the number of organizations to display for release current/diff
	maxReleaseIndex := len(filteredData)
	if maxReleaseIndex > len(currentReleaseCells) {
		maxReleaseIndex = len(currentReleaseCells)
	}

	// Populate cells with current release and release difference
	for i := 0; i < maxReleaseIndex; i++ {
		org := filteredData[i]
		set(currentReleaseCells[i], org.Release.Current)
		set(releaseDifferenceCells[i], org.Release.Current-org.Release.Previous)
		set(pastYearReleaseCells[i], org.Release.YearAgo)
		set(twoYearsAgoReleaseCells[i], org.Release.TwoYearsAgo)
	}

	// Populate incoming volume (total income) data in cells L6-M22
	currentYearIncomingVolumeCells := []string{"L6", "L8", "L10", "L12", "L14", "L16", "L18", "L22"}
	pastYearIncomingVolumeCells := []string{"M6", "M8", "M10", "M12", "M14", "M16", "M18", "M22"}

	// Calculate the number of organizations to display for incoming volume
	maxIncomingVolumeIndex := len(filteredData)
	if maxIncomingVolumeIndex > len(currentYearIncomingVolumeCells) {
		maxIncomingVolumeIndex = len(currentYearIncomingVolumeCells)
	}

	// Populate cells with incoming volume data
	for i := 0; i < maxIncomingVolumeIndex; i++ {
		org := filteredData[i]
		set(currentYearIncomingVolumeCells[i], org.IncomingVolume)
		set(pastYearIncomingVolumeCells[i], org.IncomingVolumePrevYear)
	}

	// Populate modsnow data in cells N6-O22. Per-org gated by
	// reservoir_summary_config.modsnow_enabled: false / missing config →
	// leave the cell empty (previously: hardcoded `if i == 2 { continue }`
	// for the Сардоба slot). Organisations with OrganizationID==nil are
	// already filtered out above (ИТОГО row).
	currentYearModsnowCells := []string{"N6", "N8", "N10", "N12", "N14", "N16", "N18", "N22"}
	pastYearModsnowCells := []string{"O6", "O8", "O10", "O12", "O14", "O16", "O18", "O22"}

	maxModsnowIndex := len(filteredData)
	if maxModsnowIndex > len(currentYearModsnowCells) {
		maxModsnowIndex = len(currentYearModsnowCells)
	}

	for i := 0; i < maxModsnowIndex; i++ {
		org := filteredData[i]
		// nil map → legacy "render everything" path for callers that
		// haven't been updated to pass per-org config.
		enabled := true
		if configByOrgID != nil {
			enabled = false
			if org.OrganizationID != nil {
				if cfg, ok := configByOrgID[*org.OrganizationID]; ok && cfg.ModsnowEnabled {
					enabled = true
				}
			}
		}
		if !enabled {
			// Explicitly clear whatever the template might have had so a
			// disabled org always shows blank, not stale template data.
			set(currentYearModsnowCells[i], "")
			set(pastYearModsnowCells[i], "")
			continue
		}
		set(currentYearModsnowCells[i], org.Modsnow.Current)
		set(pastYearModsnowCells[i], org.Modsnow.YearAgo)
	}

	set("K25", authorShortName)

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
