package dto

import "time"

type AddDischargeRequest struct {
	OrganizationID int64      `json:"organization_id"`
	StartTime      time.Time  `json:"start_time"`
	EndTime        *time.Time `json:"end_time,omitempty"`
	FlowRate       float64    `json:"flow_rate"`
	Reason         *string    `json:"reason,omitempty"`
	FileIDs        []int64    `json:"file_ids,omitempty"`
}

type EditDischargeRequest struct {
	OrganizationID *int64     `json:"organization_id,omitempty"`
	StartTime      *time.Time `json:"start_time,omitempty"`
	EndTime        *time.Time `json:"end_time,omitempty"`
	FlowRate       *float64   `json:"flow_rate,omitempty"`
	Reason         *string    `json:"reason,omitempty"`
	Approved       *bool      `json:"approved,omitempty"`
	FileIDs        []int64    `json:"file_ids,omitempty"`
}
