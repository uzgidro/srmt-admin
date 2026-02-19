package dto

import "time"

type AddVisitRequest struct {
	OrganizationID  int64     `json:"organization_id" validate:"required"`
	VisitDate       time.Time `json:"visit_date" validate:"required"`
	Description     string    `json:"description" validate:"required"`
	ResponsibleName string    `json:"responsible_name" validate:"required"`
	CreatedByUserID int64     `json:"-"`
	FileIDs         []int64   `json:"file_ids,omitempty"`
}

type EditVisitRequest struct {
	OrganizationID  *int64     `json:"organization_id,omitempty"`
	VisitDate       *time.Time `json:"visit_date,omitempty"`
	Description     *string    `json:"description,omitempty"`
	ResponsibleName *string    `json:"responsible_name,omitempty"`
	FileIDs         []int64    `json:"file_ids,omitempty"`
}
