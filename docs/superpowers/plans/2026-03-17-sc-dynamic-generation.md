# SC Report Dynamic Generation Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace hardcoded org IDs in SC Excel template with dynamic generation sorted by parent_id → org_id.

**Architecture:** Each section (discharges, ges, mini, micro, visits, incidents) uses a single template row. At generation time, organizations are fetched from DB, sorted depth-first by parent hierarchy, and rows are duplicated dynamically. The handler passes flat data + org metadata to the generator, which handles grouping internally.

**Tech Stack:** Go 1.25, excelize/v2, chi/v5, Wire DI, PostgreSQL

**Spec:** `docs/superpowers/specs/2026-03-17-sc-dynamic-generation-design.md`

---

## Chunk 1: Sorting utility + repo method

### Task 1: Add `sortOrgIDs` sorting utility with tests

**Files:**
- Create: `internal/lib/service/excel/sc/sort.go`
- Create: `internal/lib/service/excel/sc/sort_test.go`

- [ ] **Step 1: Write failing tests for `sortOrgIDs`**

File: `internal/lib/service/excel/sc/sort_test.go`

```go
package sc

import (
	"slices"
	"testing"
)

func TestSortOrgIDs(t *testing.T) {
	tests := []struct {
		name      string
		orgIDs    []int64
		parentMap map[int64]*int64
		expected  []int64
	}{
		{
			name:      "empty input",
			orgIDs:    []int64{},
			parentMap: map[int64]*int64{},
			expected:  []int64{},
		},
		{
			name:      "single org no parent",
			orgIDs:    []int64{5},
			parentMap: map[int64]*int64{5: nil},
			expected:  []int64{5},
		},
		{
			name:   "flat orgs sorted by id",
			orgIDs: []int64{12, 5, 23},
			parentMap: map[int64]*int64{
				5:  nil,
				12: nil,
				23: nil,
			},
			expected: []int64{5, 12, 23},
		},
		{
			name:   "children grouped after parent depth-first",
			orgIDs: []int64{10, 5, 12, 3, 11},
			parentMap: map[int64]*int64{
				3:  nil,
				5:  nil,
				10: ptr(int64(3)),
				11: ptr(int64(3)),
				12: ptr(int64(5)),
			},
			// cascade 3, then children 10, 11, then cascade 5, then child 12
			expected: []int64{3, 10, 11, 5, 12},
		},
		{
			name:   "children without parent in list still sorted",
			orgIDs: []int64{10, 11},
			parentMap: map[int64]*int64{
				10: ptr(int64(3)),
				11: ptr(int64(3)),
			},
			// parent 3 not in orgIDs, children grouped under virtual parent 3
			expected: []int64{10, 11},
		},
		{
			name:   "org missing from parentMap treated as root",
			orgIDs: []int64{99, 5},
			parentMap: map[int64]*int64{
				5: nil,
			},
			expected: []int64{5, 99},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sortOrgIDs(tt.orgIDs, tt.parentMap)
			if !slices.Equal(result, tt.expected) {
				t.Errorf("sortOrgIDs() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -v -run TestSortOrgIDs ./internal/lib/service/excel/sc/...`
Expected: FAIL — `sortOrgIDs` undefined

- [ ] **Step 3: Implement `sortOrgIDs`**

File: `internal/lib/service/excel/sc/sort.go`

