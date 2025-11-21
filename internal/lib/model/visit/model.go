package visit

import "time"

type ResponseModel struct {
	ID               int64     `json:"id"`
	OrganizationID   int64     `json:"organization_id"`
	OrganizationName string    `json:"organization_name"`
	VisitDate        time.Time `json:"visit_date"`
	Description      string    `json:"description"`
	ResponsibleName  string    `json:"responsible_name"`
	CreatedAt        time.Time `json:"created_at"`
	CreatedByUserID  int64     `json:"created_by_user_id"`
	CreatedByUserFIO string    `json:"created_by_user"`
}
