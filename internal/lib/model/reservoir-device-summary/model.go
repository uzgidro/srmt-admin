package reservoirdevicesummary

import (
	"time"
)

type ResponseModel struct {
	ID                   int64      `json:"id"`
	OrganizationID       int64      `json:"organization_id"`
	OrganizationName     string     `json:"organization_name"`
	DeviceTypeName       string     `json:"device_type_name"`
	CountTotal           int        `json:"count_total"`
	CountInstalled       int        `json:"count_installed"`
	CountOperational     int        `json:"count_operational"`
	CountFaulty          int        `json:"count_faulty"`
	CountActive          int        `json:"count_active"`
	CountAutomationScope int        `json:"count_automation_scope"`
	Criterion1           *float64   `json:"criterion_1,omitempty"`
	Criterion2           *float64   `json:"criterion_2,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            *time.Time `json:"updated_at,omitempty"`
	UpdatedByUserID      *int64     `json:"updated_by_user_id,omitempty"`
}
