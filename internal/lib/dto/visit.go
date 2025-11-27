package dto

import "time"

type AddVisitRequest struct {
	OrganizationID  int64
	VisitDate       time.Time
	Description     string
	ResponsibleName string
	CreatedByUserID int64
	FileIDs         []int64 `json:"file_ids,omitempty"`
}

type EditVisitRequest struct {
	OrganizationID  *int64
	VisitDate       *time.Time
	Description     *string
	ResponsibleName *string
	FileIDs         []int64 `json:"file_ids,omitempty"`
}
