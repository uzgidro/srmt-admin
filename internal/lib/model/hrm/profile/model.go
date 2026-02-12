package profile

type MyProfile struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	Position      string  `json:"position"`
	Department    string  `json:"department"`
	Organization  string  `json:"organization"`
	Email         *string `json:"email,omitempty"`
	Phone         *string `json:"phone,omitempty"`
	InternalPhone *string `json:"internal_phone,omitempty"`
	HireDate      *string `json:"hire_date,omitempty"`
	Status        *string `json:"status,omitempty"`
	ContractType  *string `json:"contract_type,omitempty"`
	Avatar        *string `json:"avatar,omitempty"`
	WorkSchedule  *string `json:"work_schedule,omitempty"`
	TabNumber     *string `json:"tab_number,omitempty"`
}
