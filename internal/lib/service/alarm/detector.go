package alarm

import (
	"regexp"
	"strings"

	"srmt-admin/internal/lib/model/asutp"
)

// alarmMap provides O(1) lookup for alarm signals
var alarmMap map[string]AlarmSignal

func init() {
	alarmMap = make(map[string]AlarmSignal, len(MonitoredAlarms))
	for _, alarm := range MonitoredAlarms {
		alarmMap[alarm.Name] = alarm
	}
}

// DetectTriggeredAlarms returns the list of triggered alarm signals
func DetectTriggeredAlarms(values []asutp.DataPoint) []AlarmSignal {
	var triggered []AlarmSignal

	for _, dp := range values {
		// Check if this data point is a monitored alarm
		alarm, isMonitored := alarmMap[dp.Name]
		if !isMonitored {
			continue
		}

		// Check if the value is true (alarm triggered)
		if isTrueValue(dp.Value) {
			triggered = append(triggered, alarm)
		}
	}

	return triggered
}

// isTrueValue checks if the value represents a triggered alarm
func isTrueValue(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return v
	case float64:
		return v != 0
	case int:
		return v != 0
	case int64:
		return v != 0
	case string:
		lower := strings.ToLower(v)
		return lower == "true" || lower == "1" || lower == "yes"
	default:
		return false
	}
}

// generatorRegex matches device IDs like "gen1", "gen2", etc.
var generatorRegex = regexp.MustCompile(`^gen(\d+)$`)

// ExtractGeneratorNumber extracts the generator number from device_id
// Examples: "gen1" -> "Г1", "gen2" -> "Г2", "other" -> ""
func ExtractGeneratorNumber(deviceID string) string {
	matches := generatorRegex.FindStringSubmatch(deviceID)
	if len(matches) == 2 {
		return "Г" + matches[1]
	}
	return ""
}

// FormatReason formats the reason in Russian with generator prefix
// Examples:
//   - "Г1: Аварийный останов, Срабатывание защиты комплекта А"
//   - "Г2: Кнопка аварийного останова №1"
func FormatReason(deviceID string, alarms []AlarmSignal) string {
	if len(alarms) == 0 {
		return ""
	}

	// Build descriptions list
	descriptions := make([]string, len(alarms))
	for i, alarm := range alarms {
		descriptions[i] = alarm.Description
	}

	reason := strings.Join(descriptions, ", ")

	// Add generator prefix if applicable
	genNum := ExtractGeneratorNumber(deviceID)
	if genNum != "" {
		return genNum + ": " + reason
	}

	return reason
}
