package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"srmt-admin/internal/lib/model/role"
	"srmt-admin/internal/lib/model/user"
	"srmt-admin/internal/storage"
	"strings"
)

type Repo struct {
	Driver       *sql.DB
	ErrorHandler storage.ErrorTranslator
}

func New(driver *storage.Driver) *Repo {
	return &Repo{driver.DB, driver.Translator}
}

func (s *Repo) Close() error {
	return s.Driver.Close()
}

func (s *Repo) AddUser(ctx context.Context, name, passHash string) (int64, error) {
	const op = "storage.repo.AddUser"

	query := `INSERT INTO users (name, pass_hash) VALUES ($1, $2) RETURNING id`

	stmt, err := s.Driver.Prepare(query)
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

func (s *Repo) GetUserByName(ctx context.Context, name string) (user.Model, error) {
	const op = "storage.repo.GetUserByName"

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
	stmt, err := s.Driver.Prepare(query)
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

func (s *Repo) EditUser(ctx context.Context, id int64, name, passHash string) error {
	const op = "storage.repo.EditUser"

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

	stmt, err := s.Driver.Prepare(query.String())
	if err != nil {
		return fmt.Errorf("%s: failed to prepare statement: %w", op, err)
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(ctx, args...)
	if err != nil {
		if err := s.ErrorHandler.Translate(err, op); err != nil {
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

func (s *Repo) AddRole(ctx context.Context, name string, description string) (int64, error) {
	const op = "storage.repo.AddRole"

	stmt, err := s.Driver.Prepare("INSERT INTO roles(name, description) VALUES($1, $2) RETURNING id")
	if err != nil {
		return 0, fmt.Errorf("%s: failed to prepare statement: %w", op, err)
	}
	defer stmt.Close()

	var id int64
	if err := stmt.QueryRowContext(ctx, name, description).Scan(&id); err != nil {
		if err := s.ErrorHandler.Translate(err, op); err != nil {
			return 0, err
		}
		return 0, fmt.Errorf("%s: failed to execute statement: %w", op, err)
	}

	return id, nil
}

func (s *Repo) EditRole(ctx context.Context, id int64, name, description string) error {
	const op = "storage.repo.EditRole"

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

	stmt, err := s.Driver.Prepare(query.String())
	if err != nil {
		return fmt.Errorf("%s: failed to prepare statement: %w", op, err)
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(ctx, args...)
	if err != nil {
		if err := s.ErrorHandler.Translate(err, op); err != nil {
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

func (s *Repo) DeleteRole(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteRole"

	stmt, err := s.Driver.Prepare("DELETE FROM roles WHERE id = $1")
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

func (s *Repo) GetRoleByName(ctx context.Context, name string) (role.Model, error) {
	const op = "storage.repo.GetRoleByName"
	stmt, err := s.Driver.Prepare("SELECT id, name FROM roles WHERE name = $1")
	if err != nil {
		return role.Model{}, fmt.Errorf("%s: failed to prepare statement: %w", op, err)
	}
	defer stmt.Close()

	row := stmt.QueryRowContext(ctx, name)

	var r role.Model
	if err := row.Scan(&r.ID, &r.Name); err != nil {
		// Если Scan вернул sql.ErrNoRows, значит запись не найдена.
		if errors.Is(err, sql.ErrNoRows) {
			return role.Model{}, storage.ErrRoleNotFound
		}
		return role.Model{}, fmt.Errorf("%s: failed to scan row: %w", op, err)
	}

	return r, nil
}

func (s *Repo) AssignRole(ctx context.Context, userID, roleID int64) error {
	const op = "storage.repo.AssignRole"

	stmt, err := s.Driver.Prepare("INSERT INTO users_roles(user_id, role_id) VALUES($1, $2)")
	if err != nil {
		return fmt.Errorf("%s: failed to prepare statement: %w", op, err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, userID, roleID)
	if err != nil {
		if err := s.ErrorHandler.Translate(err, op); err != nil {
			return err
		}
		return fmt.Errorf("%s: failed to execute statement: %w", op, err)
	}

	return nil
}

func (s *Repo) RevokeRole(ctx context.Context, userID, roleID int64) error {
	const op = "storage.repo.RevokeRole"

	stmt, err := s.Driver.Prepare("DELETE FROM users_roles WHERE user_id = $1 AND role_id = $2")
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

func (s *Repo) GetUserRoles(ctx context.Context, userID int64) ([]role.Model, error) {
	const op = "storage.repo.GetUserRoles"

	const query = `
		SELECT r.id, r.name FROM roles r
		JOIN users_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
	`
	stmt, err := s.Driver.Prepare(query)
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

func (s *Repo) AddReservoir(ctx context.Context, name string) (int64, error) {
	const op = "storage.repo.AddReservoir"

	stmt, err := s.Driver.Prepare("INSERT INTO reservoirs(name) VALUES($1) RETURNING id")
	if err != nil {
		return 0, fmt.Errorf("%s: failed to prepare statement: %w", op, err)
	}
	defer stmt.Close()

	var id int64
	if err := stmt.QueryRowContext(ctx, name).Scan(&id); err != nil {
		if err := s.ErrorHandler.Translate(err, op); err != nil {
			return 0, err
		}
		return 0, fmt.Errorf("%s: failed to execute statement: %w", op, err)
	}

	return id, nil
}

func (s *Repo) SetAndijanIndicator(ctx context.Context, height float64) (int64, error) {
	const op = "storage.repo.SetAndijanIndicator"

	query := `INSERT INTO indicator_height (height, res_id) VALUES ($1, $2) RETURNING id`

	reservoirID, err := s.GetAndijanReservoir(ctx)

	stmt, err := s.Driver.Prepare(query)
	if err != nil {
		return 0, fmt.Errorf("%s: failed to prepare statement: %w", op, err)
	}
	defer stmt.Close()

	var id int64
	if err := stmt.QueryRowContext(ctx, height, reservoirID).Scan(&id); err != nil {
		if translatedErr := s.ErrorHandler.Translate(err, op); translatedErr != nil {
			return 0, translatedErr
		}
		return 0, fmt.Errorf("%s: failed to execute statement: %w", op, err)
	}

	return id, nil
}

func (s *Repo) GetAndijanReservoir(ctx context.Context) (int64, error) {
	const op = "storage.repo.GetAndijanReservoir"

	stmt, err := s.Driver.Prepare("SELECT id FROM reservoirs WHERE name = $1")
	if err != nil {
		return 0, fmt.Errorf("%s: failed to prepare statement: %w", op, err)
	}
	defer stmt.Close()

	row := stmt.QueryRowContext(ctx, "And")

	var id int64
	if err := row.Scan(&id); err != nil {
		// Если Scan вернул sql.ErrNoRows, значит запись не найдена.
		if errors.Is(err, sql.ErrNoRows) {
			// Рекомендуется использовать специальную ошибку для этого случая,
			// по аналогии с ErrUserNotFound и ErrRoleNotFound.
			return 0, storage.ErrReservoirNotFound
		}
		return 0, fmt.Errorf("%s: failed to scan row: %w", op, err)
	}

	return id, nil
}
