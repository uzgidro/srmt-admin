package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"srmt-admin/internal/lib/dto/hrm"
	hrmmodel "srmt-admin/internal/lib/model/hrm"
	"srmt-admin/internal/storage"
)

// --- Holiday Operations ---

// AddHoliday creates a new holiday
func (r *Repo) AddHoliday(ctx context.Context, req hrm.AddHolidayRequest) (int, error) {
	const op = "storage.repo.AddHoliday"

	const query = `
		INSERT INTO hrm_holidays (name, date, year, is_working_day)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	var id int
	err := r.db.QueryRowContext(ctx, query,
		req.Name, req.Date, req.Year, req.IsWorkingDay,
	).Scan(&id)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code.Name() == "unique_violation" {
			return 0, storage.ErrDuplicate
		}
		return 0, fmt.Errorf("%s: failed to insert holiday: %w", op, err)
	}

	return id, nil
}

// GetHolidays retrieves holidays for a year
func (r *Repo) GetHolidays(ctx context.Context, year int) ([]*hrmmodel.Holiday, error) {
	const op = "storage.repo.GetHolidays"

	const query = `
		SELECT id, name, date, year, is_working_day, created_at, updated_at
		FROM hrm_holidays
		WHERE year = $1
		ORDER BY date`

	rows, err := r.db.QueryContext(ctx, query, year)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query holidays: %w", op, err)
	}
	defer rows.Close()

	var holidays []*hrmmodel.Holiday
	for rows.Next() {
		var h hrmmodel.Holiday
		var updatedAt sql.NullTime

		err := rows.Scan(&h.ID, &h.Name, &h.Date, &h.Year, &h.IsWorkingDay, &h.CreatedAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan holiday: %w", op, err)
		}

		if updatedAt.Valid {
			h.UpdatedAt = &updatedAt.Time
		}

		holidays = append(holidays, &h)
	}

	if holidays == nil {
		holidays = make([]*hrmmodel.Holiday, 0)
	}

	return holidays, nil
}

// DeleteHoliday deletes a holiday
func (r *Repo) DeleteHoliday(ctx context.Context, id int) error {
	const op = "storage.repo.DeleteHoliday"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_holidays WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("%s: failed to delete holiday: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- Timesheet Operations ---

// AddTimesheet creates a new timesheet
func (r *Repo) AddTimesheet(ctx context.Context, req hrm.AddTimesheetRequest) (int64, error) {
	const op = "storage.repo.AddTimesheet"

	const query = `
		INSERT INTO hrm_timesheets (employee_id, year, month, notes)
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
		return 0, fmt.Errorf("%s: failed to insert timesheet: %w", op, err)
	}

	return id, nil
}

// GetTimesheetByID retrieves timesheet by ID
func (r *Repo) GetTimesheetByID(ctx context.Context, id int64) (*hrmmodel.Timesheet, error) {
	const op = "storage.repo.GetTimesheetByID"

	const query = `
		SELECT id, employee_id, year, month, total_work_days, total_worked_days,
			total_hours, overtime_hours, sick_days, vacation_days, absence_days,
			status, submitted_at, approved_by, approved_at, rejection_reason,
			notes, created_at, updated_at
		FROM hrm_timesheets
		WHERE id = $1`

	ts, err := r.scanTimesheet(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get timesheet: %w", op, err)
	}

	return ts, nil
}

// GetTimesheetByPeriod retrieves timesheet for employee/period
func (r *Repo) GetTimesheetByPeriod(ctx context.Context, employeeID int64, year, month int) (*hrmmodel.Timesheet, error) {
	const op = "storage.repo.GetTimesheetByPeriod"

	const query = `
		SELECT id, employee_id, year, month, total_work_days, total_worked_days,
			total_hours, overtime_hours, sick_days, vacation_days, absence_days,
			status, submitted_at, approved_by, approved_at, rejection_reason,
			notes, created_at, updated_at
		FROM hrm_timesheets
		WHERE employee_id = $1 AND year = $2 AND month = $3`

	ts, err := r.scanTimesheet(r.db.QueryRowContext(ctx, query, employeeID, year, month))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get timesheet: %w", op, err)
	}

	return ts, nil
}

