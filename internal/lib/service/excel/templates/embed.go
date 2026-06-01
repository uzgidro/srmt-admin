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
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xuri/excelize/v2"
)

// maxTemplateSize caps the size of an override .xlsx that Open will read.
// Real templates in this project are well under 200 KB; 16 MB is generous
// headroom and a hard ceiling against zip-bomb-style override files that a
// compromised override directory might contain. The embedded path is not
// gated by this — those bytes are baked into the binary at compile time.
const maxTemplateSize = 16 * 1024 * 1024

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
//  1. overrideDir != "" AND overrideDir/name opens cleanly → use it.
//  2. Otherwise → excelize.OpenReader on the embedded copy.
//
// A missing override file is NOT an error — falls through to embed. This
// keeps the dev workflow frictionless (delete the override → it just works
// again from embed). Any OTHER error from opening or parsing the override
// (corrupt file, permission denied, oversize) IS surfaced to the caller —
// silently using embed in that case would hide a real misconfiguration.
//
// Security: override reads go through os.Open (not OpenFile-by-path) to
// avoid a Stat-then-Open TOCTOU window, and through excelize.Options with
// UnzipSizeLimit set to maxTemplateSize so an oversized or zip-bomb-style
// override cannot exhaust memory.
func Open(name, overrideDir string) (*excelize.File, error) {
	if overrideDir != "" {
		f, err := openOverride(filepath.Join(overrideDir, name))
		if err == nil {
			return f, nil
		}
		// Missing file → fall through to embed. Anything else surfaces.
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
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

// openOverride opens an override .xlsx by path. It returns os.ErrNotExist
// (wrapped) when the file is missing so Open() can fall back to embed.
// All other errors (parse error, oversize, permission) are reported as-is.
func openOverride(p string) (*excelize.File, error) {
	fh, err := os.Open(p)
	if err != nil {
		// os.Open already returns *PathError wrapping os.ErrNotExist; pass through.
		return nil, err
	}
	defer fh.Close()

	// Cap file size before excelize touches it — zip-bomb defence.
	info, err := fh.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat override %s: %w", p, err)
	}
	if info.Size() > maxTemplateSize {
		return nil, fmt.Errorf("override %s exceeds %d bytes (got %d)",
			p, maxTemplateSize, info.Size())
	}

	f, err := excelize.OpenReader(fh, excelize.Options{UnzipSizeLimit: maxTemplateSize})
	if err != nil {
		return nil, fmt.Errorf("open override %s: %w", p, err)
	}
	return f, nil
}