```go
package sc

import (
	"slices"

	"srmt-admin/internal/lib/model/incident"
	"srmt-admin/internal/lib/model/visit"
)

// sortOrgIDs sorts organization IDs depth-first by parent hierarchy.
// Parent orgs come first, immediately followed by their children sorted by ID.
// Root orgs (no parent) are sorted by ID among themselves.
func sortOrgIDs(orgIDs []int64, parentMap map[int64]*int64) []int64 {
	if len(orgIDs) == 0 {
		return orgIDs
	}

	// Build tree: parentID -> sorted children
	children := make(map[int64][]int64)  // parentID -> child orgIDs
	var roots []int64

	for _, id := range orgIDs {
		parentPtr, exists := parentMap[id]
		if !exists || parentPtr == nil {
			roots = append(roots, id)
		} else {
			children[*parentPtr] = append(children[*parentPtr], id)
		}
	}

	slices.Sort(roots)
	for k := range children {
		slices.Sort(children[k])
	}

	// Depth-first walk
	orgIDSet := make(map[int64]bool, len(orgIDs))
	for _, id := range orgIDs {
		orgIDSet[id] = true
	}

	result := make([]int64, 0, len(orgIDs))

	var walk func(id int64)
	walk = func(id int64) {
		if orgIDSet[id] {
			result = append(result, id)
		}
		for _, child := range children[id] {
			walk(child)
		}
	}

	// Walk from roots
	for _, root := range roots {
		walk(root)
	}

	// Handle orphans: children whose parent is not in orgIDs and not a root
	// Group them by parent and emit in order
	emitted := make(map[int64]bool, len(result))
	for _, id := range result {
		emitted[id] = true
	}

	var orphanParents []int64
	for parentID := range children {
		// Check if any child of this parent was not emitted
		if !emitted[children[parentID][0]] {
			orphanParents = append(orphanParents, parentID)
		}
	}
	slices.Sort(orphanParents)

	for _, parentID := range orphanParents {
		for _, child := range children[parentID] {
			if !emitted[child] {
				result = append(result, child)
				emitted[child] = true
			}
		}
	}

	return result
}

// determineOrgType determines the organization type from a list of types.
// Priority: micro > mini > ges (more specific wins).
func determineOrgType(types []string) string {
	for _, t := range types {
		if t == "micro" {
			return "micro"
		}
	}
	for _, t := range types {
		if t == "mini" {
			return "mini"
		}
	}
	for _, t := range types {
		if t == "ges" {
			return "ges"
		}
	}
	return ""
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -v -run TestSortOrgIDs ./internal/lib/service/excel/sc/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/lib/service/excel/sc/sort.go internal/lib/service/excel/sc/sort_test.go
git commit -m "Add sortOrgIDs depth-first sorting utility for SC report"
```

### Task 2: Add `GetOrganizationParentMap` repo method

**Files:**
- Modify: `internal/storage/repo/organization.go` (append method)

- [ ] **Step 1: Implement `GetOrganizationParentMap`**

Add to end of `internal/storage/repo/organization.go`:

```go
// GetOrganizationParentMap returns a map of org_id -> parent_org_id for all organizations.
func (r *Repo) GetOrganizationParentMap(ctx context.Context) (map[int64]*int64, error) {
	const op = "storage.repo.GetOrganizationParentMap"
	const query = `SELECT id, parent_organization_id FROM organizations`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	result := make(map[int64]*int64)
	for rows.Next() {
		var id int64
		var parentID *int64
		if err := rows.Scan(&id, &parentID); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		result[id] = parentID
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows: %w", op, err)
	}

	return result, nil
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/storage/repo/...`
Expected: success

- [ ] **Step 3: Commit**

```bash
git add internal/storage/repo/organization.go
git commit -m "Add GetOrganizationParentMap repo method for SC report sorting"
```

---

## Chunk 2: Rewrite generator

### Task 3: Simplify `SectionInfo` and `scanSections`

**Files:**
- Modify: `internal/lib/service/excel/sc/generator.go`

- [ ] **Step 1: Update `SectionInfo` struct**

Replace the `SectionInfo` struct (lines 18-22):

```go
// SectionInfo holds information about a section in the template
type SectionInfo struct {
	Tag         string // "discharges", "ges", "mini", "micro", "visits", "incidents"
	HeaderRow   int    // row number of section header (the tag row)
	TemplateRow int    // HeaderRow + 1 (the template data row)
}
```

- [ ] **Step 2: Rewrite `scanSections`**

Replace `scanSections` method (lines 282-325):

```go
// scanSections reads column P to identify sections and their template rows
func (g *Generator) scanSections(f *excelize.File, sheet string) (map[string]*SectionInfo, error) {
	sections := make(map[string]*SectionInfo)

	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows: %w", err)
	}

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

		switch cellValue {
		case "ges", "mini", "micro", "discharges", "visits", "incidents", "res":
			sections[cellValue] = &SectionInfo{
				Tag:         cellValue,
				HeaderRow:   rowNum,
				TemplateRow: rowNum + 1,
			}
		}
	}

	return sections, nil
}
```

- [ ] **Step 3: Remove `sortReverse` and `sortAsc`**

Delete the `sortReverse` (lines 765-773) and `sortAsc` (lines 776-784) functions. They will be replaced by `slices.Sort` / `slices.SortFunc` in the rewritten process methods.

- [ ] **Step 4: Remove `GroupedShutdowns` struct**

Delete the `GroupedShutdowns` struct (lines 25-29).

