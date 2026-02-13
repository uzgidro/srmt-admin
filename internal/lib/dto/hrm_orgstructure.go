package dto

// --- Org Structure ---

type CreateOrgUnitRequest struct {
	Name         string `json:"name" validate:"required"`
	Type         string `json:"type" validate:"required,oneof=company branch division department section group team"`
	ParentID     *int64 `json:"parent_id,omitempty"`
	HeadID       *int64 `json:"head_id,omitempty"`
	DepartmentID *int64 `json:"department_id,omitempty"`
}

type UpdateOrgUnitRequest struct {
	Name         *string `json:"name,omitempty"`
	Type         *string `json:"type,omitempty" validate:"omitempty,oneof=company branch division department section group team"`
	ParentID     *int64  `json:"parent_id,omitempty"`
	HeadID       *int64  `json:"head_id,omitempty"`
	DepartmentID *int64  `json:"department_id,omitempty"`
}
