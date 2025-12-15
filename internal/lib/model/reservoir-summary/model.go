package reservoirsummary

type ValueResponse struct {
	Current     float64 `json:"current"`
	Previous    float64 `json:"prev"`
	YearAgo     float64 `json:"year_ago"`
	TwoYearsAgo float64 `json:"two_years_ago"`
}

type ResponseModel struct {
	OrganizationID   *int64 `json:"organization_id"`
	OrganizationName string `json:"organization_name"`

	Income  ValueResponse `json:"income"`
	Volume  ValueResponse `json:"volume"`
	Level   ValueResponse `json:"level"`
	Release ValueResponse `json:"release"`
	Modsnow ValueResponse `json:"modsnow"`

	IncomingVolume         float64 `json:"incoming_volume"`
	IncomingVolumePrevYear float64 `json:"incoming_volume_prev_year"`
}
