package alarm

import (
	"testing"

	"srmt-admin/internal/lib/model/asutp"
)

func TestDetectTriggeredAlarms(t *testing.T) {
	tests := []struct {
		name     string
		values   []asutp.DataPoint
		expected int
	}{
		{
			name:     "no values",
			values:   []asutp.DataPoint{},
			expected: 0,
		},
		{
			name: "no alarms in values",
			values: []asutp.DataPoint{
				{Name: "temperature", Value: 75.5, Quality: "good"},
				{Name: "pressure", Value: 100.0, Quality: "good"},
			},
			expected: 0,
		},
		{
			name: "alarm not triggered (false)",
			values: []asutp.DataPoint{
				{Name: "emergency_stop", Value: false, Quality: "good"},
			},
			expected: 0,
		},
		{
			name: "single alarm triggered (bool true)",
			values: []asutp.DataPoint{
				{Name: "emergency_stop", Value: true, Quality: "good"},
			},
			expected: 1,
		},
		{
			name: "single alarm triggered (float64 1)",
			values: []asutp.DataPoint{
				{Name: "emergency_stop", Value: float64(1), Quality: "good"},
			},
			expected: 1,
		},
		{
			name: "single alarm triggered (string true)",
			values: []asutp.DataPoint{
				{Name: "emergency_stop", Value: "true", Quality: "good"},
			},
			expected: 1,
		},
		{
			name: "multiple alarms triggered",
			values: []asutp.DataPoint{
				{Name: "emergency_stop", Value: true, Quality: "good"},
				{Name: "protection_set_a_trip", Value: true, Quality: "good"},
				{Name: "protection_general_trip", Value: true, Quality: "good"},
			},
			expected: 3,
		},
		{
			name: "mixed triggered and not triggered",
			values: []asutp.DataPoint{
				{Name: "emergency_stop", Value: true, Quality: "good"},
				{Name: "emergency_stop_button1", Value: false, Quality: "good"},
				{Name: "protection_set_a_trip", Value: true, Quality: "good"},
			},
			expected: 2,
		},
		{
			name: "all alarms triggered",
			values: []asutp.DataPoint{
				{Name: "emergency_stop", Value: true, Quality: "good"},
				{Name: "emergency_stop_button1", Value: true, Quality: "good"},
				{Name: "emergency_stop_button2", Value: true, Quality: "good"},
				{Name: "protection_set_a_trip", Value: true, Quality: "good"},
				{Name: "protection_set_b_trip", Value: true, Quality: "good"},
				{Name: "protection_general_trip", Value: true, Quality: "good"},
				{Name: "manual_emergency_stop_mosaic", Value: true, Quality: "good"},
			},
			expected: 7,
		},
		{
			name: "alarm with int value 0 (not triggered)",
			values: []asutp.DataPoint{
				{Name: "emergency_stop", Value: 0, Quality: "good"},
			},
			expected: 0,
		},
		{
			name: "alarm with int value 1 (triggered)",
			values: []asutp.DataPoint{
				{Name: "emergency_stop", Value: 1, Quality: "good"},
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectTriggeredAlarms(tt.values)
			if len(result) != tt.expected {
				t.Errorf("DetectTriggeredAlarms() got %d alarms, want %d", len(result), tt.expected)
			}
		})
	}
}

func TestExtractGeneratorNumber(t *testing.T) {
	tests := []struct {
		deviceID string
		expected string
	}{
		{"gen1", "Г1"},
		{"gen2", "Г2"},
		{"gen10", "Г10"},
		{"gen123", "Г123"},
		{"generator1", ""},
		{"GEN1", ""},
		{"gen", ""},
		{"pump1", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.deviceID, func(t *testing.T) {
			result := ExtractGeneratorNumber(tt.deviceID)
			if result != tt.expected {
				t.Errorf("ExtractGeneratorNumber(%q) = %q, want %q", tt.deviceID, result, tt.expected)
			}
		})
	}
}

func TestFormatReason(t *testing.T) {
	tests := []struct {
		name     string
		deviceID string
		alarms   []AlarmSignal
		expected string
	}{
		{
			name:     "empty alarms",
			deviceID: "gen1",
			alarms:   []AlarmSignal{},
			expected: "",
		},
		{
			name:     "single alarm with generator",
			deviceID: "gen1",
			alarms: []AlarmSignal{
				{Name: "emergency_stop", Description: "Аварийный останов"},
			},
			expected: "Г1: Аварийный останов",
		},
		{
			name:     "multiple alarms with generator",
			deviceID: "gen2",
			alarms: []AlarmSignal{
				{Name: "emergency_stop", Description: "Аварийный останов"},
				{Name: "protection_set_a_trip", Description: "Срабатывание защиты комплекта А"},
			},
			expected: "Г2: Аварийный останов, Срабатывание защиты комплекта А",
		},
		{
			name:     "alarm without generator prefix",
			deviceID: "pump1",
			alarms: []AlarmSignal{
				{Name: "emergency_stop", Description: "Аварийный останов"},
			},
			expected: "Аварийный останов",
		},
		{
			name:     "multiple alarms without generator",
			deviceID: "unknown_device",
			alarms: []AlarmSignal{
				{Name: "emergency_stop", Description: "Аварийный останов"},
				{Name: "protection_general_trip", Description: "Срабатывание общей защиты"},
			},
			expected: "Аварийный останов, Срабатывание общей защиты",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatReason(tt.deviceID, tt.alarms)
			if result != tt.expected {
				t.Errorf("FormatReason(%q, %v) = %q, want %q", tt.deviceID, tt.alarms, result, tt.expected)
			}
		})
	}
}

func TestIsTrueValue(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{"bool true", true, true},
		{"bool false", false, false},
		{"float64 1", float64(1), true},
		{"float64 0", float64(0), false},
		{"float64 -1", float64(-1), true},
		{"int 1", 1, true},
		{"int 0", 0, false},
		{"int64 1", int64(1), true},
		{"int64 0", int64(0), false},
		{"string true", "true", true},
		{"string TRUE", "TRUE", true},
		{"string 1", "1", true},
		{"string yes", "yes", true},
		{"string false", "false", false},
		{"string 0", "0", false},
		{"nil", nil, false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTrueValue(tt.value)
			if result != tt.expected {
				t.Errorf("isTrueValue(%v) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}
