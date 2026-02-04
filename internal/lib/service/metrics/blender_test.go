package metrics

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/asutp"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockASCUEFetcher is a mock implementation of dto.ASCUEFetcher
type mockASCUEFetcher struct {
	result map[int64]*dto.ASCUEMetrics
	err    error
}

func (m *mockASCUEFetcher) FetchAll(ctx context.Context) (map[int64]*dto.ASCUEMetrics, error) {
	return m.result, m.err
}

// mockTelemetryGetter is a mock implementation of TelemetryGetter
type mockTelemetryGetter struct {
	envelopes map[int64][]*asutp.Envelope
	err       error
}

func (m *mockTelemetryGetter) GetStationTelemetry(ctx context.Context, stationDBID int64) ([]*asutp.Envelope, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.envelopes[stationDBID], nil
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestMetricsBlender_FetchAll_BlendASUTPMetrics(t *testing.T) {
	active := 10.0
	reactive := 5.0

	ascueFetcher := &mockASCUEFetcher{
		result: map[int64]*dto.ASCUEMetrics{
			BlendOrganizationID: {
				Active:   &active,
				Reactive: &reactive,
			},
		},
	}

	telemetryGetter := &mockTelemetryGetter{
		envelopes: map[int64][]*asutp.Envelope{
			BlendOrganizationID: {
				{
					DeviceID:    "gen1",
					DeviceGroup: DeviceGroupGenerators,
					Values: []asutp.DataPoint{
						{Name: DataPointActivePower, Value: 8000.0},   // 8 MW
						{Name: DataPointReactivePower, Value: 3000.0}, // 3 MVAr
					},
				},
				{
					DeviceID:    "line1",
					DeviceGroup: DeviceGroupLines35kV,
					Values: []asutp.DataPoint{
						{Name: DataPointActivePower, Value: 7500.0}, // 7.5 MW export
					},
				},
			},
		},
	}

	blender := NewMetricsBlender(ascueFetcher, telemetryGetter, newTestLogger())
	result, err := blender.FetchAll(context.Background())

	require.NoError(t, err)
	require.NotNil(t, result)

	metrics := result[BlendOrganizationID]
	require.NotNil(t, metrics)

	// ASUTP values should replace ASCUE values
	assert.InDelta(t, 8.0, *metrics.Active, 0.001)      // From ASUTP
	assert.InDelta(t, 3.0, *metrics.Reactive, 0.001)    // From ASUTP
	assert.InDelta(t, 7.5, *metrics.PowerExport, 0.001) // From ASUTP
	assert.Equal(t, 1, *metrics.ActiveAggCount)
	assert.Equal(t, 0, *metrics.PendingAggCount)
}

func TestMetricsBlender_FetchAll_ASUTPUnavailable(t *testing.T) {
	active := 10.0
	reactive := 5.0

	ascueFetcher := &mockASCUEFetcher{
		result: map[int64]*dto.ASCUEMetrics{
			BlendOrganizationID: {
				Active:   &active,
				Reactive: &reactive,
			},
		},
	}

	telemetryGetter := &mockTelemetryGetter{
		err: errors.New("redis connection failed"),
	}

	blender := NewMetricsBlender(ascueFetcher, telemetryGetter, newTestLogger())
	result, err := blender.FetchAll(context.Background())

	require.NoError(t, err)
	require.NotNil(t, result)

	metrics := result[BlendOrganizationID]
	require.NotNil(t, metrics)

	// Should return original ASCUE values when ASUTP unavailable
	assert.InDelta(t, 10.0, *metrics.Active, 0.001)
	assert.InDelta(t, 5.0, *metrics.Reactive, 0.001)
}

func TestMetricsBlender_FetchAll_NoASUTPTelemetry(t *testing.T) {
	active := 10.0
	reactive := 5.0

	ascueFetcher := &mockASCUEFetcher{
		result: map[int64]*dto.ASCUEMetrics{
			BlendOrganizationID: {
				Active:   &active,
				Reactive: &reactive,
			},
		},
	}

	telemetryGetter := &mockTelemetryGetter{
		envelopes: map[int64][]*asutp.Envelope{
			BlendOrganizationID: {}, // Empty envelopes
		},
	}

	blender := NewMetricsBlender(ascueFetcher, telemetryGetter, newTestLogger())
	result, err := blender.FetchAll(context.Background())

	require.NoError(t, err)
	require.NotNil(t, result)

	metrics := result[BlendOrganizationID]
	require.NotNil(t, metrics)

	// Should return original ASCUE values when no ASUTP telemetry
	assert.InDelta(t, 10.0, *metrics.Active, 0.001)
	assert.InDelta(t, 5.0, *metrics.Reactive, 0.001)
}

func TestMetricsBlender_FetchAll_ASCUEError(t *testing.T) {
	ascueFetcher := &mockASCUEFetcher{
		err: errors.New("ASCUE fetch failed"),
	}

	telemetryGetter := &mockTelemetryGetter{}

	blender := NewMetricsBlender(ascueFetcher, telemetryGetter, newTestLogger())
	result, err := blender.FetchAll(context.Background())

	require.Error(t, err)
	require.Nil(t, result)
}

func TestMetricsBlender_FetchAll_NewOrganization(t *testing.T) {
	// Test when organization doesn't exist in ASCUE but has ASUTP data
	ascueFetcher := &mockASCUEFetcher{
		result: map[int64]*dto.ASCUEMetrics{}, // Empty result
	}

	telemetryGetter := &mockTelemetryGetter{
		envelopes: map[int64][]*asutp.Envelope{
			BlendOrganizationID: {
				{
					DeviceID:    "gen1",
					DeviceGroup: DeviceGroupGenerators,
					Values: []asutp.DataPoint{
						{Name: DataPointActivePower, Value: 5000.0},
						{Name: DataPointReactivePower, Value: 2000.0},
					},
				},
			},
		},
	}

	blender := NewMetricsBlender(ascueFetcher, telemetryGetter, newTestLogger())
	result, err := blender.FetchAll(context.Background())

	require.NoError(t, err)
	require.NotNil(t, result)

	metrics := result[BlendOrganizationID]
	require.NotNil(t, metrics)

	// Should create new metrics from ASUTP data
	assert.InDelta(t, 5.0, *metrics.Active, 0.001)
	assert.InDelta(t, 2.0, *metrics.Reactive, 0.001)
}

func TestMetricsBlender_FetchAll_OtherOrganizationsUnaffected(t *testing.T) {
	active1 := 10.0
	active2 := 20.0

	ascueFetcher := &mockASCUEFetcher{
		result: map[int64]*dto.ASCUEMetrics{
			BlendOrganizationID: {Active: &active1},
			99:                  {Active: &active2}, // Different organization
		},
	}

	telemetryGetter := &mockTelemetryGetter{
		envelopes: map[int64][]*asutp.Envelope{
			BlendOrganizationID: {
				{
					DeviceID:    "gen1",
					DeviceGroup: DeviceGroupGenerators,
					Values: []asutp.DataPoint{
						{Name: DataPointActivePower, Value: 8000.0},
					},
				},
			},
		},
	}

	blender := NewMetricsBlender(ascueFetcher, telemetryGetter, newTestLogger())
	result, err := blender.FetchAll(context.Background())

	require.NoError(t, err)
	require.NotNil(t, result)

	// BlendOrganizationID should be updated
	assert.InDelta(t, 8.0, *result[BlendOrganizationID].Active, 0.001)

	// Other organization should be unchanged
	assert.InDelta(t, 20.0, *result[99].Active, 0.001)
}
