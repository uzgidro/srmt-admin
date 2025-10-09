package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"srmt-admin/internal/lib/model/role"
	"srmt-admin/internal/storage"
	"strings"
)

func (r *Repo) AddRole(ctx context.Context, name string, description string) (int64, error) {
	const op = "storage.role.AddRole"

	stmt, err := r.db.Prepare("INSERT INTO roles(name, description) VALUES($1, $2) RETURNING id")
	if err != nil {
		return 0, fmt.Errorf("%s: failed to prepare statement: %w", op, err)
	}
	defer stmt.Close()

	var id int64
	if err := stmt.QueryRowContext(ctx, name, description).Scan(&id); err != nil {
		if err := r.translator.Translate(err, op); err != nil {
			return 0, err
		}
		return 0, fmt.Errorf("%s: failed to execute statement: %w", op, err)
	}

	return id, nil
}

func (r *Repo) GetAllRoles(ctx context.Context) ([]role.Model, error) {
	const op = "storage.repo.GetAllRoles"

	const query = `SELECT id, name, description FROM roles WHERE id != 1 ORDER BY name`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query roles: %w", op, err)
	}
	defer rows.Close()

	var roles []role.Model
	for rows.Next() {
		var rl role.Model
		if err := rows.Scan(&rl.ID, &rl.Name, &rl.Description); err != nil {
			return nil, fmt.Errorf("%s: failed to scan role row: %w", op, err)
		}
		roles = append(roles, rl)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if roles == nil {
		roles = make([]role.Model, 0)
	}

	return roles, nil
}

func (r *Repo) EditRole(ctx context.Context, id int64, name, description string) error {
	const op = "storage.role.EditRole"

	var query strings.Builder
	query.WriteString("UPDATE roles SET")

	var args []interface{}
	var setClauses []string
	argID := 1

	if name != "" {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argID))
		args = append(args, name)
		argID++
	}
	if description != "" {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argID))
		args = append(args, description)
		argID++
	}

	if len(setClauses) == 0 {
		return fmt.Errorf("%s: fields are empty", op)
	}

	query.WriteString(strings.Join(setClauses, ","))
	query.WriteString(fmt.Sprintf("WHERE id = $%d", argID))
	args = append(args, id)

	stmt, err := r.db.Prepare(query.String())
	if err != nil {
		return fmt.Errorf("%s: failed to prepare statement: %w", op, err)
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(ctx, args...)
	if err != nil {
		if err := r.translator.Translate(err, op); err != nil {
			return err
		}
		return fmt.Errorf("%s: failed to execute statement: %w", op, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: failed to set affected rows: %w", op, err)
	}
	if rowsAffected == 0 {
		return storage.ErrRoleNotFound
	}

	return nil
}

func (r *Repo) DeleteRole(ctx context.Context, id int64) error {
	const op = "storage.role.DeleteRole"

	stmt, err := r.db.Prepare("DELETE FROM roles WHERE id = $1")
	if err != nil {
		return fmt.Errorf("%s: failed to prepare statement: %w", op, err)
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(ctx, id)
	if err != nil {
		return fmt.Errorf("%s: failed to execute statement: %w", op, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: failed to set affected rows: %w", op, err)
	}
	if rowsAffected == 0 {
		return storage.ErrRoleNotFound
	}

	return nil
}

func (r *Repo) GetRoleByName(ctx context.Context, name string) (role.Model, error) {
	const op = "storage.role.GetRoleByName"
	stmt, err := r.db.Prepare("SELECT id, name FROM roles WHERE name = $1")
	if err != nil {
		return role.Model{}, fmt.Errorf("%s: failed to prepare statement: %w", op, err)
	}
	defer stmt.Close()

	row := stmt.QueryRowContext(ctx, name)

	var resp role.Model
	if err := row.Scan(&resp.ID, &resp.Name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return role.Model{}, storage.ErrRoleNotFound
		}
		return role.Model{}, fmt.Errorf("%s: failed to scan row: %w", op, err)
	}

	return resp, nil
}

func (r *Repo) AssignRole(ctx context.Context, userID, roleID int64) error {
	const op = "storage.role.AssignRole"

	stmt, err := r.db.Prepare("INSERT INTO users_roles(user_id, role_id) VALUES($1, $2)")
	if err != nil {
		return fmt.Errorf("%s: failed to prepare statement: %w", op, err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, userID, roleID)
	if err != nil {
		if err := r.translator.Translate(err, op); err != nil {
			return err
		}
		return fmt.Errorf("%s: failed to execute statement: %w", op, err)
	}

	return nil
}

func (r *Repo) AssignRoleToUsers(ctx context.Context, roleID int64, userIDs []int64) error {
	const op = "storage.role.AssignRoleToUsers"

	if len(userIDs) == 0 {
		return nil
	}

	var valueStrings []string
	var valueArgs []interface{}
	paramIndex := 1
	for _, userID := range userIDs {
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d)", paramIndex, paramIndex+1))
		valueArgs = append(valueArgs, userID)
		valueArgs = append(valueArgs, roleID)
		paramIndex += 2
	}

	fullQuery := "INSERT INTO users_roles (user_id, role_id) VALUES " + strings.Join(valueStrings, ",") + " ON CONFLICT DO NOTHING"

	_, err := r.db.ExecContext(ctx, fullQuery, valueArgs...)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to execute bulk insert: %w", op, err)
	}

	return nil
}

func (r *Repo) AssignRolesToUser(ctx context.Context, userID int64, roleIDs []int64) error {
	const op = "storage.role.AssignRolesToUser"

	if len(roleIDs) == 0 {
		return nil
	}

	query := "INSERT INTO users_roles (user_id, role_id) VALUES "

	var valueStrings []string
	var valueArgs []interface{}
	paramIndex := 1
	for _, roleID := range roleIDs {
		// Для каждой роли добавляем пару (userID, roleID)
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d)", paramIndex, paramIndex+1))
		valueArgs = append(valueArgs, userID)
		valueArgs = append(valueArgs, roleID)
		paramIndex += 2
	}

	fullQuery := query + strings.Join(valueStrings, ",") + " ON CONFLICT DO NOTHING"

	_, err := r.db.ExecContext(ctx, fullQuery, valueArgs...)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to execute bulk insert: %w", op, err)
	}

	return nil
}

func (r *Repo) RevokeRole(ctx context.Context, userID, roleID int64) error {
	const op = "storage.role.RevokeRole"

	stmt, err := r.db.Prepare("DELETE FROM users_roles WHERE user_id = $1 AND role_id = $2")
	if err != nil {
		return fmt.Errorf("%s: failed to prepare statement: %w", op, err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, userID, roleID)
	if err != nil {
		return fmt.Errorf("%s: failed to execute statement: %w", op, err)
	}

	return nil
}

func (r *Repo) GetUserRoles(ctx context.Context, userID int64) ([]role.Model, error) {
	const op = "storage.role.GetUserRoles"

	const query = `
		SELECT r.id, r.name FROM roles r
		JOIN users_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
	`
	stmt, err := r.db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to prepare statement: %w", op, err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query roles: %w", op, err)
	}
	defer rows.Close()

	var roles []role.Model
	for rows.Next() {
		var r role.Model
		if err := rows.Scan(&r.ID, &r.Name); err != nil {
			return nil, fmt.Errorf("%s: failed to scan role row: %w", op, err)
		}
		roles = append(roles, r)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	return roles, nil
}