// GetTimesheets retrieves timesheets with filters
func (r *Repo) GetTimesheets(ctx context.Context, filter hrm.TimesheetFilter) ([]*hrmmodel.Timesheet, error) {
	const op = "storage.repo.GetTimesheets"

	var query strings.Builder
	query.WriteString(`
		SELECT t.id, t.employee_id, t.year, t.month, t.total_work_days, t.total_worked_days,
			t.total_hours, t.overtime_hours, t.sick_days, t.vacation_days, t.absence_days,
			t.status, t.submitted_at, t.approved_by, t.approved_at, t.rejection_reason,
			t.notes, t.created_at, t.updated_at
		FROM hrm_timesheets t
		LEFT JOIN hrm_employees e ON t.employee_id = e.id
		LEFT JOIN contacts c ON e.contact_id = c.id
		WHERE 1=1
	`)

	var args []interface{}
	argIdx := 1

	if filter.EmployeeID != nil {
		query.WriteString(fmt.Sprintf(" AND t.employee_id = $%d", argIdx))
		args = append(args, *filter.EmployeeID)
		argIdx++
	}
	if filter.Year != nil {
		query.WriteString(fmt.Sprintf(" AND t.year = $%d", argIdx))
		args = append(args, *filter.Year)
		argIdx++
	}
	if filter.Month != nil {
		query.WriteString(fmt.Sprintf(" AND t.month = $%d", argIdx))
		args = append(args, *filter.Month)
		argIdx++
	}
	if filter.Status != nil {
		query.WriteString(fmt.Sprintf(" AND t.status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.DepartmentID != nil {
		query.WriteString(fmt.Sprintf(" AND c.department_id = $%d", argIdx))
		args = append(args, *filter.DepartmentID)
		argIdx++
	}

	query.WriteString(" ORDER BY t.year DESC, t.month DESC")

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
		return nil, fmt.Errorf("%s: failed to query timesheets: %w", op, err)
	}
	defer rows.Close()

	var timesheets []*hrmmodel.Timesheet
	for rows.Next() {
		ts, err := r.scanTimesheetRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan timesheet: %w", op, err)
		}
		timesheets = append(timesheets, ts)
	}

	if timesheets == nil {
		timesheets = make([]*hrmmodel.Timesheet, 0)
	}

	return timesheets, nil
}

