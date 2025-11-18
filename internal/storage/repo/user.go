package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/contact"
	"srmt-admin/internal/lib/model/department"
	"srmt-admin/internal/lib/model/organization"
	"srmt-admin/internal/lib/model/position"
	"srmt-admin/internal/lib/model/user"
	"srmt-admin/internal/storage"
	"strings"
)

func (r *Repo) AddUser(ctx context.Context, login string, passwordHash []byte, contactID int64) (int64, error) {
	const op = "storage.repo.AddUser"

	const query = `
		INSERT INTO users (login, pass_hash, contact_id, is_active)
		VALUES ($1, $2, $3, TRUE)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query, login, passwordHash, contactID).Scan(&id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return 0, translatedErr // (ErrDuplicate или ErrForeignKeyViolation)
		}
		return 0, fmt.Errorf("%s: failed to insert user: %w", op, err)
	}

	return id, nil
}

func (r *Repo) IsContactLinked(ctx context.Context, contactID int64) (bool, error) {
	const op = "storage.repo.IsContactLinked"

	const query = "SELECT 1 FROM users WHERE contact_id = $1"

	var exists int
	err := r.db.QueryRowContext(ctx, query, contactID).Scan(&exists)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil // Не связан - это не ошибка
		}
		return false, fmt.Errorf("%s: failed to check contact link: %w", op, err)
	}

	return true, nil // Связан
}

func (r *Repo) GetAllUsers(ctx context.Context, filters dto.GetAllUsersFilters) ([]*user.Model, error) {
	const op = "storage.repo.GetAllUsers"

	var query strings.Builder
	query.WriteString(selectUserFields)
	query.WriteString(fromUserJoins)

	var whereClauses []string
	var args []interface{}
	argID := 1

	if filters.OrganizationID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("c.organization_id = $%d", argID))
		args = append(args, *filters.OrganizationID)
		argID++
	}
	if filters.DepartmentID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("c.department_id = $%d", argID))
		args = append(args, *filters.DepartmentID)
		argID++
	}
	if filters.IsActive != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("u.is_active = $%d", argID))
		args = append(args, *filters.IsActive)
		argID++
	}

	if len(whereClauses) > 0 {
		query.WriteString(" WHERE " + strings.Join(whereClauses, " AND "))
	}

	query.WriteString(" ORDER BY c.fio")

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query users: %w", op, err)
	}
	defer rows.Close()

	var users []*user.Model
	var discardPassHash sql.NullString // Пароль не нужен

	for rows.Next() {
		u, err := scanUserRow(rows, &discardPassHash)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan user row: %w", op, err)
		}
		users = append(users, u)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if users == nil {
		users = make([]*user.Model, 0)
	}

	return users, nil
}

func (r *Repo) GetUserByLogin(ctx context.Context, login string) (*user.Model, string, error) {
	const op = "storage.repo.GetUserByLogin"

	query := selectUserFields + fromUserJoins + " WHERE u.login = $1"

	var passHash sql.NullString // (pass_hash может быть NULL)
	row := r.db.QueryRowContext(ctx, query, login)

	u, err := scanUserRow(row, &passHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", storage.ErrNotFound
		}
		return nil, "", fmt.Errorf("%s: failed to scan user: %w", op, err)
	}

	return u, passHash.String, nil
}

func (r *Repo) GetUserByID(ctx context.Context, id int64) (*user.Model, error) {
	const op = "storage.repo.GetUserByID"

	query := selectUserFields + fromUserJoins + " WHERE u.id = $1"

	var discardPassHash sql.NullString // Пароль не нужен
	row := r.db.QueryRowContext(ctx, query, id)

	u, err := scanUserRow(row, &discardPassHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to scan user: %w", op, err)
	}

	return u, nil
}

func (r *Repo) GetUsersByRole(ctx context.Context, roleID int64) ([]user.Model, error) {
	const op = "storage.user.GetUserByRole"

	const query = `
		SELECT u.id, u.login FROM users u
		JOIN users_roles ur ON u.id = ur.user_id
		WHERE ur.role_id = $1
	`

	rows, err := r.db.QueryContext(ctx, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query users: %w", op, err)
	}
	defer rows.Close()

	var users []user.Model
	for rows.Next() {
		var u user.Model
		if err := rows.Scan(&u.ID, &u.Login); err != nil {
			return nil, fmt.Errorf("%s: failed to scan user row: %w", op, err)
		}
		users = append(users, u)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	return users, nil
}

func (r *Repo) EditUser(ctx context.Context, userID int64, passwordHash []byte, req dto.EditUserRequest) error {
	const op = "storage.repo.EditUser"

	// (Динамический UPDATE, как в твоем старом коде)
	var updates []string
	var args []interface{}
	argID := 1

	if req.Login != nil {
		updates = append(updates, fmt.Sprintf("login = $%d", argID))
		args = append(args, *req.Login)
		argID++
	}
	if req.IsActive != nil {
		updates = append(updates, fmt.Sprintf("is_active = $%d", argID))
		args = append(args, *req.IsActive)
		argID++
	}
	if passwordHash != nil { // (Пароль пришел из хендлера)
		updates = append(updates, fmt.Sprintf("pass_hash = $%d", argID))
		args = append(args, passwordHash)
		argID++
	}

	if len(updates) == 0 {
		return nil // Нечего обновлять
	}

	updates = append(updates, "updated_at = NOW()")

	query := fmt.Sprintf("UPDATE users SET %s WHERE id = $%d",
		strings.Join(updates, ", "),
		argID,
	)
	args = append(args, userID)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr // (storage.ErrDuplicate)
		}
		return fmt.Errorf("%s: failed to update users table: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

func (r *Repo) DeleteUser(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteUser"

	// (Удаляем ТОЛЬКО `User`. `Contact` остается.)
	res, err := r.db.ExecContext(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			// (ErrForeignKeyViolation, если на user ссылаются)
			return translatedErr
		}
		return fmt.Errorf("%s: failed to delete user: %w", op, err)
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

const (
	selectUserFields = `
		SELECT
			u.id, u.is_active, u.login, u.pass_hash, u.created_at, u.updated_at, u.contact_id,
			
			c.fio, c.email, c.phone, c.ip_phone, c.dob, c.external_organization_name,
			
			o.id as org_id, o.name as org_name,
			
			d.id as dept_id, d.name as dept_name,
			
			p.id as pos_id, p.name as pos_name,
			
			COALESCE(r.roles_json, '[]'::json) as roles_json
	`
	fromUserJoins = `
		FROM
			users u
		JOIN
			contacts c ON u.contact_id = c.id
		LEFT JOIN
			organizations o ON c.organization_id = o.id
		LEFT JOIN
			departments d ON c.department_id = d.id
		LEFT JOIN
			positions p ON c.position_id = p.id
		LEFT JOIN (
            SELECT
               ur.user_id,
               json_agg(ro.name ORDER BY ro.name) as roles_json
            FROM
               users_roles ur
            JOIN
               roles ro ON ur.role_id = ro.id
            GROUP BY
               ur.user_id
        ) r ON u.id = r.user_id
	`
)

// scanUserRow - Хелпер для сканирования (включая PassHash)
func scanUserRow(scanner interface {
	Scan(dest ...interface{}) error
}, passHashDest interface{}) (*user.Model, error) {
	var u user.Model
	var (
		rolesJSON                     []byte
		email, phone, ipPhone, extOrg sql.NullString
		dob                           sql.NullTime
		orgID, deptID, posID          sql.NullInt64
		orgName, deptName, posName    sql.NullString
	)

	err := scanner.Scan(
		&u.ID, &u.IsActive, &u.Login, passHashDest, &u.CreatedAt, &u.UpdatedAt, &u.ContactID,
		&u.Name, &email, &phone, &ipPhone, &dob, &extOrg,
		&orgID, &orgName,
		&deptID, &deptName,
		&posID, &posName,
		&rolesJSON,
	)
	if err != nil {
		return nil, err
	}

	// Nullable поля
	if email.Valid {
		u.Email = &email.String
	}
	if phone.Valid {
		u.Phone = &phone.String
	}
	if ipPhone.Valid {
		u.IPPhone = &ipPhone.String
	}
	if extOrg.Valid {
		u.ExternalOrgName = &extOrg.String
	}
	if dob.Valid {
		u.DOB = &dob.Time
	}

	// Вложенные структуры
	if orgID.Valid && orgName.Valid {
		u.Organization = &organization.Model{ID: orgID.Int64, Name: orgName.String}
	}
	if deptID.Valid && deptName.Valid {
		u.Department = &department.Model{ID: deptID.Int64, Name: deptName.String}
	}
	if posID.Valid && posName.Valid {
		u.Position = &position.Model{ID: posID.Int64, Name: posName.String}
	}

	// Вложенная контактная информация
	u.Contact = &contact.Model{
		ID:              u.ContactID,
		Name:            u.Name,
		Email:           u.Email,
		Phone:           u.Phone,
		IPPhone:         u.IPPhone,
		DOB:             u.DOB,
		ExternalOrgName: u.ExternalOrgName,
		Organization:    u.Organization,
		Department:      u.Department,
		Position:        u.Position,
	}

	// Роли
	if err := json.Unmarshal(rolesJSON, &u.Roles); err != nil {
		return nil, fmt.Errorf("scanUserRow: failed to unmarshal roles: %w", err)
	}
	if u.Roles == nil {
		u.Roles = make([]string, 0)
	}

	return &u, nil
}
