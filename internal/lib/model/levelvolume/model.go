package levelvolume

import "time"

type Model struct {
	ID             int64      `json:"id"`
	Level          float64    `json:"level"`
	Volume         float64    `json:"volume"`
	OrganizationID int64      `json:"organization_id"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
}
