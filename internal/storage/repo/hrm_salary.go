package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"srmt-admin/internal/lib/dto/hrm"
	hrmmodel "srmt-admin/internal/lib/model/hrm"
	"srmt-admin/internal/storage"
)

// --- Salary Structure Operations ---

// AddSalaryStructure creates a new salary structure
func (r *Repo) AddSalaryStructure(ctx context.Context, req hrm.AddSalaryStructureRequest) (int64, error) {
	const op = "storage.repo.AddSalaryStructure"

	const query = `
		INSERT INTO hrm_salary_structures (
			employee_id, base_salary, currency, pay_frequency,
			allowances, effective_from, effective_to, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.EmployeeID, req.BaseSalary, req.Currency, req.PayFrequency,
		req.Allowances, req.EffectiveFrom, req.EffectiveTo, req.Notes,
	).Scan(&id)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code.Name() == "foreign_key_violation" {
			return 0, storage.ErrForeignKeyViolation
		}
		return 0, fmt.Errorf("%s: failed to insert salary structure: %w", op, err)
	}

	return id, nil
}

// GetSalaryStructureByID retrieves salary structure by ID
func (r *Repo) GetSalaryStructureByID(ctx context.Context, id int64) (*hrmmodel.SalaryStructure, error) {
	const op = "storage.repo.GetSalaryStructureByID"

	const query = `
		SELECT id, employee_id, base_salary, currency, pay_frequency,
			allowances, effective_from, effective_to, notes, created_at, updated_at
		FROM hrm_salary_structures
		WHERE id = $1`

	var ss hrmmodel.SalaryStructure
	var allowances []byte
	var effectiveTo, updatedAt sql.NullTime
	var notes sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&ss.ID, &ss.EmployeeID, &ss.BaseSalary, &ss.Currency, &ss.PayFrequency,
		&allowances, &ss.EffectiveFrom, &effectiveTo, &notes, &ss.CreatedAt, &updatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get salary structure: %w", op, err)
	}

	if len(allowances) > 0 {
		ss.Allowances = json.RawMessage(allowances)
	}
	if effectiveTo.Valid {
		ss.EffectiveTo = &effectiveTo.Time
	}
	if notes.Valid {
		ss.Notes = &notes.String
	}
	if updatedAt.Valid {
		ss.UpdatedAt = &updatedAt.Time
	}

	return &ss, nil
}

// GetCurrentSalaryStructure retrieves current salary structure for employee
func (r *Repo) GetCurrentSalaryStructure(ctx context.Context, employeeID int64) (*hrmmodel.SalaryStructure, error) {
	const op = "storage.repo.GetCurrentSalaryStructure"

	const query = `
		SELECT id, employee_id, base_salary, currency, pay_frequency,
			allowances, effective_from, effective_to, notes, created_at, updated_at
		FROM hrm_salary_structures
		WHERE employee_id = $1
		AND effective_from <= CURRENT_DATE
		AND (effective_to IS NULL OR effective_to >= CURRENT_DATE)
		ORDER BY effective_from DESC
		LIMIT 1`

	var ss hrmmodel.SalaryStructure
	var allowances []byte
	var effectiveTo, updatedAt sql.NullTime
	var notes sql.NullString

	err := r.db.QueryRowContext(ctx, query, employeeID).Scan(
		&ss.ID, &ss.EmployeeID, &ss.BaseSalary, &ss.Currency, &ss.PayFrequency,
		&allowances, &ss.EffectiveFrom, &effectiveTo, &notes, &ss.CreatedAt, &updatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get salary structure: %w", op, err)
	}

	if len(allowances) > 0 {
		ss.Allowances = json.RawMessage(allowances)
	}
	if effectiveTo.Valid {
		ss.EffectiveTo = &effectiveTo.Time
	}
	if notes.Valid {
		ss.Notes = &notes.String
	}
	if updatedAt.Valid {
		ss.UpdatedAt = &updatedAt.Time
	}

	return &ss, nil
}

// GetSalaryStructures retrieves salary structures with filters
func (r *Repo) GetSalaryStructures(ctx context.Context, filter hrm.SalaryStructureFilter) ([]*hrmmodel.SalaryStructure, error) {
	const op = "storage.repo.GetSalaryStructures"

	var query strings.Builder
	query.WriteString(`
		SELECT id, employee_id, base_salary, currency, pay_frequency,
			allowances, effective_from, effective_to, notes, created_at, updated_at
		FROM hrm_salary_structures
		WHERE 1=1
	`)

	var args []interface{}
	argIdx := 1

	if filter.EmployeeID != nil {
		query.WriteString(fmt.Sprintf(" AND employee_id = $%d", argIdx))
		args = append(args, *filter.EmployeeID)
		argIdx++
	}
	if filter.ActiveOnly {
		query.WriteString(" AND effective_from <= CURRENT_DATE AND (effective_to IS NULL OR effective_to >= CURRENT_DATE)")
	}

	query.WriteString(" ORDER BY effective_from DESC")

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query salary structures: %w", op, err)
	}
	defer rows.Close()

	var structures []*hrmmodel.SalaryStructure
	for rows.Next() {
		var ss hrmmodel.SalaryStructure
		var allowances []byte
		var effectiveTo, updatedAt sql.NullTime
		var notes sql.NullString

		err := rows.Scan(
			&ss.ID, &ss.EmployeeID, &ss.BaseSalary, &ss.Currency, &ss.PayFrequency,
			&allowances, &ss.EffectiveFrom, &effectiveTo, &notes, &ss.CreatedAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan salary structure: %w", op, err)
		}

		if len(allowances) > 0 {
			ss.Allowances = json.RawMessage(allowances)
		}
		if effectiveTo.Valid {
			ss.EffectiveTo = &effectiveTo.Time
		}
		if notes.Valid {
			ss.Notes = &notes.String
		}
		if updatedAt.Valid {
			ss.UpdatedAt = &updatedAt.Time
		}

		structures = append(structures, &ss)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if structures == nil {
		structures = make([]*hrmmodel.SalaryStructure, 0)
	}

	return structures, nil
}

// EditSalaryStructure updates salary structure
func (r *Repo) EditSalaryStructure(ctx context.Context, id int64, req hrm.EditSalaryStructureRequest) error {
	const op = "storage.repo.EditSalaryStructure"

	var updates []string
	var args []interface{}
	argIdx := 1

	if req.BaseSalary != nil {
		updates = append(updates, fmt.Sprintf("base_salary = $%d", argIdx))
		args = append(args, *req.BaseSalary)
		argIdx++
	}
	if req.Currency != nil {
		updates = append(updates, fmt.Sprintf("currency = $%d", argIdx))
		args = append(args, *req.Currency)
		argIdx++
	}
	if req.PayFrequency != nil {
		updates = append(updates, fmt.Sprintf("pay_frequency = $%d", argIdx))
		args = append(args, *req.PayFrequency)
		argIdx++
	}
	if req.Allowances != nil {
		updates = append(updates, fmt.Sprintf("allowances = $%d", argIdx))
		args = append(args, req.Allowances)
		argIdx++
	}
	if req.EffectiveFrom != nil {
		updates = append(updates, fmt.Sprintf("effective_from = $%d", argIdx))
		args = append(args, *req.EffectiveFrom)
		argIdx++
	}
	if req.EffectiveTo != nil {
		updates = append(updates, fmt.Sprintf("effective_to = $%d", argIdx))
		args = append(args, *req.EffectiveTo)
		argIdx++
	}
	if req.Notes != nil {
		updates = append(updates, fmt.Sprintf("notes = $%d", argIdx))
		args = append(args, *req.Notes)
		argIdx++
	}

	if len(updates) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE hrm_salary_structures SET %s WHERE id = $%d",
		strings.Join(updates, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to update salary structure: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteSalaryStructure deletes salary structure
func (r *Repo) DeleteSalaryStructure(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteSalaryStructure"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_salary_structures WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("%s: failed to delete salary structure: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- Salary Operations ---

// AddSalary creates a new salary record
func (r *Repo) AddSalary(ctx context.Context, req hrm.AddSalaryRequest) (int64, error) {
	const op = "storage.repo.AddSalary"

	const query = `
		INSERT INTO hrm_salaries (employee_id, year, month, notes)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.EmployeeID, req.Year, req.Month, req.Notes,
	).Scan(&id)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code.Name() == "unique_violation" {
				return 0, storage.ErrDuplicate
			}
			if pqErr.Code.Name() == "foreign_key_violation" {
				return 0, storage.ErrForeignKeyViolation
			}
		}
		return 0, fmt.Errorf("%s: failed to insert salary: %w", op, err)
	}

	return id, nil
}

