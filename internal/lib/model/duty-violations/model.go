// Package dutyviolations defines the data model for the duty-officer
// violations CRUD feature (запись о прогуле/нарушении дежурного с
// привязанными файлами).
package dutyviolations

import (
	"time"

	"srmt-admin/internal/lib/model/file"
)

// DutyViolation is the response shape returned by the HTTP layer.
// Always includes the attached files (possibly empty) so the frontend can
// render the record without an extra round-trip.
type DutyViolation struct {
	ID               int64        `json:"id"`
	OrganizationID   int64        `json:"organization_id"`
	OrganizationName string       `json:"organization_name,omitempty"`
	StartTime        time.Time    `json:"start_time"`
	EndTime          time.Time    `json:"end_time"`
	DutyOfficerName  string       `json:"duty_officer_name"`
	Reason           string       `json:"reason"`
	Files            []file.Model `json:"files"`
	CreatedAt        time.Time    `json:"created_at"`
	CreatedByUserID  *int64       `json:"created_by_user_id,omitempty"`
	UpdatedAt        time.Time    `json:"updated_at"`
}

// CreateRequest is the POST body. file_ids must come from a prior
// POST /upload/files call (frontend uploads files first, then submits the
// record with the returned IDs).
type CreateRequest struct {
	OrganizationID  int64     `json:"organization_id" validate:"required,gt=0"`
	StartTime       time.Time `json:"start_time" validate:"required"`
	EndTime         time.Time `json:"end_time" validate:"required,gtfield=StartTime"`
	DutyOfficerName string    `json:"duty_officer_name" validate:"required,min=1,max=200"`
	Reason          string    `json:"reason" validate:"required,min=1,max=2000"`
	FileIDs         []int64   `json:"file_ids" validate:"omitempty,dive,gt=0"`
}

// UpdateRequest is the PATCH body. file_ids is treated as a full
// replacement of the current attachment list (not a delta). To clear all
// attachments pass an empty array; to add one pass [...old, new].
type UpdateRequest struct {
	OrganizationID  int64     `json:"organization_id" validate:"required,gt=0"`
	StartTime       time.Time `json:"start_time" validate:"required"`
	EndTime         time.Time `json:"end_time" validate:"required,gtfield=StartTime"`
	DutyOfficerName string    `json:"duty_officer_name" validate:"required,min=1,max=200"`
	Reason          string    `json:"reason" validate:"required,min=1,max=2000"`
	FileIDs         []int64   `json:"file_ids" validate:"omitempty,dive,gt=0"`
}

// ListFilter holds optional query-string filters for the GET list endpoint.
// All fields are nil when not provided; the repo composes a dynamic WHERE
// clause from the non-nil ones.
//
// Day is the operational day (anchored at 05:00 local time, Asia/Tashkent).
// The repo translates it to a half-open `[Day, Day+24h)` window — matching
// incidents/visits/shutdowns. A nil Day means "no day filter at all" (used
// when listing without a date param).
type ListFilter struct {
	OrganizationID *int64
	Day            *time.Time
}
