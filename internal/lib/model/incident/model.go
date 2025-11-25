package incident

import (
	"srmt-admin/internal/lib/model/user"
	"time"
)

type ResponseModel struct {
	ID               int64           `json:"id"`
	IncidentTime     time.Time       `json:"incident_date"`
	Description      string          `json:"description"`
	CreatedAt        time.Time       `json:"created_at"`
	OrganizationID   *int64          `json:"organization_id,omitempty"`
	OrganizationName *string         `json:"organization,omitempty"`
	CreatedByUser    *user.ShortInfo `json:"created_by"`
}
