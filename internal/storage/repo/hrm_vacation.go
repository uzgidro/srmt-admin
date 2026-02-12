package repo

import (
	"context"
	"database/sql"
	"fmt"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/hrm/vacation"
	"srmt-admin/internal/storage"
	"strings"
)

// --- Vacations ---

func (r *Repo) CreateVacation(ctx context.Context, req dto.CreateVacationRequest, days int, createdBy int64) (int64, error) {
	const op = "repo.CreateVacation"

	query := `
		INSERT INTO vacations (employee_id, vacation_type, start_date, end_date, days, status, reason, substitute_id, created_by)
		VALUES ($1, $2, $3, $4, $5, 'draft', $6, $7, $8)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.EmployeeID, req.VacationType, req.StartDate, req.EndDate,
		days, req.Reason, req.SubstituteID, createdBy,
	).Scan(&id)
	if err != nil {
		if translated := r.translator.Translate(err, op); translated != nil {
			return 0, translated
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

func (r *Repo) GetVacationByID(ctx context.Context, id int64) (*vacation.Vacation, error) {
	const op = "repo.GetVacationByID"

	query := `
		SELECT v.id, v.employee_id, c.fio, v.vacation_type,
			   v.start_date::text, v.end_date::text, v.days, v.status,
			   v.reason, v.rejection_reason, v.approved_by,
			   v.approved_at::text, v.substitute_id,
			   sc.fio, v.created_at, v.updated_at
		FROM vacations v
		JOIN contacts c ON v.employee_id = c.id
		LEFT JOIN contacts sc ON v.substitute_id = sc.id
		WHERE v.id = $1`

	vac, err := scanVacation(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrVacationNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return vac, nil
}

func (r *Repo) GetAllVacations(ctx context.Context, filters dto.VacationFilters) ([]*vacation.Vacation, error) {
	const op = "repo.GetAllVacations"

	query := `
		SELECT v.id, v.employee_id, c.fio, v.vacation_type,
			   v.start_date::text, v.end_date::text, v.days, v.status,
			   v.reason, v.rejection_reason, v.approved_by,
			   v.approved_at::text, v.substitute_id,
			   sc.fio, v.created_at, v.updated_at
		FROM vacations v
		JOIN contacts c ON v.employee_id = c.id
		LEFT JOIN contacts sc ON v.substitute_id = sc.id`

	var conditions []string
	var args []interface{}
	argIdx := 1

	if filters.EmployeeID != nil {
		conditions = append(conditions, fmt.Sprintf("v.employee_id = $%d", argIdx))
		args = append(args, *filters.EmployeeID)
		argIdx++
	}
	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("v.status = $%d", argIdx))
		args = append(args, *filters.Status)
		argIdx++
	}
	if filters.VacationType != nil {
		conditions = append(conditions, fmt.Sprintf("v.vacation_type = $%d", argIdx))
		args = append(args, *filters.VacationType)
		argIdx++
	}
	if filters.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("v.start_date >= $%d", argIdx))
		args = append(args, *filters.StartDate)
		argIdx++
	}
	if filters.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("v.end_date <= $%d", argIdx))
		args = append(args, *filters.EndDate)
		argIdx++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY v.created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var vacations []*vacation.Vacation
	for rows.Next() {
		vac, err := scanVacation(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		vacations = append(vacations, vac)
	}
	return vacations, rows.Err()
}

func (r *Repo) UpdateVacation(ctx context.Context, id int64, req dto.EditVacationRequest, days *int) error {
	const op = "repo.UpdateVacation"

	var setClauses []string
	var args []interface{}
	argIdx := 1

	if req.VacationType != nil {
		setClauses = append(setClauses, fmt.Sprintf("vacation_type = $%d", argIdx))
		args = append(args, *req.VacationType)
		argIdx++
	}
	if req.StartDate != nil {
		setClauses = append(setClauses, fmt.Sprintf("start_date = $%d", argIdx))
		args = append(args, *req.StartDate)
		argIdx++
	}
	if req.EndDate != nil {
		setClauses = append(setClauses, fmt.Sprintf("end_date = $%d", argIdx))
		args = append(args, *req.EndDate)
		argIdx++
	}
	if req.Reason != nil {
		setClauses = append(setClauses, fmt.Sprintf("reason = $%d", argIdx))
		args = append(args, *req.Reason)
		argIdx++
	}
	if req.SubstituteID != nil {
		setClauses = append(setClauses, fmt.Sprintf("substitute_id = $%d", argIdx))
		args = append(args, *req.SubstituteID)
		argIdx++
	}
	if days != nil {
		setClauses = append(setClauses, fmt.Sprintf("days = $%d", argIdx))
		args = append(args, *days)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE vacations SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if translated := r.translator.Translate(err, op); translated != nil {
			return translated
		}
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrVacationNotFound
	}
	return nil
}

func (r *Repo) DeleteVacation(ctx context.Context, id int64) error {
	const op = "repo.DeleteVacation"

	result, err := r.db.ExecContext(ctx, "DELETE FROM vacations WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrVacationNotFound
	}
	return nil
}

func (r *Repo) UpdateVacationStatus(ctx context.Context, id int64, status string) error {
	const op = "repo.UpdateVacationStatus"

	result, err := r.db.ExecContext(ctx, "UPDATE vacations SET status = $1 WHERE id = $2", status, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrVacationNotFound
	}
	return nil
}

func (r *Repo) ApproveVacation(ctx context.Context, id int64, approvedBy int64) error {
	const op = "repo.ApproveVacation"

	query := `UPDATE vacations SET status = 'approved', approved_by = $1, approved_at = NOW() WHERE id = $2`
	result, err := r.db.ExecContext(ctx, query, approvedBy, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrVacationNotFound
	}
	return nil
}

func (r *Repo) RejectVacation(ctx context.Context, id int64, rejectedBy int64, reason string) error {
	const op = "repo.RejectVacation"

	query := `UPDATE vacations SET status = 'rejected', approved_by = $1, approved_at = NOW(), rejection_reason = $2 WHERE id = $3`
	result, err := r.db.ExecContext(ctx, query, rejectedBy, reason, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrVacationNotFound
	}
	return nil
}

// --- Vacation Overlap Detection ---

func (r *Repo) CheckVacationOverlap(ctx context.Context, employeeID int64, startDate, endDate string, excludeID *int64) (bool, error) {
	const op = "repo.CheckVacationOverlap"

	query := `
		SELECT COUNT(*) FROM vacations
		WHERE employee_id = $1
		  AND status IN ('approved', 'active', 'pending')
		  AND daterange(start_date, end_date, '[]') && daterange($2::date, $3::date, '[]')`

	args := []interface{}{employeeID, startDate, endDate}
	if excludeID != nil {
		query += " AND id != $4"
		args = append(args, *excludeID)
	}

	var count int
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	return count > 0, nil
}

// --- Blocked Period Check ---

func (r *Repo) CheckBlockedPeriod(ctx context.Context, departmentID int64, startDate, endDate string) (bool, error) {
	const op = "repo.CheckBlockedPeriod"

	query := `
		SELECT COUNT(*) FROM department_blocked_periods
		WHERE department_id = $1
		  AND daterange(start_date, end_date, '[]') && daterange($2::date, $3::date, '[]')`

	var count int
	if err := r.db.QueryRowContext(ctx, query, departmentID, startDate, endDate).Scan(&count); err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	return count > 0, nil
}

// --- Vacation Balances ---

func (r *Repo) GetVacationBalance(ctx context.Context, employeeID int64, year int) (*vacation.Balance, error) {
	const op = "repo.GetVacationBalance"

	query := `
		SELECT employee_id, year, total_days, used_days, pending_days, remaining_days, carried_over
		FROM vacation_balances
		WHERE employee_id = $1 AND year = $2`

	var b vacation.Balance
	err := r.db.QueryRowContext(ctx, query, employeeID, year).Scan(
		&b.EmployeeID, &b.Year, &b.TotalDays, &b.UsedDays,
		&b.PendingDays, &b.RemainingDays, &b.CarriedOver,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrBalanceNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &b, nil
}

func (r *Repo) GetAllVacationBalances(ctx context.Context, year int) ([]*vacation.Balance, error) {
	const op = "repo.GetAllVacationBalances"

	query := `
		SELECT employee_id, year, total_days, used_days, pending_days, remaining_days, carried_over
		FROM vacation_balances
		WHERE year = $1
		ORDER BY employee_id`

	rows, err := r.db.QueryContext(ctx, query, year)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var balances []*vacation.Balance
	for rows.Next() {
		var b vacation.Balance
		if err := rows.Scan(&b.EmployeeID, &b.Year, &b.TotalDays, &b.UsedDays,
			&b.PendingDays, &b.RemainingDays, &b.CarriedOver); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		balances = append(balances, &b)
	}
	return balances, rows.Err()
}

func (r *Repo) UpdateVacationBalancePending(ctx context.Context, employeeID int64, year int, deltaPending int) error {
	const op = "repo.UpdateVacationBalancePending"

	query := `
		UPDATE vacation_balances
		SET pending_days = pending_days + $1,
		    remaining_days = remaining_days - $1
		WHERE employee_id = $2 AND year = $3`

	result, err := r.db.ExecContext(ctx, query, deltaPending, employeeID, year)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrBalanceNotFound
	}
	return nil
}

func (r *Repo) UpdateVacationBalanceApprove(ctx context.Context, employeeID int64, year int, days int) error {
	const op = "repo.UpdateVacationBalanceApprove"

	query := `
		UPDATE vacation_balances
		SET pending_days = pending_days - $1,
		    used_days = used_days + $1
		WHERE employee_id = $2 AND year = $3`

	result, err := r.db.ExecContext(ctx, query, days, employeeID, year)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrBalanceNotFound
	}
	return nil
}

func (r *Repo) UpdateVacationBalanceReject(ctx context.Context, employeeID int64, year int, days int) error {
	const op = "repo.UpdateVacationBalanceReject"

	query := `
		UPDATE vacation_balances
		SET pending_days = pending_days - $1,
		    remaining_days = remaining_days + $1
		WHERE employee_id = $2 AND year = $3`

	result, err := r.db.ExecContext(ctx, query, days, employeeID, year)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrBalanceNotFound
	}
	return nil
}

func (r *Repo) UpdateVacationBalanceCancelApproved(ctx context.Context, employeeID int64, year int, days int) error {
	const op = "repo.UpdateVacationBalanceCancelApproved"

	query := `
		UPDATE vacation_balances
		SET used_days = used_days - $1,
		    remaining_days = remaining_days + $1
		WHERE employee_id = $2 AND year = $3`

	result, err := r.db.ExecContext(ctx, query, days, employeeID, year)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrBalanceNotFound
	}
	return nil
}

// --- Pending Vacations ---

func (r *Repo) GetPendingVacations(ctx context.Context) ([]*vacation.Vacation, error) {
	const op = "repo.GetPendingVacations"

	query := `
		SELECT v.id, v.employee_id, c.fio, v.vacation_type,
			   v.start_date::text, v.end_date::text, v.days, v.status,
			   v.reason, v.rejection_reason, v.approved_by,
			   v.approved_at::text, v.substitute_id,
			   sc.fio, v.created_at, v.updated_at
		FROM vacations v
		JOIN contacts c ON v.employee_id = c.id
		LEFT JOIN contacts sc ON v.substitute_id = sc.id
		WHERE v.status = 'pending'
		ORDER BY v.created_at ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var vacations []*vacation.Vacation
	for rows.Next() {
		vac, err := scanVacation(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		vacations = append(vacations, vac)
	}
	return vacations, rows.Err()
}

// --- Vacation Calendar ---

func (r *Repo) GetVacationCalendar(ctx context.Context, filters dto.VacationCalendarFilters) ([]*vacation.CalendarEntry, error) {
	const op = "repo.GetVacationCalendar"

	query := `
		SELECT v.id, v.employee_id, c.fio, COALESCE(d.name, ''),
			   v.vacation_type, v.start_date::text, v.end_date::text, v.days, v.status
		FROM vacations v
		JOIN contacts c ON v.employee_id = c.id
		LEFT JOIN personnel_records pr ON v.employee_id = pr.employee_id
		LEFT JOIN departments d ON pr.department_id = d.id
		WHERE v.status IN ('approved', 'active')`

	var args []interface{}
	argIdx := 1

	if filters.DepartmentID != nil {
		query += fmt.Sprintf(" AND pr.department_id = $%d", argIdx)
		args = append(args, *filters.DepartmentID)
		argIdx++
	}
	if filters.StartDate != nil {
		query += fmt.Sprintf(" AND v.end_date >= $%d::date", argIdx)
		args = append(args, *filters.StartDate)
		argIdx++
	}
	if filters.EndDate != nil {
		query += fmt.Sprintf(" AND v.start_date <= $%d::date", argIdx)
		args = append(args, *filters.EndDate)
		argIdx++
	}

	query += " ORDER BY v.start_date"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var entries []*vacation.CalendarEntry
	for rows.Next() {
		var e vacation.CalendarEntry
		if err := rows.Scan(&e.ID, &e.EmployeeID, &e.EmployeeName, &e.Department,
			&e.VacationType, &e.StartDate, &e.EndDate, &e.Days, &e.Status); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		entries = append(entries, &e)
	}
	return entries, rows.Err()
}

// --- Helpers ---

func scanVacation(scanner interface {
	Scan(dest ...interface{}) error
}) (*vacation.Vacation, error) {
	var v vacation.Vacation
	var (
		reason, rejectionReason sql.NullString
		approvedBy              sql.NullInt64
		approvedAt              sql.NullString
		substituteID            sql.NullInt64
		substituteName          sql.NullString
	)

	err := scanner.Scan(
		&v.ID, &v.EmployeeID, &v.EmployeeName, &v.VacationType,
		&v.StartDate, &v.EndDate, &v.Days, &v.Status,
		&reason, &rejectionReason, &approvedBy,
		&approvedAt, &substituteID,
		&substituteName, &v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if reason.Valid {
		v.Reason = &reason.String
	}
	if rejectionReason.Valid {
		v.RejectionReason = &rejectionReason.String
	}
	if approvedBy.Valid {
		v.ApprovedBy = &approvedBy.Int64
	}
	if approvedAt.Valid {
		v.ApprovedAt = &approvedAt.String
	}
	if substituteID.Valid {
		v.SubstituteID = &substituteID.Int64
	}
	if substituteName.Valid {
		v.SubstituteName = &substituteName.String
	}

	return &v, nil
}
