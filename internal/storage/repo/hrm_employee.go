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
	"srmt-admin/internal/lib/model/contact"
	"srmt-admin/internal/lib/model/department"
	hrmmodel "srmt-admin/internal/lib/model/hrm"
	"srmt-admin/internal/lib/model/organization"
	"srmt-admin/internal/lib/model/position"
	"srmt-admin/internal/storage"
)

// --- Employee Operations ---

// AddEmployee creates a new HRM employee record
func (r *Repo) AddEmployee(ctx context.Context, req hrm.AddEmployeeRequest) (int64, error) {
	const op = "storage.repo.AddEmployee"

	const query = `
		INSERT INTO hrm_employees (
			contact_id, user_id, employee_number, hire_date, employment_type,
			employment_status, work_schedule, work_hours_per_week, manager_id,
			probation_end_date, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`

	status := req.EmploymentStatus
	if status == "" {
		status = hrmmodel.EmploymentStatusActive
	}

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.ContactID,
		req.UserID,
		req.EmployeeNumber,
		req.HireDate,
		req.EmploymentType,
		status,
		req.WorkSchedule,
		req.WorkHoursPerWeek,
		req.ManagerID,
		req.ProbationEndDate,
		req.Notes,
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
		return 0, fmt.Errorf("%s: failed to insert employee: %w", op, err)
	}

	return id, nil
}

// GetEmployeeByID retrieves employee by ID with enriched data
func (r *Repo) GetEmployeeByID(ctx context.Context, id int64) (*hrmmodel.Employee, error) {
	const op = "storage.repo.GetEmployeeByID"

	const query = `
		SELECT
			e.id, e.contact_id, e.user_id, e.employee_number, e.hire_date,
			e.termination_date, e.employment_type, e.employment_status,
			e.work_schedule, e.work_hours_per_week, e.manager_id,
			e.probation_end_date, e.probation_passed, e.notes,
			e.created_at, e.updated_at,
			c.id, c.fio, c.email, c.phone, c.ip_phone, c.dob,
			o.id, o.name,
			d.id, d.name,
			p.id, p.name
		FROM hrm_employees e
		LEFT JOIN contacts c ON e.contact_id = c.id
		LEFT JOIN organizations o ON c.organization_id = o.id
		LEFT JOIN departments d ON c.department_id = d.id
		LEFT JOIN positions p ON c.position_id = p.id
		WHERE e.id = $1`

	emp, err := r.scanEmployee(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get employee: %w", op, err)
	}

	return emp, nil
}

