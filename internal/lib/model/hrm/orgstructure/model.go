package orgstructure

import "time"

type OrgUnit struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	ParentID      *int64    `json:"parent_id,omitempty"`
	HeadID        *int64    `json:"head_id,omitempty"`
	HeadName      *string   `json:"head_name,omitempty"`
	DepartmentID  *int64    `json:"department_id,omitempty"`
	EmployeeCount int       `json:"employee_count"`
	Level         int       `json:"level"`
	Children      []OrgUnit `json:"children"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type OrgEmployee struct {
	ID                int64   `json:"id"`
	Name              string  `json:"name"`
	Position          string  `json:"position"`
	Department        string  `json:"department"`
	UnitID            *int64  `json:"unit_id,omitempty"`
	IsHead            bool    `json:"is_head"`
	ManagerID         *int64  `json:"manager_id,omitempty"`
	ManagerName       *string `json:"manager_name,omitempty"`
	SubordinatesCount int     `json:"subordinates_count"`
	Avatar            *string `json:"avatar,omitempty"`
	Phone             *string `json:"phone,omitempty"`
	Email             *string `json:"email,omitempty"`
}
