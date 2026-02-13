package access

import (
	"encoding/json"
	"time"
)

type AccessCard struct {
	ID           int64           `json:"id"`
	EmployeeID   int64           `json:"employee_id"`
	EmployeeName string          `json:"employee_name"`
	CardNumber   string          `json:"card_number"`
	Status       string          `json:"status"`
	IssuedDate   string          `json:"issued_date"`
	ExpiryDate   string          `json:"expiry_date"`
	AccessZones  json.RawMessage `json:"access_zones"`
	AccessLevel  *string         `json:"access_level,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type AccessZone struct {
	ID               int64           `json:"id"`
	Name             string          `json:"name"`
	Description      *string         `json:"description,omitempty"`
	SecurityLevel    string          `json:"security_level"`
	Building         *string         `json:"building,omitempty"`
	Floor            *string         `json:"floor,omitempty"`
	MaxOccupancy     int             `json:"max_occupancy"`
	CurrentOccupancy int             `json:"current_occupancy"`
	Readers          json.RawMessage `json:"readers"`
	Schedules        json.RawMessage `json:"schedules"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type AccessLog struct {
	ID           int64     `json:"id"`
	EmployeeID   int64     `json:"employee_id"`
	EmployeeName string    `json:"employee_name"`
	CardNumber   string    `json:"card_number"`
	ZoneID       *int64    `json:"zone_id,omitempty"`
	ZoneName     string    `json:"zone_name"`
	ReaderID     *int      `json:"reader_id,omitempty"`
	Direction    string    `json:"direction"`
	Timestamp    time.Time `json:"timestamp"`
	Status       string    `json:"status"`
	DenialReason *string   `json:"denial_reason,omitempty"`
}

type AccessRequest struct {
	ID              int64     `json:"id"`
	EmployeeID      int64     `json:"employee_id"`
	EmployeeName    string    `json:"employee_name"`
	ZoneID          *int64    `json:"zone_id,omitempty"`
	ZoneName        *string   `json:"zone_name,omitempty"`
	Reason          string    `json:"reason"`
	Status          string    `json:"status"`
	ApprovedBy      *int64    `json:"approved_by,omitempty"`
	RejectionReason *string   `json:"rejection_reason,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
