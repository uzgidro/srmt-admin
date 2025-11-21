package dto

import "time"

type AddVisitRequest struct {
	OrganizationID  int64
	VisitDate       time.Time
	Description     string
	ResponsibleName string
	CreatedByUserID int64
}

type EditVisitRequest struct {
	OrganizationID  *int64
	VisitDate       *time.Time
	Description     *string
	ResponsibleName *string
}
