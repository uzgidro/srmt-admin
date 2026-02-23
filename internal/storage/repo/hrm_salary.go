package repo

import (
	"context"
	"database/sql"
	"fmt"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/hrm/salary"
	"srmt-admin/internal/storage"
	"strings"
)

// --- Salary CRUD ---

func (r *Repo) CreateSalary(ctx context.Context, req dto.CreateSalaryRequest) (int64, error) {
	const op = "repo.CreateSalary"

	query := `INSERT INTO salaries (employee_id, period_month, period_year) VALUES ($1, $2, $3) RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query, req.EmployeeID, req.PeriodMonth, req.PeriodYear).Scan(&id)
	if err != nil {
		if translated := r.translator.Translate(err, op); translated != nil {
			if translated == storage.ErrUniqueViolation {
				return 0, storage.ErrSalaryAlreadyExists
			}
			return 0, translated
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

func (r *Repo) GetSalaryByID(ctx context.Context, id int64) (*salary.Salary, error) {
	const op = "repo.GetSalaryByID"

	query := `
		SELECT s.id, s.employee_id, COALESCE(c.fio, ''), COALESCE(d.name, ''), COALESCE(p.name, ''),
			   s.period_month, s.period_year,
			   s.base_salary, s.regional_allowance, s.seniority_allowance,
			   s.qualification_allowance, s.hazard_allowance, s.night_shift_allowance,
			   s.overtime_amount, s.bonus_amount, s.gross_salary,
			   s.ndfl, s.social_tax, s.pension_fund, s.health_insurance, s.trade_union,
			   s.total_deductions, s.net_salary,
			   s.work_days, s.actual_days, s.overtime_hours,
			   s.status, s.calculated_at, s.approved_by, s.approved_at, s.paid_at,
			   s.created_at, s.updated_at
		FROM salaries s
		JOIN contacts c ON s.employee_id = c.id
		LEFT JOIN personnel_records pr ON pr.employee_id = s.employee_id AND pr.status = 'active'
		LEFT JOIN departments d ON pr.department_id = d.id
		LEFT JOIN positions p ON pr.position_id = p.id
		WHERE s.id = $1`

	sal, err := scanSalary(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrSalaryNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return sal, nil
}

func (r *Repo) GetAllSalaries(ctx context.Context, filters dto.SalaryFilters) ([]*salary.Salary, error) {
	const op = "repo.GetAllSalaries"

	query := `
		SELECT s.id, s.employee_id, COALESCE(c.fio, ''), COALESCE(d.name, ''), COALESCE(p.name, ''),
			   s.period_month, s.period_year,
			   s.base_salary, s.regional_allowance, s.seniority_allowance,
			   s.qualification_allowance, s.hazard_allowance, s.night_shift_allowance,
			   s.overtime_amount, s.bonus_amount, s.gross_salary,
			   s.ndfl, s.social_tax, s.pension_fund, s.health_insurance, s.trade_union,
			   s.total_deductions, s.net_salary,
			   s.work_days, s.actual_days, s.overtime_hours,
			   s.status, s.calculated_at, s.approved_by, s.approved_at, s.paid_at,
			   s.created_at, s.updated_at
		FROM salaries s
		JOIN contacts c ON s.employee_id = c.id
		LEFT JOIN personnel_records pr ON pr.employee_id = s.employee_id AND pr.status = 'active'
		LEFT JOIN departments d ON pr.department_id = d.id
		LEFT JOIN positions p ON pr.position_id = p.id`

	var conditions []string
	var args []interface{}
	argIdx := 1

	if filters.EmployeeID != nil {
		conditions = append(conditions, fmt.Sprintf("s.employee_id = $%d", argIdx))
		args = append(args, *filters.EmployeeID)
		argIdx++
	}
	if filters.PeriodYear != nil {
		conditions = append(conditions, fmt.Sprintf("s.period_year = $%d", argIdx))
		args = append(args, *filters.PeriodYear)
		argIdx++
	}
	if filters.PeriodMonth != nil {
		conditions = append(conditions, fmt.Sprintf("s.period_month = $%d", argIdx))
		args = append(args, *filters.PeriodMonth)
		argIdx++
	}
	if filters.DepartmentID != nil {
		conditions = append(conditions, fmt.Sprintf("pr.department_id = $%d", argIdx))
		args = append(args, *filters.DepartmentID)
		argIdx++
	}
	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("s.status = $%d", argIdx))
		args = append(args, *filters.Status)
		argIdx++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY s.created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var salaries []*salary.Salary
	for rows.Next() {
		sal, err := scanSalary(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		salaries = append(salaries, sal)
	}
	return salaries, rows.Err()
}

func (r *Repo) UpdateSalary(ctx context.Context, id int64, req dto.UpdateSalaryRequest) error {
	const op = "repo.UpdateSalary"

	var setClauses []string
	var args []interface{}
	argIdx := 1

	if req.WorkDays != nil {
		setClauses = append(setClauses, fmt.Sprintf("work_days = $%d", argIdx))
		args = append(args, *req.WorkDays)
		argIdx++
	}
	if req.ActualDays != nil {
		setClauses = append(setClauses, fmt.Sprintf("actual_days = $%d", argIdx))
		args = append(args, *req.ActualDays)
		argIdx++
	}
	if req.OvertimeHours != nil {
		setClauses = append(setClauses, fmt.Sprintf("overtime_hours = $%d", argIdx))
		args = append(args, *req.OvertimeHours)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE salaries SET %s WHERE id = $%d AND status = 'draft'",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrSalaryNotFound
	}
	return nil
}

func (r *Repo) DeleteSalary(ctx context.Context, id int64) error {
	const op = "repo.DeleteSalary"

	result, err := r.db.ExecContext(ctx, "DELETE FROM salaries WHERE id = $1 AND status = 'draft'", id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrSalaryNotFound
	}
	return nil
}

// --- Status Machine ---

func (r *Repo) UpdateSalaryCalculation(ctx context.Context, sal *salary.Salary) error {
	const op = "repo.UpdateSalaryCalculation"

	query := `
		UPDATE salaries SET
			base_salary = $1, regional_allowance = $2, seniority_allowance = $3,
			qualification_allowance = $4, hazard_allowance = $5, night_shift_allowance = $6,
			overtime_amount = $7, bonus_amount = $8, gross_salary = $9,
			ndfl = $10, social_tax = $11, pension_fund = $12, health_insurance = $13, trade_union = $14,
			total_deductions = $15, net_salary = $16,
			work_days = $17, actual_days = $18, overtime_hours = $19,
			status = 'calculated', calculated_at = NOW()
		WHERE id = $20`

	result, err := r.db.ExecContext(ctx, query,
		sal.BaseSalary, sal.RegionalAllowance, sal.SeniorityAllowance,
		sal.QualificationAllow, sal.HazardAllowance, sal.NightShiftAllowance,
		sal.OvertimeAmount, sal.BonusAmount, sal.GrossSalary,
		sal.NDFL, sal.SocialTax, sal.PensionFund, sal.HealthInsurance, sal.TradeUnion,
		sal.TotalDeductions, sal.NetSalary,
		sal.WorkDays, sal.ActualDays, sal.OvertimeHours,
		sal.ID,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrSalaryNotFound
	}
	return nil
}

func (r *Repo) ApproveSalary(ctx context.Context, id int64, approvedBy int64) error {
	const op = "repo.ApproveSalary"

	query := `UPDATE salaries SET status = 'approved', approved_by = $1, approved_at = NOW() WHERE id = $2 AND status = 'calculated'`
	result, err := r.db.ExecContext(ctx, query, approvedBy, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrSalaryNotFound
	}
	return nil
}

func (r *Repo) MarkSalaryPaid(ctx context.Context, id int64) error {
	const op = "repo.MarkSalaryPaid"

	query := `UPDATE salaries SET status = 'paid', paid_at = NOW() WHERE id = $1 AND status = 'approved'`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrSalaryNotFound
	}
	return nil
}

// --- Salary Structures ---

func (r *Repo) GetActiveSalaryStructure(ctx context.Context, employeeID int64, forDate string) (*salary.SalaryStructure, error) {
	const op = "repo.GetActiveSalaryStructure"

	query := `
		SELECT id, employee_id, base_salary, regional_allowance, seniority_allowance,
			   qualification_allowance, hazard_allowance, night_shift_allowance,
			   effective_from::text, effective_to::text, created_at, updated_at
		FROM salary_structures
		WHERE employee_id = $1 AND effective_from <= $2::date
		  AND (effective_to IS NULL OR effective_to >= $2::date)
		ORDER BY effective_from DESC
		LIMIT 1`

	ss, err := scanSalaryStructure(r.db.QueryRowContext(ctx, query, employeeID, forDate))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrSalaryStructureNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return ss, nil
}

func (r *Repo) GetSalaryStructureByEmployee(ctx context.Context, employeeID int64) ([]*salary.SalaryStructure, error) {
	const op = "repo.GetSalaryStructureByEmployee"

	query := `
		SELECT id, employee_id, base_salary, regional_allowance, seniority_allowance,
			   qualification_allowance, hazard_allowance, night_shift_allowance,
			   effective_from::text, effective_to::text, created_at, updated_at
		FROM salary_structures
		WHERE employee_id = $1
		ORDER BY effective_from DESC`

	rows, err := r.db.QueryContext(ctx, query, employeeID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var structures []*salary.SalaryStructure
	for rows.Next() {
		ss, err := scanSalaryStructure(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		structures = append(structures, ss)
	}
	return structures, rows.Err()
}

// --- Bonuses & Deductions ---

func (r *Repo) CreateBonuses(ctx context.Context, salaryID int64, bonuses []dto.BonusInput) error {
	const op = "repo.CreateBonuses"

	if len(bonuses) == 0 {
		return nil
	}

	query := `INSERT INTO salary_bonuses (salary_id, bonus_type, amount, description) VALUES `
	var args []interface{}
	argIdx := 1
	var placeholders []string

	for _, b := range bonuses {
		placeholders = append(placeholders, fmt.Sprintf("($%d, $%d, $%d, $%d)", argIdx, argIdx+1, argIdx+2, argIdx+3))
		args = append(args, salaryID, b.Type, b.Amount, b.Description)
		argIdx += 4
	}
	query += strings.Join(placeholders, ", ")

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *Repo) CreateDeductions(ctx context.Context, salaryID int64, deductions []dto.DeductionInput) error {
	const op = "repo.CreateDeductions"

	if len(deductions) == 0 {
		return nil
	}

	query := `INSERT INTO salary_deductions (salary_id, deduction_type, amount, description) VALUES `
	var args []interface{}
	argIdx := 1
	var placeholders []string

	for _, d := range deductions {
		placeholders = append(placeholders, fmt.Sprintf("($%d, $%d, $%d, $%d)", argIdx, argIdx+1, argIdx+2, argIdx+3))
		args = append(args, salaryID, d.Type, d.Amount, d.Description)
		argIdx += 4
	}
	query += strings.Join(placeholders, ", ")

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *Repo) GetBonuses(ctx context.Context, salaryID int64) ([]*salary.Bonus, error) {
	const op = "repo.GetBonuses"

	query := `SELECT id, salary_id, bonus_type, amount, description, created_at FROM salary_bonuses WHERE salary_id = $1 ORDER BY id`

	rows, err := r.db.QueryContext(ctx, query, salaryID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var bonuses []*salary.Bonus
	for rows.Next() {
		b, err := scanBonus(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		bonuses = append(bonuses, b)
	}
	return bonuses, rows.Err()
}

func (r *Repo) GetDeductions(ctx context.Context, salaryID int64) ([]*salary.Deduction, error) {
	const op = "repo.GetDeductions"

	query := `SELECT id, salary_id, deduction_type, amount, description, created_at FROM salary_deductions WHERE salary_id = $1 ORDER BY id`

	rows, err := r.db.QueryContext(ctx, query, salaryID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var deductions []*salary.Deduction
	for rows.Next() {
		d, err := scanDeduction(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		deductions = append(deductions, d)
	}
	return deductions, rows.Err()
}

// --- Helpers ---

func (r *Repo) GetActiveEmployeesByDepartment(ctx context.Context, departmentID *int64) ([]int64, error) {
	const op = "repo.GetActiveEmployeesByDepartment"

	query := `SELECT employee_id FROM personnel_records WHERE status = 'active'`
	var args []interface{}

	if departmentID != nil {
		query += " AND department_id = $1"
		args = append(args, *departmentID)
	}
	query += " ORDER BY employee_id"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *Repo) SalaryExists(ctx context.Context, employeeID int64, year, month int) (bool, error) {
	const op = "repo.SalaryExists"

	var exists bool
	err := r.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM salaries WHERE employee_id = $1 AND period_year = $2 AND period_month = $3)",
		employeeID, year, month,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	return exists, nil
}

// --- Collection Getters ---

func (r *Repo) GetAllSalaryStructures(ctx context.Context) ([]*salary.SalaryStructure, error) {
	const op = "repo.GetAllSalaryStructures"

	query := `
		SELECT ss.id, ss.employee_id,
			   ss.base_salary, ss.regional_allowance, ss.seniority_allowance,
			   ss.qualification_allowance, ss.hazard_allowance, ss.night_shift_allowance,
			   ss.effective_from::text, ss.effective_to::text, ss.created_at, ss.updated_at
		FROM salary_structures ss
		ORDER BY ss.effective_from DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var structures []*salary.SalaryStructure
	for rows.Next() {
		ss, err := scanSalaryStructure(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		structures = append(structures, ss)
	}
	return structures, rows.Err()
}

func (r *Repo) GetAllBonuses(ctx context.Context) ([]*salary.Bonus, error) {
	const op = "repo.GetAllBonuses"

	query := `SELECT id, salary_id, bonus_type, amount, description, created_at FROM salary_bonuses ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var bonuses []*salary.Bonus
	for rows.Next() {
		b, err := scanBonus(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		bonuses = append(bonuses, b)
	}
	return bonuses, rows.Err()
}

func (r *Repo) GetAllDeductions(ctx context.Context) ([]*salary.Deduction, error) {
	const op = "repo.GetAllDeductions"

	query := `SELECT id, salary_id, deduction_type, amount, description, created_at FROM salary_deductions ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var deductions []*salary.Deduction
	for rows.Next() {
		d, err := scanDeduction(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		deductions = append(deductions, d)
	}
	return deductions, rows.Err()
}

// --- Scan Helpers ---

func scanSalary(scanner interface {
	Scan(dest ...interface{}) error
}) (*salary.Salary, error) {
	var s salary.Salary
	var (
		calculatedAt sql.NullString
		approvedBy   sql.NullInt64
		approvedAt   sql.NullString
		paidAt       sql.NullString
	)

	err := scanner.Scan(
		&s.ID, &s.EmployeeID, &s.EmployeeName, &s.Department, &s.Position,
		&s.PeriodMonth, &s.PeriodYear,
		&s.BaseSalary, &s.RegionalAllowance, &s.SeniorityAllowance,
		&s.QualificationAllow, &s.HazardAllowance, &s.NightShiftAllowance,
		&s.OvertimeAmount, &s.BonusAmount, &s.GrossSalary,
		&s.NDFL, &s.SocialTax, &s.PensionFund, &s.HealthInsurance, &s.TradeUnion,
		&s.TotalDeductions, &s.NetSalary,
		&s.WorkDays, &s.ActualDays, &s.OvertimeHours,
		&s.Status, &calculatedAt, &approvedBy, &approvedAt, &paidAt,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if calculatedAt.Valid {
		s.CalculatedAt = &calculatedAt.String
	}
	if approvedBy.Valid {
		s.ApprovedBy = &approvedBy.Int64
	}
	if approvedAt.Valid {
		s.ApprovedAt = &approvedAt.String
	}
	if paidAt.Valid {
		s.PaidAt = &paidAt.String
	}

	return &s, nil
}

func scanSalaryStructure(scanner interface {
	Scan(dest ...interface{}) error
}) (*salary.SalaryStructure, error) {
	var ss salary.SalaryStructure
	var effectiveTo sql.NullString

	err := scanner.Scan(
		&ss.ID, &ss.EmployeeID,
		&ss.BaseSalary, &ss.RegionalAllowance, &ss.SeniorityAllowance,
		&ss.QualificationAllow, &ss.HazardAllowance, &ss.NightShiftAllowance,
		&ss.EffectiveFrom, &effectiveTo,
		&ss.CreatedAt, &ss.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if effectiveTo.Valid {
		ss.EffectiveTo = &effectiveTo.String
	}

	return &ss, nil
}

func scanBonus(scanner interface {
	Scan(dest ...interface{}) error
}) (*salary.Bonus, error) {
	var b salary.Bonus
	var desc sql.NullString

	err := scanner.Scan(&b.ID, &b.SalaryID, &b.BonusType, &b.Amount, &desc, &b.CreatedAt)
	if err != nil {
		return nil, err
	}

	if desc.Valid {
		b.Description = &desc.String
	}
	return &b, nil
}

func scanDeduction(scanner interface {
	Scan(dest ...interface{}) error
}) (*salary.Deduction, error) {
	var d salary.Deduction
	var desc sql.NullString

	err := scanner.Scan(&d.ID, &d.SalaryID, &d.DeductionType, &d.Amount, &desc, &d.CreatedAt)
	if err != nil {
		return nil, err
	}

	if desc.Valid {
		d.Description = &desc.String
	}
	return &d, nil
}
