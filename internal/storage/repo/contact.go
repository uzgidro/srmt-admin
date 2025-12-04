package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/contact"
	"srmt-admin/internal/lib/model/department"
	"srmt-admin/internal/lib/model/organization"
	"srmt-admin/internal/lib/model/position"
	"srmt-admin/internal/storage"
	"strings"
)

// (Предполагается, что `r.translator` и `r.db` есть в `Repo`)

// --- 1. CREATE ---

// AddContact
func (r *Repo) AddContact(ctx context.Context, req dto.AddContactRequest) (int64, error) {
	const op = "storage.repo.AddContact"

	const query = `
		INSERT INTO contacts (
			fio, email, phone, ip_phone, external_organization_name, icon_id,
			organization_id, department_id, position_id, dob
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.Name, req.Email, req.Phone, req.IPPhone, req.ExternalOrgName, req.IconID,
		req.OrganizationID, req.DepartmentID, req.PositionID, req.DOB,
	).Scan(&id)

	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return 0, translatedErr
		}
		return 0, fmt.Errorf("%s: failed to insert contact: %w", op, err)
	}

	return id, nil
}

// --- 2. READ (Блок хелперов) ---

const (
	// (Джойним всё)
	selectContactFields = `
		SELECT
			c.id, c.fio, c.email, c.phone, c.ip_phone, c.dob,
			c.external_organization_name, c.icon_id, c.created_at, c.updated_at,

			o.id as org_id, o.name as org_name,

			d.id as dept_id, d.name as dept_name,

			p.id as pos_id, p.name as pos_name, p.description as pos_description
	`
	fromContactJoins = `
		FROM
			contacts c
		LEFT JOIN
			organizations o ON c.organization_id = o.id
		LEFT JOIN
			departments d ON c.department_id = d.id
		LEFT JOIN
			positions p ON c.position_id = p.id
	`
)

// scanContactRow - Хелпер для сканирования обогащенной модели Contact
func scanContactRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*contact.Model, error) {
	var c contact.Model
	var (
		email, phone, ipPhone, extOrg sql.NullString
		dob                           sql.NullTime
		iconID                        sql.NullInt64
		orgID, deptID, posID          sql.NullInt64
		orgName, deptName, posName    sql.NullString
		posDescription                sql.NullString
	)

	err := scanner.Scan(
		&c.ID, &c.Name, &email, &phone, &ipPhone, &dob, &extOrg, &iconID,
		&c.CreatedAt, &c.UpdatedAt,
		&orgID, &orgName,
		&deptID, &deptName,
		&posID, &posName, &posDescription,
	)
	if err != nil {
		return nil, err
	}

	// Nullable поля
	if email.Valid {
		c.Email = &email.String
	}
	if phone.Valid {
		c.Phone = &phone.String
	}
	if ipPhone.Valid {
		c.IPPhone = &ipPhone.String
	}
	if extOrg.Valid {
		c.ExternalOrgName = &extOrg.String
	}
	if iconID.Valid {
		c.IconID = &iconID.Int64
	}
	if dob.Valid {
		c.DOB = &dob.Time
	}

	// Вложенные структуры
	if orgID.Valid && orgName.Valid {
		c.Organization = &organization.Model{ID: orgID.Int64, Name: orgName.String}
	}
	if deptID.Valid && deptName.Valid {
		c.Department = &department.Model{ID: deptID.Int64, Name: deptName.String}
	}
	if posID.Valid && posName.Valid {
		var desc *string
		if posDescription.Valid {
			desc = &posDescription.String
		}
		c.Position = &position.Model{ID: posID.Int64, Name: posName.String, Description: desc}
	}

	return &c, nil
}

// GetContactByID
func (r *Repo) GetContactByID(ctx context.Context, id int64) (*contact.Model, error) {
	const op = "storage.repo.GetContactByID"

	query := selectContactFields + fromContactJoins + " WHERE c.id = $1"

	row := r.db.QueryRowContext(ctx, query, id)
	c, err := scanContactRow(row)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to scan contact: %w", op, err)
	}

	return c, nil
}

// GetAllContacts
func (r *Repo) GetAllContacts(ctx context.Context, filters dto.GetAllContactsFilters) ([]*contact.Model, error) {
	const op = "storage.repo.GetAllContacts"

	var query strings.Builder
	query.WriteString(selectContactFields)
	query.WriteString(fromContactJoins)

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

	if len(whereClauses) > 0 {
		query.WriteString(" WHERE " + strings.Join(whereClauses, " AND "))
	}

	query.WriteString(" ORDER BY c.fio")

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query contacts: %w", op, err)
	}
	defer rows.Close()

	var contacts []*contact.Model
	for rows.Next() {
		c, err := scanContactRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan contact row: %w", op, err)
		}
		contacts = append(contacts, c)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if contacts == nil {
		contacts = make([]*contact.Model, 0)
	}

	return contacts, nil
}

// --- 3. UPDATE ---

// EditContact
func (r *Repo) EditContact(ctx context.Context, contactID int64, req dto.EditContactRequest) error {
	const op = "storage.repo.EditContact"

	// Динамический UPDATE
	var updates []string
	var args []interface{}
	argID := 1

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("fio = $%d", argID))
		args = append(args, *req.Name)
		argID++
	}
	if req.Email != nil {
		updates = append(updates, fmt.Sprintf("email = $%d", argID))
		args = append(args, *req.Email)
		argID++
	}
	if req.Phone != nil {
		updates = append(updates, fmt.Sprintf("phone = $%d", argID))
		args = append(args, *req.Phone)
		argID++
	}
	if req.IPPhone != nil {
		updates = append(updates, fmt.Sprintf("ip_phone = $%d", argID))
		args = append(args, *req.IPPhone)
		argID++
	}
	if req.DOB != nil {
		updates = append(updates, fmt.Sprintf("dob = $%d", argID))
		args = append(args, *req.DOB)
		argID++
	}
	if req.ExternalOrgName != nil {
		updates = append(updates, fmt.Sprintf("external_organization_name = $%d", argID))
		args = append(args, *req.ExternalOrgName)
		argID++
	}
	if req.IconID != nil {
		updates = append(updates, fmt.Sprintf("icon_id = $%d", argID))
		args = append(args, *req.IconID)
		argID++
	}
	if req.OrganizationID != nil {
		updates = append(updates, fmt.Sprintf("organization_id = $%d", argID))
		args = append(args, *req.OrganizationID)
		argID++
	}
	if req.DepartmentID != nil {
		updates = append(updates, fmt.Sprintf("department_id = $%d", argID))
		args = append(args, *req.DepartmentID)
		argID++
	}
	if req.PositionID != nil {
		updates = append(updates, fmt.Sprintf("position_id = $%d", argID))
		args = append(args, *req.PositionID)
		argID++
	}

	if len(updates) == 0 {
		return nil // Нечего обновлять
	}

	updates = append(updates, "updated_at = NOW()") // (Обновляем `updated_at`)

	query := fmt.Sprintf("UPDATE contacts SET %s WHERE id = $%d",
		strings.Join(updates, ", "),
		argID,
	)
	args = append(args, contactID)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr // (ErrDuplicate или ErrForeignKeyViolation)
		}
		return fmt.Errorf("%s: failed to update contacts: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- 4. DELETE ---

// DeleteContact
func (r *Repo) DeleteContact(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteContact"

	res, err := r.db.ExecContext(ctx, "DELETE FROM contacts WHERE id = $1", id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			// (Если на контакт ссылается `users.contact_id` - будет ErrForeignKeyViolation)
			return translatedErr
		}
		return fmt.Errorf("%s: failed to delete contact: %w", op, err)
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