// SubmitTimesheet submits timesheet for approval
func (r *Repo) SubmitTimesheet(ctx context.Context, id int64) error {
	const op = "storage.repo.SubmitTimesheet"

	const query = `
		UPDATE hrm_timesheets
		SET status = 'submitted', submitted_at = $1
		WHERE id = $2 AND status = 'draft'`

	res, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("%s: failed to submit timesheet: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// ApproveTimesheet approves or rejects timesheet
func (r *Repo) ApproveTimesheet(ctx context.Context, id int64, approvedBy int64, approved bool, rejectionReason *string) error {
	const op = "storage.repo.ApproveTimesheet"

	var status string
	if approved {
		status = hrmmodel.TimesheetStatusApproved
	} else {
		status = hrmmodel.TimesheetStatusRejected
	}

	const query = `
		UPDATE hrm_timesheets
		SET status = $1, approved_by = $2, approved_at = $3, rejection_reason = $4
		WHERE id = $5 AND status = 'submitted'`

	res, err := r.db.ExecContext(ctx, query, status, approvedBy, time.Now(), rejectionReason, id)
	if err != nil {
		return fmt.Errorf("%s: failed to approve timesheet: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// UpdateTimesheetSummary updates timesheet summary
func (r *Repo) UpdateTimesheetSummary(ctx context.Context, id int64, req hrm.EditTimesheetRequest) error {
	const op = "storage.repo.UpdateTimesheetSummary"

	var updates []string
	var args []interface{}
	argIdx := 1

	if req.TotalWorkDays != nil {
		updates = append(updates, fmt.Sprintf("total_work_days = $%d", argIdx))
		args = append(args, *req.TotalWorkDays)
		argIdx++
	}
	if req.TotalWorkedDays != nil {
		updates = append(updates, fmt.Sprintf("total_worked_days = $%d", argIdx))
		args = append(args, *req.TotalWorkedDays)
		argIdx++
	}
	if req.TotalHours != nil {
		updates = append(updates, fmt.Sprintf("total_hours = $%d", argIdx))
		args = append(args, *req.TotalHours)
		argIdx++
	}
	if req.OvertimeHours != nil {
		updates = append(updates, fmt.Sprintf("overtime_hours = $%d", argIdx))
		args = append(args, *req.OvertimeHours)
		argIdx++
	}
	if req.SickDays != nil {
		updates = append(updates, fmt.Sprintf("sick_days = $%d", argIdx))
		args = append(args, *req.SickDays)
		argIdx++
	}
	if req.VacationDays != nil {
		updates = append(updates, fmt.Sprintf("vacation_days = $%d", argIdx))
		args = append(args, *req.VacationDays)
		argIdx++
	}
	if req.AbsenceDays != nil {
		updates = append(updates, fmt.Sprintf("absence_days = $%d", argIdx))
		args = append(args, *req.AbsenceDays)
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

	query := fmt.Sprintf("UPDATE hrm_timesheets SET %s WHERE id = $%d",
		strings.Join(updates, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to update timesheet: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteTimesheet deletes timesheet
func (r *Repo) DeleteTimesheet(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteTimesheet"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_timesheets WHERE id = $1 AND status = 'draft'", id)
	if err != nil {
		return fmt.Errorf("%s: failed to delete timesheet: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// timesheetScanner interface for sql.Row and sql.Rows compatibility
type timesheetScanner interface {
	Scan(dest ...interface{}) error
}

// scanTimesheetFromScanner scans timesheet data from a scanner interface
func (r *Repo) scanTimesheetFromScanner(s timesheetScanner) (*hrmmodel.Timesheet, error) {
	var ts hrmmodel.Timesheet
	var submittedAt, approvedAt, updatedAt sql.NullTime
	var approvedBy sql.NullInt64
	var rejectionReason, notes sql.NullString

	err := s.Scan(
		&ts.ID, &ts.EmployeeID, &ts.Year, &ts.Month,
		&ts.TotalWorkDays, &ts.TotalWorkedDays, &ts.TotalHours, &ts.OvertimeHours,
		&ts.SickDays, &ts.VacationDays, &ts.AbsenceDays,
		&ts.Status, &submittedAt, &approvedBy, &approvedAt, &rejectionReason,
		&notes, &ts.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if submittedAt.Valid {
		ts.SubmittedAt = &submittedAt.Time
	}
	if approvedBy.Valid {
		ts.ApprovedBy = &approvedBy.Int64
	}
	if approvedAt.Valid {
		ts.ApprovedAt = &approvedAt.Time
	}
	if rejectionReason.Valid {
		ts.RejectionReason = &rejectionReason.String
	}
	if notes.Valid {
		ts.Notes = &notes.String
	}
	if updatedAt.Valid {
		ts.UpdatedAt = &updatedAt.Time
	}

	return &ts, nil
}

// scanTimesheet scans timesheet from sql.Row
func (r *Repo) scanTimesheet(row *sql.Row) (*hrmmodel.Timesheet, error) {
	return r.scanTimesheetFromScanner(row)
}

// scanTimesheetRow scans timesheet from sql.Rows
func (r *Repo) scanTimesheetRow(rows *sql.Rows) (*hrmmodel.Timesheet, error) {
	return r.scanTimesheetFromScanner(rows)
}

// --- Timesheet Entry Operations ---

// AddTimesheetEntry creates a new timesheet entry
func (r *Repo) AddTimesheetEntry(ctx context.Context, req hrm.AddTimesheetEntryRequest) (int64, error) {
	const op = "storage.repo.AddTimesheetEntry"

	const query = `
		INSERT INTO hrm_timesheet_entries (
			timesheet_id, employee_id, date, check_in, check_out,
			break_minutes, worked_hours, overtime_hours, day_type, is_remote, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.TimesheetID, req.EmployeeID, req.Date, req.CheckIn, req.CheckOut,
		req.BreakMinutes, req.WorkedHours, req.OvertimeHours, req.DayType, req.IsRemote, req.Notes,
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
		return 0, fmt.Errorf("%s: failed to insert timesheet entry: %w", op, err)
	}

	return id, nil
}

// GetTimesheetEntries retrieves entries for a timesheet
func (r *Repo) GetTimesheetEntries(ctx context.Context, timesheetID int64) ([]*hrmmodel.TimesheetEntry, error) {
	const op = "storage.repo.GetTimesheetEntries"

	const query = `
		SELECT id, timesheet_id, employee_id, date, check_in, check_out,
			break_minutes, worked_hours, overtime_hours, day_type, is_remote,
			notes, created_at, updated_at
		FROM hrm_timesheet_entries
		WHERE timesheet_id = $1
		ORDER BY date`

	rows, err := r.db.QueryContext(ctx, query, timesheetID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query entries: %w", op, err)
	}
	defer rows.Close()

	var entries []*hrmmodel.TimesheetEntry
	for rows.Next() {
		var e hrmmodel.TimesheetEntry
		var checkIn, checkOut, notes sql.NullString
		var updatedAt sql.NullTime

		err := rows.Scan(
			&e.ID, &e.TimesheetID, &e.EmployeeID, &e.Date, &checkIn, &checkOut,
			&e.BreakMinutes, &e.WorkedHours, &e.OvertimeHours, &e.DayType, &e.IsRemote,
			&notes, &e.CreatedAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan entry: %w", op, err)
		}

		if checkIn.Valid {
			e.CheckIn = &checkIn.String
		}
		if checkOut.Valid {
			e.CheckOut = &checkOut.String
		}
		if notes.Valid {
			e.Notes = &notes.String
		}
		if updatedAt.Valid {
			e.UpdatedAt = &updatedAt.Time
		}

		entries = append(entries, &e)
	}

	if entries == nil {
		entries = make([]*hrmmodel.TimesheetEntry, 0)
	}

	return entries, nil
}

// EditTimesheetEntry updates timesheet entry
func (r *Repo) EditTimesheetEntry(ctx context.Context, id int64, req hrm.EditTimesheetEntryRequest) error {
	const op = "storage.repo.EditTimesheetEntry"

	var updates []string
	var args []interface{}
	argIdx := 1

	if req.CheckIn != nil {
		updates = append(updates, fmt.Sprintf("check_in = $%d", argIdx))
		args = append(args, *req.CheckIn)
		argIdx++
	}
	if req.CheckOut != nil {
		updates = append(updates, fmt.Sprintf("check_out = $%d", argIdx))
		args = append(args, *req.CheckOut)
		argIdx++
	}
	if req.BreakMinutes != nil {
		updates = append(updates, fmt.Sprintf("break_minutes = $%d", argIdx))
		args = append(args, *req.BreakMinutes)
		argIdx++
	}
	if req.WorkedHours != nil {
		updates = append(updates, fmt.Sprintf("worked_hours = $%d", argIdx))
		args = append(args, *req.WorkedHours)
		argIdx++
	}
	if req.OvertimeHours != nil {
		updates = append(updates, fmt.Sprintf("overtime_hours = $%d", argIdx))
		args = append(args, *req.OvertimeHours)
		argIdx++
	}
	if req.DayType != nil {
		updates = append(updates, fmt.Sprintf("day_type = $%d", argIdx))
		args = append(args, *req.DayType)
		argIdx++
	}
	if req.IsRemote != nil {
		updates = append(updates, fmt.Sprintf("is_remote = $%d", argIdx))
		args = append(args, *req.IsRemote)
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

	query := fmt.Sprintf("UPDATE hrm_timesheet_entries SET %s WHERE id = $%d",
		strings.Join(updates, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to update entry: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteTimesheetEntry deletes timesheet entry
func (r *Repo) DeleteTimesheetEntry(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteTimesheetEntry"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_timesheet_entries WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("%s: failed to delete entry: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- Timesheet Correction Operations ---

// AddTimesheetCorrection creates a correction request
func (r *Repo) AddTimesheetCorrection(ctx context.Context, req hrm.AddTimesheetCorrectionRequest, employeeID int64, originalCheckIn, originalCheckOut, originalDayType *string) (int64, error) {
	const op = "storage.repo.AddTimesheetCorrection"

	const query = `
		INSERT INTO hrm_timesheet_corrections (
			entry_id, employee_id, original_check_in, original_check_out, original_day_type,
			requested_check_in, requested_check_out, requested_day_type, reason
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.EntryID, employeeID, originalCheckIn, originalCheckOut, originalDayType,
		req.RequestedCheckIn, req.RequestedCheckOut, req.RequestedDayType, req.Reason,
	).Scan(&id)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code.Name() == "foreign_key_violation" {
			return 0, storage.ErrForeignKeyViolation
		}
		return 0, fmt.Errorf("%s: failed to insert correction: %w", op, err)
	}

	return id, nil
}

// ApproveTimesheetCorrection approves or rejects correction
func (r *Repo) ApproveTimesheetCorrection(ctx context.Context, id int64, approvedBy int64, approved bool, rejectionReason *string) error {
	const op = "storage.repo.ApproveTimesheetCorrection"

	var status string
	if approved {
		status = hrmmodel.CorrectionStatusApproved
	} else {
		status = hrmmodel.CorrectionStatusRejected
	}

	const query = `
		UPDATE hrm_timesheet_corrections
		SET status = $1, approved_by = $2, approved_at = $3, rejection_reason = $4
		WHERE id = $5 AND status = 'pending'`

	res, err := r.db.ExecContext(ctx, query, status, approvedBy, time.Now(), rejectionReason, id)
	if err != nil {
		return fmt.Errorf("%s: failed to approve correction: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// GetTimesheetCorrections retrieves corrections with filters
func (r *Repo) GetTimesheetCorrections(ctx context.Context, filter hrm.CorrectionFilter) ([]*hrmmodel.TimesheetCorrection, error) {
	const op = "storage.repo.GetTimesheetCorrections"

	var query strings.Builder
	query.WriteString(`
		SELECT id, entry_id, employee_id, original_check_in, original_check_out, original_day_type,
			requested_check_in, requested_check_out, requested_day_type, reason,
			status, approved_by, approved_at, rejection_reason, created_at, updated_at
		FROM hrm_timesheet_corrections
		WHERE 1=1
	`)

	var args []interface{}
	argIdx := 1

	if filter.EmployeeID != nil {
		query.WriteString(fmt.Sprintf(" AND employee_id = $%d", argIdx))
		args = append(args, *filter.EmployeeID)
		argIdx++
	}
	if filter.Status != nil {
		query.WriteString(fmt.Sprintf(" AND status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}

	query.WriteString(" ORDER BY created_at DESC")

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
		return nil, fmt.Errorf("%s: failed to query corrections: %w", op, err)
	}
	defer rows.Close()

	var corrections []*hrmmodel.TimesheetCorrection
	for rows.Next() {
		var c hrmmodel.TimesheetCorrection
		var origCheckIn, origCheckOut, origDayType sql.NullString
		var reqCheckIn, reqCheckOut, reqDayType sql.NullString
		var approvedBy sql.NullInt64
		var approvedAt, updatedAt sql.NullTime
		var rejectionReason sql.NullString

		err := rows.Scan(
			&c.ID, &c.EntryID, &c.EmployeeID, &origCheckIn, &origCheckOut, &origDayType,
			&reqCheckIn, &reqCheckOut, &reqDayType, &c.Reason,
			&c.Status, &approvedBy, &approvedAt, &rejectionReason, &c.CreatedAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan correction: %w", op, err)
		}

		if origCheckIn.Valid {
			c.OriginalCheckIn = &origCheckIn.String
		}
		if origCheckOut.Valid {
			c.OriginalCheckOut = &origCheckOut.String
		}
		if origDayType.Valid {
			c.OriginalDayType = &origDayType.String
		}
		if reqCheckIn.Valid {
			c.RequestedCheckIn = &reqCheckIn.String
		}
		if reqCheckOut.Valid {
			c.RequestedCheckOut = &reqCheckOut.String
		}
		if reqDayType.Valid {
			c.RequestedDayType = &reqDayType.String
		}
		if approvedBy.Valid {
			c.ApprovedBy = &approvedBy.Int64
		}
		if approvedAt.Valid {
			c.ApprovedAt = &approvedAt.Time
		}
		if rejectionReason.Valid {
			c.RejectionReason = &rejectionReason.String
		}
		if updatedAt.Valid {
			c.UpdatedAt = &updatedAt.Time
		}

		corrections = append(corrections, &c)
	}

	if corrections == nil {
		corrections = make([]*hrmmodel.TimesheetCorrection, 0)
	}

	return corrections, nil
}
