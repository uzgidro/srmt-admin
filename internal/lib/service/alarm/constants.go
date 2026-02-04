package alarm

// AlarmSignal describes an emergency alarm signal
type AlarmSignal struct {
	Name        string
	Description string // Russian description
}

// MonitoredAlarms is the list of alarm signals to monitor
var MonitoredAlarms = []AlarmSignal{
	{Name: "emergency_stop", Description: "Аварийный останов"},
	{Name: "emergency_stop_button1", Description: "Кнопка аварийного останова №1"},
	{Name: "emergency_stop_button2", Description: "Кнопка аварийного останова №2"},
	{Name: "protection_set_a_trip", Description: "Срабатывание защиты комплекта А"},
	{Name: "protection_set_b_trip", Description: "Срабатывание защиты комплекта Б"},
	{Name: "protection_general_trip", Description: "Срабатывание общей защиты"},
	{Name: "manual_emergency_stop_mosaic", Description: "Ручной аварийный останов (мозаика)"},
}

// SystemUserID is the user ID for automatically created records
const SystemUserID = int64(0)
