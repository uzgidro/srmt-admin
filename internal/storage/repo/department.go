package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/lib/pq"
	"srmt-admin/internal/lib/model/department"
	"srmt-admin/internal/lib/model/organization"
	"srmt-admin/internal/storage"
	"strings"
)

// AddDepartment реализует интерфейс handlers.department.add.DepartmentAdder
func (r *Repo) AddDepartment(ctx context.Context, name string, description *string, orgID int64) (int64, error) {
	const op = "storage.repo.AddDepartment"

	const query = `
		INSERT INTO departments (name, description, organization_id)
		VALUES ($1, $2, $3)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query, name, description, orgID).Scan(&id)
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
		return 0, fmt.Errorf("%s: failed to insert department: %w", op, err)
	}

	return id, nil
}

// GetAllDepartments реализует интерфейс handlers.department.get_all.DepartmentGetter
func (r *Repo) GetAllDepartments(ctx context.Context, orgID *int64) ([]*department.Model, error) {
	const op = "storage.repo.GetAllDepartments"

	var query strings.Builder
	query.WriteString(`
		SELECT
			d.id, d.name, d.description, d.organization_id, d.created_at, d.updated_at,
			o.id as org_id, o.name as org_name
		FROM
			departments d
		LEFT JOIN
			organizations o ON d.organization_id = o.id
	`)

	var args []interface{}
	if orgID != nil {
		query.WriteString(" WHERE d.organization_id = $1")
		args = append(args, *orgID)
	}

	query.WriteString(" ORDER BY d.name")

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query departments: %w", op, err)
	}
	defer rows.Close()

	var departments []*department.Model
	for rows.Next() {
		var dep department.Model
		var desc sql.NullString

		var orgID sql.NullInt64
		var orgName sql.NullString

		if err := rows.Scan(
			&dep.ID,
			&dep.Name,
			&desc,
			&dep.OrganizationID,
			&dep.CreatedAt,
			&dep.UpdatedAt,
			&orgID,
			&orgName,
		); err != nil {
			return nil, fmt.Errorf("%s: failed to scan department: %w", op, err)
		}

		if desc.Valid {
			dep.Description = &desc.String
		}

		if orgID.Valid && orgName.Valid {
			dep.Organization = &organization.Model{
				ID:   orgID.Int64,
				Name: orgName.String,
			}
		}

		departments = append(departments, &dep)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if departments == nil {
		departments = make([]*department.Model, 0)
	}

	return departments, nil
}

func (r *Repo) GetDepartmentByID(ctx context.Context, id int64) (*department.Model, error) {
	const op = "storage.repo.GetDepartmentByID"

	const query = `
		SELECT
			d.id, d.name, d.description, d.organization_id, d.created_at, d.updated_at,
			o.id as org_id, o.name as org_name
		FROM
			departments d
		LEFT JOIN
			organizations o ON d.organization_id = o.id
		WHERE
			d.id = $1`

	var dep department.Model
	var desc sql.NullString

	var orgID sql.NullInt64
	var orgName sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&dep.ID,
		&dep.Name,
		&desc,
		&dep.OrganizationID,
		&dep.CreatedAt,
		&dep.UpdatedAt,
		&orgID,
		&orgName,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to scan department: %w", op, err)
	}

	if desc.Valid {
		dep.Description = &desc.String
	}

	if orgID.Valid && orgName.Valid {
		dep.Organization = &organization.Model{
			ID:   orgID.Int64,
			Name: orgName.String,
		}
	}

	return &dep, nil
}

// EditDepartment реализует интерфейс handlers.department.update.DepartmentUpdater
func (r *Repo) EditDepartment(ctx context.Context, id int64, name *string, description *string) error {
	const op = "storage.repo.EditDepartment"

	var updates []string
	var args []interface{}
	argID := 1

	if name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argID))
		args = append(args, *name)
		argID++
	}
	if description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argID))
		args = append(args, *description)
		argID++
	}

	if len(updates) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE departments SET %s WHERE id = $%d",
		strings.Join(updates, ", "),
		argID,
	)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code.Name() == "unique_violation" {
				return storage.ErrDuplicate
			}
		}
		return fmt.Errorf("%s: failed to update department: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteDepartment реализует интерфейс handlers.department.delete.DepartmentDeleter
func (r *Repo) DeleteDepartment(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteDepartment"

	res, err := r.db.ExecContext(ctx, "DELETE FROM departments WHERE id = $1", id)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code.Name() == "foreign_key_violation" {
			return storage.ErrForeignKeyViolation
		}
		return fmt.Errorf("%s: failed to delete department: %w", op, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: failed to get affected rows: %w", op, err)
	}

	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}
