package discharge

import (
	"srmt-admin/internal/lib/model/organization"
	"srmt-admin/internal/lib/model/user"
	"time"
)

type Model struct {
	ID int64 `json:"id"`

	OrganizationID *int64 `json:"-"`
	CreatedBy      *int64 `json:"-"`
	ApprovedBy     *int64 `json:"-"`

	Organization   *organization.Model `json:"organization"`
	CreatedByUser  *user.ShortInfo     `json:"created_by"`
	ApprovedByUser *user.ShortInfo     `json:"updated_by,omitempty"`

	StartedAt   time.Time  `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at"`
	FlowRate    float64    `json:"flow_rate"`
	TotalVolume float64    `json:"total_volume"`
	Reason      string     `json:"reason"`
	IsOngoing   bool       `json:"is_ongoing"`
	Approved    *bool      `json:"approved"`
}
