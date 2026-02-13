package salary

import "time"

type Salary struct {
	ID                  int64     `json:"id"`
	EmployeeID          int64     `json:"employee_id"`
	EmployeeName        string    `json:"employee_name"`
	Department          string    `json:"department"`
	Position            string    `json:"position"`
	PeriodMonth         int       `json:"period_month"`
	PeriodYear          int       `json:"period_year"`
	BaseSalary          float64   `json:"base_salary"`
	RegionalAllowance   float64   `json:"regional_allowance"`
	SeniorityAllowance  float64   `json:"seniority_allowance"`
	QualificationAllow  float64   `json:"qualification_allowance"`
	HazardAllowance     float64   `json:"hazard_allowance"`
	NightShiftAllowance float64   `json:"night_shift_allowance"`
	OvertimeAmount      float64   `json:"overtime_amount"`
	BonusAmount         float64   `json:"bonus_amount"`
	GrossSalary         float64   `json:"gross_salary"`
	NDFL                float64   `json:"ndfl"`
	SocialTax           float64   `json:"social_tax"`
	PensionFund         float64   `json:"pension_fund"`
	HealthInsurance     float64   `json:"health_insurance"`
	TradeUnion          float64   `json:"trade_union"`
	TotalDeductions     float64   `json:"total_deductions"`
	NetSalary           float64   `json:"net_salary"`
	WorkDays            int       `json:"work_days"`
	ActualDays          int       `json:"actual_days"`
	OvertimeHours       float64   `json:"overtime_hours"`
	Status              string    `json:"status"`
	CalculatedAt        *string   `json:"calculated_at,omitempty"`
	ApprovedBy          *int64    `json:"approved_by,omitempty"`
	ApprovedAt          *string   `json:"approved_at,omitempty"`
	PaidAt              *string   `json:"paid_at,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type SalaryStructure struct {
	ID                  int64     `json:"id"`
	EmployeeID          int64     `json:"employee_id"`
	BaseSalary          float64   `json:"base_salary"`
	RegionalAllowance   float64   `json:"regional_allowance"`
	SeniorityAllowance  float64   `json:"seniority_allowance"`
	QualificationAllow  float64   `json:"qualification_allowance"`
	HazardAllowance     float64   `json:"hazard_allowance"`
	NightShiftAllowance float64   `json:"night_shift_allowance"`
	EffectiveFrom       string    `json:"effective_from"`
	EffectiveTo         *string   `json:"effective_to,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type Bonus struct {
	ID          int64     `json:"id"`
	SalaryID    int64     `json:"salary_id"`
	BonusType   string    `json:"bonus_type"`
	Amount      float64   `json:"amount"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type Deduction struct {
	ID            int64     `json:"id"`
	SalaryID      int64     `json:"salary_id"`
	DeductionType string    `json:"deduction_type"`
	Amount        float64   `json:"amount"`
	Description   *string   `json:"description,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type TaxRates struct {
	NDFL       float64
	Social     float64
	Pension    float64
	Health     float64
	TradeUnion float64
}

var DefaultTaxRates = TaxRates{
	NDFL:       0.12,
	Social:     0.005,
	Pension:    0.03,
	Health:     0.005,
	TradeUnion: 0.01,
}
