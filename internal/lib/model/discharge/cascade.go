package discharge

type HPP struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	Discharges []Model `json:"discharges"`
}

type Cascade struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	HPPs []HPP  `json:"hpps"`
}
