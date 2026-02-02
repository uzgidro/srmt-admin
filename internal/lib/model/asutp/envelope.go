package asutp

import "time"

type DataPoint struct {
	Name     string      `json:"name"`
	Value    interface{} `json:"value"`
	Unit     string      `json:"unit,omitempty"`
	Quality  string      `json:"quality"`
	Severity string      `json:"severity,omitempty"`
}

type Envelope struct {
	ID          string      `json:"id"`
	StationID   string      `json:"station_id"`
	StationName string      `json:"station_name"`
	Timestamp   time.Time   `json:"timestamp"`
	DeviceID    string      `json:"device_id"`
	DeviceName  string      `json:"device_name"`
	DeviceGroup string      `json:"device_group"`
	Values      []DataPoint `json:"values"`
}
