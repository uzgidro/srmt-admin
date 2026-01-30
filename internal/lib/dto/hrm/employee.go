package hrm

import "time"

// --- Employee DTOs ---

// AddEmployeeRequest represents request to create an employee
type AddEmployeeRequest struct {
	ContactID        int64      `json:"contact_id" validate:"required"`
	UserID           *int64     `json:"user_id,omitempty"`
	EmployeeNumber   *string    `json:"employee_number,omitempty"`
	HireDate         time.Time  `json:"hire_date" validate:"required"`
	EmploymentType   string     `json:"employment_type" validate:"required,oneof=full_time part_time contract intern"`
	EmploymentStatus string     `json:"employment_status" validate:"omitempty,oneof=active on_leave suspended terminated"`
	WorkSchedule     *string    `json:"work_schedule,omitempty"`
	WorkHoursPerWeek *float64   `json:"work_hours_per_week,omitempty"`
	ManagerID        *int64     `json:"manager_id,omitempty"`
	ProbationEndDate *time.Time `json:"probation_end_date,omitempty"`
	Notes            *string    `json:"notes,omitempty"`
}

// EditEmployeeRequest represents request to update an employee
type EditEmployeeRequest struct {
	ContactID        *int64     `json:"contact_id,omitempty"`
	UserID           *int64     `json:"user_id,omitempty"`
	EmployeeNumber   *string    `json:"employee_number,omitempty"`
	HireDate         *time.Time `json:"hire_date,omitempty"`
	TerminationDate  *time.Time `json:"termination_date,omitempty"`
	EmploymentType   *string    `json:"employment_type,omitempty" validate:"omitempty,oneof=full_time part_time contract intern"`
	EmploymentStatus *string    `json:"employment_status,omitempty" validate:"omitempty,oneof=active on_leave suspended terminated"`
	WorkSchedule     *string    `json:"work_schedule,omitempty"`
	WorkHoursPerWeek *float64   `json:"work_hours_per_week,omitempty"`
	ManagerID        *int64     `json:"manager_id,omitempty"`
	ProbationEndDate *time.Time `json:"probation_end_date,omitempty"`
	ProbationPassed  *bool      `json:"probation_passed,omitempty"`
	Notes            *string    `json:"notes,omitempty"`
}

// EmployeeFilter represents filter for employee list
type EmployeeFilter struct {
	OrganizationID   *int64  `json:"organization_id,omitempty"`
	DepartmentID     *int64  `json:"department_id,omitempty"`
	PositionID       *int64  `json:"position_id,omitempty"`
	ManagerID        *int64  `json:"manager_id,omitempty"`
	EmploymentType   *string `json:"employment_type,omitempty"`
	EmploymentStatus *string `json:"employment_status,omitempty"`
	Search           *string `json:"search,omitempty"` // Name, email, employee_number
	Limit            int     `json:"limit,omitempty"`
	Offset           int     `json:"offset,omitempty"`
}

// --- Personnel Document DTOs ---

// AddPersonnelDocumentRequest represents request to add personnel document
type AddPersonnelDocumentRequest struct {
	EmployeeID     int64      `json:"employee_id" validate:"required"`
	DocumentType   string     `json:"document_type" validate:"required"`
	DocumentNumber *string    `json:"document_number,omitempty"`
	DocumentSeries *string    `json:"document_series,omitempty"`
	IssuedBy       *string    `json:"issued_by,omitempty"`
	IssuedDate     *time.Time `json:"issued_date,omitempty"`
	ExpiryDate     *time.Time `json:"expiry_date,omitempty"`
	FileID         *int64     `json:"file_id,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
}

// EditPersonnelDocumentRequest represents request to edit personnel document
type EditPersonnelDocumentRequest struct {
	DocumentType   *string    `json:"document_type,omitempty"`
	DocumentNumber *string    `json:"document_number,omitempty"`
	DocumentSeries *string    `json:"document_series,omitempty"`
	IssuedBy       *string    `json:"issued_by,omitempty"`
	IssuedDate     *time.Time `json:"issued_date,omitempty"`
	ExpiryDate     *time.Time `json:"expiry_date,omitempty"`
	FileID         *int64     `json:"file_id,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
	IsVerified     *bool      `json:"is_verified,omitempty"`
}

// PersonnelDocumentFilter represents filter for personnel documents
type PersonnelDocumentFilter struct {
	EmployeeID   *int64  `json:"employee_id,omitempty"`
	DocumentType *string `json:"document_type,omitempty"`
	IsVerified   *bool   `json:"is_verified,omitempty"`
	ExpiringDays *int    `json:"expiring_days,omitempty"` // Expiring within N days
}

// --- Transfer DTOs ---

// AddTransferRequest represents request to add transfer record
type AddTransferRequest struct {
	EmployeeID         int64      `json:"employee_id" validate:"required"`
	FromDepartmentID   *int64     `json:"from_department_id,omitempty"`
	FromPositionID     *int64     `json:"from_position_id,omitempty"`
	FromOrganizationID *int64     `json:"from_organization_id,omitempty"`
	ToDepartmentID     *int64     `json:"to_department_id,omitempty"`
	ToPositionID       *int64     `json:"to_position_id,omitempty"`
	ToOrganizationID   *int64     `json:"to_organization_id,omitempty"`
	TransferType       string     `json:"transfer_type" validate:"required,oneof=promotion demotion lateral relocation"`
	TransferReason     *string    `json:"transfer_reason,omitempty"`
	EffectiveDate      time.Time  `json:"effective_date" validate:"required"`
	OrderNumber        *string    `json:"order_number,omitempty"`
	OrderDate          *time.Time `json:"order_date,omitempty"`
	OrderFileID        *int64     `json:"order_file_id,omitempty"`
}

// TransferFilter represents filter for transfers
type TransferFilter struct {
	EmployeeID     *int64     `json:"employee_id,omitempty"`
	TransferType   *string    `json:"transfer_type,omitempty"`
	FromDate       *time.Time `json:"from_date,omitempty"`
	ToDate         *time.Time `json:"to_date,omitempty"`
	DepartmentID   *int64     `json:"department_id,omitempty"`
	OrganizationID *int64     `json:"organization_id,omitempty"`
}
