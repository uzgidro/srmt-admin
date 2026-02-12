package dto

type CreatePersonnelRecordRequest struct {
	EmployeeID      int64   `json:"employee_id" validate:"required"`
	TabNumber       string  `json:"tab_number" validate:"required"`
	HireDate        string  `json:"hire_date" validate:"required"`
	DepartmentID    int64   `json:"department_id" validate:"required"`
	PositionID      int64   `json:"position_id" validate:"required"`
	ContractType    string  `json:"contract_type" validate:"required,oneof=permanent temporary contract"`
	ContractEndDate *string `json:"contract_end_date,omitempty"`
	Status          string  `json:"status" validate:"required,oneof=active on_leave dismissed"`
}

type EditPersonnelRecordRequest struct {
	TabNumber       *string `json:"tab_number,omitempty"`
	HireDate        *string `json:"hire_date,omitempty"`
	DepartmentID    *int64  `json:"department_id,omitempty"`
	PositionID      *int64  `json:"position_id,omitempty"`
	ContractType    *string `json:"contract_type,omitempty" validate:"omitempty,oneof=permanent temporary contract"`
	ContractEndDate *string `json:"contract_end_date,omitempty"`
	Status          *string `json:"status,omitempty" validate:"omitempty,oneof=active on_leave dismissed"`
}

type PersonnelRecordFilters struct {
	DepartmentID *int64
	PositionID   *int64
	Status       *string
	Search       *string
}
