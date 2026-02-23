package salary

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/hrm/salary"
	"srmt-admin/internal/storage"
	"time"
)

type RepoInterface interface {
	// Salary CRUD
	CreateSalary(ctx context.Context, req dto.CreateSalaryRequest) (int64, error)
	GetSalaryByID(ctx context.Context, id int64) (*salary.Salary, error)
	GetAllSalaries(ctx context.Context, filters dto.SalaryFilters) ([]*salary.Salary, error)
	UpdateSalary(ctx context.Context, id int64, req dto.UpdateSalaryRequest) error
	DeleteSalary(ctx context.Context, id int64) error

	// Status machine
	UpdateSalaryCalculation(ctx context.Context, sal *salary.Salary) error
	ApproveSalary(ctx context.Context, id int64, approvedBy int64) error
	MarkSalaryPaid(ctx context.Context, id int64) error

	// Structures
	GetActiveSalaryStructure(ctx context.Context, employeeID int64, forDate string) (*salary.SalaryStructure, error)
	GetSalaryStructureByEmployee(ctx context.Context, employeeID int64) ([]*salary.SalaryStructure, error)
	GetAllSalaryStructures(ctx context.Context) ([]*salary.SalaryStructure, error)

	// Bonuses/Deductions
	CreateBonuses(ctx context.Context, salaryID int64, bonuses []dto.BonusInput) error
	CreateDeductions(ctx context.Context, salaryID int64, deductions []dto.DeductionInput) error
	GetBonuses(ctx context.Context, salaryID int64) ([]*salary.Bonus, error)
	GetDeductions(ctx context.Context, salaryID int64) ([]*salary.Deduction, error)
	GetAllBonuses(ctx context.Context) ([]*salary.Bonus, error)
	GetAllDeductions(ctx context.Context) ([]*salary.Deduction, error)

	// Helpers
	GetActiveEmployeesByDepartment(ctx context.Context, departmentID *int64) ([]int64, error)
	SalaryExists(ctx context.Context, employeeID int64, year, month int) (bool, error)
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) Create(ctx context.Context, req dto.CreateSalaryRequest) (int64, error) {
	return s.repo.CreateSalary(ctx, req)
}

func (s *Service) GetByID(ctx context.Context, id int64) (*salary.Salary, error) {
	return s.repo.GetSalaryByID(ctx, id)
}

func (s *Service) GetAll(ctx context.Context, filters dto.SalaryFilters) ([]*salary.Salary, error) {
	salaries, err := s.repo.GetAllSalaries(ctx, filters)
	if err != nil {
		return nil, err
	}
	if salaries == nil {
		salaries = []*salary.Salary{}
	}
	return salaries, nil
}

func (s *Service) Update(ctx context.Context, id int64, req dto.UpdateSalaryRequest) error {
	return s.repo.UpdateSalary(ctx, id, req)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	sal, err := s.repo.GetSalaryByID(ctx, id)
	if err != nil {
		return err
	}
	if sal.Status != "draft" {
		return storage.ErrInvalidStatus
	}
	return s.repo.DeleteSalary(ctx, id)
}

