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

// --- Vacation Type Operations ---

// AddVacationType creates a new vacation type
func (r *Repo) AddVacationType(ctx context.Context, req hrm.AddVacationTypeRequest) (int, error) {
	const op = "storage.repo.AddVacationType"

	const query = `
		INSERT INTO hrm_vacation_types (
			name, code, description, default_days_per_year, is_paid,
			requires_approval, can_carry_over, max_carry_over_days, sort_order
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`

	var id int
	err := r.db.QueryRowContext(ctx, query,
		req.Name, req.Code, req.Description, req.DefaultDaysPerYear,
		req.IsPaid, req.RequiresApproval, req.CanCarryOver,
		req.MaxCarryOverDays, req.SortOrder,
	).Scan(&id)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code.Name() == "unique_violation" {
			return 0, storage.ErrDuplicate
		}
		return 0, fmt.Errorf("%s: failed to insert vacation type: %w", op, err)
	}

	return id, nil
}

// GetVacationTypeByID retrieves vacation type by ID
func (r *Repo) GetVacationTypeByID(ctx context.Context, id int) (*hrmmodel.VacationType, error) {
	const op = "storage.repo.GetVacationTypeByID"

	const query = `
		SELECT id, name, code, description, default_days_per_year, is_paid,
			requires_approval, can_carry_over, max_carry_over_days,
			is_active, sort_order, created_at, updated_at
		FROM hrm_vacation_types
		WHERE id = $1`

	var vt hrmmodel.VacationType
	var desc sql.NullString
	var updatedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&vt.ID, &vt.Name, &vt.Code, &desc, &vt.DefaultDaysPerYear,
		&vt.IsPaid, &vt.RequiresApproval, &vt.CanCarryOver, &vt.MaxCarryOverDays,
		&vt.IsActive, &vt.SortOrder, &vt.CreatedAt, &updatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get vacation type: %w", op, err)
	}

	if desc.Valid {
		vt.Description = &desc.String
	}
	if updatedAt.Valid {
		vt.UpdatedAt = &updatedAt.Time
	}

	return &vt, nil
}

// GetAllVacationTypes retrieves all vacation types
func (r *Repo) GetAllVacationTypes(ctx context.Context, activeOnly bool) ([]*hrmmodel.VacationType, error) {
	const op = "storage.repo.GetAllVacationTypes"

	var query strings.Builder
	query.WriteString(`
		SELECT id, name, code, description, default_days_per_year, is_paid,
			requires_approval, can_carry_over, max_carry_over_days,
			is_active, sort_order, created_at, updated_at
		FROM hrm_vacation_types
	`)

	if activeOnly {
		query.WriteString(" WHERE is_active = TRUE")
	}
	query.WriteString(" ORDER BY sort_order, name")

	rows, err := r.db.QueryContext(ctx, query.String())
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query vacation types: %w", op, err)
	}
	defer rows.Close()

	var types []*hrmmodel.VacationType
	for rows.Next() {
		var vt hrmmodel.VacationType
		var desc sql.NullString
		var updatedAt sql.NullTime

		err := rows.Scan(
			&vt.ID, &vt.Name, &vt.Code, &desc, &vt.DefaultDaysPerYear,
			&vt.IsPaid, &vt.RequiresApproval, &vt.CanCarryOver, &vt.MaxCarryOverDays,
			&vt.IsActive, &vt.SortOrder, &vt.CreatedAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan vacation type: %w", op, err)
		}

		if desc.Valid {
			vt.Description = &desc.String
		}
		if updatedAt.Valid {
			vt.UpdatedAt = &updatedAt.Time
		}

		types = append(types, &vt)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if types == nil {
		types = make([]*hrmmodel.VacationType, 0)
	}

	return types, nil
}