// GetEmployeeByContactID retrieves employee by contact ID
func (r *Repo) GetEmployeeByContactID(ctx context.Context, contactID int64) (*hrmmodel.Employee, error) {
	const op = "storage.repo.GetEmployeeByContactID"

	const query = `
		SELECT
			e.id, e.contact_id, e.user_id, e.employee_number, e.hire_date,
			e.termination_date, e.employment_type, e.employment_status,
			e.work_schedule, e.work_hours_per_week, e.manager_id,
			e.probation_end_date, e.probation_passed, e.notes,
			e.created_at, e.updated_at,
			c.id, c.fio, c.email, c.phone, c.ip_phone, c.dob,
			o.id, o.name,
			d.id, d.name,
			p.id, p.name
		FROM hrm_employees e
		LEFT JOIN contacts c ON e.contact_id = c.id
		LEFT JOIN organizations o ON c.organization_id = o.id
		LEFT JOIN departments d ON c.department_id = d.id
		LEFT JOIN positions p ON c.position_id = p.id
		WHERE e.contact_id = $1`

	emp, err := r.scanEmployee(r.db.QueryRowContext(ctx, query, contactID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get employee: %w", op, err)
	}

	return emp, nil
}

// GetEmployeeByUserID retrieves employee by user ID
func (r *Repo) GetEmployeeByUserID(ctx context.Context, userID int64) (*hrmmodel.Employee, error) {
	const op = "storage.repo.GetEmployeeByUserID"

	const query = `
		SELECT
			e.id, e.contact_id, e.user_id, e.employee_number, e.hire_date,
			e.termination_date, e.employment_type, e.employment_status,
			e.work_schedule, e.work_hours_per_week, e.manager_id,
			e.probation_end_date, e.probation_passed, e.notes,
			e.created_at, e.updated_at,
			c.id, c.fio, c.email, c.phone, c.ip_phone, c.dob,
			o.id, o.name,
			d.id, d.name,
			p.id, p.name
		FROM hrm_employees e
		LEFT JOIN contacts c ON e.contact_id = c.id
		LEFT JOIN organizations o ON c.organization_id = o.id
		LEFT JOIN departments d ON c.department_id = d.id
		LEFT JOIN positions p ON c.position_id = p.id
		WHERE e.user_id = $1`

	emp, err := r.scanEmployee(r.db.QueryRowContext(ctx, query, userID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get employee: %w", op, err)
	}

	return emp, nil
}

// GetAllEmployees retrieves all employees with filters
func (r *Repo) GetAllEmployees(ctx context.Context, filter hrm.EmployeeFilter) ([]*hrmmodel.Employee, error) {
	const op = "storage.repo.GetAllEmployees"

	var query strings.Builder
	query.WriteString(`
		SELECT
			e.id, e.contact_id, e.user_id, e.employee_number, e.hire_date,
			e.termination_date, e.employment_type, e.employment_status,
			e.work_schedule, e.work_hours_per_week, e.manager_id,
			e.probation_end_date, e.probation_passed, e.notes,
			e.created_at, e.updated_at,
			c.id, c.fio, c.email, c.phone, c.ip_phone, c.dob,
			o.id, o.name,
			d.id, d.name,
			p.id, p.name
		FROM hrm_employees e
		LEFT JOIN contacts c ON e.contact_id = c.id
		LEFT JOIN organizations o ON c.organization_id = o.id
		LEFT JOIN departments d ON c.department_id = d.id
		LEFT JOIN positions p ON c.position_id = p.id
		WHERE 1=1
	`)

	var args []interface{}
	argIdx := 1

	if filter.OrganizationID != nil {
		query.WriteString(fmt.Sprintf(" AND c.organization_id = $%d", argIdx))
		args = append(args, *filter.OrganizationID)
		argIdx++
	}
	if filter.DepartmentID != nil {
		query.WriteString(fmt.Sprintf(" AND c.department_id = $%d", argIdx))
		args = append(args, *filter.DepartmentID)
		argIdx++
	}
	if filter.PositionID != nil {
		query.WriteString(fmt.Sprintf(" AND c.position_id = $%d", argIdx))
		args = append(args, *filter.PositionID)
		argIdx++
	}
	if filter.ManagerID != nil {
		query.WriteString(fmt.Sprintf(" AND e.manager_id = $%d", argIdx))
		args = append(args, *filter.ManagerID)
		argIdx++
	}
	if filter.EmploymentType != nil {
		query.WriteString(fmt.Sprintf(" AND e.employment_type = $%d", argIdx))
		args = append(args, *filter.EmploymentType)
		argIdx++
	}
	if filter.EmploymentStatus != nil {
		query.WriteString(fmt.Sprintf(" AND e.employment_status = $%d", argIdx))
		args = append(args, *filter.EmploymentStatus)
		argIdx++
	}
	if filter.Search != nil && *filter.Search != "" {
		query.WriteString(fmt.Sprintf(" AND (c.fio ILIKE $%d OR c.email ILIKE $%d OR e.employee_number ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+*filter.Search+"%")
		argIdx++
	}

	query.WriteString(" ORDER BY c.fio")

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
		return nil, fmt.Errorf("%s: failed to query employees: %w", op, err)
	}
	defer rows.Close()

	var employees []*hrmmodel.Employee
	for rows.Next() {
		emp, err := r.scanEmployeeRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan employee: %w", op, err)
		}
		employees = append(employees, emp)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if employees == nil {
		employees = make([]*hrmmodel.Employee, 0)
	}

	return employees, nil
}

// EditEmployee updates an employee record
func (r *Repo) EditEmployee(ctx context.Context, id int64, req hrm.EditEmployeeRequest) error {
	const op = "storage.repo.EditEmployee"

	var updates []string
	var args []interface{}
	argIdx := 1

	if req.ContactID != nil {
		updates = append(updates, fmt.Sprintf("contact_id = $%d", argIdx))
		args = append(args, *req.ContactID)
		argIdx++
	}
	if req.UserID != nil {
		updates = append(updates, fmt.Sprintf("user_id = $%d", argIdx))
		args = append(args, *req.UserID)
		argIdx++
	}
	if req.EmployeeNumber != nil {
		updates = append(updates, fmt.Sprintf("employee_number = $%d", argIdx))
		args = append(args, *req.EmployeeNumber)
		argIdx++
	}
	if req.HireDate != nil {
		updates = append(updates, fmt.Sprintf("hire_date = $%d", argIdx))
		args = append(args, *req.HireDate)
		argIdx++
	}
	if req.TerminationDate != nil {
		updates = append(updates, fmt.Sprintf("termination_date = $%d", argIdx))
		args = append(args, *req.TerminationDate)
		argIdx++
	}
	if req.EmploymentType != nil {
		updates = append(updates, fmt.Sprintf("employment_type = $%d", argIdx))
		args = append(args, *req.EmploymentType)
		argIdx++
	}
	if req.EmploymentStatus != nil {
		updates = append(updates, fmt.Sprintf("employment_status = $%d", argIdx))
		args = append(args, *req.EmploymentStatus)
		argIdx++
	}
	if req.WorkSchedule != nil {
		updates = append(updates, fmt.Sprintf("work_schedule = $%d", argIdx))
		args = append(args, *req.WorkSchedule)
		argIdx++
	}
	if req.WorkHoursPerWeek != nil {
		updates = append(updates, fmt.Sprintf("work_hours_per_week = $%d", argIdx))
		args = append(args, *req.WorkHoursPerWeek)
		argIdx++
	}
	if req.ManagerID != nil {
		updates = append(updates, fmt.Sprintf("manager_id = $%d", argIdx))
		args = append(args, *req.ManagerID)
		argIdx++
	}
	if req.ProbationEndDate != nil {
		updates = append(updates, fmt.Sprintf("probation_end_date = $%d", argIdx))
		args = append(args, *req.ProbationEndDate)
		argIdx++
	}
	if req.ProbationPassed != nil {
		updates = append(updates, fmt.Sprintf("probation_passed = $%d", argIdx))
		args = append(args, *req.ProbationPassed)
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

	query := fmt.Sprintf("UPDATE hrm_employees SET %s WHERE id = $%d",
		strings.Join(updates, ", "),
		argIdx,
	)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code.Name() == "unique_violation" {
				return storage.ErrDuplicate
			}
			if pqErr.Code.Name() == "foreign_key_violation" {
				return storage.ErrForeignKeyViolation
			}
		}
		return fmt.Errorf("%s: failed to update employee: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteEmployee deletes an employee record
func (r *Repo) DeleteEmployee(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteEmployee"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_employees WHERE id = $1", id)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code.Name() == "foreign_key_violation" {
			return storage.ErrForeignKeyViolation
		}
		return fmt.Errorf("%s: failed to delete employee: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// TerminateEmployee terminates an employee by setting termination date and changing status
func (r *Repo) TerminateEmployee(ctx context.Context, id int64, terminationDate time.Time) error {
	const op = "storage.repo.TerminateEmployee"

	query := `
		UPDATE hrm_employees
		SET termination_date = $2, employment_status = 'terminated', updated_at = NOW()
		WHERE id = $1 AND termination_date IS NULL
	`

	res, err := r.db.ExecContext(ctx, query, id, terminationDate)
	if err != nil {
		return fmt.Errorf("%s: failed to terminate employee: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// CountEmployees returns count of employees matching filter
func (r *Repo) CountEmployees(ctx context.Context, filter hrm.EmployeeFilter) (int, error) {
	const op = "storage.repo.CountEmployees"

	var query strings.Builder
	query.WriteString(`
		SELECT COUNT(e.id)
		FROM hrm_employees e
		LEFT JOIN contacts c ON e.contact_id = c.id
		WHERE 1=1
	`)

	var args []interface{}
	argIdx := 1

	if filter.OrganizationID != nil {
		query.WriteString(fmt.Sprintf(" AND c.organization_id = $%d", argIdx))
		args = append(args, *filter.OrganizationID)
		argIdx++
	}
	if filter.DepartmentID != nil {
		query.WriteString(fmt.Sprintf(" AND c.department_id = $%d", argIdx))
		args = append(args, *filter.DepartmentID)
		argIdx++
	}
	if filter.EmploymentStatus != nil {
		query.WriteString(fmt.Sprintf(" AND e.employment_status = $%d", argIdx))
		args = append(args, *filter.EmploymentStatus)
		argIdx++
	}

	var count int
	err := r.db.QueryRowContext(ctx, query.String(), args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("%s: failed to count employees: %w", op, err)
	}

	return count, nil
}

// Helper function to scan employee from row
func (r *Repo) scanEmployee(row *sql.Row) (*hrmmodel.Employee, error) {
	var emp hrmmodel.Employee
	var userID, managerID sql.NullInt64
	var employeeNumber, workSchedule, notes sql.NullString
	var workHoursPerWeek sql.NullFloat64
	var terminationDate, probationEndDate, updatedAt sql.NullTime

	// Contact fields
	var contactID sql.NullInt64
	var contactName, contactEmail, contactPhone, contactIPPhone sql.NullString
	var contactDOB sql.NullTime

	// Organization, Department, Position
	var orgID sql.NullInt64
	var orgName sql.NullString
	var deptID sql.NullInt64
	var deptName sql.NullString
	var posID sql.NullInt64
	var posName sql.NullString

	err := row.Scan(
		&emp.ID, &emp.ContactID, &userID, &employeeNumber, &emp.HireDate,
		&terminationDate, &emp.EmploymentType, &emp.EmploymentStatus,
		&workSchedule, &workHoursPerWeek, &managerID,
		&probationEndDate, &emp.ProbationPassed, &notes,
		&emp.CreatedAt, &updatedAt,
		&contactID, &contactName, &contactEmail, &contactPhone, &contactIPPhone, &contactDOB,
		&orgID, &orgName,
		&deptID, &deptName,
		&posID, &posName,
	)
	if err != nil {
		return nil, err
	}

	// Map nullable fields
	if userID.Valid {
		emp.UserID = &userID.Int64
	}
	if employeeNumber.Valid {
		emp.EmployeeNumber = &employeeNumber.String
	}
	if terminationDate.Valid {
		emp.TerminationDate = &terminationDate.Time
	}
	if workSchedule.Valid {
		emp.WorkSchedule = &workSchedule.String
	}
	if workHoursPerWeek.Valid {
		emp.WorkHoursPerWeek = &workHoursPerWeek.Float64
	}
	if managerID.Valid {
		emp.ManagerID = &managerID.Int64
	}
	if probationEndDate.Valid {
		emp.ProbationEndDate = &probationEndDate.Time
	}
	if notes.Valid {
		emp.Notes = &notes.String
	}
	if updatedAt.Valid {
		emp.UpdatedAt = &updatedAt.Time
	}

	// Build contact
	if contactID.Valid {
		emp.Contact = &contact.Model{
			ID:   contactID.Int64,
			Name: contactName.String,
		}
		if contactEmail.Valid {
			emp.Contact.Email = &contactEmail.String
		}
		if contactPhone.Valid {
			emp.Contact.Phone = &contactPhone.String
		}
		if contactIPPhone.Valid {
			emp.Contact.IPPhone = &contactIPPhone.String
		}
		if contactDOB.Valid {
			emp.Contact.DOB = &contactDOB.Time
		}
	}

	// Build organization
	if orgID.Valid && orgName.Valid {
		emp.Organization = &organization.Model{
			ID:   orgID.Int64,
			Name: orgName.String,
		}
	}

	// Build department
	if deptID.Valid && deptName.Valid {
		emp.Department = &department.Model{
			ID:   deptID.Int64,
			Name: deptName.String,
		}
	}

	// Build position
	if posID.Valid && posName.Valid {
		emp.Position = &position.Model{
			ID:   posID.Int64,
			Name: posName.String,
		}
	}

	return &emp, nil
}

// Helper to scan employee from rows
func (r *Repo) scanEmployeeRow(rows *sql.Rows) (*hrmmodel.Employee, error) {
	var emp hrmmodel.Employee
	var userID, managerID sql.NullInt64
	var employeeNumber, workSchedule, notes sql.NullString
	var workHoursPerWeek sql.NullFloat64
	var terminationDate, probationEndDate, updatedAt sql.NullTime

	// Contact fields
	var contactID sql.NullInt64
	var contactName, contactEmail, contactPhone, contactIPPhone sql.NullString
	var contactDOB sql.NullTime

	// Organization, Department, Position
	var orgID sql.NullInt64
	var orgName sql.NullString
	var deptID sql.NullInt64
	var deptName sql.NullString
	var posID sql.NullInt64
	var posName sql.NullString

	err := rows.Scan(
		&emp.ID, &emp.ContactID, &userID, &employeeNumber, &emp.HireDate,
		&terminationDate, &emp.EmploymentType, &emp.EmploymentStatus,
		&workSchedule, &workHoursPerWeek, &managerID,
		&probationEndDate, &emp.ProbationPassed, &notes,
		&emp.CreatedAt, &updatedAt,
		&contactID, &contactName, &contactEmail, &contactPhone, &contactIPPhone, &contactDOB,
		&orgID, &orgName,
		&deptID, &deptName,
		&posID, &posName,
	)
	if err != nil {
		return nil, err
	}

	// Map nullable fields (same as scanEmployee)
	if userID.Valid {
		emp.UserID = &userID.Int64
	}
	if employeeNumber.Valid {
		emp.EmployeeNumber = &employeeNumber.String
	}
	if terminationDate.Valid {
		emp.TerminationDate = &terminationDate.Time
	}
	if workSchedule.Valid {
		emp.WorkSchedule = &workSchedule.String
	}
	if workHoursPerWeek.Valid {
		emp.WorkHoursPerWeek = &workHoursPerWeek.Float64
	}
	if managerID.Valid {
		emp.ManagerID = &managerID.Int64
	}
	if probationEndDate.Valid {
		emp.ProbationEndDate = &probationEndDate.Time
	}
	if notes.Valid {
		emp.Notes = &notes.String
	}
	if updatedAt.Valid {
		emp.UpdatedAt = &updatedAt.Time
	}

	// Build contact
	if contactID.Valid {
		emp.Contact = &contact.Model{
			ID:   contactID.Int64,
			Name: contactName.String,
		}
		if contactEmail.Valid {
			emp.Contact.Email = &contactEmail.String
		}
		if contactPhone.Valid {
			emp.Contact.Phone = &contactPhone.String
		}
		if contactIPPhone.Valid {
			emp.Contact.IPPhone = &contactIPPhone.String
		}
		if contactDOB.Valid {
			emp.Contact.DOB = &contactDOB.Time
		}
	}

	// Build organization
	if orgID.Valid && orgName.Valid {
		emp.Organization = &organization.Model{
			ID:   orgID.Int64,
			Name: orgName.String,
		}
	}

	// Build department
	if deptID.Valid && deptName.Valid {
		emp.Department = &department.Model{
			ID:   deptID.Int64,
			Name: deptName.String,
		}
	}

	// Build position
	if posID.Valid && posName.Valid {
		emp.Position = &position.Model{
			ID:   posID.Int64,
			Name: posName.String,
		}
	}

	return &emp, nil
}

// --- Personnel Document Operations ---

// AddPersonnelDocument creates a new personnel document
func (r *Repo) AddPersonnelDocument(ctx context.Context, req hrm.AddPersonnelDocumentRequest) (int64, error) {
	const op = "storage.repo.AddPersonnelDocument"

	const query = `
		INSERT INTO hrm_personnel_documents (
			employee_id, document_type, document_number, document_series,
			issued_by, issued_date, expiry_date, file_id, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.EmployeeID,
		req.DocumentType,
		req.DocumentNumber,
		req.DocumentSeries,
		req.IssuedBy,
		req.IssuedDate,
		req.ExpiryDate,
		req.FileID,
		req.Notes,
	).Scan(&id)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code.Name() == "foreign_key_violation" {
				return 0, storage.ErrForeignKeyViolation
			}
		}
		return 0, fmt.Errorf("%s: failed to insert personnel document: %w", op, err)
	}

	return id, nil
}

// GetPersonnelDocumentByID retrieves a personnel document by ID
func (r *Repo) GetPersonnelDocumentByID(ctx context.Context, id int64) (*hrmmodel.PersonnelDocument, error) {
	const op = "storage.repo.GetPersonnelDocumentByID"

	const query = `
		SELECT id, employee_id, document_type, document_number, document_series,
			issued_by, issued_date, expiry_date, file_id, notes,
			is_verified, verified_by, verified_at, created_at, updated_at
		FROM hrm_personnel_documents
		WHERE id = $1`

	var doc hrmmodel.PersonnelDocument
	var docNumber, docSeries, issuedBy, notes sql.NullString
	var issuedDate, expiryDate, verifiedAt, updatedAt sql.NullTime
	var fileID, verifiedBy sql.NullInt64

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&doc.ID, &doc.EmployeeID, &doc.DocumentType, &docNumber, &docSeries,
		&issuedBy, &issuedDate, &expiryDate, &fileID, &notes,
		&doc.IsVerified, &verifiedBy, &verifiedAt, &doc.CreatedAt, &updatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get personnel document: %w", op, err)
	}

	if docNumber.Valid {
		doc.DocumentNumber = &docNumber.String
	}
	if docSeries.Valid {
		doc.DocumentSeries = &docSeries.String
	}
	if issuedBy.Valid {
		doc.IssuedBy = &issuedBy.String
	}
	if issuedDate.Valid {
		doc.IssuedDate = &issuedDate.Time
	}
	if expiryDate.Valid {
		doc.ExpiryDate = &expiryDate.Time
	}
	if fileID.Valid {
		doc.FileID = &fileID.Int64
	}
	if notes.Valid {
		doc.Notes = &notes.String
	}
	if verifiedBy.Valid {
		doc.VerifiedBy = &verifiedBy.Int64
	}
	if verifiedAt.Valid {
		doc.VerifiedAt = &verifiedAt.Time
	}
	if updatedAt.Valid {
		doc.UpdatedAt = &updatedAt.Time
	}

	return &doc, nil
}

// GetPersonnelDocuments retrieves personnel documents with filters
func (r *Repo) GetPersonnelDocuments(ctx context.Context, filter hrm.PersonnelDocumentFilter) ([]*hrmmodel.PersonnelDocument, error) {
	const op = "storage.repo.GetPersonnelDocuments"

	var query strings.Builder
	query.WriteString(`
		SELECT id, employee_id, document_type, document_number, document_series,
			issued_by, issued_date, expiry_date, file_id, notes,
			is_verified, verified_by, verified_at, created_at, updated_at
		FROM hrm_personnel_documents
		WHERE 1=1
	`)

	var args []interface{}
	argIdx := 1

	if filter.EmployeeID != nil {
		query.WriteString(fmt.Sprintf(" AND employee_id = $%d", argIdx))
		args = append(args, *filter.EmployeeID)
		argIdx++
	}
	if filter.DocumentType != nil {
		query.WriteString(fmt.Sprintf(" AND document_type = $%d", argIdx))
		args = append(args, *filter.DocumentType)
		argIdx++
	}
	if filter.IsVerified != nil {
		query.WriteString(fmt.Sprintf(" AND is_verified = $%d", argIdx))
		args = append(args, *filter.IsVerified)
		argIdx++
	}
	if filter.ExpiringDays != nil {
		query.WriteString(fmt.Sprintf(" AND expiry_date IS NOT NULL AND expiry_date <= CURRENT_DATE + INTERVAL '%d days'", *filter.ExpiringDays))
	}

	query.WriteString(" ORDER BY created_at DESC")

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query personnel documents: %w", op, err)
	}
	defer rows.Close()

	var docs []*hrmmodel.PersonnelDocument
	for rows.Next() {
		var doc hrmmodel.PersonnelDocument
		var docNumber, docSeries, issuedBy, notes sql.NullString
		var issuedDate, expiryDate, verifiedAt, updatedAt sql.NullTime
		var fileID, verifiedBy sql.NullInt64

		err := rows.Scan(
			&doc.ID, &doc.EmployeeID, &doc.DocumentType, &docNumber, &docSeries,
			&issuedBy, &issuedDate, &expiryDate, &fileID, &notes,
			&doc.IsVerified, &verifiedBy, &verifiedAt, &doc.CreatedAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan personnel document: %w", op, err)
		}

		if docNumber.Valid {
			doc.DocumentNumber = &docNumber.String
		}
		if docSeries.Valid {
			doc.DocumentSeries = &docSeries.String
		}
		if issuedBy.Valid {
			doc.IssuedBy = &issuedBy.String
		}
		if issuedDate.Valid {
			doc.IssuedDate = &issuedDate.Time
		}
		if expiryDate.Valid {
			doc.ExpiryDate = &expiryDate.Time
		}
		if fileID.Valid {
			doc.FileID = &fileID.Int64
		}
		if notes.Valid {
			doc.Notes = &notes.String
		}
		if verifiedBy.Valid {
			doc.VerifiedBy = &verifiedBy.Int64
		}
		if verifiedAt.Valid {
			doc.VerifiedAt = &verifiedAt.Time
		}
		if updatedAt.Valid {
			doc.UpdatedAt = &updatedAt.Time
		}

		docs = append(docs, &doc)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if docs == nil {
		docs = make([]*hrmmodel.PersonnelDocument, 0)
	}

	return docs, nil
}

// EditPersonnelDocument updates a personnel document
func (r *Repo) EditPersonnelDocument(ctx context.Context, id int64, req hrm.EditPersonnelDocumentRequest) error {
	const op = "storage.repo.EditPersonnelDocument"

	var updates []string
	var args []interface{}
	argIdx := 1

	if req.DocumentType != nil {
		updates = append(updates, fmt.Sprintf("document_type = $%d", argIdx))
		args = append(args, *req.DocumentType)
		argIdx++
	}
	if req.DocumentNumber != nil {
		updates = append(updates, fmt.Sprintf("document_number = $%d", argIdx))
		args = append(args, *req.DocumentNumber)
		argIdx++
	}
	if req.DocumentSeries != nil {
		updates = append(updates, fmt.Sprintf("document_series = $%d", argIdx))
		args = append(args, *req.DocumentSeries)
		argIdx++
	}
	if req.IssuedBy != nil {
		updates = append(updates, fmt.Sprintf("issued_by = $%d", argIdx))
		args = append(args, *req.IssuedBy)
		argIdx++
	}
	if req.IssuedDate != nil {
		updates = append(updates, fmt.Sprintf("issued_date = $%d", argIdx))
		args = append(args, *req.IssuedDate)
		argIdx++
	}
	if req.ExpiryDate != nil {
		updates = append(updates, fmt.Sprintf("expiry_date = $%d", argIdx))
		args = append(args, *req.ExpiryDate)
		argIdx++
	}
	if req.FileID != nil {
		updates = append(updates, fmt.Sprintf("file_id = $%d", argIdx))
		args = append(args, *req.FileID)
		argIdx++
	}
	if req.Notes != nil {
		updates = append(updates, fmt.Sprintf("notes = $%d", argIdx))
		args = append(args, *req.Notes)
		argIdx++
	}
	if req.IsVerified != nil {
		updates = append(updates, fmt.Sprintf("is_verified = $%d", argIdx))
		args = append(args, *req.IsVerified)
		argIdx++
	}

	if len(updates) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE hrm_personnel_documents SET %s WHERE id = $%d",
		strings.Join(updates, ", "),
		argIdx,
	)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to update personnel document: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// VerifyPersonnelDocument marks a document as verified
func (r *Repo) VerifyPersonnelDocument(ctx context.Context, id int64, verifiedBy int64) error {
	const op = "storage.repo.VerifyPersonnelDocument"

	const query = `
		UPDATE hrm_personnel_documents
		SET is_verified = TRUE, verified_by = $1, verified_at = $2
		WHERE id = $3`

	res, err := r.db.ExecContext(ctx, query, verifiedBy, time.Now(), id)
	if err != nil {
		return fmt.Errorf("%s: failed to verify document: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeletePersonnelDocument deletes a personnel document
func (r *Repo) DeletePersonnelDocument(ctx context.Context, id int64) error {
	const op = "storage.repo.DeletePersonnelDocument"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_personnel_documents WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("%s: failed to delete personnel document: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- Transfer Operations ---

// AddTransfer creates a new transfer record
func (r *Repo) AddTransfer(ctx context.Context, req hrm.AddTransferRequest) (int64, error) {
	const op = "storage.repo.AddTransfer"

	const query = `
		INSERT INTO hrm_transfers (
			employee_id, from_department_id, from_position_id, from_organization_id,
			to_department_id, to_position_id, to_organization_id,
			transfer_type, transfer_reason, effective_date,
			order_number, order_date, order_file_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.EmployeeID,
		req.FromDepartmentID,
		req.FromPositionID,
		req.FromOrganizationID,
		req.ToDepartmentID,
		req.ToPositionID,
		req.ToOrganizationID,
		req.TransferType,
		req.TransferReason,
		req.EffectiveDate,
		req.OrderNumber,
		req.OrderDate,
		req.OrderFileID,
	).Scan(&id)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code.Name() == "foreign_key_violation" {
				return 0, storage.ErrForeignKeyViolation
			}
		}
		return 0, fmt.Errorf("%s: failed to insert transfer: %w", op, err)
	}

	return id, nil
}

// GetTransferByID retrieves a transfer by ID
func (r *Repo) GetTransferByID(ctx context.Context, id int64) (*hrmmodel.Transfer, error) {
	const op = "storage.repo.GetTransferByID"

	const query = `
		SELECT id, employee_id, from_department_id, from_position_id, from_organization_id,
			to_department_id, to_position_id, to_organization_id,
			transfer_type, transfer_reason, effective_date,
			order_number, order_date, order_file_id,
			approved_by, approved_at, created_at, updated_at
		FROM hrm_transfers
		WHERE id = $1`

	var t hrmmodel.Transfer
	var fromDeptID, fromPosID, fromOrgID sql.NullInt64
	var toDeptID, toPosID, toOrgID sql.NullInt64
	var transferReason, orderNumber sql.NullString
	var orderDate, approvedAt, updatedAt sql.NullTime
	var orderFileID, approvedBy sql.NullInt64

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&t.ID, &t.EmployeeID, &fromDeptID, &fromPosID, &fromOrgID,
		&toDeptID, &toPosID, &toOrgID,
		&t.TransferType, &transferReason, &t.EffectiveDate,
		&orderNumber, &orderDate, &orderFileID,
		&approvedBy, &approvedAt, &t.CreatedAt, &updatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get transfer: %w", op, err)
	}

	if fromDeptID.Valid {
		t.FromDepartmentID = &fromDeptID.Int64
	}
	if fromPosID.Valid {
		t.FromPositionID = &fromPosID.Int64
	}
	if fromOrgID.Valid {
		t.FromOrganizationID = &fromOrgID.Int64
	}
	if toDeptID.Valid {
		t.ToDepartmentID = &toDeptID.Int64
	}
	if toPosID.Valid {
		t.ToPositionID = &toPosID.Int64
	}
	if toOrgID.Valid {
		t.ToOrganizationID = &toOrgID.Int64
	}
	if transferReason.Valid {
		t.TransferReason = &transferReason.String
	}
	if orderNumber.Valid {
		t.OrderNumber = &orderNumber.String
	}
	if orderDate.Valid {
		t.OrderDate = &orderDate.Time
	}
	if orderFileID.Valid {
		t.OrderFileID = &orderFileID.Int64
	}
	if approvedBy.Valid {
		t.ApprovedBy = &approvedBy.Int64
	}
	if approvedAt.Valid {
		t.ApprovedAt = &approvedAt.Time
	}
	if updatedAt.Valid {
		t.UpdatedAt = &updatedAt.Time
	}

	return &t, nil
}

// GetTransfers retrieves transfers with filters
func (r *Repo) GetTransfers(ctx context.Context, filter hrm.TransferFilter) ([]*hrmmodel.Transfer, error) {
	const op = "storage.repo.GetTransfers"

	var query strings.Builder
	query.WriteString(`
		SELECT id, employee_id, from_department_id, from_position_id, from_organization_id,
			to_department_id, to_position_id, to_organization_id,
			transfer_type, transfer_reason, effective_date,
			order_number, order_date, order_file_id,
			approved_by, approved_at, created_at, updated_at
		FROM hrm_transfers
		WHERE 1=1
	`)

	var args []interface{}
	argIdx := 1

	if filter.EmployeeID != nil {
		query.WriteString(fmt.Sprintf(" AND employee_id = $%d", argIdx))
		args = append(args, *filter.EmployeeID)
		argIdx++
	}
	if filter.TransferType != nil {
		query.WriteString(fmt.Sprintf(" AND transfer_type = $%d", argIdx))
		args = append(args, *filter.TransferType)
		argIdx++
	}
	if filter.FromDate != nil {
		query.WriteString(fmt.Sprintf(" AND effective_date >= $%d", argIdx))
		args = append(args, *filter.FromDate)
		argIdx++
	}
	if filter.ToDate != nil {
		query.WriteString(fmt.Sprintf(" AND effective_date <= $%d", argIdx))
		args = append(args, *filter.ToDate)
		argIdx++
	}

	query.WriteString(" ORDER BY effective_date DESC")

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query transfers: %w", op, err)
	}
	defer rows.Close()

	var transfers []*hrmmodel.Transfer
	for rows.Next() {
		var t hrmmodel.Transfer
		var fromDeptID, fromPosID, fromOrgID sql.NullInt64
		var toDeptID, toPosID, toOrgID sql.NullInt64
		var transferReason, orderNumber sql.NullString
		var orderDate, approvedAt, updatedAt sql.NullTime
		var orderFileID, approvedBy sql.NullInt64

		err := rows.Scan(
			&t.ID, &t.EmployeeID, &fromDeptID, &fromPosID, &fromOrgID,
			&toDeptID, &toPosID, &toOrgID,
			&t.TransferType, &transferReason, &t.EffectiveDate,
			&orderNumber, &orderDate, &orderFileID,
			&approvedBy, &approvedAt, &t.CreatedAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan transfer: %w", op, err)
		}

		if fromDeptID.Valid {
			t.FromDepartmentID = &fromDeptID.Int64
		}
		if fromPosID.Valid {
			t.FromPositionID = &fromPosID.Int64
		}
		if fromOrgID.Valid {
			t.FromOrganizationID = &fromOrgID.Int64
		}
		if toDeptID.Valid {
			t.ToDepartmentID = &toDeptID.Int64
		}
		if toPosID.Valid {
			t.ToPositionID = &toPosID.Int64
		}
		if toOrgID.Valid {
			t.ToOrganizationID = &toOrgID.Int64
		}
		if transferReason.Valid {
			t.TransferReason = &transferReason.String
		}
		if orderNumber.Valid {
			t.OrderNumber = &orderNumber.String
		}
		if orderDate.Valid {
			t.OrderDate = &orderDate.Time
		}
		if orderFileID.Valid {
			t.OrderFileID = &orderFileID.Int64
		}
		if approvedBy.Valid {
			t.ApprovedBy = &approvedBy.Int64
		}
		if approvedAt.Valid {
			t.ApprovedAt = &approvedAt.Time
		}
		if updatedAt.Valid {
			t.UpdatedAt = &updatedAt.Time
		}

		transfers = append(transfers, &t)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if transfers == nil {
		transfers = make([]*hrmmodel.Transfer, 0)
	}

	return transfers, nil
}

// ApproveTransfer approves a transfer
func (r *Repo) ApproveTransfer(ctx context.Context, id int64, approvedBy int64) error {
	const op = "storage.repo.ApproveTransfer"

	const query = `
		UPDATE hrm_transfers
		SET approved_by = $1, approved_at = $2
		WHERE id = $3`

	res, err := r.db.ExecContext(ctx, query, approvedBy, time.Now(), id)
	if err != nil {
		return fmt.Errorf("%s: failed to approve transfer: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteTransfer deletes a transfer record
func (r *Repo) DeleteTransfer(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteTransfer"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_transfers WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("%s: failed to delete transfer: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// GetPendingApprovalsForManager returns pending approvals for a manager
// This includes vacation requests, document approvals, etc.
func (r *Repo) GetPendingApprovalsForManager(ctx context.Context, managerID int64) ([]interface{}, error) {
	const op = "storage.repo.GetPendingApprovalsForManager"

	var results []interface{}

	// Get pending vacation requests for employees managed by this manager
	vacationQuery := `
		SELECT v.id, 'vacation_request' as type,
			CONCAT('Vacation request from ', c.name) as title,
			'pending' as status
		FROM hrm_vacations v
		JOIN hrm_employees e ON v.employee_id = e.id
		JOIN contacts c ON e.contact_id = c.id
		WHERE e.manager_id = $1 AND v.status = 'pending'
		ORDER BY v.created_at DESC
		LIMIT 20`

	vacRows, err := r.db.QueryContext(ctx, vacationQuery, managerID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query vacation approvals: %w", op, err)
	}
	defer vacRows.Close()

	for vacRows.Next() {
		var id int64
		var taskType, title, status string
		if err := vacRows.Scan(&id, &taskType, &title, &status); err != nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"id":     id,
			"type":   taskType,
			"title":  title,
			"status": status,
		})
	}

	// Get pending timesheet approvals
	timesheetQuery := `
		SELECT t.id, 'timesheet_approval' as type,
			CONCAT('Timesheet for ', c.name, ' - ', t.year, '/', t.month) as title,
			'pending' as status
		FROM hrm_timesheets t
		JOIN hrm_employees e ON t.employee_id = e.id
		JOIN contacts c ON e.contact_id = c.id
		WHERE e.manager_id = $1 AND t.status = 'submitted'
		ORDER BY t.created_at DESC
		LIMIT 20`

	tsRows, err := r.db.QueryContext(ctx, timesheetQuery, managerID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query timesheet approvals: %w", op, err)
	}
	defer tsRows.Close()

	for tsRows.Next() {
		var id int64
		var taskType, title, status string
		if err := tsRows.Scan(&id, &taskType, &title, &status); err != nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"id":     id,
			"type":   taskType,
			"title":  title,
			"status": status,
		})
	}

	if results == nil {
		results = make([]interface{}, 0)
	}

	return results, nil
}