// GetSalaryByID retrieves salary by ID
func (r *Repo) GetSalaryByID(ctx context.Context, id int64) (*hrmmodel.Salary, error) {
	const op = "storage.repo.GetSalaryByID"

	const query = `
		SELECT id, employee_id, year, month, base_amount, allowances_amount,
			bonuses_amount, deductions_amount, gross_amount, tax_amount, net_amount,
			worked_days, total_work_days, overtime_hours, status,
			calculated_at, approved_by, approved_at, paid_at, notes,
			created_at, updated_at
		FROM hrm_salaries
		WHERE id = $1`

	return r.scanSalary(r.db.QueryRowContext(ctx, query, id))
}

// GetSalaryByPeriod retrieves salary for employee/period
func (r *Repo) GetSalaryByPeriod(ctx context.Context, employeeID int64, year, month int) (*hrmmodel.Salary, error) {
	const op = "storage.repo.GetSalaryByPeriod"

	const query = `
		SELECT id, employee_id, year, month, base_amount, allowances_amount,
			bonuses_amount, deductions_amount, gross_amount, tax_amount, net_amount,
			worked_days, total_work_days, overtime_hours, status,
			calculated_at, approved_by, approved_at, paid_at, notes,
			created_at, updated_at
		FROM hrm_salaries
		WHERE employee_id = $1 AND year = $2 AND month = $3`

	sal, err := r.scanSalary(r.db.QueryRowContext(ctx, query, employeeID, year, month))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get salary: %w", op, err)
	}

	return sal, nil
}

