package dto

import "encoding/json"

// --- Access Cards ---

type CreateAccessCardRequest struct {
	EmployeeID  int64           `json:"employee_id" validate:"required"`
	CardNumber  string          `json:"card_number" validate:"required"`
	IssuedDate  string          `json:"issued_date" validate:"required"`
	ExpiryDate  string          `json:"expiry_date" validate:"required"`
	AccessZones json.RawMessage `json:"access_zones,omitempty"`
	AccessLevel *string         `json:"access_level,omitempty"`
}

type UpdateAccessCardRequest struct {
	CardNumber  *string          `json:"card_number,omitempty"`
	IssuedDate  *string          `json:"issued_date,omitempty"`
	ExpiryDate  *string          `json:"expiry_date,omitempty"`
	AccessZones *json.RawMessage `json:"access_zones,omitempty"`
	AccessLevel *string          `json:"access_level,omitempty"`
}

type AccessCardFilters struct {
	EmployeeID *int64
	Status     *string
	Search     *string
}

// --- Access Zones ---

type CreateAccessZoneRequest struct {
	Name          string          `json:"name" validate:"required"`
	Description   *string         `json:"description,omitempty"`
	SecurityLevel string          `json:"security_level" validate:"required,oneof=low medium high restricted"`
	Building      *string         `json:"building,omitempty"`
	Floor         *string         `json:"floor,omitempty"`
	MaxOccupancy  *int            `json:"max_occupancy,omitempty"`
	Readers       json.RawMessage `json:"readers,omitempty"`
	Schedules     json.RawMessage `json:"schedules,omitempty"`
}

type UpdateAccessZoneRequest struct {
	Name          *string          `json:"name,omitempty"`
	Description   *string          `json:"description,omitempty"`
	SecurityLevel *string          `json:"security_level,omitempty" validate:"omitempty,oneof=low medium high restricted"`
	Building      *string          `json:"building,omitempty"`
	Floor         *string          `json:"floor,omitempty"`
	MaxOccupancy  *int             `json:"max_occupancy,omitempty"`
	Readers       *json.RawMessage `json:"readers,omitempty"`
	Schedules     *json.RawMessage `json:"schedules,omitempty"`
}

// --- Access Logs ---

type AccessLogFilters struct {
	EmployeeID *int64
	ZoneID     *int64
	Direction  *string
	Status     *string
	DateFrom   *string
	DateTo     *string
}

// --- Access Requests ---

type CreateAccessRequestReq struct {
	ZoneID int64  `json:"zone_id" validate:"required"`
	Reason string `json:"reason" validate:"required"`
}

type RejectAccessRequestReq struct {
	Reason string `json:"reason" validate:"required"`
}
