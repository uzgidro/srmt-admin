package metrics

import (
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/asutp"
)

// CalculateFromEnvelopes calculates ASCUE metrics from ASUTP telemetry envelopes
func CalculateFromEnvelopes(envelopes []*asutp.Envelope) *dto.ASCUEMetrics {
	var (
		active      float64
		reactive    float64
		powerExport float64
		activeAgg   int
		pendingAgg  int
	)

	for _, env := range envelopes {
		switch env.DeviceGroup {
		case DeviceGroupGenerators:
			for _, dp := range env.Values {
				switch dp.Name {
				case DataPointActivePower:
					val := toFloat64(dp.Value)
					active += val
					if val > 0 {
						activeAgg++
					} else {
						pendingAgg++
					}
				case DataPointReactivePower:
					reactive += toFloat64(dp.Value)
				}
			}
		case DeviceGroupLines35kV:
			for _, dp := range env.Values {
				if dp.Name == DataPointActivePower {
					powerExport += toFloat64(dp.Value)
				}
			}
		}
	}

	// Convert kW to MW
	active /= KWtoMW
	reactive /= KWtoMW
	powerExport /= KWtoMW

	repairAgg := 0

	return &dto.ASCUEMetrics{
		Active:          &active,
		Reactive:        &reactive,
		PowerExport:     &powerExport,
		ActiveAggCount:  &activeAgg,
		PendingAggCount: &pendingAgg,
		RepairAggCount:  &repairAgg,
	}
}

// toFloat64 safely converts interface{} to float64
func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case int32:
		return float64(val)
	default:
		return 0
	}
}