// GetSalaries retrieves salaries with filters
func (r *Repo) GetSalaries(ctx context.Context, filter hrm.SalaryFilter) ([]*hrmmodel.Salary, error) {
	const op = "storage.repo.GetSalaries"

	var query strings.Builder
	query.WriteString(`
		SELECT s.id, s.employee_id, s.year, s.month, s.base_amount, s.allowances_amount,
			s.bonuses_amount, s.deductions_amount, s.gross_amount, s.tax_amount, s.net_amount,
			s.worked_days, s.total_work_days, s.overtime_hours, s.status,
			s.calculated_at, s.approved_by, s.approved_at, s.paid_at, s.notes,
			s.created_at, s.updated_at
		FROM hrm_salaries s
		LEFT JOIN hrm_employees e ON s.employee_id = e.id
		LEFT JOIN contacts c ON e.contact_id = c.id
		WHERE 1=1
	`)

	var args []interface{}
	argIdx := 1

	if filter.EmployeeID != nil {
		query.WriteString(fmt.Sprintf(" AND s.employee_id = $%d", argIdx))
		args = append(args, *filter.EmployeeID)
		argIdx++
	}
	if filter.Year != nil {
		query.WriteString(fmt.Sprintf(" AND s.year = $%d", argIdx))
		args = append(args, *filter.Year)
		argIdx++
	}
	if filter.Month != nil {
		query.WriteString(fmt.Sprintf(" AND s.month = $%d", argIdx))
		args = append(args, *filter.Month)
		argIdx++
	}
	if filter.Status != nil {
		query.WriteString(fmt.Sprintf(" AND s.status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.DepartmentID != nil {
		query.WriteString(fmt.Sprintf(" AND c.department_id = $%d", argIdx))
		args = append(args, *filter.DepartmentID)
		argIdx++
	}

	query.WriteString(" ORDER BY s.year DESC, s.month DESC")

	if filter.Limit > 0 {
		query.WriteString(fmt.Sprintf(" LIMIT $%d", argIdx))
		args = append(args, filter.Limit)
		argIdx++
	}
	if filter.Offset > 0 {
		query.WriteString(fmt.Sprintf(" OFFSET $%d", argIdx))
		args = append(args, filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query salaries: %w", op, err)
	}
	defer rows.Close()

	var salaries []*hrmmodel.Salary
	for rows.Next() {
		sal, err := r.scanSalaryRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan salary: %w", op, err)
		}
		salaries = append(salaries, sal)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if salaries == nil {
		salaries = make([]*hrmmodel.Salary, 0)
	}

	return salaries, nil
}

// UpdateSalaryCalculation updates salary amounts
func (r *Repo) UpdateSalaryCalculation(ctx context.Context, id int64, baseAmount, allowancesAmount, bonusesAmount, deductionsAmount, taxAmount float64, workedDays, totalWorkDays int, overtimeHours float64) error {
	const op = "storage.repo.UpdateSalaryCalculation"

	grossAmount := baseAmount + allowancesAmount + bonusesAmount
	netAmount := grossAmount - deductionsAmount - taxAmount

	// Validate net amount is not negative
	if netAmount < 0 {
		return storage.ErrNegativeNetAmount
	}

	const query = `
		UPDATE hrm_salaries
		SET base_amount = $1, allowances_amount = $2, bonuses_amount = $3,
			deductions_amount = $4, gross_amount = $5, tax_amount = $6, net_amount = $7,
			worked_days = $8, total_work_days = $9, overtime_hours = $10,
			status = 'calculated', calculated_at = $11
		WHERE id = $12`

	res, err := r.db.ExecContext(ctx, query,
		baseAmount, allowancesAmount, bonusesAmount, deductionsAmount,
		grossAmount, taxAmount, netAmount, workedDays, totalWorkDays,
		overtimeHours, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("%s: failed to update salary calculation: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// ApproveSalary approves a salary record
func (r *Repo) ApproveSalary(ctx context.Context, id int64, approvedBy int64) error {
	const op = "storage.repo.ApproveSalary"

	const query = `
		UPDATE hrm_salaries
		SET status = 'approved', approved_by = $1, approved_at = $2
		WHERE id = $3 AND status = 'calculated'`

	res, err := r.db.ExecContext(ctx, query, approvedBy, time.Now(), id)
	if err != nil {
		return fmt.Errorf("%s: failed to approve salary: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// MarkSalaryPaid marks salary as paid
func (r *Repo) MarkSalaryPaid(ctx context.Context, id int64) error {
	const op = "storage.repo.MarkSalaryPaid"

	const query = `
		UPDATE hrm_salaries
		SET status = 'paid', paid_at = $1
		WHERE id = $2 AND status = 'approved'`

	res, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("%s: failed to mark salary paid: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteSalary deletes salary record
func (r *Repo) DeleteSalary(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteSalary"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_salaries WHERE id = $1 AND status = 'draft'", id)
	if err != nil {
		return fmt.Errorf("%s: failed to delete salary: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// salaryScanner interface for sql.Row and sql.Rows compatibility
type salaryScanner interface {
	Scan(dest ...interface{}) error
}

// scanSalaryFromScanner scans salary data from a scanner interface
func (r *Repo) scanSalaryFromScanner(s salaryScanner) (*hrmmodel.Salary, error) {
	var sal hrmmodel.Salary
	var calculatedAt, approvedAt, paidAt, updatedAt sql.NullTime
	var approvedBy sql.NullInt64
	var notes sql.NullString

	err := s.Scan(
		&sal.ID, &sal.EmployeeID, &sal.Year, &sal.Month,
		&sal.BaseAmount, &sal.AllowancesAmount, &sal.BonusesAmount,
		&sal.DeductionsAmount, &sal.GrossAmount, &sal.TaxAmount, &sal.NetAmount,
		&sal.WorkedDays, &sal.TotalWorkDays, &sal.OvertimeHours, &sal.Status,
		&calculatedAt, &approvedBy, &approvedAt, &paidAt, &notes,
		&sal.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if calculatedAt.Valid {
		sal.CalculatedAt = &calculatedAt.Time
	}
	if approvedBy.Valid {
		sal.ApprovedBy = &approvedBy.Int64
	}
	if approvedAt.Valid {
		sal.ApprovedAt = &approvedAt.Time
	}
	if paidAt.Valid {
		sal.PaidAt = &paidAt.Time
	}
	if notes.Valid {
		sal.Notes = &notes.String
	}
	if updatedAt.Valid {
		sal.UpdatedAt = &updatedAt.Time
	}

	return &sal, nil
}

// scanSalary scans salary from sql.Row
func (r *Repo) scanSalary(row *sql.Row) (*hrmmodel.Salary, error) {
	return r.scanSalaryFromScanner(row)
}

// scanSalaryRow scans salary from sql.Rows
func (r *Repo) scanSalaryRow(rows *sql.Rows) (*hrmmodel.Salary, error) {
	return r.scanSalaryFromScanner(rows)
}

// --- Bonus Operations ---

// AddBonus creates a new bonus
func (r *Repo) AddBonus(ctx context.Context, req hrm.AddBonusRequest) (int64, error) {
	const op = "storage.repo.AddBonus"

	const query = `
		INSERT INTO hrm_salary_bonuses (
			salary_id, employee_id, bonus_type, amount, description, year, month
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.SalaryID, req.EmployeeID, req.BonusType, req.Amount,
		req.Description, req.Year, req.Month,
	).Scan(&id)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code.Name() == "foreign_key_violation" {
			return 0, storage.ErrForeignKeyViolation
		}
		return 0, fmt.Errorf("%s: failed to insert bonus: %w", op, err)
	}

	return id, nil
}

// GetBonuses retrieves bonuses with filters
func (r *Repo) GetBonuses(ctx context.Context, filter hrm.BonusFilter) ([]*hrmmodel.SalaryBonus, error) {
	const op = "storage.repo.GetBonuses"

	var query strings.Builder
	query.WriteString(`
		SELECT id, salary_id, employee_id, bonus_type, amount, description,
			year, month, approved_by, approved_at, created_at, updated_at
		FROM hrm_salary_bonuses
		WHERE 1=1
	`)

	var args []interface{}
	argIdx := 1

	if filter.EmployeeID != nil {
		query.WriteString(fmt.Sprintf(" AND employee_id = $%d", argIdx))
		args = append(args, *filter.EmployeeID)
		argIdx++
	}
	if filter.SalaryID != nil {
		query.WriteString(fmt.Sprintf(" AND salary_id = $%d", argIdx))
		args = append(args, *filter.SalaryID)
		argIdx++
	}
	if filter.Year != nil {
		query.WriteString(fmt.Sprintf(" AND year = $%d", argIdx))
		args = append(args, *filter.Year)
		argIdx++
	}
	if filter.Month != nil {
		query.WriteString(fmt.Sprintf(" AND month = $%d", argIdx))
		args = append(args, *filter.Month)
		argIdx++
	}

	query.WriteString(" ORDER BY created_at DESC")

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query bonuses: %w", op, err)
	}
	defer rows.Close()

	var bonuses []*hrmmodel.SalaryBonus
	for rows.Next() {
		var b hrmmodel.SalaryBonus
		var salaryID, approvedBy sql.NullInt64
		var description sql.NullString
		var year, month sql.NullInt32
		var approvedAt, updatedAt sql.NullTime

		err := rows.Scan(
			&b.ID, &salaryID, &b.EmployeeID, &b.BonusType, &b.Amount, &description,
			&year, &month, &approvedBy, &approvedAt, &b.CreatedAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan bonus: %w", op, err)
		}

		if salaryID.Valid {
			b.SalaryID = &salaryID.Int64
		}
		if description.Valid {
			b.Description = &description.String
		}
		if year.Valid {
			y := int(year.Int32)
			b.Year = &y
		}
		if month.Valid {
			m := int(month.Int32)
			b.Month = &m
		}
		if approvedBy.Valid {
			b.ApprovedBy = &approvedBy.Int64
		}
		if approvedAt.Valid {
			b.ApprovedAt = &approvedAt.Time
		}
		if updatedAt.Valid {
			b.UpdatedAt = &updatedAt.Time
		}

		bonuses = append(bonuses, &b)
	}

	if bonuses == nil {
		bonuses = make([]*hrmmodel.SalaryBonus, 0)
	}

	return bonuses, nil
}

// ApproveBonus approves a bonus
func (r *Repo) ApproveBonus(ctx context.Context, id int64, approvedBy int64) error {
	const op = "storage.repo.ApproveBonus"

	const query = `
		UPDATE hrm_salary_bonuses
		SET approved_by = $1, approved_at = $2
		WHERE id = $3`

	res, err := r.db.ExecContext(ctx, query, approvedBy, time.Now(), id)
	if err != nil {
		return fmt.Errorf("%s: failed to approve bonus: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteBonus deletes a bonus
func (r *Repo) DeleteBonus(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteBonus"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_salary_bonuses WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("%s: failed to delete bonus: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- Deduction Operations ---

// AddDeduction creates a new deduction
func (r *Repo) AddDeduction(ctx context.Context, req hrm.AddDeductionRequest) (int64, error) {
	const op = "storage.repo.AddDeduction"

	const query = `
		INSERT INTO hrm_salary_deductions (
			salary_id, employee_id, deduction_type, amount, description,
			year, month, is_recurring, recurring_until
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.SalaryID, req.EmployeeID, req.DeductionType, req.Amount,
		req.Description, req.Year, req.Month, req.IsRecurring, req.RecurringUntil,
	).Scan(&id)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code.Name() == "foreign_key_violation" {
			return 0, storage.ErrForeignKeyViolation
		}
		return 0, fmt.Errorf("%s: failed to insert deduction: %w", op, err)
	}

	return id, nil
}

// GetDeductions retrieves deductions with filters
func (r *Repo) GetDeductions(ctx context.Context, filter hrm.DeductionFilter) ([]*hrmmodel.SalaryDeduction, error) {
	const op = "storage.repo.GetDeductions"

	var query strings.Builder
	query.WriteString(`
		SELECT id, salary_id, employee_id, deduction_type, amount, description,
			year, month, is_recurring, recurring_until, created_at, updated_at
		FROM hrm_salary_deductions
		WHERE 1=1
	`)

	var args []interface{}
	argIdx := 1

	if filter.EmployeeID != nil {
		query.WriteString(fmt.Sprintf(" AND employee_id = $%d", argIdx))
		args = append(args, *filter.EmployeeID)
		argIdx++
	}
	if filter.SalaryID != nil {
		query.WriteString(fmt.Sprintf(" AND salary_id = $%d", argIdx))
		args = append(args, *filter.SalaryID)
		argIdx++
	}
	if filter.Year != nil {
		query.WriteString(fmt.Sprintf(" AND year = $%d", argIdx))
		args = append(args, *filter.Year)
		argIdx++
	}
	if filter.IsRecurring != nil {
		query.WriteString(fmt.Sprintf(" AND is_recurring = $%d", argIdx))
		args = append(args, *filter.IsRecurring)
		argIdx++
	}

	query.WriteString(" ORDER BY created_at DESC")

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query deductions: %w", op, err)
	}
	defer rows.Close()

	var deductions []*hrmmodel.SalaryDeduction
	for rows.Next() {
		var d hrmmodel.SalaryDeduction
		var salaryID sql.NullInt64
		var description sql.NullString
		var year, month sql.NullInt32
		var recurringUntil, updatedAt sql.NullTime

		err := rows.Scan(
			&d.ID, &salaryID, &d.EmployeeID, &d.DeductionType, &d.Amount, &description,
			&year, &month, &d.IsRecurring, &recurringUntil, &d.CreatedAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan deduction: %w", op, err)
		}

		if salaryID.Valid {
			d.SalaryID = &salaryID.Int64
		}
		if description.Valid {
			d.Description = &description.String
		}
		if year.Valid {
			y := int(year.Int32)
			d.Year = &y
		}
		if month.Valid {
			m := int(month.Int32)
			d.Month = &m
		}
		if recurringUntil.Valid {
			d.RecurringUntil = &recurringUntil.Time
		}
		if updatedAt.Valid {
			d.UpdatedAt = &updatedAt.Time
		}

		deductions = append(deductions, &d)
	}

	if deductions == nil {
		deductions = make([]*hrmmodel.SalaryDeduction, 0)
	}

	return deductions, nil
}

// DeleteDeduction deletes a deduction
func (r *Repo) DeleteDeduction(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteDeduction"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_salary_deductions WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("%s: failed to delete deduction: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// GetSalarySummary retrieves salary summary for period
func (r *Repo) GetSalarySummary(ctx context.Context, year, month int) (*hrmmodel.SalarySummary, error) {
	const op = "storage.repo.GetSalarySummary"

	const query = `
		SELECT
			$1 as year, $2 as month,
			COALESCE(SUM(gross_amount), 0) as total_gross,
			COALESCE(SUM(net_amount), 0) as total_net,
			COALESCE(SUM(tax_amount), 0) as total_tax,
			COALESCE(SUM(bonuses_amount), 0) as total_bonuses,
			COUNT(*) as employee_count
		FROM hrm_salaries
		WHERE year = $1 AND month = $2`

	var summary hrmmodel.SalarySummary
	err := r.db.QueryRowContext(ctx, query, year, month).Scan(
		&summary.Year, &summary.Month,
		&summary.TotalGross, &summary.TotalNet, &summary.TotalTax,
		&summary.TotalBonuses, &summary.EmployeeCount,
	)

	if err != nil {
		return nil, fmt.Errorf("%s: failed to get salary summary: %w", op, err)
	}

	return &summary, nil
}
