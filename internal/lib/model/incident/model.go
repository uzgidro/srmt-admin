package incident

import "time"

type ResponseModel struct {
	ID               int64     `json:"id"`
	IncidentTime     time.Time `json:"incident_date"`
	Description      string    `json:"description"`
	CreatedAt        time.Time `json:"created_at"`
	OrganizationID   int64     `json:"organization_id"`
	OrganizationName string    `json:"organization"`
	CreatedByUserID  int64     `json:"created_by_user_id"`
	CreatedByUserFIO string    `json:"created_by_user"`
}
