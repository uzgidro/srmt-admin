package metrics

const (
	// Device groups from ASUTP telemetry
	DeviceGroupGenerators = "generators"
	DeviceGroupLines35kV  = "lines_35kv"

	// Data point names
	DataPointActivePower   = "active_power_kw"
	DataPointReactivePower = "reactive_power_kvar"

	// Organization ID for blending (GES-1)
	BlendOrganizationID = int64(32)

	// Conversion factor from kW to MW
	KWtoMW = 1000.0
)
