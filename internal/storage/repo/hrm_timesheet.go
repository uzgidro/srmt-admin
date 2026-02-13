package repo

import (
	"context"
	"database/sql"
	"fmt"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/hrm/timesheet"
	"srmt-admin/internal/storage"
	"strings"
)

// --- Timesheet Entries ---

func (r *Repo) GetTimesheetEntries(ctx context.Context, employeeID int64, year, month int) ([]*timesheet.Day, error) {
	const op = "repo.GetTimesheetEntries"

	query := `
		SELECT id, employee_id, date::text, status,
			   check_in::text, check_out::text,
			   hours_worked, overtime, is_weekend, is_holiday, note
		FROM timesheet_entries
		WHERE employee_id = $1
		  AND EXTRACT(YEAR FROM date) = $2
		  AND EXTRACT(MONTH FROM date) = $3
		ORDER BY date`

	rows, err := r.db.QueryContext(ctx, query, employeeID, year, month)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var entries []*timesheet.Day
	for rows.Next() {
		d, err := scanTimesheetDay(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		entries = append(entries, d)
	}
	return entries, rows.Err()
}

func (r *Repo) GetTimesheetEntry(ctx context.Context, id int64) (*timesheet.Day, error) {
	const op = "repo.GetTimesheetEntry"

	query := `
		SELECT id, employee_id, date::text, status,
			   check_in::text, check_out::text,
			   hours_worked, overtime, is_weekend, is_holiday, note
		FROM timesheet_entries
		WHERE id = $1`

	d, err := scanTimesheetDay(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrTimesheetEntryNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return d, nil
}

func (r *Repo) GetTimesheetEntryByEmployeeDate(ctx context.Context, employeeID int64, date string) (*timesheet.Day, error) {
	const op = "repo.GetTimesheetEntryByEmployeeDate"

	query := `
		SELECT id, employee_id, date::text, status,
			   check_in::text, check_out::text,
			   hours_worked, overtime, is_weekend, is_holiday, note
		FROM timesheet_entries
		WHERE employee_id = $1 AND date = $2::date`

	d, err := scanTimesheetDay(r.db.QueryRowContext(ctx, query, employeeID, date))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrTimesheetEntryNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return d, nil
}

func (r *Repo) UpsertTimesheetEntry(ctx context.Context, employeeID int64, date, status string, checkIn, checkOut *string, hoursWorked, overtime *float64, isWeekend, isHoliday bool, note *string) (int64, error) {
	const op = "repo.UpsertTimesheetEntry"

	query := `
		INSERT INTO timesheet_entries (employee_id, date, status, check_in, check_out, hours_worked, overtime, is_weekend, is_holiday, note)
		VALUES ($1, $2::date, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (employee_id, date) DO UPDATE SET
			status = EXCLUDED.status,
			check_in = EXCLUDED.check_in,
			check_out = EXCLUDED.check_out,
			hours_worked = EXCLUDED.hours_worked,
			overtime = EXCLUDED.overtime,
			is_weekend = EXCLUDED.is_weekend,
			is_holiday = EXCLUDED.is_holiday,
			note = EXCLUDED.note
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		employeeID, date, status, checkIn, checkOut, hoursWorked, overtime, isWeekend, isHoliday, note,
	).Scan(&id)
	if err != nil {
		if translated := r.translator.Translate(err, op); translated != nil {
			return 0, translated
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

func (r *Repo) UpdateTimesheetEntry(ctx context.Context, id int64, req dto.UpdateTimesheetEntryRequest) error {
	const op = "repo.UpdateTimesheetEntry"

	var setClauses []string
	var args []interface{}
	argIdx := 1

	if req.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
	}
	if req.CheckIn != nil {
		setClauses = append(setClauses, fmt.Sprintf("check_in = $%d", argIdx))
		args = append(args, *req.CheckIn)
		argIdx++
	}
	if req.CheckOut != nil {
		setClauses = append(setClauses, fmt.Sprintf("check_out = $%d", argIdx))
		args = append(args, *req.CheckOut)
		argIdx++
	}
	if req.HoursWorked != nil {
		setClauses = append(setClauses, fmt.Sprintf("hours_worked = $%d", argIdx))
		args = append(args, *req.HoursWorked)
		argIdx++
	}
	if req.Overtime != nil {
		setClauses = append(setClauses, fmt.Sprintf("overtime = $%d", argIdx))
		args = append(args, *req.Overtime)
		argIdx++
	}
	if req.Note != nil {
		setClauses = append(setClauses, fmt.Sprintf("note = $%d", argIdx))
		args = append(args, *req.Note)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE timesheet_entries SET %s WHERE id = $%d",
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
		return storage.ErrTimesheetEntryNotFound
	}
	return nil
}

func (r *Repo) GetEmployeesForTimesheet(ctx context.Context, filters dto.TimesheetFilters) ([]*timesheet.EmployeeInfo, error) {
	const op = "repo.GetEmployeesForTimesheet"

	query := `
		SELECT pr.employee_id, c.fio,
			   COALESCE(d.name, ''), COALESCE(p.name, ''),
			   COALESCE(pr.tab_number, '')
		FROM personnel_records pr
		JOIN contacts c ON pr.employee_id = c.id
		LEFT JOIN departments d ON pr.department_id = d.id
		LEFT JOIN positions p ON pr.position_id = p.id
		WHERE pr.status = 'active'`

	var args []interface{}
	argIdx := 1

	if filters.DepartmentID != nil {
		query += fmt.Sprintf(" AND pr.department_id = $%d", argIdx)
		args = append(args, *filters.DepartmentID)
		argIdx++
	}
	if filters.EmployeeID != nil {
		query += fmt.Sprintf(" AND pr.employee_id = $%d", argIdx)
		args = append(args, *filters.EmployeeID)
		argIdx++
	}

	query += " ORDER BY c.fio"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var employees []*timesheet.EmployeeInfo
	for rows.Next() {
		var e timesheet.EmployeeInfo
		if err := rows.Scan(&e.EmployeeID, &e.Name, &e.Department, &e.Position, &e.TabNumber); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		employees = append(employees, &e)
	}
	return employees, rows.Err()
}

// --- Holidays ---

func (r *Repo) GetHolidays(ctx context.Context, year int) ([]*timesheet.Holiday, error) {
	const op = "repo.GetHolidays"

	query := `
		SELECT id, name, date::text, type, description, created_at
		FROM holidays
		WHERE EXTRACT(YEAR FROM date) = $1
		ORDER BY date`

	rows, err := r.db.QueryContext(ctx, query, year)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var holidays []*timesheet.Holiday
	for rows.Next() {
		var h timesheet.Holiday
		var desc sql.NullString
		if err := rows.Scan(&h.ID, &h.Name, &h.Date, &h.Type, &desc, &h.CreatedAt); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		if desc.Valid {
			h.Description = &desc.String
		}
		holidays = append(holidays, &h)
	}
	return holidays, rows.Err()
}

func (r *Repo) GetHolidaysByMonth(ctx context.Context, year, month int) ([]*timesheet.Holiday, error) {
	const op = "repo.GetHolidaysByMonth"

	query := `
		SELECT id, name, date::text, type, description, created_at
		FROM holidays
		WHERE EXTRACT(YEAR FROM date) = $1 AND EXTRACT(MONTH FROM date) = $2
		ORDER BY date`

	rows, err := r.db.QueryContext(ctx, query, year, month)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var holidays []*timesheet.Holiday
	for rows.Next() {
		var h timesheet.Holiday
		var desc sql.NullString
		if err := rows.Scan(&h.ID, &h.Name, &h.Date, &h.Type, &desc, &h.CreatedAt); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		if desc.Valid {
			h.Description = &desc.String
		}
		holidays = append(holidays, &h)
	}
	return holidays, rows.Err()
}

func (r *Repo) CreateHoliday(ctx context.Context, req dto.CreateHolidayRequest) (int64, error) {
	const op = "repo.CreateHoliday"

	query := `INSERT INTO holidays (name, date, type, description) VALUES ($1, $2::date, $3, $4) RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query, req.Name, req.Date, req.Type, req.Description).Scan(&id)
	if err != nil {
		if translated := r.translator.Translate(err, op); translated != nil {
			if translated == storage.ErrUniqueViolation {
				return 0, storage.ErrHolidayAlreadyExists
			}
			return 0, translated
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

func (r *Repo) DeleteHoliday(ctx context.Context, id int64) error {
	const op = "repo.DeleteHoliday"

	result, err := r.db.ExecContext(ctx, "DELETE FROM holidays WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrHolidayNotFound
	}
	return nil
}

// --- Corrections ---

func (r *Repo) GetTimesheetCorrections(ctx context.Context, filters dto.CorrectionFilters) ([]*timesheet.Correction, error) {
	const op = "repo.GetTimesheetCorrections"

	query := `
		SELECT tc.id, tc.employee_id, c.fio, tc.date::text,
			   tc.original_status, tc.new_status,
			   tc.original_check_in::text, tc.new_check_in::text,
			   tc.original_check_out::text, tc.new_check_out::text,
			   tc.reason, tc.status, tc.requested_by, tc.approved_by,
			   tc.created_at, tc.updated_at
		FROM timesheet_corrections tc
		JOIN contacts c ON tc.employee_id = c.id`

	var conditions []string
	var args []interface{}
	argIdx := 1

	if filters.EmployeeID != nil {
		conditions = append(conditions, fmt.Sprintf("tc.employee_id = $%d", argIdx))
		args = append(args, *filters.EmployeeID)
		argIdx++
	}
	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("tc.status = $%d", argIdx))
		args = append(args, *filters.Status)
		argIdx++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY tc.created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var corrections []*timesheet.Correction
	for rows.Next() {
		cor, err := scanCorrection(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		corrections = append(corrections, cor)
	}
	return corrections, rows.Err()
}

func (r *Repo) GetTimesheetCorrectionByID(ctx context.Context, id int64) (*timesheet.Correction, error) {
	const op = "repo.GetTimesheetCorrectionByID"

	query := `
		SELECT tc.id, tc.employee_id, c.fio, tc.date::text,
			   tc.original_status, tc.new_status,
			   tc.original_check_in::text, tc.new_check_in::text,
			   tc.original_check_out::text, tc.new_check_out::text,
			   tc.reason, tc.status, tc.requested_by, tc.approved_by,
			   tc.created_at, tc.updated_at
		FROM timesheet_corrections tc
		JOIN contacts c ON tc.employee_id = c.id
		WHERE tc.id = $1`

	cor, err := scanCorrection(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrCorrectionNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return cor, nil
}

func (r *Repo) CreateTimesheetCorrection(ctx context.Context, req dto.CreateTimesheetCorrectionRequest, originalStatus, originalCheckIn, originalCheckOut *string, requestedBy int64) (int64, error) {
	const op = "repo.CreateTimesheetCorrection"

	query := `
		INSERT INTO timesheet_corrections
			(employee_id, date, original_status, new_status,
			 original_check_in, new_check_in, original_check_out, new_check_out,
			 reason, requested_by)
		VALUES ($1, $2::date, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.EmployeeID, req.Date, originalStatus, req.NewStatus,
		originalCheckIn, req.NewCheckIn, originalCheckOut, req.NewCheckOut,
		req.Reason, requestedBy,
	).Scan(&id)
	if err != nil {
		if translated := r.translator.Translate(err, op); translated != nil {
			return 0, translated
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

func (r *Repo) ApproveTimesheetCorrection(ctx context.Context, id int64, approvedBy int64) error {
	const op = "repo.ApproveTimesheetCorrection"

	query := `UPDATE timesheet_corrections SET status = 'approved', approved_by = $1 WHERE id = $2 AND status = 'pending'`
	result, err := r.db.ExecContext(ctx, query, approvedBy, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrCorrectionNotFound
	}
	return nil
}

func (r *Repo) RejectTimesheetCorrection(ctx context.Context, id int64, approvedBy int64, reason string) error {
	const op = "repo.RejectTimesheetCorrection"

	query := `UPDATE timesheet_corrections SET status = 'rejected', approved_by = $1 WHERE id = $2 AND status = 'pending'`
	result, err := r.db.ExecContext(ctx, query, approvedBy, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrCorrectionNotFound
	}
	return nil
}

// --- Scan Helpers ---

func scanTimesheetDay(scanner interface {
	Scan(dest ...interface{}) error
}) (*timesheet.Day, error) {
	var d timesheet.Day
	var (
		checkIn     sql.NullString
		checkOut    sql.NullString
		hoursWorked sql.NullFloat64
		overtime    sql.NullFloat64
		note        sql.NullString
	)

	err := scanner.Scan(
		&d.ID, &d.EmployeeID, &d.Date, &d.Status,
		&checkIn, &checkOut,
		&hoursWorked, &overtime, &d.IsWeekend, &d.IsHoliday, &note,
	)
	if err != nil {
		return nil, err
	}

	if checkIn.Valid {
		d.CheckIn = &checkIn.String
	}
	if checkOut.Valid {
		d.CheckOut = &checkOut.String
	}
	if hoursWorked.Valid {
		d.HoursWorked = &hoursWorked.Float64
	}
	if overtime.Valid {
		d.Overtime = &overtime.Float64
	}
	if note.Valid {
		d.Note = &note.String
	}

	return &d, nil
}

func scanCorrection(scanner interface {
	Scan(dest ...interface{}) error
}) (*timesheet.Correction, error) {
	var c timesheet.Correction
	var (
		originalStatus   sql.NullString
		originalCheckIn  sql.NullString
		newCheckIn       sql.NullString
		originalCheckOut sql.NullString
		newCheckOut      sql.NullString
		approvedBy       sql.NullInt64
	)

	err := scanner.Scan(
		&c.ID, &c.EmployeeID, &c.EmployeeName, &c.Date,
		&originalStatus, &c.NewStatus,
		&originalCheckIn, &newCheckIn,
		&originalCheckOut, &newCheckOut,
		&c.Reason, &c.Status, &c.RequestedBy, &approvedBy,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if originalStatus.Valid {
		c.OriginalStatus = &originalStatus.String
	}
	if originalCheckIn.Valid {
		c.OriginalCheckIn = &originalCheckIn.String
	}
	if newCheckIn.Valid {
		c.NewCheckIn = &newCheckIn.String
	}
	if originalCheckOut.Valid {
		c.OriginalCheckOut = &originalCheckOut.String
	}
	if newCheckOut.Valid {
		c.NewCheckOut = &newCheckOut.String
	}
	if approvedBy.Valid {
		c.ApprovedBy = &approvedBy.Int64
	}

	return &c, nil
}
