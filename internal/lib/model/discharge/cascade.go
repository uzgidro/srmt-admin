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