// Calculate performs full salary calculation for a draft salary record.
func (s *Service) Calculate(ctx context.Context, id int64, req dto.CalculateSalaryRequest) (*salary.Salary, error) {
	sal, err := s.repo.GetSalaryByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if sal.Status != "draft" {
		return nil, storage.ErrInvalidStatus
	}

	// Get salary structure effective for the period
	periodDate := fmt.Sprintf("%d-%02d-01", sal.PeriodYear, sal.PeriodMonth)
	structure, err := s.repo.GetActiveSalaryStructure(ctx, sal.EmployeeID, periodDate)
	if err != nil {
		return nil, fmt.Errorf("get salary structure: %w", err)
	}

	// Proportional ratio
	ratio := 1.0
	if req.WorkDays > 0 {
		ratio = float64(req.ActualDays) / float64(req.WorkDays)
	}

	// Proportional allowances
	sal.BaseSalary = round2(structure.BaseSalary * ratio)
	sal.RegionalAllowance = round2(structure.RegionalAllowance * ratio)
	sal.SeniorityAllowance = round2(structure.SeniorityAllowance * ratio)
	sal.QualificationAllow = round2(structure.QualificationAllow * ratio)
	sal.HazardAllowance = round2(structure.HazardAllowance * ratio)
	sal.NightShiftAllowance = round2(structure.NightShiftAllowance * ratio)

	// Overtime: hourlyRate = base_salary / (work_days * 8) * 1.5 * overtime_hours
	sal.OvertimeAmount = 0
	if req.WorkDays > 0 && req.OvertimeHours > 0 {
		hourlyRate := structure.BaseSalary / (float64(req.WorkDays) * 8)
		sal.OvertimeAmount = round2(hourlyRate * 1.5 * req.OvertimeHours)
	}

	// Bonuses
	var bonusTotal float64
	for _, b := range req.Bonuses {
		bonusTotal += b.Amount
	}
	sal.BonusAmount = round2(bonusTotal)

	// Gross salary
	sal.GrossSalary = round2(
		sal.BaseSalary + sal.RegionalAllowance + sal.SeniorityAllowance +
			sal.QualificationAllow + sal.HazardAllowance + sal.NightShiftAllowance +
			sal.OvertimeAmount + sal.BonusAmount,
	)

	// Taxes (Uzbekistan rates)
	taxes := salary.DefaultTaxRates
	sal.NDFL = round2(sal.GrossSalary * taxes.NDFL)
	sal.SocialTax = round2(sal.GrossSalary * taxes.Social)
	sal.PensionFund = round2(sal.GrossSalary * taxes.Pension)
	sal.HealthInsurance = round2(sal.GrossSalary * taxes.Health)
	sal.TradeUnion = round2(sal.GrossSalary * taxes.TradeUnion)

	taxDeductions := sal.NDFL + sal.SocialTax + sal.PensionFund + sal.HealthInsurance + sal.TradeUnion

	// Additional deductions
	var extraDeductions float64
	for _, d := range req.Deductions {
		extraDeductions += d.Amount
	}

	sal.TotalDeductions = round2(taxDeductions + extraDeductions)
	sal.NetSalary = round2(sal.GrossSalary - sal.TotalDeductions)

	sal.WorkDays = req.WorkDays
	sal.ActualDays = req.ActualDays
	sal.OvertimeHours = req.OvertimeHours

	// Save calculation
	if err := s.repo.UpdateSalaryCalculation(ctx, sal); err != nil {
		return nil, fmt.Errorf("save calculation: %w", err)
	}

	// Save bonuses and deductions
	if len(req.Bonuses) > 0 {
		if err := s.repo.CreateBonuses(ctx, id, req.Bonuses); err != nil {
			s.log.Error("failed to save bonuses", "error", err, "salary_id", id)
		}
	}
	if len(req.Deductions) > 0 {
		if err := s.repo.CreateDeductions(ctx, id, req.Deductions); err != nil {
			s.log.Error("failed to save deductions", "error", err, "salary_id", id)
		}
	}

	return sal, nil
}

