package reservoirsummary

import (
	"context"
	"errors"
	"log/slog"

	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	reservoirsummary "srmt-admin/internal/lib/model/reservoir-summary"
	"srmt-admin/internal/storage"
)

// volumeByLevelByOrg is the slice of the repo this package needs to recompute
// reservoir volume from the configured level→volume curve.
type volumeByLevelByOrg interface {
	GetVolumeByLevelByOrg(ctx context.Context, orgID int64, level float64) (float64, error)
}

// ConfigLookup gives applyStaticFallbacks per-organization access to the
// reservoir_summary_config row. Returning ok=false means "no config row";
// callers treat that as defaults (modsnow_enabled=true, volume_source=static).
// Introduced as a no-op param ahead of the modsnow_enabled / volume_source
// split: this seam is what lets those two features land in parallel without
// touching every call-site again.
type ConfigLookup interface {
	Get(orgID int64) (reservoirsummary.ReservoirSummaryConfig, bool)
}

// MapConfigLookup is the obvious in-memory ConfigLookup keyed by org id.
type MapConfigLookup map[int64]reservoirsummary.ReservoirSummaryConfig

func (m MapConfigLookup) Get(orgID int64) (reservoirsummary.ReservoirSummaryConfig, bool) {
	v, ok := m[orgID]
	return v, ok
}

// computeVolumeFromLevel asks the repo for the interpolated volume at the
// given level for the organization. Returns (value, true) on success. Returns
// (0, false) if there is no curve for this org, the level falls outside it,
// or any other error occurs — all three cases are treated as "fall back to
// whatever the caller had before". Out-of-range and unexpected errors are
// logged at warn/error; not-configured is silent because it is the expected
// state for any reservoir we have not calibrated yet.
func computeVolumeFromLevel(ctx context.Context, log *slog.Logger, repo volumeByLevelByOrg, orgID int64, level float64) (float64, bool) {
	v, err := repo.GetVolumeByLevelByOrg(ctx, orgID, level)
	if err == nil {
		return v, true
	}
	if errors.Is(err, storage.ErrLevelVolumeNotConfigured) {
		return 0, false
	}
	if errors.Is(err, storage.ErrLevelOutOfCurveRange) {
		log.Warn("level outside level_volume curve range",
			slog.Int64("organization_id", orgID),
			slog.Float64("level", level),
		)
		return 0, false
	}
	log.Error("failed to compute volume from level",
		sl.Err(err),
		slog.Int64("organization_id", orgID),
		slog.Float64("level", level),
	)
	return 0, false
}

// Supported values for ReservoirSummaryConfig.VolumeSource. Defined here so
// the strategy switch and the handler default agree on the exact strings —
// drift would silently send writes through that the CHECK constraint rejects.
const (
	volumeSourceStatic      = "static"
	volumeSourceLevelVolume = "level_volume"
)

