package templates

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

// Every template advertised by AllNames must be present in the embedded FS
// and look like a real xlsx (more than 1 KB — a Zip header alone is ~30 B).
// This catches a missing file, a typo in a constant, and an accidentally
// truncated copy.
func TestEmbeddedAllPresent(t *testing.T) {
	names := AllNames()
	if len(names) != 8 {
		t.Fatalf("AllNames len: want 8, got %d", len(names))
	}
	for _, name := range names {
		data, err := FS.ReadFile(name)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if len(data) < 1024 {
			t.Errorf("%s: suspiciously small (%d bytes)", name, len(data))
		}
		// Don't trust size alone — a 1 KB blob of garbage with a Zip header
		// would pass that check. Round-trip through excelize and make sure
		// the workbook actually parses and has at least one sheet. This is
		// the fail-fast guarantee that every embedded template is renderable.
		f, err := Open(name, "")
		if err != nil {
			t.Errorf("%s: Open failed: %v", name, err)
			continue
		}
		if sheets := f.GetSheetList(); len(sheets) == 0 {
			t.Errorf("%s: parsed file has no sheets", name)
		}
		_ = f.Close()
	}
}

// Open with overrideDir="" must return the embedded copy and yield a valid
// excelize file (i.e. the bytes round-trip through excelize.OpenReader).
func TestOpen_EmbedOnly(t *testing.T) {
	f, err := Open(Sel, "")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()
	if got := f.GetSheetList(); len(got) == 0 {
		t.Errorf("opened file has no sheets")
	}
}

// When the override directory contains the requested file, Open must read
// from disk — proved by writing a distinctive cell value to a temp xlsx and
// reading it back after Open.
func TestOpen_OverridePresent(t *testing.T) {
	dir := t.TempDir()

	// Build a minimal xlsx on disk with a unique marker.
	src := excelize.NewFile()
	const marker = "OVERRIDE_MARKER_42"
	if err := src.SetCellValue(src.GetSheetName(0), "A1", marker); err != nil {
		t.Fatalf("SetCellValue: %v", err)
	}
	overridePath := filepath.Join(dir, Sel)
	if err := src.SaveAs(overridePath); err != nil {
		t.Fatalf("SaveAs: %v", err)
	}
	src.Close()

	got, err := Open(Sel, dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer got.Close()

	v, err := got.GetCellValue(got.GetSheetName(0), "A1")
	if err != nil {
		t.Fatalf("GetCellValue: %v", err)
	}
	if v != marker {
		t.Errorf("A1: want %q (override), got %q (likely embed)", marker, v)
	}
}

// CountOverrideFiles reports how many embedded templates exist on disk in
// overrideDir. Used by cmd/main.go to warn when an override directory is set
// but contains no templates (typo / wrong path / forgotten edit).
//
// "" overrideDir → 0 (no warning needed, embed-only mode is normal).
// Non-empty + every file missing → 0 (warn at startup).
// Non-empty + ≥1 file present → that count (silent; partial overrides are
// supported by Open via per-file fallback).
func TestCountOverrideFiles(t *testing.T) {
	t.Run("empty overrideDir returns 0", func(t *testing.T) {
		if got := CountOverrideFiles(""); got != 0 {
			t.Errorf("want 0, got %d", got)
		}
	})

	t.Run("non-empty but no files returns 0", func(t *testing.T) {
		dir := t.TempDir()
		if got := CountOverrideFiles(dir); got != 0 {
			t.Errorf("want 0, got %d", got)
		}
	})

	t.Run("two templates present returns 2", func(t *testing.T) {
		dir := t.TempDir()
		for _, name := range []string{Sel, GESProd} {
			f := excelize.NewFile()
			if err := f.SaveAs(filepath.Join(dir, name)); err != nil {
				t.Fatalf("SaveAs %s: %v", name, err)
			}
			f.Close()
		}
		if got := CountOverrideFiles(dir); got != 2 {
			t.Errorf("want 2, got %d", got)
		}
	})

	t.Run("foreign files do not count", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "stranger.xlsx"), []byte("zip"), 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		if got := CountOverrideFiles(dir); got != 0 {
			t.Errorf("want 0 (stranger.xlsx is not an embedded name), got %d", got)
		}
	})
}

// If overrideDir is non-empty but the file is missing there, Open silently
// falls back to the embedded copy. This is the dev convenience the plan
// explicitly chose — deleting an override file must not break rendering.
func TestOpen_OverrideMissing_FallbackToEmbed(t *testing.T) {
	dir := t.TempDir()
	// Sanity-check the override path really is empty.
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatalf("t.TempDir not empty: %d entries", len(entries))
	}

	f, err := Open(Sel, dir)
	if err != nil {
		t.Fatalf("Open: %v (should fall back to embed)", err)
	}
	defer f.Close()
	if got := f.GetSheetList(); len(got) == 0 {
		t.Errorf("fallback embed file has no sheets")
	}
}
