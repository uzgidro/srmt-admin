package shutdown

import (
	"srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/model/user"
	"time"
)

type ResponseModel struct {
	ID                int64           `json:"id"`
	OrganizationID    int64           `json:"organization_id"`
	OrganizationName  string          `json:"organization_name"`
	StartedAt         time.Time       `json:"started_at"`
	EndedAt           *time.Time      `json:"ended_at,omitempty"`
	Reason            *string         `json:"reason,omitempty"`
	CreatedByUser     *user.ShortInfo `json:"created_by"`
	GenerationLossMwh *float64        `json:"generation_loss,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`

	IdleDischargeVolumeThousandM3 *float64     `json:"idle_discharge_volume,omitempty"`
	Files                         []file.Model `json:"files,omitempty"`
}

type GroupedResponse struct {
	Ges   []*ResponseModel `json:"ges"`
	Mini  []*ResponseModel `json:"mini"`
	Micro []*ResponseModel `json:"micro"`
	Other []*ResponseModel `json:"other,omitempty"`
}