// EditVacationType updates a vacation type
func (r *Repo) EditVacationType(ctx context.Context, id int, req hrm.EditVacationTypeRequest) error {
	const op = "storage.repo.EditVacationType"

	var updates []string
	var args []interface{}
	argIdx := 1

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Code != nil {
		updates = append(updates, fmt.Sprintf("code = $%d", argIdx))
		args = append(args, *req.Code)
		argIdx++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.DefaultDaysPerYear != nil {
		updates = append(updates, fmt.Sprintf("default_days_per_year = $%d", argIdx))
		args = append(args, *req.DefaultDaysPerYear)
		argIdx++
	}
	if req.IsPaid != nil {
		updates = append(updates, fmt.Sprintf("is_paid = $%d", argIdx))
		args = append(args, *req.IsPaid)
		argIdx++
	}
	if req.RequiresApproval != nil {
		updates = append(updates, fmt.Sprintf("requires_approval = $%d", argIdx))
		args = append(args, *req.RequiresApproval)
		argIdx++
	}
	if req.CanCarryOver != nil {
		updates = append(updates, fmt.Sprintf("can_carry_over = $%d", argIdx))
		args = append(args, *req.CanCarryOver)
		argIdx++
	}
	if req.MaxCarryOverDays != nil {
		updates = append(updates, fmt.Sprintf("max_carry_over_days = $%d", argIdx))
		args = append(args, *req.MaxCarryOverDays)
		argIdx++
	}
	if req.IsActive != nil {
		updates = append(updates, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *req.IsActive)
		argIdx++
	}
	if req.SortOrder != nil {
		updates = append(updates, fmt.Sprintf("sort_order = $%d", argIdx))
		args = append(args, *req.SortOrder)
		argIdx++
	}

	if len(updates) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE hrm_vacation_types SET %s WHERE id = $%d",
		strings.Join(updates, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code.Name() == "unique_violation" {
			return storage.ErrDuplicate
		}
		return fmt.Errorf("%s: failed to update vacation type: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteVacationType deletes a vacation type
func (r *Repo) DeleteVacationType(ctx context.Context, id int) error {
	const op = "storage.repo.DeleteVacationType"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_vacation_types WHERE id = $1", id)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code.Name() == "foreign_key_violation" {
			return storage.ErrForeignKeyViolation
		}
		return fmt.Errorf("%s: failed to delete vacation type: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- Vacation Balance Operations ---

// AddVacationBalance creates a new vacation balance
func (r *Repo) AddVacationBalance(ctx context.Context, req hrm.AddVacationBalanceRequest) (int64, error) {
	const op = "storage.repo.AddVacationBalance"

	const query = `
		INSERT INTO hrm_vacation_balances (
			employee_id, vacation_type_id, year, entitled_days,
			carried_over_days, adjustment_days, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.EmployeeID, req.VacationTypeID, req.Year,
		req.EntitledDays, req.CarriedOverDays, req.AdjustmentDays, req.Notes,
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
		return 0, fmt.Errorf("%s: failed to insert vacation balance: %w", op, err)
	}

	return id, nil
}

// GetVacationBalanceByID retrieves vacation balance by ID
func (r *Repo) GetVacationBalanceByID(ctx context.Context, id int64) (*hrmmodel.VacationBalance, error) {
	const op = "storage.repo.GetVacationBalanceByID"

	const query = `
		SELECT b.id, b.employee_id, b.vacation_type_id, b.year,
			b.entitled_days, b.used_days, b.carried_over_days, b.adjustment_days,
			b.notes, b.created_at, b.updated_at,
			t.name, t.code
		FROM hrm_vacation_balances b
		LEFT JOIN hrm_vacation_types t ON b.vacation_type_id = t.id
		WHERE b.id = $1`

	var vb hrmmodel.VacationBalance
	var notes sql.NullString
	var updatedAt sql.NullTime
	var typeName, typeCode sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&vb.ID, &vb.EmployeeID, &vb.VacationTypeID, &vb.Year,
		&vb.EntitledDays, &vb.UsedDays, &vb.CarriedOverDays, &vb.AdjustmentDays,
		&notes, &vb.CreatedAt, &updatedAt,
		&typeName, &typeCode,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get vacation balance: %w", op, err)
	}

	if notes.Valid {
		vb.Notes = &notes.String
	}
	if updatedAt.Valid {
		vb.UpdatedAt = &updatedAt.Time
	}
	if typeName.Valid {
		vb.VacationType = &hrmmodel.VacationType{
			ID:   vb.VacationTypeID,
			Name: typeName.String,
			Code: typeCode.String,
		}
	}

	vb.RemainingDays = vb.CalculateRemaining()

	return &vb, nil
}

// GetVacationBalances retrieves vacation balances with filters
func (r *Repo) GetVacationBalances(ctx context.Context, filter hrm.VacationBalanceFilter) ([]*hrmmodel.VacationBalance, error) {
	const op = "storage.repo.GetVacationBalances"

	var query strings.Builder
	query.WriteString(`
		SELECT b.id, b.employee_id, b.vacation_type_id, b.year,
			b.entitled_days, b.used_days, b.carried_over_days, b.adjustment_days,
			b.notes, b.created_at, b.updated_at,
			t.name, t.code
		FROM hrm_vacation_balances b
		LEFT JOIN hrm_vacation_types t ON b.vacation_type_id = t.id
		WHERE 1=1
	`)

	var args []interface{}
	argIdx := 1

	if filter.EmployeeID != nil {
		query.WriteString(fmt.Sprintf(" AND b.employee_id = $%d", argIdx))
		args = append(args, *filter.EmployeeID)
		argIdx++
	}
	if filter.VacationTypeID != nil {
		query.WriteString(fmt.Sprintf(" AND b.vacation_type_id = $%d", argIdx))
		args = append(args, *filter.VacationTypeID)
		argIdx++
	}
	if filter.Year != nil {
		query.WriteString(fmt.Sprintf(" AND b.year = $%d", argIdx))
		args = append(args, *filter.Year)
		argIdx++
	}

	query.WriteString(" ORDER BY b.year DESC, t.sort_order")

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query vacation balances: %w", op, err)
	}
	defer rows.Close()

	var balances []*hrmmodel.VacationBalance
	for rows.Next() {
		var vb hrmmodel.VacationBalance
		var notes sql.NullString
		var updatedAt sql.NullTime
		var typeName, typeCode sql.NullString

		err := rows.Scan(
			&vb.ID, &vb.EmployeeID, &vb.VacationTypeID, &vb.Year,
			&vb.EntitledDays, &vb.UsedDays, &vb.CarriedOverDays, &vb.AdjustmentDays,
			&notes, &vb.CreatedAt, &updatedAt,
			&typeName, &typeCode,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan vacation balance: %w", op, err)
		}

		if notes.Valid {
			vb.Notes = &notes.String
		}
		if updatedAt.Valid {
			vb.UpdatedAt = &updatedAt.Time
		}
		if typeName.Valid {
			vb.VacationType = &hrmmodel.VacationType{
				ID:   vb.VacationTypeID,
				Name: typeName.String,
				Code: typeCode.String,
			}
		}

		vb.RemainingDays = vb.CalculateRemaining()

		balances = append(balances, &vb)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if balances == nil {
		balances = make([]*hrmmodel.VacationBalance, 0)
	}

	return balances, nil
}

// EditVacationBalance updates vacation balance
func (r *Repo) EditVacationBalance(ctx context.Context, id int64, req hrm.EditVacationBalanceRequest) error {
	const op = "storage.repo.EditVacationBalance"

	var updates []string
	var args []interface{}
	argIdx := 1

	if req.EntitledDays != nil {
		updates = append(updates, fmt.Sprintf("entitled_days = $%d", argIdx))
		args = append(args, *req.EntitledDays)
		argIdx++
	}
	if req.UsedDays != nil {
		updates = append(updates, fmt.Sprintf("used_days = $%d", argIdx))
		args = append(args, *req.UsedDays)
		argIdx++
	}
	if req.CarriedOverDays != nil {
		updates = append(updates, fmt.Sprintf("carried_over_days = $%d", argIdx))
		args = append(args, *req.CarriedOverDays)
		argIdx++
	}
	if req.AdjustmentDays != nil {
		updates = append(updates, fmt.Sprintf("adjustment_days = $%d", argIdx))
		args = append(args, *req.AdjustmentDays)
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

	query := fmt.Sprintf("UPDATE hrm_vacation_balances SET %s WHERE id = $%d",
		strings.Join(updates, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to update vacation balance: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// UpdateVacationBalanceUsedDays updates the used days for a balance
func (r *Repo) UpdateVacationBalanceUsedDays(ctx context.Context, employeeID int64, vacationTypeID int, year int, daysToAdd float64) error {
	const op = "storage.repo.UpdateVacationBalanceUsedDays"

	const query = `
		UPDATE hrm_vacation_balances
		SET used_days = used_days + $1
		WHERE employee_id = $2 AND vacation_type_id = $3 AND year = $4`

	res, err := r.db.ExecContext(ctx, query, daysToAdd, employeeID, vacationTypeID, year)
	if err != nil {
		return fmt.Errorf("%s: failed to update used days: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- Vacation Request Operations ---

// AddVacation creates a new vacation request
func (r *Repo) AddVacation(ctx context.Context, req hrm.AddVacationRequest) (int64, error) {
	const op = "storage.repo.AddVacation"

	const query = `
		INSERT INTO hrm_vacations (
			employee_id, vacation_type_id, start_date, end_date, days_count,
			reason, substitute_employee_id, supporting_document_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.EmployeeID, req.VacationTypeID, req.StartDate, req.EndDate, req.DaysCount,
		req.Reason, req.SubstituteEmployeeID, req.SupportingDocumentID,
	).Scan(&id)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code.Name() == "foreign_key_violation" {
			return 0, storage.ErrForeignKeyViolation
		}
		return 0, fmt.Errorf("%s: failed to insert vacation: %w", op, err)
	}

	return id, nil
}

// GetVacationByID retrieves vacation by ID
func (r *Repo) GetVacationByID(ctx context.Context, id int64) (*hrmmodel.Vacation, error) {
	const op = "storage.repo.GetVacationByID"

	const query = `
		SELECT v.id, v.employee_id, v.vacation_type_id, v.start_date, v.end_date,
			v.days_count, v.status, v.reason, v.rejection_reason,
			v.requested_at, v.approved_by, v.approved_at,
			v.substitute_employee_id, v.supporting_document_id,
			v.created_at, v.updated_at,
			t.name, t.code
		FROM hrm_vacations v
		LEFT JOIN hrm_vacation_types t ON v.vacation_type_id = t.id
		WHERE v.id = $1`

	var vac hrmmodel.Vacation
	var reason, rejectionReason sql.NullString
	var approvedBy, substituteID, docID sql.NullInt64
	var approvedAt, updatedAt sql.NullTime
	var typeName, typeCode sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&vac.ID, &vac.EmployeeID, &vac.VacationTypeID, &vac.StartDate, &vac.EndDate,
		&vac.DaysCount, &vac.Status, &reason, &rejectionReason,
		&vac.RequestedAt, &approvedBy, &approvedAt,
		&substituteID, &docID,
		&vac.CreatedAt, &updatedAt,
		&typeName, &typeCode,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get vacation: %w", op, err)
	}

	if reason.Valid {
		vac.Reason = &reason.String
	}
	if rejectionReason.Valid {
		vac.RejectionReason = &rejectionReason.String
	}
	if approvedBy.Valid {
		vac.ApprovedBy = &approvedBy.Int64
	}
	if approvedAt.Valid {
		vac.ApprovedAt = &approvedAt.Time
	}
	if substituteID.Valid {
		vac.SubstituteEmployeeID = &substituteID.Int64
	}
	if docID.Valid {
		vac.SupportingDocumentID = &docID.Int64
	}
	if updatedAt.Valid {
		vac.UpdatedAt = &updatedAt.Time
	}
	if typeName.Valid {
		vac.VacationType = &hrmmodel.VacationType{
			ID:   vac.VacationTypeID,
			Name: typeName.String,
			Code: typeCode.String,
		}
	}

	return &vac, nil
}

// GetVacations retrieves vacations with filters
func (r *Repo) GetVacations(ctx context.Context, filter hrm.VacationFilter) ([]*hrmmodel.Vacation, error) {
	const op = "storage.repo.GetVacations"

	var query strings.Builder
	query.WriteString(`
		SELECT v.id, v.employee_id, v.vacation_type_id, v.start_date, v.end_date,
			v.days_count, v.status, v.reason, v.rejection_reason,
			v.requested_at, v.approved_by, v.approved_at,
			v.substitute_employee_id, v.supporting_document_id,
			v.created_at, v.updated_at,
			t.name, t.code,
			c.fio as employee_name
		FROM hrm_vacations v
		LEFT JOIN hrm_vacation_types t ON v.vacation_type_id = t.id
		LEFT JOIN hrm_employees e ON v.employee_id = e.id
		LEFT JOIN contacts c ON e.contact_id = c.id
		WHERE 1=1
	`)

	var args []interface{}
	argIdx := 1

	if filter.EmployeeID != nil {
		query.WriteString(fmt.Sprintf(" AND v.employee_id = $%d", argIdx))
		args = append(args, *filter.EmployeeID)
		argIdx++
	}
	if filter.VacationTypeID != nil {
		query.WriteString(fmt.Sprintf(" AND v.vacation_type_id = $%d", argIdx))
		args = append(args, *filter.VacationTypeID)
		argIdx++
	}
	if filter.Status != nil {
		query.WriteString(fmt.Sprintf(" AND v.status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.FromDate != nil {
		query.WriteString(fmt.Sprintf(" AND v.start_date >= $%d", argIdx))
		args = append(args, *filter.FromDate)
		argIdx++
	}
	if filter.ToDate != nil {
		query.WriteString(fmt.Sprintf(" AND v.end_date <= $%d", argIdx))
		args = append(args, *filter.ToDate)
		argIdx++
	}
	if filter.DepartmentID != nil {
		query.WriteString(fmt.Sprintf(" AND c.department_id = $%d", argIdx))
		args = append(args, *filter.DepartmentID)
		argIdx++
	}

	query.WriteString(" ORDER BY v.requested_at DESC")

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
		return nil, fmt.Errorf("%s: failed to query vacations: %w", op, err)
	}
	defer rows.Close()

	var vacations []*hrmmodel.Vacation
	for rows.Next() {
		var vac hrmmodel.Vacation
		var reason, rejectionReason sql.NullString
		var approvedBy, substituteID, docID sql.NullInt64
		var approvedAt, updatedAt sql.NullTime
		var typeName, typeCode, employeeName sql.NullString

		err := rows.Scan(
			&vac.ID, &vac.EmployeeID, &vac.VacationTypeID, &vac.StartDate, &vac.EndDate,
			&vac.DaysCount, &vac.Status, &reason, &rejectionReason,
			&vac.RequestedAt, &approvedBy, &approvedAt,
			&substituteID, &docID,
			&vac.CreatedAt, &updatedAt,
			&typeName, &typeCode,
			&employeeName,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan vacation: %w", op, err)
		}

		if reason.Valid {
			vac.Reason = &reason.String
		}
		if rejectionReason.Valid {
			vac.RejectionReason = &rejectionReason.String
		}
		if approvedBy.Valid {
			vac.ApprovedBy = &approvedBy.Int64
		}
		if approvedAt.Valid {
			vac.ApprovedAt = &approvedAt.Time
		}
		if substituteID.Valid {
			vac.SubstituteEmployeeID = &substituteID.Int64
		}
		if docID.Valid {
			vac.SupportingDocumentID = &docID.Int64
		}
		if updatedAt.Valid {
			vac.UpdatedAt = &updatedAt.Time
		}
		if typeName.Valid {
			vac.VacationType = &hrmmodel.VacationType{
				ID:   vac.VacationTypeID,
				Name: typeName.String,
				Code: typeCode.String,
			}
		}

		vacations = append(vacations, &vac)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if vacations == nil {
		vacations = make([]*hrmmodel.Vacation, 0)
	}

	return vacations, nil
}

// EditVacation updates a vacation request
func (r *Repo) EditVacation(ctx context.Context, id int64, req hrm.EditVacationRequest) error {
	const op = "storage.repo.EditVacation"

	var updates []string
	var args []interface{}
	argIdx := 1

	if req.VacationTypeID != nil {
		updates = append(updates, fmt.Sprintf("vacation_type_id = $%d", argIdx))
		args = append(args, *req.VacationTypeID)
		argIdx++
	}
	if req.StartDate != nil {
		updates = append(updates, fmt.Sprintf("start_date = $%d", argIdx))
		args = append(args, *req.StartDate)
		argIdx++
	}
	if req.EndDate != nil {
		updates = append(updates, fmt.Sprintf("end_date = $%d", argIdx))
		args = append(args, *req.EndDate)
		argIdx++
	}
	if req.DaysCount != nil {
		updates = append(updates, fmt.Sprintf("days_count = $%d", argIdx))
		args = append(args, *req.DaysCount)
		argIdx++
	}
	if req.Reason != nil {
		updates = append(updates, fmt.Sprintf("reason = $%d", argIdx))
		args = append(args, *req.Reason)
		argIdx++
	}
	if req.SubstituteEmployeeID != nil {
		updates = append(updates, fmt.Sprintf("substitute_employee_id = $%d", argIdx))
		args = append(args, *req.SubstituteEmployeeID)
		argIdx++
	}
	if req.SupportingDocumentID != nil {
		updates = append(updates, fmt.Sprintf("supporting_document_id = $%d", argIdx))
		args = append(args, *req.SupportingDocumentID)
		argIdx++
	}

	if len(updates) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE hrm_vacations SET %s WHERE id = $%d AND status = 'pending'",
		strings.Join(updates, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to update vacation: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// ApproveVacation approves or rejects a vacation request
func (r *Repo) ApproveVacation(ctx context.Context, id int64, approvedBy int64, approved bool, rejectionReason *string) error {
	const op = "storage.repo.ApproveVacation"

	var status string
	if approved {
		status = hrmmodel.VacationStatusApproved
	} else {
		status = hrmmodel.VacationStatusRejected
	}

	const query = `
		UPDATE hrm_vacations
		SET status = $1, approved_by = $2, approved_at = $3, rejection_reason = $4
		WHERE id = $5 AND status = 'pending'`

	res, err := r.db.ExecContext(ctx, query, status, approvedBy, time.Now(), rejectionReason, id)
	if err != nil {
		return fmt.Errorf("%s: failed to approve vacation: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// CancelVacation cancels a vacation request
func (r *Repo) CancelVacation(ctx context.Context, id int64) error {
	const op = "storage.repo.CancelVacation"

	const query = `
		UPDATE hrm_vacations
		SET status = $1
		WHERE id = $2 AND status IN ('pending', 'approved')`

	res, err := r.db.ExecContext(ctx, query, hrmmodel.VacationStatusCancelled, id)
	if err != nil {
		return fmt.Errorf("%s: failed to cancel vacation: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteVacation deletes a vacation request
func (r *Repo) DeleteVacation(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteVacation"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_vacations WHERE id = $1 AND status = 'pending'", id)
	if err != nil {
		return fmt.Errorf("%s: failed to delete vacation: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// CountPendingVacations counts pending vacation requests
func (r *Repo) CountPendingVacations(ctx context.Context, approverID *int64) (int, error) {
	const op = "storage.repo.CountPendingVacations"

	var query strings.Builder
	query.WriteString(`
		SELECT COUNT(*) FROM hrm_vacations v
		JOIN hrm_employees e ON v.employee_id = e.id
		WHERE v.status = 'pending'
	`)

	var args []interface{}
	if approverID != nil {
		query.WriteString(" AND e.manager_id = $1")
		args = append(args, *approverID)
	}

	var count int
	err := r.db.QueryRowContext(ctx, query.String(), args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("%s: failed to count pending vacations: %w", op, err)
	}

	return count, nil
}

// GetVacationCalendar retrieves vacation calendar for a month
func (r *Repo) GetVacationCalendar(ctx context.Context, filter hrm.VacationCalendarFilter) ([]*hrmmodel.VacationCalendarEntry, error) {
	const op = "storage.repo.GetVacationCalendar"

	const query = `
		SELECT v.employee_id, c.fio, v.start_date, v.end_date, t.name, v.status
		FROM hrm_vacations v
		JOIN hrm_employees e ON v.employee_id = e.id
		JOIN contacts c ON e.contact_id = c.id
		JOIN hrm_vacation_types t ON v.vacation_type_id = t.id
		WHERE v.status IN ('approved', 'pending')
		AND (
			(v.start_date <= $1 AND v.end_date >= $2)
			OR (v.start_date BETWEEN $2 AND $1)
			OR (v.end_date BETWEEN $2 AND $1)
		)
		ORDER BY c.fio, v.start_date`

	// Calculate month boundaries
	startOfMonth := time.Date(filter.Year, time.Month(filter.Month), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, -1)

	rows, err := r.db.QueryContext(ctx, query, endOfMonth, startOfMonth)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query vacation calendar: %w", op, err)
	}
	defer rows.Close()

	var entries []*hrmmodel.VacationCalendarEntry
	for rows.Next() {
		var employeeID int64
		var employeeName, vacationType, status string
		var startDate, endDate time.Time

		err := rows.Scan(&employeeID, &employeeName, &startDate, &endDate, &vacationType, &status)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan calendar entry: %w", op, err)
		}

		// Generate entries for each day
		for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
			if d.Month() == time.Month(filter.Month) && d.Year() == filter.Year {
				entries = append(entries, &hrmmodel.VacationCalendarEntry{
					Date:         d,
					EmployeeID:   employeeID,
					EmployeeName: employeeName,
					VacationType: vacationType,
					Status:       status,
				})
			}
		}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if entries == nil {
		entries = make([]*hrmmodel.VacationCalendarEntry, 0)
	}

	return entries, nil
}
