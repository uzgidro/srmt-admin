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

// applyStaticFallbacks mutates summaries in place: pulls income/release/level
// from the static.uz day-begin snapshot when DB values are zero, and recomputes
// Volume.Current from the (possibly just-pulled) Level via the level_volume
// curve. If the curve isn't available, falls back to the static.uz Volume — the
// pre-existing behaviour before the curve recompute was added.
//
// configs is consulted to mask Modsnow.Current / Modsnow.YearAgo for any org
// whose ReservoirSummaryConfig has ModsnowEnabled=false. This is the JSON-side
// counterpart of the empty-modsnow-cell behaviour in the Excel generator: a
// reservoir with modsnow disabled should never expose stored modsnow values
// to the frontend regardless of what the SQL query returned.
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

		if summary.Volume.Current != 0 {
			continue
		}
		// Prefer the calibration curve over the static.uz size field — operators
		// reported that static.uz volume disagrees with the level→volume table
		// for several reservoirs, so we recompute when we can and only fall
		// back to static.size when the curve is unavailable.
		if summary.Level.Current != 0 {
			if computed, computedOK := computeVolumeFromLevel(ctx, log, curve, *summary.OrganizationID, summary.Level.Current); computedOK {
				summary.Volume.Current = computed
				summary.Volume.IsEdited = &isEdited
				continue
			}
		}
		if staticData != nil && staticData.Volume != nil && *staticData.Volume != 0 {
			summary.Volume.Current = *staticData.Volume
			summary.Volume.IsEdited = &isEdited
		}
	}
}
