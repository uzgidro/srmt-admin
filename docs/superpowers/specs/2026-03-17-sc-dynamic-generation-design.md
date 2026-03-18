# SC Report Dynamic Generation Design

## Problem

The SC report Excel template (`template/sc.xlsx`) has hardcoded organization IDs in column P for the `discharges`, `ges`, `mini`, and `micro` sections. When new organizations are added to the system, the template must be manually updated. This is fragile and doesn't scale.

## Solution

Replace hardcoded org rows with a single template row per section. Organizations are fetched from the database at generation time and sorted by `parent_organization_id` then `organization_id`. This makes the report fully dynamic — new organizations appear automatically.

## Scope

All six sections become dynamic with template-row duplication:

| Section | Current State | Target State |
|---------|--------------|--------------|
| discharges | Hardcoded org IDs in column P | Tag + 1 template row, dynamic generation |
| ges | Hardcoded org IDs in column P | Tag + 1 template row, dynamic generation |
| mini | Hardcoded org IDs in column P | Tag + 1 template row, dynamic generation |
| micro | Hardcoded org IDs in column P | Tag + 1 template row, dynamic generation |
| visits | Already dynamic (template row duplication) | Add sorting by parent_id → org_id |
| incidents | Already dynamic (template row duplication) | Add sorting by parent_id → org_id; NULL org_id first |

## Design

### 1. Template Changes (sc.xlsx)

**Before (discharges example):**
```
Row 10: [tag: "discharges"]
Row 11: [org_id: "5"]   — formatted row for org 5
Row 12: [org_id: "12"]  — formatted row for org 12
Row 13: [org_id: "23"]  — formatted row for org 23
```

**After:**
```
Row 10: [tag: "discharges"]
Row 11: [empty P col]   — single template row with formatting/merges/borders
```

Same pattern for ges, mini, micro sections — each becomes tag + 1 template row.

Visits and incidents already have this structure — no template changes needed for them.

### 2. Data Layer Changes

#### New handler dependency: organization data with parent_id

The handler needs a map of `org_id → parent_org_id` for sorting. Use existing `GetFlatOrganizations` (returns `[]*organization.Model` with `ParentOrganizationID` field) or, more efficiently, add a lightweight method:

```go
// In repo/organization.go
func (r *Repo) GetOrganizationParentMap(ctx context.Context) (map[int64]*int64, error)
```

Returns `map[orgID]*parentOrgID`. Simple query: `SELECT id, parent_organization_id FROM organizations`.

#### Remove GroupedShutdowns

Currently the handler groups shutdowns into `Ges/Mini/Micro` slices and passes `*scgen.GroupedShutdowns` to the generator. This struct becomes unnecessary — shutdowns are passed as a flat `[]*shutdown.ResponseModel` slice, and the generator groups them internally using `orgTypesMap`.

### 3. Generator Changes

#### 3.1 SectionInfo simplification

```go
// Before
type SectionInfo struct {
    Tag       string
    HeaderRow int
    OrgRows   map[int64]int  // REMOVE
}

// After
type SectionInfo struct {
    Tag         string
    HeaderRow   int
    TemplateRow int  // HeaderRow + 1
}
```

#### 3.2 scanSections simplification

Only parses section tags from column P. No longer parses org IDs. Sets `TemplateRow = HeaderRow + 1` for each section.

#### 3.3 Sorting function

```go
// sortOrgIDs sorts organization IDs by parent_id first, then by org_id.
// Organizations with the same parent are grouped together.
func sortOrgIDs(orgIDs []int64, parentMap map[int64]*int64) []int64
```

**Algorithm:** Depth-first sort that keeps children immediately after their parent cascade:

1. Build a tree: group org IDs by their `parent_id`
2. For root orgs (`parent_id = NULL`), sort by `org_id`
3. Walk the tree depth-first: emit parent, then its children sorted by `org_id`
4. Orgs without a parent (root-level) are sorted by `org_id` among themselves

This ensures cascade 3 is immediately followed by its children (10, 11, 12) before moving to cascade 5 and its children.

For flat lists (orgs without hierarchy), this degenerates to simple `org_id` sort.

#### 3.4 GenerateExcel signature change

```go
// Before
func (g *Generator) GenerateExcel(
    dateStart, dateEnd time.Time,
    discharges []discharge.Model,
    shutdowns *GroupedShutdowns,        // grouped struct
    visits []*visit.ResponseModel,
    incidents []*incident.ResponseModel,
    loc *time.Location,
    authorShortName string,
) (*excelize.File, error)

// After
func (g *Generator) GenerateExcel(
    dateStart, dateEnd time.Time,
    discharges []discharge.Model,
    shutdowns []*shutdown.ResponseModel, // flat list
    orgTypesMap map[int64][]string,      // for ges/mini/micro grouping
    orgParentMap map[int64]*int64,       // for sorting
    visits []*visit.ResponseModel,
    incidents []*incident.ResponseModel,
    loc *time.Location,
    authorShortName string,
) (*excelize.File, error)
```

