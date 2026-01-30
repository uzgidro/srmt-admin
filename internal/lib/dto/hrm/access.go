package hrm

import (
	"encoding/json"
	"time"
)

// --- Access Zone DTOs ---

// AddAccessZoneRequest represents request to create access zone
type AddAccessZoneRequest struct {
	Name           string          `json:"name" validate:"required"`
	Code           *string         `json:"code,omitempty"`
	Description    *string         `json:"description,omitempty"`
	Building       *string         `json:"building,omitempty"`
	Floor          *string         `json:"floor,omitempty"`
	SecurityLevel  int             `json:"security_level" validate:"required,min=1,max=5"`
	AccessSchedule json.RawMessage `json:"access_schedule,omitempty"`
}

// EditAccessZoneRequest represents request to update access zone
type EditAccessZoneRequest struct {
	Name           *string         `json:"name,omitempty"`
	Code           *string         `json:"code,omitempty"`
	Description    *string         `json:"description,omitempty"`
	Building       *string         `json:"building,omitempty"`
	Floor          *string         `json:"floor,omitempty"`
	SecurityLevel  *int            `json:"security_level,omitempty"`
	AccessSchedule json.RawMessage `json:"access_schedule,omitempty"`
	IsActive       *bool           `json:"is_active,omitempty"`
}

// AccessZoneFilter represents filter for access zones
type AccessZoneFilter struct {
	Building      *string `json:"building,omitempty"`
	SecurityLevel *int    `json:"security_level,omitempty"`
	IsActive      *bool   `json:"is_active,omitempty"`
}

// --- Access Card DTOs ---

// AddAccessCardRequest represents request to create access card
type AddAccessCardRequest struct {
	EmployeeID int64      `json:"employee_id" validate:"required"`
	CardNumber string     `json:"card_number" validate:"required"`
	CardType   string     `json:"card_type" validate:"required,oneof=standard temporary visitor contractor"`
	IssuedDate time.Time  `json:"issued_date" validate:"required"`
	ExpiryDate *time.Time `json:"expiry_date,omitempty"`
	Notes      *string    `json:"notes,omitempty"`
}

// EditAccessCardRequest represents request to update access card
type EditAccessCardRequest struct {
	CardNumber *string    `json:"card_number,omitempty"`
	CardType   *string    `json:"card_type,omitempty"`
	ExpiryDate *time.Time `json:"expiry_date,omitempty"`
	IsActive   *bool      `json:"is_active,omitempty"`
	Notes      *string    `json:"notes,omitempty"`
}

// DeactivateCardRequest represents request to deactivate card
type DeactivateCardRequest struct {
	Reason string `json:"reason" validate:"required"`
}

// AccessCardFilter represents filter for access cards
type AccessCardFilter struct {
	EmployeeID *int64  `json:"employee_id,omitempty"`
	CardType   *string `json:"card_type,omitempty"`
	IsActive   *bool   `json:"is_active,omitempty"`
	Search     *string `json:"search,omitempty"` // Card number search
	Limit      int     `json:"limit,omitempty"`
	Offset     int     `json:"offset,omitempty"`
}

// --- Card Zone Access DTOs ---

// AddCardZoneAccessRequest represents request to grant zone access
type AddCardZoneAccessRequest struct {
	CardID         int64           `json:"card_id" validate:"required"`
	ZoneID         int             `json:"zone_id" validate:"required"`
	CustomSchedule json.RawMessage `json:"custom_schedule,omitempty"`
	ValidFrom      time.Time       `json:"valid_from" validate:"required"`
	ValidUntil     *time.Time      `json:"valid_until,omitempty"`
}

// EditCardZoneAccessRequest represents request to update zone access
type EditCardZoneAccessRequest struct {
	CustomSchedule json.RawMessage `json:"custom_schedule,omitempty"`
	ValidFrom      *time.Time      `json:"valid_from,omitempty"`
	ValidUntil     *time.Time      `json:"valid_until,omitempty"`
}

// BulkZoneAccessRequest represents request to grant multiple zone access
type BulkZoneAccessRequest struct {
	CardID     int64      `json:"card_id" validate:"required"`
	ZoneIDs    []int      `json:"zone_ids" validate:"required,min=1"`
	ValidFrom  time.Time  `json:"valid_from" validate:"required"`
	ValidUntil *time.Time `json:"valid_until,omitempty"`
}

// CardZoneAccessFilter represents filter for zone access
type CardZoneAccessFilter struct {
	CardID *int64 `json:"card_id,omitempty"`
	ZoneID *int   `json:"zone_id,omitempty"`
}

// --- Access Log DTOs ---

// AddAccessLogRequest represents request to log access event
type AddAccessLogRequest struct {
	CardID       *int64          `json:"card_id,omitempty"`
	EmployeeID   *int64          `json:"employee_id,omitempty"`
	ZoneID       int             `json:"zone_id" validate:"required"`
	EventType    string          `json:"event_type" validate:"required,oneof=entry exit denied"`
	EventTime    time.Time       `json:"event_time" validate:"required"`
	DeviceID     *string         `json:"device_id,omitempty"`
	DeviceName   *string         `json:"device_name,omitempty"`
	DenialReason *string         `json:"denial_reason,omitempty"`
	CardNumber   *string         `json:"card_number,omitempty"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
}

// AccessLogFilter represents filter for access logs
type AccessLogFilter struct {
	EmployeeID *int64     `json:"employee_id,omitempty"`
	CardID     *int64     `json:"card_id,omitempty"`
	ZoneID     *int       `json:"zone_id,omitempty"`
	EventType  *string    `json:"event_type,omitempty"`
	FromTime   *time.Time `json:"from_time,omitempty"`
	ToTime     *time.Time `json:"to_time,omitempty"`
	Limit      int        `json:"limit,omitempty"`
	Offset     int        `json:"offset,omitempty"`
}

// AccessReportFilter represents filter for access reports
type AccessReportFilter struct {
	EmployeeID     *int64    `json:"employee_id,omitempty"`
	DepartmentID   *int64    `json:"department_id,omitempty"`
	OrganizationID *int64    `json:"organization_id,omitempty"`
	ZoneID         *int      `json:"zone_id,omitempty"`
	FromTime       time.Time `json:"from_time" validate:"required"`
	ToTime         time.Time `json:"to_time" validate:"required"`
	GroupBy        string    `json:"group_by" validate:"omitempty,oneof=employee zone day"` // Aggregation
}
