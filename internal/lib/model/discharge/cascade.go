package discharge

type HPP struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	TotalVolume float64 `json:"total_volume"`
	Discharges  []Model `json:"discharges"`
}

type Cascade struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	TotalVolume float64 `json:"total_volume"`
	HPPs        []HPP   `json:"hpps"`
}

// HPPWithURLs is the API response model with presigned file URLs for nested discharges
type HPPWithURLs struct {
	ID          int64           `json:"id"`
	Name        string          `json:"name"`
	TotalVolume float64         `json:"total_volume"`
	Discharges  []ModelWithURLs `json:"discharges"`
}

// CascadeWithURLs is the API response model with presigned file URLs for nested structures
type CascadeWithURLs struct {
	ID          int64         `json:"id"`
	Name        string        `json:"name"`
	TotalVolume float64       `json:"total_volume"`
	HPPs        []HPPWithURLs `json:"hpps"`
}
