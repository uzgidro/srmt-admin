package filter

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"srmt-admin/internal/lib/model/filtration"

	"github.com/xuri/excelize/v2"
)

// Generator adds filtration/piezometer blocks to an already-filled reservoir summary Excel file.
type Generator struct{}

func New() *Generator {
	return &Generator{}
}

// blockStyles holds style IDs extracted from the template filtration block (rows 26–34).
type blockStyles struct {
	header       int // A26 — reservoir name header
	labelBold    int // A28 — "Жами, л/с" label
	colHeader    int // B28 — date column header
	diffHeader   int // D28 — "+,-" header
	normHeader   int // E28 — "меъёр" header
	totalFlowNum int // B29 — filtration total (bold)
	subFlowLabel int // A30 — sub-location label (not bold, left-aligned)
	subFlowNum   int // B30 — sub-location flow (not bold)
	piezoLabel   int // I29 — piezometer name (not bold)
	piezoNum     int // K29 — piezometer level (not bold)
	countsLabel  int // I30 — "Жами, дона" (bold)
	countsNum    int // K30 — count value (bold)
	normText     int // L30 — norm assessment text
	sigLeft  int // D34 — signature left
	sigRight int // L34 — signature right (author name)
}

// filtrationOrder defines the display order of filtration blocks.
var filtrationOrder = map[string]int{
	"Андижон":   1,
	"Чорвок":    2,
	"Хисорак":   3,
	"Топаланг":  4,
	"Охангарон": 5,
	"Сардоба":   6,
}

// FillFiltrationBlocks writes filtration/piezometer blocks (section 2) into the Excel file.
// The file should already have the reservoir summary filled (section 1).
func (g *Generator) FillFiltrationBlocks(
	f *excelize.File, sheet string,
	comparisons []filtration.OrgComparison,
	authorShort string,
) error {
	styles := extractStyles(f, sheet)
	_ = f.SetCellValue(sheet, "K25", "")
	clearTemplateBlock(f, sheet)
	sortComparisons(comparisons)

	cursor := 26
	for _, comp := range comparisons {
		cursor = writeBlock(f, sheet, cursor, comp, styles)
		cursor++ // blank row between blocks
	}

	// Signature
	_ = f.MergeCell(sheet, cell("C", cursor), cell("G", cursor))
	_ = f.SetCellValue(sheet, cell("C", cursor), "Вазиятлар маркази \nтезкор навбатчиси")
	_ = f.SetCellStyle(sheet, cell("C", cursor), cell("G", cursor), styles.sigLeft)
	_ = f.SetRowHeight(sheet, cursor, 35)

	_ = f.MergeCell(sheet, cell("J", cursor), cell("N", cursor))
	_ = f.SetCellValue(sheet, cell("J", cursor), authorShort)
	_ = f.SetCellStyle(sheet, cell("J", cursor), cell("N", cursor), styles.sigRight)

	// Print area — delete any existing one first, then set new
	_ = f.DeleteDefinedName(&excelize.DefinedName{
		Name:  "_xlnm.Print_Area",
		Scope: sheet,
	})
	_ = f.SetDefinedName(&excelize.DefinedName{
		Name:     "_xlnm.Print_Area",
		RefersTo: fmt.Sprintf("'%s'!$A$1:$O$%d", sheet, cursor+1),
		Scope:    sheet,
	})

	return nil
}

// ---------------------------------------------------------------------------
// Template cleanup
// ---------------------------------------------------------------------------

func extractStyles(f *excelize.File, sheet string) blockStyles {
	get := func(c string) int {
		id, _ := f.GetCellStyle(sheet, c)
		return id
	}
	return blockStyles{
		header:       get("A26"),
		labelBold:    get("A28"),
		colHeader:    get("B28"),
		diffHeader:   get("D28"),
		normHeader:   get("E28"),
		totalFlowNum: get("B29"),
		subFlowLabel: get("A30"),
		subFlowNum:   get("B30"),
		piezoLabel:   get("I29"),
		piezoNum:     get("K29"),
		countsLabel:  get("I30"),
		countsNum:    get("K30"),
		normText:     get("L30"),
		sigLeft:  get("D34"),
		sigRight: get("L34"),
	}
}

func clearTemplateBlock(f *excelize.File, sheet string) {
	merges := [][2]string{
		{"A26", "O26"}, {"A28", "A29"},
		{"I28", "J28"}, {"I29", "J29"}, {"I30", "J30"},
		{"I31", "J31"}, {"I32", "J32"}, {"L30", "O32"},
		{"D34", "G34"}, {"L34", "N34"},
	}
	for _, m := range merges {
		_ = f.UnmergeCell(sheet, m[0], m[1])
	}
	for row := 26; row <= 34; row++ {
		for col := 1; col <= 15; col++ {
			c, _ := excelize.CoordinatesToCellName(col, row)
			_ = f.SetCellValue(sheet, c, "")
		}
	}
}