// BulkCalculate creates and calculates salary for all active employees in a department.
func (s *Service) BulkCalculate(ctx context.Context, req dto.BulkCalculateRequest) (int, error) {
	employees, err := s.repo.GetActiveEmployeesByDepartment(ctx, req.DepartmentID)
	if err != nil {
		return 0, fmt.Errorf("get employees: %w", err)
	}

	workDays := calculateWorkDays(req.PeriodYear, req.PeriodMonth)
	calculated := 0

	for _, empID := range employees {
		// Skip if salary already exists
		exists, err := s.repo.SalaryExists(ctx, empID, req.PeriodYear, req.PeriodMonth)
		if err != nil {
			s.log.Error("failed to check salary existence", "error", err, "employee_id", empID)
			continue
		}
		if exists {
			continue
		}

		// Create draft
		createReq := dto.CreateSalaryRequest{
			EmployeeID:  empID,
			PeriodMonth: req.PeriodMonth,
			PeriodYear:  req.PeriodYear,
		}
		salaryID, err := s.repo.CreateSalary(ctx, createReq)
		if err != nil {
			s.log.Error("failed to create salary", "error", err, "employee_id", empID)
			continue
		}

		// Calculate with full attendance
		calcReq := dto.CalculateSalaryRequest{
			WorkDays:   workDays,
			ActualDays: workDays,
		}
		if _, err := s.Calculate(ctx, salaryID, calcReq); err != nil {
			s.log.Error("failed to calculate salary", "error", err, "employee_id", empID, "salary_id", salaryID)
			continue
		}

		calculated++
	}

	return calculated, nil
}

func (s *Service) Approve(ctx context.Context, id int64, approvedBy int64) error {
	sal, err := s.repo.GetSalaryByID(ctx, id)
	if err != nil {
		return err
	}
	if sal.Status != "calculated" {
		return storage.ErrInvalidStatus
	}
	return s.repo.ApproveSalary(ctx, id, approvedBy)
}

func (s *Service) MarkPaid(ctx context.Context, id int64) error {
	sal, err := s.repo.GetSalaryByID(ctx, id)
	if err != nil {
		return err
	}
	if sal.Status != "approved" {
		return storage.ErrInvalidStatus
	}
	return s.repo.MarkSalaryPaid(ctx, id)
}

func (s *Service) GetStructure(ctx context.Context, employeeID int64) ([]*salary.SalaryStructure, error) {
	structures, err := s.repo.GetSalaryStructureByEmployee(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if structures == nil {
		structures = []*salary.SalaryStructure{}
	}
	return structures, nil
}

func (s *Service) GetBonuses(ctx context.Context, salaryID int64) ([]*salary.Bonus, error) {
	bonuses, err := s.repo.GetBonuses(ctx, salaryID)
	if err != nil {
		return nil, err
	}
	if bonuses == nil {
		bonuses = []*salary.Bonus{}
	}
	return bonuses, nil
}

func (s *Service) GetDeductions(ctx context.Context, salaryID int64) ([]*salary.Deduction, error) {
	deductions, err := s.repo.GetDeductions(ctx, salaryID)
	if err != nil {
		return nil, err
	}
	if deductions == nil {
		deductions = []*salary.Deduction{}
	}
	return deductions, nil
}

func (s *Service) GetAllStructures(ctx context.Context) ([]*salary.SalaryStructure, error) {
	structures, err := s.repo.GetAllSalaryStructures(ctx)
	if err != nil {
		return nil, err
	}
	if structures == nil {
		structures = []*salary.SalaryStructure{}
	}
	return structures, nil
}

func (s *Service) GetAllBonuses(ctx context.Context) ([]*salary.Bonus, error) {
	bonuses, err := s.repo.GetAllBonuses(ctx)
	if err != nil {
		return nil, err
	}
	if bonuses == nil {
		bonuses = []*salary.Bonus{}
	}
	return bonuses, nil
}

func (s *Service) GetAllDeductions(ctx context.Context) ([]*salary.Deduction, error) {
	deductions, err := s.repo.GetAllDeductions(ctx)
	if err != nil {
		return nil, err
	}
	if deductions == nil {
		deductions = []*salary.Deduction{}
	}
	return deductions, nil
}

// calculateWorkDays counts Mon-Sat days in a month (Uzbekistan work week: Mon-Sat, Sunday off).
func calculateWorkDays(year, month int) int {
	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, -1)
	days := 0
	for d := firstDay; !d.After(lastDay); d = d.AddDate(0, 0, 1) {
		if d.Weekday() != time.Sunday {
			days++
		}
	}
	return days
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
