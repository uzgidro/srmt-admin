package reservoirdata

// ReservoirDataItem represents a single reservoir data record
type ReservoirDataItem struct {
	OrganizationID int64   `json:"organization_id" validate:"required"`
	Date           string  `json:"date" validate:"required"`
	Income         float64 `json:"income"`
	Level          float64 `json:"level"`
	Release        float64 `json:"release"`
	Volume         float64 `json:"volume"`
}