// ---------------------------------------------------------------------------
// Block writer
// ---------------------------------------------------------------------------

func writeBlock(
	f *excelize.File, sheet string, startRow int,
	comp filtration.OrgComparison, st blockStyles,
) int {
	cursor := startRow

	// Historical lookup maps
	histLocMap := make(map[int64]*float64)
	histPiezoMap := make(map[int64]*float64)
	if comp.Historical != nil {
		for _, loc := range comp.Historical.Locations {
			histLocMap[loc.ID] = loc.FlowRate
		}
		for _, p := range comp.Historical.Piezometers {
			histPiezoMap[p.ID] = p.Level
		}
	}

	// Filtration totals
	var totalCurr, totalHist, totalNorm float64
	for _, loc := range comp.Current.Locations {
		totalCurr += pval(loc.FlowRate)
		totalNorm += pval(loc.Norm)
		if hf, ok := histLocMap[loc.ID]; ok {
			totalHist += pval(hf)
		}
	}

	// --- Header row (merged A:O) ---
	headerText := fmt.Sprintf("%s сатҳи - %.2f м,  ҳажм - %.2f млн м3",
		comp.OrganizationName, pval(comp.Current.Level), pval(comp.Current.Volume))
	_ = f.MergeCell(sheet, cell("A", cursor), cell("O", cursor))
	_ = f.SetCellValue(sheet, cell("A", cursor), headerText)
	setRowStyle(f, sheet, cursor, 1, 15, st.header)
	cursor++

	// --- Spacer row ---
	_ = f.SetRowHeight(sheet, cursor, 12.6)
	cursor++

	// --- Column headers row ---
	colRow := cursor
	_ = f.MergeCell(sheet, cell("A", colRow), cell("A", colRow+1))
	_ = f.SetCellValue(sheet, cell("A", colRow), "Жами, л/с")
	_ = f.SetCellStyle(sheet, cell("A", colRow), cell("A", colRow+1), st.labelBold)

	histDate := fmtDate(comp.Historical)
	currDate := fmtDateStr(comp.Current.Date)

	_ = f.SetCellValue(sheet, cell("B", colRow), histDate)
	_ = f.SetCellValue(sheet, cell("C", colRow), currDate)
	_ = f.SetCellValue(sheet, cell("D", colRow), " +,-")
	_ = f.SetCellValue(sheet, cell("E", colRow), "меъёр")
	_ = f.SetCellValue(sheet, cell("F", colRow), " +,-")

	_ = f.MergeCell(sheet, cell("I", colRow), cell("J", colRow))
	_ = f.SetCellValue(sheet, cell("I", colRow), "Асосий пьезометрлар №")
	_ = f.SetCellStyle(sheet, cell("I", colRow), cell("J", colRow), st.labelBold)
	_ = f.SetCellValue(sheet, cell("K", colRow), histDate)
	_ = f.SetCellValue(sheet, cell("L", colRow), currDate)
	_ = f.SetCellValue(sheet, cell("M", colRow), " +,-")
	_ = f.SetCellValue(sheet, cell("N", colRow), "меёър")
	_ = f.SetCellValue(sheet, cell("O", colRow), " +,-")

	for _, c := range []string{"B", "C", "K", "L"} {
		_ = f.SetCellStyle(sheet, cell(c, colRow), cell(c, colRow), st.colHeader)
	}
	for _, c := range []string{"D", "F", "M", "O"} {
		_ = f.SetCellStyle(sheet, cell(c, colRow), cell(c, colRow), st.diffHeader)
	}
	for _, c := range []string{"E", "N"} {
		_ = f.SetCellStyle(sheet, cell(c, colRow), cell(c, colRow), st.normHeader)
	}
	cursor++

	// --- Data rows ---
	locations := comp.Current.Locations
	piezometers := comp.Current.Piezometers

	leftRows := 1 + len(locations)
	rightRows := len(piezometers) + 3
	dataRows := leftRows
	if rightRows > dataRows {
		dataRows = rightRows
	}

	for i := 0; i < dataRows; i++ {
		row := cursor + i

		// Left side: filtration
		if i == 0 {
			// Total row (bold, from row 29 style)
			setFlowCells(f, sheet, row, totalHist, totalCurr, totalNorm, st.totalFlowNum)
		} else if i-1 < len(locations) {
			// Sub-location rows (not bold, from row 30 style)
			loc := locations[i-1]
			label := loc.Name
			if i == 1 {
				label = "шундан: " + label
			}
			_ = f.SetCellValue(sheet, cell("A", row), label)
			_ = f.SetCellStyle(sheet, cell("A", row), cell("A", row), st.subFlowLabel)
			setFlowCells(f, sheet, row, pval(histLocMap[loc.ID]), pval(loc.FlowRate), pval(loc.Norm), st.subFlowNum)
		}

		// Right side: piezometers + counts (from row 29 I-O style)
		if i < len(piezometers) {
			p := piezometers[i]
			_ = f.MergeCell(sheet, cell("I", row), cell("J", row))
			_ = f.SetCellValue(sheet, cell("I", row), p.Name)
			_ = f.SetCellStyle(sheet, cell("I", row), cell("J", row), st.piezoLabel)

			currLevel := pval(p.Level)
			histLevel := pval(histPiezoMap[p.ID])
			norm := pval(p.Norm)

			_ = f.SetCellValue(sheet, cell("K", row), histLevel)
			_ = f.SetCellValue(sheet, cell("L", row), currLevel)
			_ = f.SetCellValue(sheet, cell("M", row), round2(currLevel-histLevel))
			if norm != 0 {
				_ = f.SetCellValue(sheet, cell("N", row), norm)
				_ = f.SetCellValue(sheet, cell("O", row), round2(currLevel-norm))
			}
			for _, c := range []string{"K", "L", "M", "N", "O"} {
				_ = f.SetCellStyle(sheet, cell(c, row), cell(c, row), st.piezoNum)
			}
		} else {
			countIdx := i - len(piezometers)
			switch countIdx {
			case 0:
				total := comp.Current.PiezoCounts.Pressure + comp.Current.PiezoCounts.NonPressure
				writePiezoCountRow(f, sheet, row, "Жами, дона", total, st)
				// Norm assessment text merged across 3 count rows
				endRow := row + 2
				_ = f.MergeCell(sheet, cell("L", row), cell("O", endRow))
				_ = f.SetCellValue(sheet, cell("L", row),
					"Мезон кўрсаткичлари доирасида, аномал кўрсаткичлар мавжуд эмас. ")
				_ = f.SetCellStyle(sheet, cell("L", row), cell("O", endRow), st.normText)
			case 1:
				writePiezoCountRow(f, sheet, row, "босимли", comp.Current.PiezoCounts.Pressure, st)
			case 2:
				writePiezoCountRow(f, sheet, row, "босимсиз", comp.Current.PiezoCounts.NonPressure, st)
			}
		}
	}

	return cursor + dataRows
}