#### 3.5 processDischarges (rewrite)

1. Aggregate discharges by org_id (existing logic, unchanged)
2. Collect org IDs that have data
3. Sort by parent_id → org_id using `orgParentMap`
4. Duplicate template row `N-1` times (N = number of orgs with data)
5. Fill each row with aggregated data
6. Write org name in column B (merged B-C in current template for discharges section)
7. Number rows (column A)
8. Apply bottom border to last row
9. If no data — delete the template row entirely (do not leave blank rows)

#### 3.6 processShutdowns (rewrite)

Currently called 3 times (ges/mini/micro). Still called per section type but now:

1. Filter shutdowns for this org type using `orgTypesMap`
2. Group filtered shutdowns by org_id
3. Collect unique org IDs
4. Sort by parent_id → org_id
5. For each org: duplicate template row for N shutdowns
6. Fill data per shutdown
7. Recalculate numbering, totals ("Жами" row), borders

#### 3.7 processVisits (enhance)

Add sorting: before filling data, sort visits by parent_id of their org → org_id.

#### 3.8 processIncidents (enhance)

Add sorting: incidents with `org_id = NULL` first, then sorted by parent_id → org_id.

#### 3.9 Row offset tracking optimization

Currently `scanSections` is called 6+ times. After refactoring, since all sections use template-row duplication, we can track cumulative row offset instead of rescanning. However, rescanning is simpler and the performance cost is negligible (< 100 rows). **Keep rescanning for simplicity.**

### 4. Handler Changes (export.go)

```go
// New interface
type OrgParentMapper interface {
    GetOrganizationParentMap(ctx context.Context) (map[int64]*int64, error)
}
```

Handler flow:
1. Fetch all data (discharges, shutdowns, orgTypesMap, visits, incidents) — same as before
2. **New:** Fetch org parent map
3. Remove `groupShutdownsByType` call
4. Pass flat shutdowns + orgTypesMap + orgParentMap to generator

### 5. Code Removals

| What | Where | Reason |
|------|-------|--------|
| `GroupedShutdowns` struct | `generator.go` | Replaced by flat list + orgTypesMap |
| `groupShutdownsByType()` | `export.go` | Grouping moved to generator |
| `determineOrgType()` | `export.go` | Moved to `generator.go` as a private function `determineOrgType()` with same priority logic: micro > mini > ges |
| `SectionInfo.OrgRows` field | `generator.go` | No longer needed |
| `sortReverse()`, `sortAsc()` | `generator.go` | Replace with `slices.Sort`/`slices.SortFunc` |
| Hardcoded org IDs in template | `sc.xlsx` column P | Replaced by template rows |

### 6. Wire/DI Changes

`OrgParentMapper` interface needs to be wired. The `*Repo` already satisfies it since we add the method to `*Repo`. In `router.go`, the `scExport.New(...)` call gains a 9th positional argument — `deps.PgRepo` for `OrgParentMapper`, inserted after `incidentGetter`:

```go
r.Get("/sc/export", scExport.New(
    deps.Log,
    deps.PgRepo,           // DischargeGetter
    deps.PgRepo,           // ShutdownGetter
    deps.PgRepo,           // OrgTypesGetter
    deps.PgRepo,           // VisitGetter
    deps.PgRepo,           // IncidentGetter
    deps.PgRepo,           // OrgParentMapper (NEW)
    scExcelGen.New(deps.SCExcelTemplatePath),
    loc,
))
```

### 7. Testing Strategy

- Unit test `sortOrgIDs` with various parent_id configurations
- Unit test incidents sorting with NULL org_id positioning
- Integration test: generate Excel with known data, verify row contents and ordering
- Manual verification: compare old vs new output for the same date

### 8. Migration

No database migration needed — we're using existing columns (`parent_organization_id`).

### 9. Template Migration

The sc.xlsx template must be manually simplified:
- Remove all hardcoded org ID rows from discharges, ges, mini, micro sections
- Keep only tag row + one template row per section
- Preserve formatting/merges/borders on the template row

This is a one-time manual change to the Excel file.

### 10. Edge Cases

| Case | Behavior |
|------|----------|
| No data for a section (0 discharges, 0 shutdowns, etc.) | Delete template row entirely — no blank rows in output |
| Org in data but missing from `orgParentMap` | Treat as root org (`parent_id = NULL`), sort by `org_id` among roots |
| Org in shutdowns but missing from `orgTypesMap` | Silently skip (same as current behavior) — org has no type assignment |
| Multiple shutdowns for same org | Duplicate rows per shutdown (existing behavior, preserved) |
| Incidents with `org_id = NULL` | Placed first in the incidents section, before sorted org-specific incidents |
| Template row column P | Empty — the tag is only on the header row (HeaderRow), template row (HeaderRow+1) has no P value |
