package metrics

import (
	"context"
	"log/slog"

	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/asutp"
)

// TelemetryGetter defines the interface for getting station telemetry from Redis
type TelemetryGetter interface {
	GetStationTelemetry(ctx context.Context, stationDBID int64) ([]*asutp.Envelope, error)
}

// MetricsBlender wraps an ASCUEFetcher and enriches data with ASUTP telemetry
type MetricsBlender struct {
	ascueFetcher dto.ASCUEFetcher
	redisRepo    TelemetryGetter
	log          *slog.Logger
}

// NewMetricsBlender creates a new MetricsBlender
func NewMetricsBlender(ascueFetcher dto.ASCUEFetcher, redisRepo TelemetryGetter, log *slog.Logger) *MetricsBlender {
	return &MetricsBlender{
		ascueFetcher: ascueFetcher,
		redisRepo:    redisRepo,
		log:          log,
	}
}

// FetchAll fetches ASCUE data and enriches it with ASUTP telemetry for configured organizations
func (b *MetricsBlender) FetchAll(ctx context.Context) (map[int64]*dto.ASCUEMetrics, error) {
	const op = "metrics.blender.FetchAll"

	// Get base ASCUE data
	result, err := b.ascueFetcher.FetchAll(ctx)
	if err != nil {
		return nil, err
	}

	// Enrich with ASUTP data for BlendOrganizationID (GES-1)
	b.blendASUTPMetrics(ctx, result, BlendOrganizationID)

	return result, nil
}

// blendASUTPMetrics enriches ASCUE metrics with ASUTP telemetry for a specific organization
func (b *MetricsBlender) blendASUTPMetrics(ctx context.Context, result map[int64]*dto.ASCUEMetrics, orgID int64) {
	const op = "metrics.blender.blendASUTPMetrics"

	envelopes, err := b.redisRepo.GetStationTelemetry(ctx, orgID)
	if err != nil {
		b.log.Warn("failed to get ASUTP telemetry, using ASCUE data only",
			slog.String("op", op),
			slog.Int64("organization_id", orgID),
			slog.Any("error", err),
		)
		return
	}

	if len(envelopes) == 0 {
		b.log.Debug("no ASUTP telemetry available",
			slog.String("op", op),
			slog.Int64("organization_id", orgID),
		)
		return
	}

	// Calculate metrics from ASUTP telemetry
	asutpMetrics := CalculateFromEnvelopes(envelopes)

	// Get or create metrics for this organization
	existing, ok := result[orgID]
	if !ok {
		result[orgID] = asutpMetrics
		return
	}

	// Blend ASUTP metrics into existing ASCUE metrics
	existing.Active = asutpMetrics.Active
	existing.Reactive = asutpMetrics.Reactive
	existing.PowerExport = asutpMetrics.PowerExport
	existing.ActiveAggCount = asutpMetrics.ActiveAggCount
	existing.PendingAggCount = asutpMetrics.PendingAggCount
	existing.RepairAggCount = asutpMetrics.RepairAggCount

	b.log.Debug("successfully blended ASUTP metrics",
		slog.String("op", op),
		slog.Int64("organization_id", orgID),
		slog.Int("envelopes_count", len(envelopes)),
	)
}
