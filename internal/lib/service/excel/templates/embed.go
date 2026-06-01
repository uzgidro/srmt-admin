// Package templates provides embedded Excel templates used by the report
// generators. Templates are baked into the binary at compile time, so a
// production deployment needs nothing on disk to render reports.
//
// An optional override directory (set via Config.TemplateOverridePath) lets
// developers edit .xlsx files on disk and see changes without rebuilding.
// When the override directory is non-empty and the file exists there, it
// wins; otherwise the embedded copy is used. A missing file in the override
// directory silently falls back to embed — removing an override during dev
// must not break rendering.
package templates

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xuri/excelize/v2"
)

//go:embed *.xlsx
var FS embed.FS

// Known template filenames. Exported so callers don't pass raw strings.
const (
	Sel            = "sel.xlsx"
	GESProd        = "ges-prod.xlsx"
	Discharge      = "discharge.xlsx"
	ResSummary     = "res-summary.xlsx"
	ResSummaryFilt = "res-summary-filter.xlsx"
	ResSummaryHour = "res-summary-hourly.xlsx"
	SC             = "sc.xlsx"
	OwnNeeds       = "own-needs.xlsx"
)

// AllNames returns every embedded template filename. Used at startup to
// sanity-check an optional override directory (cmd/main.go warns if a
// non-empty override path contains none of these files).
func AllNames() []string {
	return []string{
		Sel, GESProd, Discharge, ResSummary,
		ResSummaryFilt, ResSummaryHour, SC, OwnNeeds,
	}
}

// CountOverrideFiles reports how many embedded templates exist on disk
// inside overrideDir. cmd/main.go uses this at startup to warn when the
// override path is set but contains zero templates — a likely typo.
//
// An empty overrideDir is the "embed-only" mode and returns 0 without
// touching the filesystem.
func CountOverrideFiles(overrideDir string) int {
	if overrideDir == "" {
		return 0
	}
	count := 0
	for _, name := range AllNames() {
		if _, err := os.Stat(filepath.Join(overrideDir, name)); err == nil {
			count++
		}
	}
	return count
}

// Open returns the requested template, preferring overrideDir over the
// embedded copy. Caller owns the returned *excelize.File and must Close it.
//
// Resolution:
//  1. overrideDir != "" AND overrideDir/name exists → excelize.OpenFile.
//  2. Otherwise → excelize.OpenReader on the embedded copy.
//
// A missing file in overrideDir is NOT an error — falls through to embed.
// This keeps the dev workflow frictionless (delete the override file → it
// just works again from embed).
func Open(name, overrideDir string) (*excelize.File, error) {
	if overrideDir != "" {
		p := filepath.Join(overrideDir, name)
		if _, err := os.Stat(p); err == nil {
			f, err := excelize.OpenFile(p)
			if err != nil {
				return nil, fmt.Errorf("open override %s: %w", p, err)
			}
			return f, nil
		}
	}
	data, err := FS.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("read embedded %s: %w", name, err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("open embedded %s: %w", name, err)
	}
	return f, nil
}
