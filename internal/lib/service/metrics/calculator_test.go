package metrics

import (
	"testing"

	"srmt-admin/internal/lib/model/asutp"

	"github.com/stretchr/testify/assert"
)

func TestCalculateFromEnvelopes_Generators(t *testing.T) {
	envelopes := []*asutp.Envelope{
		{
			DeviceID:    "gen1",
			DeviceGroup: DeviceGroupGenerators,
			Values: []asutp.DataPoint{
				{Name: DataPointActivePower, Value: 5000.0},   // 5 MW
				{Name: DataPointReactivePower, Value: 1000.0}, // 1 MVAr
			},
		},
		{
			DeviceID:    "gen2",
			DeviceGroup: DeviceGroupGenerators,
			Values: []asutp.DataPoint{
				{Name: DataPointActivePower, Value: 3000.0},  // 3 MW
				{Name: DataPointReactivePower, Value: 500.0}, // 0.5 MVAr
			},
		},
	}

	result := CalculateFromEnvelopes(envelopes)

	assert.NotNil(t, result)
	assert.InDelta(t, 8.0, *result.Active, 0.001)   // (5000+3000)/1000 = 8 MW
	assert.InDelta(t, 1.5, *result.Reactive, 0.001) // (1000+500)/1000 = 1.5 MVAr
	assert.Equal(t, 2, *result.ActiveAggCount)      // 2 generators with active > 0
	assert.Equal(t, 0, *result.PendingAggCount)     // 0 generators with active == 0
	assert.Equal(t, 0, *result.RepairAggCount)      // always 0 for now
}

func TestCalculateFromEnvelopes_Lines35kV(t *testing.T) {
	envelopes := []*asutp.Envelope{
		{
			DeviceID:    "line1",
			DeviceGroup: DeviceGroupLines35kV,
			Values: []asutp.DataPoint{
				{Name: DataPointActivePower, Value: 10000.0}, // 10 MW export
			},
		},
		{
			DeviceID:    "line2",
			DeviceGroup: DeviceGroupLines35kV,
			Values: []asutp.DataPoint{
				{Name: DataPointActivePower, Value: 5000.0}, // 5 MW export
			},
		},
	}

	result := CalculateFromEnvelopes(envelopes)

	assert.NotNil(t, result)
	assert.InDelta(t, 15.0, *result.PowerExport, 0.001) // (10000+5000)/1000 = 15 MW
}

func TestCalculateFromEnvelopes_MixedDevices(t *testing.T) {
	envelopes := []*asutp.Envelope{
		{
			DeviceID:    "gen1",
			DeviceGroup: DeviceGroupGenerators,
			Values: []asutp.DataPoint{
				{Name: DataPointActivePower, Value: 5000.0},
				{Name: DataPointReactivePower, Value: 1000.0},
			},
		},
		{
			DeviceID:    "gen2",
			DeviceGroup: DeviceGroupGenerators,
			Values: []asutp.DataPoint{
				{Name: DataPointActivePower, Value: 0.0}, // Pending generator
				{Name: DataPointReactivePower, Value: 0.0},
			},
		},
		{
			DeviceID:    "line1",
			DeviceGroup: DeviceGroupLines35kV,
			Values: []asutp.DataPoint{
				{Name: DataPointActivePower, Value: 4500.0},
			},
		},
	}

	result := CalculateFromEnvelopes(envelopes)

	assert.NotNil(t, result)
	assert.InDelta(t, 5.0, *result.Active, 0.001)      // 5000/1000 = 5 MW
	assert.InDelta(t, 1.0, *result.Reactive, 0.001)    // 1000/1000 = 1 MVAr
	assert.InDelta(t, 4.5, *result.PowerExport, 0.001) // 4500/1000 = 4.5 MW
	assert.Equal(t, 1, *result.ActiveAggCount)         // 1 generator with active > 0
	assert.Equal(t, 1, *result.PendingAggCount)        // 1 generator with active == 0
}

func TestCalculateFromEnvelopes_EmptyEnvelopes(t *testing.T) {
	envelopes := []*asutp.Envelope{}

	result := CalculateFromEnvelopes(envelopes)

	assert.NotNil(t, result)
	assert.InDelta(t, 0.0, *result.Active, 0.001)
	assert.InDelta(t, 0.0, *result.Reactive, 0.001)
	assert.InDelta(t, 0.0, *result.PowerExport, 0.001)
	assert.Equal(t, 0, *result.ActiveAggCount)
	assert.Equal(t, 0, *result.PendingAggCount)
	assert.Equal(t, 0, *result.RepairAggCount)
}

func TestCalculateFromEnvelopes_UnknownDeviceGroup(t *testing.T) {
	envelopes := []*asutp.Envelope{
		{
			DeviceID:    "unknown1",
			DeviceGroup: "unknown_group",
			Values: []asutp.DataPoint{
				{Name: DataPointActivePower, Value: 5000.0},
			},
		},
	}

	result := CalculateFromEnvelopes(envelopes)

	// Unknown device groups should be ignored
	assert.NotNil(t, result)
	assert.InDelta(t, 0.0, *result.Active, 0.001)
	assert.InDelta(t, 0.0, *result.PowerExport, 0.001)
}

func TestToFloat64_VariousTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected float64
	}{
		{"float64", float64(123.45), 123.45},
		{"float32", float32(123.45), float64(float32(123.45))},
		{"int", int(100), 100.0},
		{"int64", int64(100), 100.0},
		{"int32", int32(100), 100.0},
		{"string", "invalid", 0.0},
		{"nil", nil, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toFloat64(tt.input)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestCalculateFromEnvelopes_KWtoMWConversion(t *testing.T) {
	// Verify that kW values are correctly converted to MW
	envelopes := []*asutp.Envelope{
		{
			DeviceID:    "gen1",
			DeviceGroup: DeviceGroupGenerators,
			Values: []asutp.DataPoint{
				{Name: DataPointActivePower, Value: 1000.0},   // 1000 kW = 1 MW
				{Name: DataPointReactivePower, Value: 2000.0}, // 2000 kVAr = 2 MVAr
			},
		},
	}

	result := CalculateFromEnvelopes(envelopes)

	assert.InDelta(t, 1.0, *result.Active, 0.001)   // 1000 kW = 1 MW
	assert.InDelta(t, 2.0, *result.Reactive, 0.001) // 2000 kVAr = 2 MVAr
}
