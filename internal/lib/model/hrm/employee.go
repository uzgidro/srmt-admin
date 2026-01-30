package hrm

import (
	"time"

	"srmt-admin/internal/lib/model/contact"
	"srmt-admin/internal/lib/model/department"
	"srmt-admin/internal/lib/model/organization"
	"srmt-admin/internal/lib/model/position"
	"srmt-admin/internal/lib/model/user"
)

// Employee represents an extended employee record linked to contact and user
type Employee struct {
	ID        int64  `json:"id"`
	ContactID int64  `json:"contact_id"`
	UserID    *int64 `json:"user_id,omitempty"`

	// Employment info
	EmployeeNumber   *string    `json:"employee_number,omitempty"`
	HireDate         time.Time  `json:"hire_date"`
	TerminationDate  *time.Time `json:"termination_date,omitempty"`
	EmploymentType   string     `json:"employment_type"`
	EmploymentStatus string     `json:"employment_status"`

	// Work schedule
	WorkSchedule     *string  `json:"work_schedule,omitempty"`
	WorkHoursPerWeek *float64 `json:"work_hours_per_week,omitempty"`

	// Manager hierarchy
	ManagerID *int64 `json:"manager_id,omitempty"`

	// Probation
	ProbationEndDate *time.Time `json:"probation_end_date,omitempty"`
	ProbationPassed  bool       `json:"probation_passed"`

	Notes *string `json:"notes,omitempty"`

	// Nested models (enriched)
	Contact      *contact.Model      `json:"contact,omitempty"`
	User         *user.Model         `json:"user,omitempty"`
	Manager      *Employee           `json:"manager,omitempty"`
	Organization *organization.Model `json:"organization,omitempty"`
	Department   *department.Model   `json:"department,omitempty"`
	Position     *position.Model     `json:"position,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// EmploymentType constants
const (
	EmploymentTypeFullTime = "full_time"
	EmploymentTypePartTime = "part_time"
	EmploymentTypeContract = "contract"
	EmploymentTypeIntern   = "intern"
)

// EmploymentStatus constants
const (
	EmploymentStatusActive     = "active"
	EmploymentStatusOnLeave    = "on_leave"
	EmploymentStatusSuspended  = "suspended"
	EmploymentStatusTerminated = "terminated"
)

// WorkSchedule constants
const (
	WorkSchedule52       = "5/2"
	WorkSchedule22       = "2/2"
	WorkScheduleShift    = "shift"
	WorkScheduleFlexible = "flexible"
)

// PersonnelDocument represents employee documents (passport, diplomas, etc.)
type PersonnelDocument struct {
	ID         int64 `json:"id"`
	EmployeeID int64 `json:"employee_id"`

	DocumentType   string  `json:"document_type"`
	DocumentNumber *string `json:"document_number,omitempty"`
	DocumentSeries *string `json:"document_series,omitempty"`

	IssuedBy   *string    `json:"issued_by,omitempty"`
	IssuedDate *time.Time `json:"issued_date,omitempty"`
	ExpiryDate *time.Time `json:"expiry_date,omitempty"`

	FileID *int64 `json:"file_id,omitempty"`

	Notes      *string    `json:"notes,omitempty"`
	IsVerified bool       `json:"is_verified"`
	VerifiedBy *int64     `json:"verified_by,omitempty"`
	VerifiedAt *time.Time `json:"verified_at,omitempty"`

	// File info (enriched)
	FileURL *string `json:"file_url,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// PersonnelDocumentType constants
const (
	PersonnelDocTypePassport    = "passport"
	PersonnelDocTypeDiploma     = "diploma"
	PersonnelDocTypeCertificate = "certificate"
	PersonnelDocTypeContract    = "contract"
	PersonnelDocTypeMilitaryID  = "military_id"
	PersonnelDocTypeSNILS       = "snils"
	PersonnelDocTypeINN         = "inn"
)

// Transfer represents employee position/department changes
type Transfer struct {
	ID         int64 `json:"id"`
	EmployeeID int64 `json:"employee_id"`

	// From
	FromDepartmentID   *int64 `json:"from_department_id,omitempty"`
	FromPositionID     *int64 `json:"from_position_id,omitempty"`
	FromOrganizationID *int64 `json:"from_organization_id,omitempty"`

	// To
	ToDepartmentID   *int64 `json:"to_department_id,omitempty"`
	ToPositionID     *int64 `json:"to_position_id,omitempty"`
	ToOrganizationID *int64 `json:"to_organization_id,omitempty"`

	TransferType   string    `json:"transfer_type"`
	TransferReason *string   `json:"transfer_reason,omitempty"`
	EffectiveDate  time.Time `json:"effective_date"`

	OrderNumber *string    `json:"order_number,omitempty"`
	OrderDate   *time.Time `json:"order_date,omitempty"`
	OrderFileID *int64     `json:"order_file_id,omitempty"`

	ApprovedBy *int64     `json:"approved_by,omitempty"`
	ApprovedAt *time.Time `json:"approved_at,omitempty"`

	// Enriched
	FromDepartment   *department.Model   `json:"from_department,omitempty"`
	FromPosition     *position.Model     `json:"from_position,omitempty"`
	FromOrganization *organization.Model `json:"from_organization,omitempty"`
	ToDepartment     *department.Model   `json:"to_department,omitempty"`
	ToPosition       *position.Model     `json:"to_position,omitempty"`
	ToOrganization   *organization.Model `json:"to_organization,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// TransferType constants
const (
	TransferTypePromotion  = "promotion"
	TransferTypeDemotion   = "demotion"
	TransferTypeLateral    = "lateral"
	TransferTypeRelocation = "relocation"
)
