package dto

type PatchReservoirDeviceSummaryItem struct {
	OrganizationID       int64    `json:"organization_id"`
	DeviceTypeName       string   `json:"device_type_name"`
	CountTotal           *int     `json:"count_total,omitempty"`
	CountInstalled       *int     `json:"count_installed,omitempty"`
	CountOperational     *int     `json:"count_operational,omitempty"`
	CountFaulty          *int     `json:"count_faulty,omitempty"`
	CountActive          *int     `json:"count_active,omitempty"`
	CountAutomationScope *int     `json:"count_automation_scope,omitempty"`
	Criterion1           *float64 `json:"criterion_1,omitempty"`
	Criterion2           *float64 `json:"criterion_2,omitempty"`
}

type PatchReservoirDeviceSummaryRequest struct {
	Updates []PatchReservoirDeviceSummaryItem `json:"updates"`
}
