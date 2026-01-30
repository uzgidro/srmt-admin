package hrm

import (
	"encoding/json"
	"time"
)

// SalaryStructure represents employee salary configuration
type SalaryStructure struct {
	ID         int64 `json:"id"`
	EmployeeID int64 `json:"employee_id"`

	BaseSalary   float64 `json:"base_salary"`
	Currency     string  `json:"currency"`
	PayFrequency string  `json:"pay_frequency"`

	// Allowances as JSON array: [{type: "transport", amount: 5000}, ...]
	Allowances json.RawMessage `json:"allowances,omitempty"`

	EffectiveFrom time.Time  `json:"effective_from"`
	EffectiveTo   *time.Time `json:"effective_to,omitempty"`

	Notes *string `json:"notes,omitempty"`

	// Enriched
	Employee *Employee `json:"employee,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// Allowance represents a salary allowance entry
type Allowance struct {
	Type   string  `json:"type"`
	Amount float64 `json:"amount"`
}

// PayFrequency constants
const (
	PayFrequencyMonthly  = "monthly"
	PayFrequencyBiWeekly = "bi_weekly"
	PayFrequencyWeekly   = "weekly"
)

// Currency constants
const (
	CurrencyRUB = "RUB"
	CurrencyUSD = "USD"
	CurrencyEUR = "EUR"
)

// Salary represents monthly payroll record
type Salary struct {
	ID         int64 `json:"id"`
	EmployeeID int64 `json:"employee_id"`

	Year  int `json:"year"`
	Month int `json:"month"`

	// Amounts
	BaseAmount       float64 `json:"base_amount"`
	AllowancesAmount float64 `json:"allowances_amount"`
	BonusesAmount    float64 `json:"bonuses_amount"`
	DeductionsAmount float64 `json:"deductions_amount"`
	GrossAmount      float64 `json:"gross_amount"`
	TaxAmount        float64 `json:"tax_amount"`
	NetAmount        float64 `json:"net_amount"`

	// Work time
	WorkedDays    int     `json:"worked_days"`
	TotalWorkDays int     `json:"total_work_days"`
	OvertimeHours float64 `json:"overtime_hours"`

	// Status
	Status       string     `json:"status"`
	CalculatedAt *time.Time `json:"calculated_at,omitempty"`
	ApprovedBy   *int64     `json:"approved_by,omitempty"`
	ApprovedAt   *time.Time `json:"approved_at,omitempty"`
	PaidAt       *time.Time `json:"paid_at,omitempty"`

	Notes *string `json:"notes,omitempty"`

	// Enriched
	Employee     *Employee         `json:"employee,omitempty"`
	Bonuses      []SalaryBonus     `json:"bonuses,omitempty"`
	Deductions   []SalaryDeduction `json:"deductions,omitempty"`
	ApproverName *string           `json:"approver_name,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// SalaryStatus constants
const (
	SalaryStatusDraft      = "draft"
	SalaryStatusCalculated = "calculated"
	SalaryStatusApproved   = "approved"
	SalaryStatusPaid       = "paid"
)

// SalaryBonus represents a bonus payment
type SalaryBonus struct {
	ID         int64  `json:"id"`
	SalaryID   *int64 `json:"salary_id,omitempty"`
	EmployeeID int64  `json:"employee_id"`

	BonusType   string  `json:"bonus_type"`
	Amount      float64 `json:"amount"`
	Description *string `json:"description,omitempty"`

	Year  *int `json:"year,omitempty"`
	Month *int `json:"month,omitempty"`

	ApprovedBy *int64     `json:"approved_by,omitempty"`
	ApprovedAt *time.Time `json:"approved_at,omitempty"`

	// Enriched
	Employee     *Employee `json:"employee,omitempty"`
	ApproverName *string   `json:"approver_name,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// BonusType constants
const (
	BonusTypePerformance = "performance"
	BonusTypeProject     = "project"
	BonusTypeAnnual      = "annual"
	BonusTypeReferral    = "referral"
	BonusTypeSignOn      = "sign_on"
	BonusTypeRetention   = "retention"
)

// SalaryDeduction represents a deduction from salary
type SalaryDeduction struct {
	ID         int64  `json:"id"`
	SalaryID   *int64 `json:"salary_id,omitempty"`
	EmployeeID int64  `json:"employee_id"`

	DeductionType string  `json:"deduction_type"`
	Amount        float64 `json:"amount"`
	Description   *string `json:"description,omitempty"`

	Year  *int `json:"year,omitempty"`
	Month *int `json:"month,omitempty"`

	IsRecurring    bool       `json:"is_recurring"`
	RecurringUntil *time.Time `json:"recurring_until,omitempty"`

	// Enriched
	Employee *Employee `json:"employee,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// DeductionType constants
const (
	DeductionTypeTax       = "tax"
	DeductionTypeInsurance = "insurance"
	DeductionTypeLoan      = "loan"
	DeductionTypePenalty   = "penalty"
	DeductionTypeAbsence   = "absence"
	DeductionTypeAdvance   = "advance"
)

// SalarySummary represents aggregated salary data
type SalarySummary struct {
	Year          int     `json:"year"`
	Month         int     `json:"month"`
	TotalGross    float64 `json:"total_gross"`
	TotalNet      float64 `json:"total_net"`
	TotalTax      float64 `json:"total_tax"`
	TotalBonuses  float64 `json:"total_bonuses"`
	EmployeeCount int     `json:"employee_count"`
}