- [ ] **Step 5: Verify compilation (expect errors in process methods — that's OK)**

Run: `go build ./internal/lib/service/excel/sc/...`
Expected: compilation errors referencing old `OrgRows` field and `GroupedShutdowns` — these are fixed in the next steps.

- [ ] **Step 6: Commit (WIP)**

```bash
git add internal/lib/service/excel/sc/generator.go
git commit -m "WIP: simplify SectionInfo and scanSections for dynamic generation"
```

### Task 4: Rewrite `GenerateExcel` signature and orchestration

**Files:**
- Modify: `internal/lib/service/excel/sc/generator.go`

- [ ] **Step 1: Update `GenerateExcel` signature and body**

Replace the `GenerateExcel` method with the new signature. Key changes:
- `shutdowns *GroupedShutdowns` → `shutdowns []*shutdown.ResponseModel`
- Add `orgTypesMap map[int64][]string`
- Add `orgParentMap map[int64]*int64`
- Shutdown processing: filter by type using `determineOrgType` from `sort.go`, call `processShutdowns` 3 times with filtered slices

```go
// GenerateExcel creates an Excel file from the template with all SC data
func (g *Generator) GenerateExcel(
	dateStart, dateEnd time.Time,
	discharges []discharge.Model,
	shutdowns []*shutdown.ResponseModel,
	orgTypesMap map[int64][]string,
	orgParentMap map[int64]*int64,
	visits []*visit.ResponseModel,
	incidents []*incident.ResponseModel,
	loc *time.Location,
	authorShortName string,
) (*excelize.File, error) {
	f, err := excelize.OpenFile(g.templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open template: %w", err)
	}

	sheet := f.GetSheetName(0)

	if err := g.replacePlaceholders(f, sheet, dateStart, dateEnd, authorShortName); err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to replace date placeholders: %w", err)
	}

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

	// Process discharges
	if sec, ok := sections["discharges"]; ok {
		if err := g.processDischarges(f, sheet, sec, discharges, orgParentMap, loc, set); err != nil {
			f.Close()
			return nil, fmt.Errorf("failed to process discharges: %w", err)
		}
	}
	if writeErr != nil {
		f.Close()
		return nil, writeErr
	}

	// Group shutdowns by org type and process each section
	shutdownsByType := map[string][]*shutdown.ResponseModel{
		"ges":   {},
		"mini":  {},
		"micro": {},
	}
	for _, s := range shutdowns {
		orgType := determineOrgType(orgTypesMap[s.OrganizationID])
		if orgType != "" {
			shutdownsByType[orgType] = append(shutdownsByType[orgType], s)
		}
	}

	for _, sectionTag := range []string{"ges", "mini", "micro"} {
		sections, err = g.scanSections(f, sheet)
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("failed to re-scan sections: %w", err)
		}
		if sec, ok := sections[sectionTag]; ok {
			if err := g.processShutdowns(f, sheet, sec, shutdownsByType[sectionTag], orgParentMap, loc, set); err != nil {
				f.Close()
				return nil, fmt.Errorf("failed to process %s shutdowns: %w", sectionTag, err)
			}
		}
	}
	if writeErr != nil {
		f.Close()
		return nil, writeErr
	}

	// Process visits
	sections, err = g.scanSections(f, sheet)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to re-scan sections for visits: %w", err)
	}
	if sec, ok := sections["visits"]; ok {
		if err := g.processVisits(f, sheet, sec, visits, orgParentMap, loc, set); err != nil {
			f.Close()
			return nil, fmt.Errorf("failed to process visits: %w", err)
		}
	}
	if writeErr != nil {
		f.Close()
		return nil, writeErr
	}

	// Process incidents
	sections, err = g.scanSections(f, sheet)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to re-scan sections for incidents: %w", err)
	}
	if sec, ok := sections["incidents"]; ok {
		if err := g.processIncidents(f, sheet, sec, incidents, orgParentMap, loc, set); err != nil {
			f.Close()
			return nil, fmt.Errorf("failed to process incidents: %w", err)
		}
	}
	if writeErr != nil {
		f.Close()
		return nil, writeErr
	}

	// Clear column P
	if err := g.clearColumn(f, sheet, "P"); err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to clear column P: %w", err)
	}

	// Set print area
	lastRow := g.findLastDataRow(f, sheet)
	printArea := fmt.Sprintf("$A$1:$P$%d", lastRow)

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

	if err := f.UpdateLinkedValue(); err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to update linked values: %w", err)
	}

	return f, nil
}
```

- [ ] **Step 2: Verify compilation (still expect errors in old process methods)**

Run: `go build ./internal/lib/service/excel/sc/...`

- [ ] **Step 3: Commit (WIP)**

```bash
git add internal/lib/service/excel/sc/generator.go
git commit -m "WIP: update GenerateExcel signature for dynamic org generation"
```

### Task 5: Rewrite `processDischarges`

**Files:**
- Modify: `internal/lib/service/excel/sc/generator.go`

- [ ] **Step 1: Rewrite `processDischarges`**

Replace the existing `processDischarges` method. New logic: aggregate by org, sort org IDs depth-first, duplicate template row, fill data.

```go
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
	aggregated := g.aggregateDischargesByOrganization(data, loc)

	if len(aggregated) == 0 {
		// No data — delete template row
		if err := f.RemoveRow(sheet, section.TemplateRow); err != nil {
			return fmt.Errorf("failed to remove template row: %w", err)
		}
		return nil
	}

	// Collect and sort org IDs
	orgIDs := make([]int64, 0, len(aggregated))
	for orgID := range aggregated {
		orgIDs = append(orgIDs, orgID)
	}
	orgIDs = sortOrgIDs(orgIDs, orgParentMap)

	// Duplicate template row for additional orgs (N-1 times)
	for i := 1; i < len(orgIDs); i++ {
		if err := f.DuplicateRow(sheet, section.TemplateRow); err != nil {
			return fmt.Errorf("failed to duplicate row %d: %w", section.TemplateRow, err)
		}
	}

	// Fill data
	for i, orgID := range orgIDs {
		rowNum := section.TemplateRow + i
		row := aggregated[orgID]

		set(fmt.Sprintf("A%d", rowNum), i+1)
		set(fmt.Sprintf("B%d", rowNum), row.OrganizationName)
		set(fmt.Sprintf("C%d", rowNum), row.StartDate.Format("02.01.2006"))
		set(fmt.Sprintf("D%d", rowNum), row.StartTime)
		set(fmt.Sprintf("E%d", rowNum), row.TotalVolume/0.0864)

		if row.EndDate != nil {
			set(fmt.Sprintf("G%d", rowNum), row.EndDate.Format("02.01.2006"))
		}
		if row.EndTime != nil {
			set(fmt.Sprintf("H%d", rowNum), *row.EndTime)
		}

		set(fmt.Sprintf("I%d", rowNum), row.Duration)
		set(fmt.Sprintf("K%d", rowNum), row.TotalVolume)

		if row.Reason != nil {
			set(fmt.Sprintf("M%d", rowNum), *row.Reason)
		}
	}

	// Restore bottom border
	lastRow := section.TemplateRow + len(orgIDs) - 1
	if err := g.applyBottomBorder(f, sheet, lastRow, "A", "O"); err != nil {
		return fmt.Errorf("failed to apply bottom border: %w", err)
	}

	return nil
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/lib/service/excel/sc/...`

- [ ] **Step 3: Commit**

```bash
git add internal/lib/service/excel/sc/generator.go
git commit -m "Rewrite processDischarges for dynamic template-row generation"
```

### Task 6: Rewrite `processShutdowns`

**Files:**
- Modify: `internal/lib/service/excel/sc/generator.go`

- [ ] **Step 1: Rewrite `processShutdowns`**

Replace existing method. Now receives flat slice (already filtered by type), groups by org, sorts, duplicates rows.

```go
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
	if len(shutdowns) == 0 {
		// No data — delete template row
		if err := f.RemoveRow(sheet, section.TemplateRow); err != nil {
			return fmt.Errorf("failed to remove template row: %w", err)
		}
		return nil
	}

	// Group by org
	shutdownsByOrg := make(map[int64][]*shutdown.ResponseModel)
	for _, s := range shutdowns {
		shutdownsByOrg[s.OrganizationID] = append(shutdownsByOrg[s.OrganizationID], s)
	}

	// Sort org IDs
	orgIDs := make([]int64, 0, len(shutdownsByOrg))
	for orgID := range shutdownsByOrg {
		orgIDs = append(orgIDs, orgID)
	}
	orgIDs = sortOrgIDs(orgIDs, orgParentMap)

	// Calculate total rows needed
	totalRows := 0
	for _, list := range shutdownsByOrg {
		totalRows += len(list)
	}

	// Duplicate template row (totalRows - 1) times
	for i := 1; i < totalRows; i++ {
		if err := f.DuplicateRow(sheet, section.TemplateRow); err != nil {
			return fmt.Errorf("failed to duplicate row %d: %w", section.TemplateRow, err)
		}
	}

	// Fill data
	var allDataRows []int
	var totalGenerationLoss float64
	var totalIdleDischargeVolume float64
	currentRow := section.TemplateRow

	for _, orgID := range orgIDs {
		for _, s := range shutdownsByOrg[orgID] {
			allDataRows = append(allDataRows, currentRow)

			set(fmt.Sprintf("C%d", currentRow), s.StartedAt.In(loc).Format("02.01.2006 15:04"))
			if s.EndedAt != nil {
				set(fmt.Sprintf("D%d", currentRow), s.EndedAt.In(loc).Format("02.01.2006 15:04"))
			}
			if s.Reason != nil {
				set(fmt.Sprintf("E%d", currentRow), *s.Reason)
			}
			if s.GenerationLossMwh != nil {
				valueInThousands := *s.GenerationLossMwh / 1000
				set(fmt.Sprintf("N%d", currentRow), valueInThousands)
				totalGenerationLoss += valueInThousands
			}
			if s.IdleDischargeVolumeThousandM3 != nil {
				set(fmt.Sprintf("O%d", currentRow), *s.IdleDischargeVolumeThousandM3)
				totalIdleDischargeVolume += *s.IdleDischargeVolumeThousandM3
			}

			currentRow++
		}
	}

	// Numbering
	for i, rowNum := range allDataRows {
		set(fmt.Sprintf("A%d", rowNum), i+1)
	}

	// Bottom border
	if len(allDataRows) > 0 {
		lastDataRow := allDataRows[len(allDataRows)-1]
		if err := g.applyBottomBorder(f, sheet, lastDataRow, "A", "O"); err != nil {
			return fmt.Errorf("failed to apply bottom border: %w", err)
		}

		// Update "Жами" totals row
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
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/lib/service/excel/sc/...`

- [ ] **Step 3: Commit**

```bash
git add internal/lib/service/excel/sc/generator.go
git commit -m "Rewrite processShutdowns for dynamic template-row generation"
```

### Task 7: Update `processVisits` and `processIncidents` with sorting

**Files:**
- Modify: `internal/lib/service/excel/sc/generator.go`

- [ ] **Step 1: Update `processVisits` — add sorting by parent_id → org_id**

Replace existing method. Add `orgParentMap` parameter, sort visits before filling.

```go
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
	if len(visits) == 0 {
		// No data — delete template row entirely
		if err := f.RemoveRow(sheet, section.TemplateRow); err != nil {
			return fmt.Errorf("failed to remove template row: %w", err)
		}
		return nil
	}

	// Sort visits by parent_id → org_id
	sortVisitsByOrg(visits, orgParentMap)

	templateRow := section.TemplateRow

	// Duplicate for additional visits
	for i := 1; i < len(visits); i++ {
		if err := f.DuplicateRow(sheet, templateRow); err != nil {
			return fmt.Errorf("failed to duplicate row %d: %w", templateRow, err)
		}
	}

	for i, v := range visits {
		row := templateRow + i
		set(fmt.Sprintf("A%d", row), i+1)
		set(fmt.Sprintf("B%d", row), v.OrganizationName)
		set(fmt.Sprintf("F%d", row), v.Description)
		set(fmt.Sprintf("M%d", row), v.ResponsibleName)
	}

	lastRow := templateRow + len(visits) - 1
	if err := g.applyBottomBorder(f, sheet, lastRow, "A", "O"); err != nil {
		return fmt.Errorf("failed to apply bottom border: %w", err)
	}

	return nil
}
```

- [ ] **Step 2: Add sorting helper functions to `sort.go`**

Append to `internal/lib/service/excel/sc/sort.go`:

```go
// sortVisitsByOrg sorts visits by parent_id → org_id of their organization.
func sortVisitsByOrg(visits []*visit.ResponseModel, parentMap map[int64]*int64) {
	orgIDs := make([]int64, 0)
	seen := make(map[int64]bool)
	for _, v := range visits {
		if !seen[v.OrganizationID] {
			orgIDs = append(orgIDs, v.OrganizationID)
			seen[v.OrganizationID] = true
		}
	}
	sorted := sortOrgIDs(orgIDs, parentMap)
	orderMap := make(map[int64]int, len(sorted))
	for i, id := range sorted {
		orderMap[id] = i
	}
	slices.SortStableFunc(visits, func(a, b *visit.ResponseModel) int {
		return orderMap[a.OrganizationID] - orderMap[b.OrganizationID]
	})
}

// sortIncidentsByOrg sorts incidents: NULL org_id first, then by parent_id → org_id.
func sortIncidentsByOrg(incidents []*incident.ResponseModel, parentMap map[int64]*int64) {
	orgIDs := make([]int64, 0)
	seen := make(map[int64]bool)
	for _, inc := range incidents {
		if inc.OrganizationID != nil && !seen[*inc.OrganizationID] {
			orgIDs = append(orgIDs, *inc.OrganizationID)
			seen[*inc.OrganizationID] = true
		}
	}
	sorted := sortOrgIDs(orgIDs, parentMap)
	orderMap := make(map[int64]int, len(sorted))
	for i, id := range sorted {
		orderMap[id] = i + 1 // +1 so NULL (0) comes first
	}
	slices.SortStableFunc(incidents, func(a, b *incident.ResponseModel) int {
		aOrder := 0
		bOrder := 0
		if a.OrganizationID != nil {
			aOrder = orderMap[*a.OrganizationID]
		}
		if b.OrganizationID != nil {
			bOrder = orderMap[*b.OrganizationID]
		}
		return aOrder - bOrder
	})
}
```

Note: `sort.go` already has the required imports (`incident`, `visit`, `slices`) from Task 1.

- [ ] **Step 3: Update `processIncidents` — add sorting, NULL org_id first**

Replace existing method:

```go
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
	if len(incidents) == 0 {
		// No data — delete template row entirely
		if err := f.RemoveRow(sheet, section.TemplateRow); err != nil {
			return fmt.Errorf("failed to remove template row: %w", err)
		}
		return nil
	}

	// Sort: NULL org_id first, then by parent_id → org_id
	sortIncidentsByOrg(incidents, orgParentMap)

	templateRow := section.TemplateRow

	for i := 1; i < len(incidents); i++ {
		if err := f.DuplicateRow(sheet, templateRow); err != nil {
			return fmt.Errorf("failed to duplicate row %d: %w", templateRow, err)
		}
	}

	for i, inc := range incidents {
		row := templateRow + i
		set(fmt.Sprintf("A%d", row), i+1)
		set(fmt.Sprintf("B%d", row), inc.IncidentTime.In(loc).Format("02.01.2006 15:04"))

		orgName := "Энергия хосил қилувчи корхона ва сув омборлар"
		if inc.OrganizationName != nil && *inc.OrganizationName != "" {
			orgName = *inc.OrganizationName
		}
		set(fmt.Sprintf("C%d", row), orgName)
		set(fmt.Sprintf("F%d", row), inc.Description)
	}

	lastRow := templateRow + len(incidents) - 1
	if err := g.applyBottomBorder(f, sheet, lastRow, "A", "O"); err != nil {
		return fmt.Errorf("failed to apply bottom border: %w", err)
	}

	return nil
}
```

- [ ] **Step 4: Verify compilation**

Run: `go build ./internal/lib/service/excel/sc/...`
Expected: success

- [ ] **Step 5: Commit**

```bash
git add internal/lib/service/excel/sc/generator.go internal/lib/service/excel/sc/sort.go
git commit -m "Update visits and incidents with org sorting, NULL org_id first for incidents"
```

---

## Chunk 3: Handler + Router + Template

### Task 8: Update handler (export.go)

**Files:**
- Modify: `internal/http-server/handlers/sc/export/export.go`

- [ ] **Step 1: Add `OrgParentMapper` interface and update `New` signature**

Add interface after existing interfaces (~line 51):

```go
// OrgParentMapper defines the interface for fetching organization parent map
type OrgParentMapper interface {
	GetOrganizationParentMap(ctx context.Context) (map[int64]*int64, error)
}
```

Update `New` function signature — add `orgParentMapper OrgParentMapper` parameter after `incidentGetter`:

```go
func New(
	log *slog.Logger,
	dischargeGetter DischargeGetter,
	shutdownGetter ShutdownGetter,
	orgTypesGetter OrgTypesGetter,
	visitGetter VisitGetter,
	incidentGetter IncidentGetter,
	orgParentMapper OrgParentMapper,
	generator *scgen.Generator,
	loc *time.Location,
) http.HandlerFunc {
```

- [ ] **Step 2: Add org parent map fetch and update generator call**

Inside the handler function, after fetching incidents (~line 161), add:

```go
		// Fetch organization parent map for sorting
		orgParentMap, err := orgParentMapper.GetOrganizationParentMap(r.Context())
		if err != nil {
			log.Error("failed to fetch organization parent map", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to fetch organization parent map"))
			return
		}
```

Update the `generator.GenerateExcel` call — replace `groupedShutdowns` with flat shutdowns + orgTypesMap + orgParentMap:

```go
		excelFile, err := generator.GenerateExcel(
			startDate, endDate,
			discharges,
			shutdowns,      // flat list (was groupedShutdowns)
			orgTypesMap,    // NEW
			orgParentMap,   // NEW
			visits,
			incidents,
			loc,
			authorShort,
		)
```

- [ ] **Step 3: Remove `groupShutdownsByType` and `determineOrgType` functions**

Delete `groupShutdownsByType` (lines 302-328) and `determineOrgType` (lines 331-349) from `export.go`. Also remove the `groupedShutdowns := groupShutdownsByType(shutdowns, orgTypesMap)` call from the handler.

Remove the unused import of `scgen "srmt-admin/internal/lib/service/excel/sc"` if `scgen.GroupedShutdowns` was the only usage (check — `*scgen.Generator` is still used, so keep the import).

- [ ] **Step 4: Verify compilation**

Run: `go build ./internal/http-server/handlers/sc/export/...`
Expected: success

- [ ] **Step 5: Commit**

```bash
git add internal/http-server/handlers/sc/export/export.go
git commit -m "Update SC export handler for dynamic generation with flat shutdowns and orgParentMap"
```

### Task 9: Update router wiring

**Files:**
- Modify: `internal/http-server/router/router.go` (lines 466-475)

- [ ] **Step 1: Add `deps.PgRepo` for `OrgParentMapper`**

Update the SC export route registration (line 466-475):

```go
			r.Get("/sc/export", scExport.New(
				deps.Log,
				deps.PgRepo, // DischargeGetter
				deps.PgRepo, // ShutdownGetter
				deps.PgRepo, // OrgTypesGetter
				deps.PgRepo, // VisitGetter
				deps.PgRepo, // IncidentGetter
				deps.PgRepo, // OrgParentMapper (NEW)
				scExcelGen.New(deps.SCExcelTemplatePath),
				loc,
			))
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./cmd/...`
Expected: success (may need `make wire` first if Wire sees changes)

- [ ] **Step 3: Run `make wire` if needed**

Run: `make wire`
Then: `go build ./cmd/...`

- [ ] **Step 4: Commit**

```bash
git add internal/http-server/router/router.go
git commit -m "Wire OrgParentMapper dependency for SC export route"
```

### Task 10: Update Excel template

**Files:**
- Modify: `template/sc.xlsx`

- [ ] **Step 1: Simplify the template**

Open `template/sc.xlsx` and for each of these sections (discharges, ges, mini, micro):
1. Find the tag row in column P (e.g., row with "discharges" in column P)
2. Keep the row immediately after the tag (this becomes the template row)
3. Delete all other rows with numeric org IDs in column P for that section
4. Clear column P in the template row (leave it empty — only the tag row has a P value)
5. Preserve all formatting, merged cells, and borders on the template row

Visits and incidents sections already have the correct structure (tag + template row). No changes needed.

**Note:** This is a manual edit to the binary .xlsx file. Cannot be done programmatically in this plan. The developer must open the file in Excel/LibreOffice and make these changes.

- [ ] **Step 2: Verify the template has correct structure**

After editing, column P should contain exactly these tags (one per section, no numeric org IDs):
- `discharges` (followed by 1 empty template row)
- `ges` (followed by 1 empty template row)
- `mini` (followed by 1 empty template row)
- `micro` (followed by 1 empty template row)
- `visits` (followed by 1 empty template row)
- `incidents` (followed by 1 empty template row)
- `res` (if present, keep as-is)

- [ ] **Step 3: Commit**

```bash
git add template/sc.xlsx
git commit -m "Simplify SC template: replace hardcoded org rows with single template rows"
```

---

## Chunk 4: Cleanup + Tests

### Task 11: Clean up dead code

**Files:**
- Modify: `internal/lib/service/excel/sc/generator.go`

- [ ] **Step 1: Remove any remaining dead code**

Check and remove:
- Old `sortReverse()` and `sortAsc()` if still present
- Any unused imports
- Any references to `OrgRows`

Run: `go vet ./internal/lib/service/excel/sc/...`

- [ ] **Step 2: Run full build**

Run: `go build ./...`
Expected: success with no errors

- [ ] **Step 3: Run existing tests**

Run: `go test -v ./...`
Expected: all existing tests pass

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "Clean up dead code from SC report refactoring"
```

### Task 12: Add sorting tests for visits and incidents

**Files:**
- Modify: `internal/lib/service/excel/sc/sort_test.go`

- [ ] **Step 1: Add tests for `sortVisitsByOrg` and `sortIncidentsByOrg`**

Append to `sort_test.go` (no import changes needed — add `incident` and `visit` imports to the existing import block):

Replace the import block at the top of `sort_test.go` with:

```go
import (
	"slices"
	"testing"

	"srmt-admin/internal/lib/model/incident"
	"srmt-admin/internal/lib/model/visit"
)
```

Then append these test functions:

```go
func TestSortVisitsByOrg(t *testing.T) {
	parentMap := map[int64]*int64{
		3:  nil,
		5:  nil,
		10: ptr(int64(3)),
	}

	visits := []*visit.ResponseModel{
		{ID: 1, OrganizationID: 5, OrganizationName: "Org5"},
		{ID: 2, OrganizationID: 10, OrganizationName: "Org10"},
		{ID: 3, OrganizationID: 3, OrganizationName: "Org3"},
	}

	sortVisitsByOrg(visits, parentMap)

	// Expected order: cascade 3, then child 10, then 5
	expectedOrgOrder := []int64{3, 10, 5}
	for i, v := range visits {
		if v.OrganizationID != expectedOrgOrder[i] {
			t.Errorf("position %d: got org_id=%d, want %d", i, v.OrganizationID, expectedOrgOrder[i])
		}
	}
}

func TestSortIncidentsByOrg_NullFirst(t *testing.T) {
	parentMap := map[int64]*int64{
		5:  nil,
		10: ptr(int64(3)),
		3:  nil,
	}

	orgID5 := int64(5)
	orgID10 := int64(10)

	incidents := []*incident.ResponseModel{
		{ID: 1, OrganizationID: &orgID5},
		{ID: 2, OrganizationID: nil},         // NULL — should be first
		{ID: 3, OrganizationID: &orgID10},
	}

	sortIncidentsByOrg(incidents, parentMap)

	if incidents[0].ID != 2 {
		t.Errorf("expected NULL org incident first, got ID=%d", incidents[0].ID)
	}
	// After NULL: org 3's child (10), then org 5
	if incidents[1].OrganizationID == nil || *incidents[1].OrganizationID != 10 {
		t.Errorf("expected org 10 second, got %v", incidents[1].OrganizationID)
	}
	if incidents[2].OrganizationID == nil || *incidents[2].OrganizationID != 5 {
		t.Errorf("expected org 5 third, got %v", incidents[2].OrganizationID)
	}
}
```

- [ ] **Step 2: Run tests**

Run: `go test -v -run TestSort ./internal/lib/service/excel/sc/...`
Expected: all pass

- [ ] **Step 3: Commit**

```bash
git add internal/lib/service/excel/sc/sort_test.go
git commit -m "Add sorting tests for incidents with NULL org_id first"
```

### Task 13: Integration smoke test

- [ ] **Step 1: Run the application locally**

```bash
export CONFIG_PATH=config/local.yaml
make dev
```

- [ ] **Step 2: Test SC export endpoint**

```bash
curl -o test-sc.xlsx "http://localhost:PORT/sc/export?date=2026-03-16&format=excel"
```

Open `test-sc.xlsx` and verify:
- Organizations appear dynamically (not hardcoded list)
- Sorted by parent cascade → child org
- Shutdowns grouped correctly in ges/mini/micro sections
- Visits and incidents sorted by org
- Incidents with NULL org appear first
- Bottom borders present on last data rows
- "Жами" totals correct
- No blank rows in empty sections

- [ ] **Step 3: Test PDF export**

```bash
curl -o test-sc.pdf "http://localhost:PORT/sc/export?date=2026-03-16&format=pdf"
```

Verify PDF renders correctly.

- [ ] **Step 4: Final commit (squash WIP commits if desired)**

```bash
git log --oneline -10  # review commits
```
