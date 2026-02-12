package profile

type Manager struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Position string `json:"position"`
}

type MyProfile struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	Position      string   `json:"position"`
	Department    string   `json:"department"`
	Organization  string   `json:"organization"`
	Email         *string  `json:"email,omitempty"`
	Phone         *string  `json:"phone,omitempty"`
	InternalPhone *string  `json:"internal_phone,omitempty"`
	HireDate      *string  `json:"hire_date,omitempty"`
	Status        *string  `json:"status,omitempty"`
	ContractType  *string  `json:"contract_type,omitempty"`
	Avatar        *string  `json:"avatar,omitempty"`
	Manager       *Manager `json:"manager,omitempty"`
	WorkSchedule  *string  `json:"work_schedule,omitempty"`
	TabNumber     *string  `json:"tab_number,omitempty"`
}

type LeaveCategory struct {
	Total     int `json:"total"`
	Used      int `json:"used"`
	Remaining int `json:"remaining"`
}

type LeaveBalance struct {
	AnnualLeave     LeaveCategory `json:"annual_leave"`
	AdditionalLeave LeaveCategory `json:"additional_leave"`
	StudyLeave      LeaveCategory `json:"study_leave"`
	SickLeave       LeaveCategory `json:"sick_leave"`
	CompDays        LeaveCategory `json:"comp_days"`
}

type MyVacation struct {
	ID              int64   `json:"id"`
	Type            string  `json:"type"`
	StartDate       string  `json:"start_date"`
	EndDate         string  `json:"end_date"`
	Days            int     `json:"days"`
	Status          string  `json:"status"`
	Reason          *string `json:"reason,omitempty"`
	ApprovedBy      *string `json:"approved_by,omitempty"`
	ApprovedAt      *string `json:"approved_at,omitempty"`
	RejectionReason *string `json:"rejection_reason,omitempty"`
	CreatedAt       string  `json:"created_at"`
}

type MyDocument struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	Date string `json:"date"`
	Size *int64 `json:"size,omitempty"`
}
