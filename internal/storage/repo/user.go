package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"srmt-admin/internal/lib/model/user"
	"srmt-admin/internal/storage"
	"strings"
)

func (r *Repo) AddUser(ctx context.Context, name, passHash string) (int64, error) {
	const op = "storage.user.AddUser"

	query := `INSERT INTO users (name, pass_hash) VALUES ($1, $2) RETURNING id`

	stmt, err := r.db.Prepare(query)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	var id int64
	if err = stmt.QueryRowContext(ctx, name, passHash).Scan(&id); err != nil {
		return 0, fmt.Errorf("%s: execute query: %w", op, err)
	}

	return id, nil
}

func (r *Repo) GetUserByName(ctx context.Context, name string) (user.Model, error) {
	const op = "storage.user.GetUserByName"

	const query = `
		SELECT
			u.id,
			u.name,
			u.pass_hash,
			-- COALESCE нужен, чтобы вернуть пустой массив '[]', если у пользователя нет ролей, вместо NULL.
			COALESCE(
				(SELECT json_agg(r.name)
				 FROM users_roles ur
				 JOIN roles r ON ur.role_id = r.id
				 WHERE ur.user_id = u.id),
				'[]'
			) as roles_json
		FROM
			users u
		WHERE
			u.name = $1
	`
	stmt, err := r.db.Prepare(query)
	if err != nil {
		return user.Model{}, fmt.Errorf("%s: failed to prepare statement: %w", op, err)
	}
	defer stmt.Close()

	row := stmt.QueryRowContext(ctx, name)

	var u user.Model
	var rolesJSON string

	if err := row.Scan(&u.ID, &u.Name, &u.PassHash, &rolesJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user.Model{}, storage.ErrUserNotFound
		}
		return user.Model{}, fmt.Errorf("%s: failed to scan user row: %w", op, err)
	}

	if err := json.Unmarshal([]byte(rolesJSON), &u.Roles); err != nil {
		return user.Model{}, fmt.Errorf("%s: failed to unmarshal roles: %w", op, err)
	}

	return u, nil
}

func (r *Repo) GetUserByID(ctx context.Context, id int64) (user.Model, error) {
	const op = "storage.user.GetUserByName"

	const query = `
		SELECT
			u.id,
			u.name,
			u.pass_hash,
			-- COALESCE нужен, чтобы вернуть пустой массив '[]', если у пользователя нет ролей, вместо NULL.
			COALESCE(
				(SELECT json_agg(r.name)
				 FROM users_roles ur
				 JOIN roles r ON ur.role_id = r.id
				 WHERE ur.user_id = u.id),
				'[]'
			) as roles_json
		FROM
			users u
		WHERE
			u.id = $1
	`
	stmt, err := r.db.Prepare(query)
	if err != nil {
		return user.Model{}, fmt.Errorf("%s: failed to prepare statement: %w", op, err)
	}
	defer stmt.Close()

	row := stmt.QueryRowContext(ctx, id)

	var u user.Model
	var rolesJSON string

	if err := row.Scan(&u.ID, &u.Name, &u.PassHash, &rolesJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user.Model{}, storage.ErrUserNotFound
		}
		return user.Model{}, fmt.Errorf("%s: failed to scan user row: %w", op, err)
	}

	if err := json.Unmarshal([]byte(rolesJSON), &u.Roles); err != nil {
		return user.Model{}, fmt.Errorf("%s: failed to unmarshal roles: %w", op, err)
	}

	return u, nil
}

func (r *Repo) EditUser(ctx context.Context, id int64, name, passHash string) error {
	const op = "storage.user.EditUser"

	var query strings.Builder
	query.WriteString("UPDATE users SET")

	var args []interface{}
	var setClauses []string
	argID := 1

	if name != "" {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argID))
		args = append(args, name)
		argID++
	}
	if passHash != "" {
		setClauses = append(setClauses, fmt.Sprintf("pass_hash = $%d", argID))
		args = append(args, passHash)
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
		return storage.ErrUserNotFound
	}

	return nil
}