// applyStaticFallbacks mutates summaries in place: pulls income/release/level
// from the static.uz day-begin snapshot when DB values are zero, then resolves
// Volume.Current.
//
// Volume resolution always starts with the DB snapshot
// (reservoir_data.volume_mln_m3): if the operator typed a value for this day
// it wins over both sources — "what the operator entered must show in the
// report". The strategy switch only fires when the snapshot is zero
// (operator hasn't entered anything yet):
//
//   - "static" (default): static.uz Volume → curve → 0
//   - "level_volume":     curve → static.uz Volume → 0
//
// Both strategies use the other source as a fallback so partial coverage
// (curve not configured / static.uz outage) still produces a value where
// possible. Empty/unknown VolumeSource → "static" so a missing ConfigLookup
// entry or a future enum value the binary doesn't recognise degrades
// gracefully.
//
// configs is also consulted to mask Modsnow.Current / Modsnow.YearAgo for any
// org whose ReservoirSummaryConfig has ModsnowEnabled=false. This is the
// JSON-side counterpart of the empty-modsnow-cell behaviour in the Excel
// generator: a reservoir with modsnow disabled should never expose stored
// modsnow values to the frontend regardless of what the SQL query returned.
func applyStaticFallbacks(
	ctx context.Context,
	log *slog.Logger,
	summaries []*reservoirsummary.ResponseModel,
	dataAtDayBegin map[int64]*dto.OrganizationWithData,
	curve volumeByLevelByOrg,
	configs ConfigLookup,
) {
	isEdited := true
	for _, summary := range summaries {
		if summary.OrganizationID == nil {
			continue
		}

		// Modsnow masking is the first thing we do — it doesn't depend on
		// static.uz data and is cheap to short-circuit. ConfigLookup is
		// allowed to be nil (test helpers); treat that as "no override".
		if configs != nil {
			if cfg, ok := configs.Get(*summary.OrganizationID); ok && !cfg.ModsnowEnabled {
				summary.Modsnow.Current = 0
				summary.Modsnow.YearAgo = 0
			}
		}

		// static.uz snapshot is optional: a reservoir without an entry can still
		// have its Volume recomputed from a Level that's already in the DB.
		var staticData *dto.ReservoirData
		if val, ok := dataAtDayBegin[*summary.OrganizationID]; ok && val.Data != nil {
			staticData = val.Data
		}

		// Income/Release/Level fallbacks are independent of volume_source —
		// they're driven entirely by the static.uz snapshot and apply to both
		// strategies identically.
		if staticData != nil {
			if summary.Income.Current == 0 && staticData.AvgIncome != nil {
				summary.Income.Current = *staticData.AvgIncome
				summary.Income.IsEdited = &isEdited
			}
			if summary.Release.Current == 0 && staticData.AvgRelease != nil {
				summary.Release.Current = *staticData.AvgRelease
				summary.Release.IsEdited = &isEdited
			}
			if staticData.Level != nil && *staticData.Level != 0 && summary.Level.Current == 0 {
				summary.Level.Current = *staticData.Level
				summary.Level.IsEdited = &isEdited
			}
		}

		// Snapshot wins universally — operator's manual POST always surfaces.
		if summary.Volume.Current != 0 {
			continue
		}

		source := volumeSourceStatic
		if configs != nil {
			if cfg, ok := configs.Get(*summary.OrganizationID); ok && cfg.VolumeSource != "" {
				source = cfg.VolumeSource
			}
		}

		// Strategy fires only when snapshot is zero. Both strategies are
		// symmetric: try the configured primary source, then the other one,
		// then leave Volume at 0.
		var primary, fallback volumeProvider
		switch source {
		case volumeSourceLevelVolume:
			primary = curveProvider{ctx: ctx, log: log, curve: curve}
			fallback = staticUzProvider{data: staticData}
		default: // volumeSourceStatic + unknown values
			primary = staticUzProvider{data: staticData}
			fallback = curveProvider{ctx: ctx, log: log, curve: curve}
		}

		if v, ok := primary.volume(*summary.OrganizationID, summary.Level.Current); ok {
			summary.Volume.Current = v
			summary.Volume.IsEdited = &isEdited
			continue
		}
		if v, ok := fallback.volume(*summary.OrganizationID, summary.Level.Current); ok {
			summary.Volume.Current = v
			summary.Volume.IsEdited = &isEdited
		}
		// Both sources empty → Volume stays 0 (the SQL default).
	}
}

// volumeProvider abstracts the two sources of volume so the strategy switch
// can compose them without duplicating per-source error handling.
type volumeProvider interface {
	volume(orgID int64, level float64) (float64, bool)
}

type staticUzProvider struct {
	data *dto.ReservoirData
}

func (s staticUzProvider) volume(_ int64, _ float64) (float64, bool) {
	if s.data == nil || s.data.Volume == nil || *s.data.Volume == 0 {
		return 0, false
	}
	return *s.data.Volume, true
}

type curveProvider struct {
	ctx   context.Context
	log   *slog.Logger
	curve volumeByLevelByOrg
}

func (c curveProvider) volume(orgID int64, level float64) (float64, bool) {
	if level == 0 {
		return 0, false
	}
	return computeVolumeFromLevel(c.ctx, c.log, c.curve, orgID, level)
}
