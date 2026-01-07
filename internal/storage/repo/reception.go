package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/contact"
	"srmt-admin/internal/lib/model/reception"
	"srmt-admin/internal/lib/model/user"
	"srmt-admin/internal/storage"
	"strings"
)

// --- 1. CREATE ---

// AddReception creates a new reception record in the database
func (r *Repo) AddReception(ctx context.Context, req dto.AddReceptionRequest) (int64, error) {
	const op = "storage.repo.AddReception"

	const query = `
		INSERT INTO receptions (
			name, date, description, visitor, created_by_user_id
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.Name, req.Date, req.Description, req.Visitor, req.CreatedByID,
	).Scan(&id)

	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return 0, translatedErr
		}
		return 0, fmt.Errorf("%s: failed to insert reception: %w", op, err)
	}

	return id, nil
}

// --- 2. READ ---

const (
	selectReceptionFields = `
		SELECT
			r.id, r.name, r.date, r.description, r.visitor, r.status, r.status_change_reason,
			r.informed, r.informed_by_user_id,
			r.created_at, r.updated_at, r.created_by_user_id, r.updated_by_user_id,

			cu.id as created_user_id, cu.login as created_login,
			cu_contact.id as created_contact_id, cu_contact.fio as created_contact_fio,
			cu_contact.phone as created_contact_phone, cu_contact.email as created_contact_email,

			uu.id as updated_user_id, uu.login as updated_login,
			uu_contact.id as updated_contact_id, uu_contact.fio as updated_contact_fio,
			uu_contact.phone as updated_contact_phone, uu_contact.email as updated_contact_email,

			iu.id as informed_user_id, iu.login as informed_login,
			iu_contact.id as informed_contact_id, iu_contact.fio as informed_contact_fio,
			iu_contact.phone as informed_contact_phone, iu_contact.email as informed_contact_email
	`
	fromReceptionJoins = `
		FROM receptions r
		LEFT JOIN users cu ON r.created_by_user_id = cu.id
		LEFT JOIN contacts cu_contact ON cu.contact_id = cu_contact.id
		LEFT JOIN users uu ON r.updated_by_user_id = uu.id
		LEFT JOIN contacts uu_contact ON uu.contact_id = uu_contact.id
		LEFT JOIN users iu ON r.informed_by_user_id = iu.id
		LEFT JOIN contacts iu_contact ON iu.contact_id = iu_contact.id
	`
)

// scanReceptionRow scans an enriched reception model from a database row
func scanReceptionRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*reception.Model, error) {
	var rec reception.Model
	var (
		description        sql.NullString
		statusChangeReason sql.NullString
		updatedAt          sql.NullTime
		updatedByUserID    sql.NullInt64
		informedByUserID   sql.NullInt64

		// Created User fields
		createdUserID       sql.NullInt64
		createdLogin        sql.NullString
		createdContactID    sql.NullInt64
		createdContactFIO   sql.NullString
		createdContactPhone sql.NullString
		createdContactEmail sql.NullString

		// Updated User fields
		updatedUserID       sql.NullInt64
		updatedLogin        sql.NullString
		updatedContactID    sql.NullInt64
		updatedContactFIO   sql.NullString
		updatedContactPhone sql.NullString
		updatedContactEmail sql.NullString

		// Informed User fields
		informedUserID       sql.NullInt64
		informedLogin        sql.NullString
		informedContactID    sql.NullInt64
		informedContactFIO   sql.NullString
		informedContactPhone sql.NullString
		informedContactEmail sql.NullString
	)

	err := scanner.Scan(
		&rec.ID, &rec.Name, &rec.Date, &description, &rec.Visitor, &rec.Status, &statusChangeReason,
		&rec.Informed, &informedByUserID,
		&rec.CreatedAt, &updatedAt, &rec.CreatedByID, &updatedByUserID,

		&createdUserID, &createdLogin,
		&createdContactID, &createdContactFIO, &createdContactPhone, &createdContactEmail,

		&updatedUserID, &updatedLogin,
		&updatedContactID, &updatedContactFIO, &updatedContactPhone, &updatedContactEmail,

		&informedUserID, &informedLogin,
		&informedContactID, &informedContactFIO, &informedContactPhone, &informedContactEmail,
	)

	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if description.Valid {
		rec.Description = &description.String
	}
	if statusChangeReason.Valid {
		rec.StatusChangeReason = &statusChangeReason.String
	}
	if updatedAt.Valid {
		rec.UpdatedAt = &updatedAt.Time
	}
	if updatedByUserID.Valid {
		rec.UpdatedByID = &updatedByUserID.Int64
	}
	if informedByUserID.Valid {
		rec.InformedByUserID = &informedByUserID.Int64
	}

	// Helper to build user model with contact
	buildUser := func(id int64, login, contactFIO, contactPhone, contactEmail string, contactID int64) *user.Model {
		u := &user.Model{
			ID:    id,
			Login: login,
		}
		if contactID != 0 {
			u.Contact = &contact.Model{
				ID:   contactID,
				Name: contactFIO,
			}
			if contactPhone != "" {
				u.Contact.Phone = &contactPhone
			}
			if contactEmail != "" {
				u.Contact.Email = &contactEmail
			}
		}
		return u
	}

	// Build nested user models
	if createdUserID.Valid {
		rec.CreatedBy = buildUser(
			createdUserID.Int64,
			createdLogin.String,
			createdContactFIO.String,
			createdContactPhone.String,
			createdContactEmail.String,
			createdContactID.Int64,
		)
	}

	if updatedUserID.Valid {
		rec.UpdatedBy = buildUser(
			updatedUserID.Int64,
			updatedLogin.String,
			updatedContactFIO.String,
			updatedContactPhone.String,
			updatedContactEmail.String,
			updatedContactID.Int64,
		)
	}

	if informedUserID.Valid {
		rec.InformedBy = buildUser(
			informedUserID.Int64,
			informedLogin.String,
			informedContactFIO.String,
			informedContactPhone.String,
			informedContactEmail.String,
			informedContactID.Int64,
		)
	}

	return &rec, nil
}

// GetReceptionByID retrieves a single reception by its ID with all related data
func (r *Repo) GetReceptionByID(ctx context.Context, id int64) (*reception.Model, error) {
	const op = "storage.repo.GetReceptionByID"

	query := selectReceptionFields + fromReceptionJoins + " WHERE r.id = $1"
	row := r.db.QueryRowContext(ctx, query, id)
	rec, err := scanReceptionRow(row)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to scan reception: %w", op, err)
	}

	return rec, nil
}

// GetAllReceptions retrieves all receptions with optional filters
func (r *Repo) GetAllReceptions(ctx context.Context, filters dto.GetAllReceptionsFilters) ([]*reception.Model, error) {
	const op = "storage.repo.GetAllReceptions"

	query := selectReceptionFields + fromReceptionJoins

	// Build WHERE clause dynamically based on filters
	var whereClauses []string
	var args []interface{}
	argID := 1

	if filters.StartDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("r.date >= $%d", argID))
		args = append(args, *filters.StartDate)
		argID++
	}

	if filters.EndDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("r.date <= $%d", argID))
		args = append(args, *filters.EndDate)
		argID++
	}

	if filters.Status != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("r.status = $%d", argID))
		args = append(args, *filters.Status)
		argID++
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	query += " ORDER BY r.date DESC, r.created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query receptions: %w", op, err)
	}
	defer rows.Close()

	var receptions []*reception.Model
	for rows.Next() {
		rec, err := scanReceptionRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan reception: %w", op, err)
		}
		receptions = append(receptions, rec)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows error: %w", op, err)
	}

	if receptions == nil {
		receptions = make([]*reception.Model, 0)
	}

	return receptions, nil
}

// --- 3. UPDATE ---

// EditReception updates an existing reception record
func (r *Repo) EditReception(ctx context.Context, receptionID int64, req dto.EditReceptionRequest) error {
	const op = "storage.repo.EditReception"

	var updates []string
	var args []interface{}
	argID := 1

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argID))
		args = append(args, *req.Name)
		argID++
	}

	if req.Date != nil {
		updates = append(updates, fmt.Sprintf("date = $%d", argID))
		args = append(args, *req.Date)
		argID++
	}

	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argID))
		args = append(args, *req.Description)
		argID++
	}

	if req.Visitor != nil {
		updates = append(updates, fmt.Sprintf("visitor = $%d", argID))
		args = append(args, *req.Visitor)
		argID++
	}

	if req.Status != nil {
		updates = append(updates, fmt.Sprintf("status = $%d", argID))
		args = append(args, *req.Status)
		argID++
	}

	if req.StatusChangeReason != nil {
		updates = append(updates, fmt.Sprintf("status_change_reason = $%d", argID))
		args = append(args, *req.StatusChangeReason)
		argID++
	}

	if req.Informed != nil {
		updates = append(updates, fmt.Sprintf("informed = $%d", argID))
		args = append(args, *req.Informed)
		argID++
	}

	if req.InformedByUserID != nil {
		updates = append(updates, fmt.Sprintf("informed_by_user_id = $%d", argID))
		args = append(args, *req.InformedByUserID)
		argID++
	}

	if len(updates) == 0 {
		return fmt.Errorf("%s: no fields to update", op)
	}

	// Always set updated_by and updated_at
	updates = append(updates, fmt.Sprintf("updated_by_user_id = $%d", argID))
	args = append(args, req.UpdatedByID)
	argID++
	updates = append(updates, "updated_at = NOW()")

	query := fmt.Sprintf("UPDATE receptions SET %s WHERE id = $%d",
		strings.Join(updates, ", "),
		argID,
	)
	args = append(args, receptionID)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to update reception: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- 4. DELETE ---

// DeleteReception deletes a reception record by its ID
func (r *Repo) DeleteReception(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteReception"

	res, err := r.db.ExecContext(ctx, "DELETE FROM receptions WHERE id = $1", id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to delete reception: %w", op, err)
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
