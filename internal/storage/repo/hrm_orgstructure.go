package repo

import (
	"context"
	"database/sql"
	"fmt"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/hrm/orgstructure"
	"srmt-admin/internal/storage"
	"strings"
)

// ==================== Org Units ====================

func (r *Repo) CreateOrgUnit(ctx context.Context, req dto.CreateOrgUnitRequest) (int64, error) {
	const op = "repo.CreateOrgUnit"

	query := `
		INSERT INTO org_units (name, type, parent_id, head_id, department_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.Name, req.Type, req.ParentID, req.HeadID, req.DepartmentID,
	).Scan(&id)
	if err != nil {
		if translated := r.translator.Translate(err, op); translated != nil {
			return 0, translated
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

func (r *Repo) GetOrgUnitByID(ctx context.Context, id int64) (*orgstructure.OrgUnit, error) {
	const op = "repo.GetOrgUnitByID"

	query := `
		SELECT ou.id, ou.name, ou.type, ou.parent_id, ou.head_id, COALESCE(c.fio, ''),
			   ou.department_id, ou.level, ou.created_at, ou.updated_at
		FROM org_units ou
		LEFT JOIN contacts c ON ou.head_id = c.id
		WHERE ou.id = $1`

	unit, err := scanOrgUnit(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrOrgUnitNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return unit, nil
}

func (r *Repo) GetAllOrgUnits(ctx context.Context) ([]*orgstructure.OrgUnit, error) {
	const op = "repo.GetAllOrgUnits"

	query := `
		SELECT ou.id, ou.name, ou.type, ou.parent_id, ou.head_id, COALESCE(c.fio, ''),
			   ou.department_id, ou.level, ou.created_at, ou.updated_at
		FROM org_units ou
		LEFT JOIN contacts c ON ou.head_id = c.id
		ORDER BY ou.level, ou.name`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var units []*orgstructure.OrgUnit
	for rows.Next() {
		unit, err := scanOrgUnit(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		units = append(units, unit)
	}
	return units, rows.Err()
}

func (r *Repo) UpdateOrgUnit(ctx context.Context, id int64, req dto.UpdateOrgUnitRequest) error {
	const op = "repo.UpdateOrgUnit"

	var setClauses []string
	var args []interface{}
	argIdx := 1

	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Type != nil {
		setClauses = append(setClauses, fmt.Sprintf("type = $%d", argIdx))
		args = append(args, *req.Type)
		argIdx++
	}
	if req.ParentID != nil {
		setClauses = append(setClauses, fmt.Sprintf("parent_id = $%d", argIdx))
		args = append(args, *req.ParentID)
		argIdx++
	}
	if req.HeadID != nil {
		setClauses = append(setClauses, fmt.Sprintf("head_id = $%d", argIdx))
		args = append(args, *req.HeadID)
		argIdx++
	}
	if req.DepartmentID != nil {
		setClauses = append(setClauses, fmt.Sprintf("department_id = $%d", argIdx))
		args = append(args, *req.DepartmentID)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE org_units SET %s WHERE id = $%d",
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
		return storage.ErrOrgUnitNotFound
	}
	return nil
}

func (r *Repo) DeleteOrgUnit(ctx context.Context, id int64) error {
	const op = "repo.DeleteOrgUnit"

	result, err := r.db.ExecContext(ctx, "DELETE FROM org_units WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrOrgUnitNotFound
	}
	return nil
}

func (r *Repo) HasChildOrgUnits(ctx context.Context, id int64) (bool, error) {
	const op = "repo.HasChildOrgUnits"

	var count int
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM org_units WHERE parent_id = $1", id).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	return count > 0, nil
}

// ==================== Employees ====================

func (r *Repo) GetUnitEmployees(ctx context.Context, unitID int64) ([]*orgstructure.OrgEmployee, error) {
	const op = "repo.GetUnitEmployees"

	// Get the department_id for this unit
	var deptID *int64
	err := r.db.QueryRowContext(ctx,
		"SELECT department_id FROM org_units WHERE id = $1", unitID).Scan(&deptID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrOrgUnitNotFound
		}
		return nil, fmt.Errorf("%s: get unit: %w", op, err)
	}

	if deptID == nil {
		return []*orgstructure.OrgEmployee{}, nil
	}

	query := `
		SELECT c.id, COALESCE(c.fio, ''), COALESCE(p.name, ''), COALESCE(d.name, ''),
			   ou.id,
			   CASE WHEN ou.head_id = c.id THEN true ELSE false END,
			   f.object_key, c.phone, c.email
		FROM contacts c
		LEFT JOIN personnel_records pr ON pr.employee_id = c.id
		LEFT JOIN positions p ON c.position_id = p.id
		LEFT JOIN departments d ON c.department_id = d.id
		LEFT JOIN org_units ou ON ou.department_id = c.department_id
		LEFT JOIN files f ON c.icon_id = f.id
		WHERE c.department_id = $1
		ORDER BY c.fio`

	rows, err := r.db.QueryContext(ctx, query, *deptID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var employees []*orgstructure.OrgEmployee
	for rows.Next() {
		emp, err := scanOrgEmployee(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		employees = append(employees, emp)
	}
	return employees, rows.Err()
}

func (r *Repo) GetAllOrgEmployees(ctx context.Context) ([]*orgstructure.OrgEmployee, error) {
	const op = "repo.GetAllOrgEmployees"

	query := `
		SELECT c.id, COALESCE(c.fio, ''), COALESCE(p.name, ''), COALESCE(d.name, ''),
			   ou.id,
			   CASE WHEN ou.head_id = c.id THEN true ELSE false END,
			   f.object_key, c.phone, c.email
		FROM contacts c
		LEFT JOIN personnel_records pr ON pr.employee_id = c.id
		LEFT JOIN positions p ON c.position_id = p.id
		LEFT JOIN departments d ON c.department_id = d.id
		LEFT JOIN org_units ou ON ou.department_id = c.department_id
		LEFT JOIN files f ON c.icon_id = f.id
		ORDER BY c.fio`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var employees []*orgstructure.OrgEmployee
	for rows.Next() {
		emp, err := scanOrgEmployee(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		employees = append(employees, emp)
	}
	return employees, rows.Err()
}

// ==================== Scanners ====================

func scanOrgUnit(s scannable) (*orgstructure.OrgUnit, error) {
	var ou orgstructure.OrgUnit
	var headName *string
	err := s.Scan(
		&ou.ID, &ou.Name, &ou.Type, &ou.ParentID, &ou.HeadID, &headName,
		&ou.DepartmentID, &ou.Level, &ou.CreatedAt, &ou.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if headName != nil && *headName != "" {
		ou.HeadName = headName
	}
	ou.Children = []orgstructure.OrgUnit{}
	return &ou, nil
}

func scanOrgEmployee(s scannable) (*orgstructure.OrgEmployee, error) {
	var e orgstructure.OrgEmployee
	err := s.Scan(
		&e.ID, &e.Name, &e.Position, &e.Department,
		&e.UnitID, &e.IsHead,
		&e.Avatar, &e.Phone, &e.Email,
	)
	if err != nil {
		return nil, err
	}
	return &e, nil
}
