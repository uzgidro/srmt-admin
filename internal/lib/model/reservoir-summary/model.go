package reservoirsummary

// ResponseModel represents reservoir summary data for a single organization or summary row
type ResponseModel struct {
	// OrganizationID is nil for the summary row (ИТОГО)
	OrganizationID   *int64 `json:"organization_id"`
	OrganizationName string `json:"organization_name"`

	// Level data (meters) - current and previous day only
	LevelCurrent float64 `json:"level_current"`
	LevelPrev    float64 `json:"level_prev"`

	// Volume data (million m³) - 4 data points
	VolumeCurrent     float64 `json:"volume_current"`
	VolumePrev        float64 `json:"volume_prev"`
	VolumeYearAgo     float64 `json:"volume_year_ago"`
	VolumeTwoYearsAgo float64 `json:"volume_two_years_ago"`

	// Income data (m³/s) - 4 data points
	IncomeCurrent     float64 `json:"income_current"`
	IncomePrev        float64 `json:"income_prev"`
	IncomeYearAgo     float64 `json:"income_year_ago"`
	IncomeTwoYearsAgo float64 `json:"income_two_years_ago"`

	// Release data (m³/s) - 4 data points
	ReleaseCurrent     float64 `json:"release_current"`
	ReleasePrev        float64 `json:"release_prev"`
	ReleaseYearAgo     float64 `json:"release_year_ago"`
	ReleaseTwoYearsAgo float64 `json:"release_two_years_ago"`

	// Incoming volume (million m³)
	IncomingVolumeMlnM3         float64 `json:"incoming_volume_mln_m3"`
	IncomingVolumeMlnM3PrevYear float64 `json:"incoming_volume_mln_m3_prev_year"`
}