func writePiezoCountRow(f *excelize.File, sheet string, row int, label string, count int, st blockStyles) {
	_ = f.MergeCell(sheet, cell("I", row), cell("J", row))
	_ = f.SetCellValue(sheet, cell("I", row), label)
	_ = f.SetCellStyle(sheet, cell("I", row), cell("J", row), st.countsLabel)
	_ = f.SetCellValue(sheet, cell("K", row), count)
	_ = f.SetCellStyle(sheet, cell("K", row), cell("K", row), st.countsNum)
}

func setFlowCells(f *excelize.File, sheet string, row int, hist, curr, norm float64, style int) {
	_ = f.SetCellValue(sheet, cell("B", row), round2(hist))
	_ = f.SetCellValue(sheet, cell("C", row), round2(curr))
	_ = f.SetCellValue(sheet, cell("D", row), round2(curr-hist))
	_ = f.SetCellValue(sheet, cell("E", row), round2(norm))
	_ = f.SetCellValue(sheet, cell("F", row), round2(curr-norm))
	for _, c := range []string{"B", "C", "D", "E", "F"} {
		_ = f.SetCellStyle(sheet, cell(c, row), cell(c, row), style)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func cell(col string, row int) string {
	return fmt.Sprintf("%s%d", col, row)
}

func setRowStyle(f *excelize.File, sheet string, row, startCol, endCol, style int) {
	for col := startCol; col <= endCol; col++ {
		c, _ := excelize.CoordinatesToCellName(col, row)
		_ = f.SetCellStyle(sheet, c, c, style)
	}
}

func pval(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func fmtDate(snap *filtration.ComparisonSnapshot) string {
	if snap == nil || snap.Date == "" {
		return ""
	}
	return fmtDateStr(snap.Date)
}

func fmtDateStr(dateStr string) string {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dateStr
	}
	return fmt.Sprintf("%02d.%02d.%02d й", t.Day(), t.Month(), t.Year()%100)
}

func sortComparisons(comparisons []filtration.OrgComparison) {
	sort.Slice(comparisons, func(i, j int) bool {
		return getOrder(comparisons[i].OrganizationName) < getOrder(comparisons[j].OrganizationName)
	})
}

func getOrder(name string) int {
	for key, val := range filtrationOrder {
		if strings.Contains(name, key) {
			return val
		}
	}
	return 99
}
