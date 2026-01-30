package hrm

import (
	"encoding/json"
	"time"
)

// AccessZone represents a physical access zone
type AccessZone struct {
	ID   int     `json:"id"`
	Name string  `json:"name"`
	Code *string `json:"code,omitempty"`

	Description *string `json:"description,omitempty"`

	// Location
	Building *string `json:"building,omitempty"`
	Floor    *string `json:"floor,omitempty"`

	// Security
	SecurityLevel int `json:"security_level"` // 1-5

	// Schedule
	AccessSchedule json.RawMessage `json:"access_schedule,omitempty"`

	IsActive bool `json:"is_active"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// AccessSchedule represents access schedule for a zone
type AccessSchedule struct {
	Monday    *TimeRange `json:"mon,omitempty"`
	Tuesday   *TimeRange `json:"tue,omitempty"`
	Wednesday *TimeRange `json:"wed,omitempty"`
	Thursday  *TimeRange `json:"thu,omitempty"`
	Friday    *TimeRange `json:"fri,omitempty"`
	Saturday  *TimeRange `json:"sat,omitempty"`
	Sunday    *TimeRange `json:"sun,omitempty"`
}

// TimeRange represents start/end time
type TimeRange struct {
	Start string `json:"start"` // HH:MM
	End   string `json:"end"`   // HH:MM
}

// AccessCard represents employee access card
type AccessCard struct {
	ID         int64 `json:"id"`
	EmployeeID int64 `json:"employee_id"`

	// Card info
	CardNumber string `json:"card_number"`
	CardType   string `json:"card_type"`

	// Validity
	IssuedDate time.Time  `json:"issued_date"`
	ExpiryDate *time.Time `json:"expiry_date,omitempty"`
	IsActive   bool       `json:"is_active"`

	// Deactivation
	DeactivatedAt      *time.Time `json:"deactivated_at,omitempty"`
	DeactivationReason *string    `json:"deactivation_reason,omitempty"`

	Notes *string `json:"notes,omitempty"`

	// Enriched
	Employee   *Employee        `json:"employee,omitempty"`
	ZoneAccess []CardZoneAccess `json:"zone_access,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// CardType constants
const (
	CardTypeStandard   = "standard"
	CardTypeTemporary  = "temporary"
	CardTypeVisitor    = "visitor"
	CardTypeContractor = "contractor"
)

// CardZoneAccess represents card-zone permissions
type CardZoneAccess struct {
	ID     int64 `json:"id"`
	CardID int64 `json:"card_id"`
	ZoneID int   `json:"zone_id"`

	// Custom schedule
	CustomSchedule json.RawMessage `json:"custom_schedule,omitempty"`

	// Validity
	ValidFrom  time.Time  `json:"valid_from"`
	ValidUntil *time.Time `json:"valid_until,omitempty"`

	GrantedBy *int64    `json:"granted_by,omitempty"`
	GrantedAt time.Time `json:"granted_at"`

	// Enriched
	Zone        *AccessZone `json:"zone,omitempty"`
	GranterName *string     `json:"granter_name,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// AccessLog represents access event log
type AccessLog struct {
	ID         int64  `json:"id"`
	CardID     *int64 `json:"card_id,omitempty"`
	EmployeeID *int64 `json:"employee_id,omitempty"`
	ZoneID     int    `json:"zone_id"`

	// Event
	EventType string    `json:"event_type"`
	EventTime time.Time `json:"event_time"`

	// Device
	DeviceID   *string `json:"device_id,omitempty"`
	DeviceName *string `json:"device_name,omitempty"`

	// Denial reason
	DenialReason *string `json:"denial_reason,omitempty"`

	// Denormalized
	CardNumber *string `json:"card_number,omitempty"`

	// Metadata
	Metadata json.RawMessage `json:"metadata,omitempty"`

	// Enriched
	Zone         *AccessZone `json:"zone,omitempty"`
	Employee     *Employee   `json:"employee,omitempty"`
	EmployeeName *string     `json:"employee_name,omitempty"`
}

// EventType constants
const (
	EventTypeEntry  = "entry"
	EventTypeExit   = "exit"
	EventTypeDenied = "denied"
)

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

// AccessStats represents access control metrics
type AccessStats struct {
	TotalActiveCards     int `json:"total_active_cards"`
	TotalZones           int `json:"total_zones"`
	TodayEntries         int `json:"today_entries"`
	TodayExits           int `json:"today_exits"`
	TodayDenied          int `json:"today_denied"`
	CurrentlyInsideCount int `json:"currently_inside_count"`
}

// EmployeeAccessSummary represents employee's current access info
type EmployeeAccessSummary struct {
	EmployeeID        int64        `json:"employee_id"`
	Employee          *Employee    `json:"employee,omitempty"`
	ActiveCard        *AccessCard  `json:"active_card,omitempty"`
	AccessZones       []AccessZone `json:"access_zones,omitempty"`
	LastEntry         *AccessLog   `json:"last_entry,omitempty"`
	LastExit          *AccessLog   `json:"last_exit,omitempty"`
	IsCurrentlyInside bool         `json:"is_currently_inside"`
}
