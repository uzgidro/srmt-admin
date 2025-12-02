package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/contact"
	"srmt-admin/internal/lib/model/fast_call"
	"srmt-admin/internal/storage"
	"strings"
)

// --- 1. CREATE ---

// AddFastCall creates a new fast call entry in the database
func (r *Repo) AddFastCall(ctx context.Context, req dto.AddFastCallRequest) (int64, error) {
	const op = "storage.repo.AddFastCall"

	const query = `
		INSERT INTO fast_calls (contact_id, position)
		VALUES ($1, $2)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query, req.ContactID, req.Position).Scan(&id)

	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return 0, translatedErr
		}
		return 0, fmt.Errorf("%s: failed to insert fast call: %w", op, err)
	}

	return id, nil
}

// --- 2. READ ---

const (
	selectFastCallFields = `
		SELECT
			fc.id, fc.contact_id, fc.position,
			c.id as contact_id, c.fio as contact_name, c.phone as contact_phone,
			c.email as contact_email, c.ip_phone as contact_ip_phone
	`
	fromFastCallJoins = `
		FROM fast_calls fc
		INNER JOIN contacts c ON fc.contact_id = c.id
	`
)

// scanFastCallRow scans an enriched fast call model from a database row
func scanFastCallRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*fast_call.Model, error) {
	var fc fast_call.Model
	var (
		contactID      int64
		contactName    string
		contactPhone   sql.NullString
		contactEmail   sql.NullString
		contactIPPhone sql.NullString
	)

	err := scanner.Scan(
		&fc.ID, &fc.ContactID, &fc.Position,
		&contactID, &contactName, &contactPhone,
		&contactEmail, &contactIPPhone,
	)

	if err != nil {
		return nil, err
	}

	// Build nested contact model
	fc.Contact = &contact.Model{
		ID:   contactID,
		Name: contactName,
	}

	if contactPhone.Valid {
		fc.Contact.Phone = &contactPhone.String
	}
	if contactEmail.Valid {
		fc.Contact.Email = &contactEmail.String
	}
	if contactIPPhone.Valid {
		fc.Contact.IPPhone = &contactIPPhone.String
	}

	return &fc, nil
}

// GetFastCallByID retrieves a single fast call by its ID with contact data
func (r *Repo) GetFastCallByID(ctx context.Context, id int64) (*fast_call.Model, error) {
	const op = "storage.repo.GetFastCallByID"

	query := selectFastCallFields + fromFastCallJoins + " WHERE fc.id = $1"
	row := r.db.QueryRowContext(ctx, query, id)
	fc, err := scanFastCallRow(row)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to scan fast call: %w", op, err)
	}

	return fc, nil
}

// GetAllFastCalls retrieves all fast calls ordered by position
func (r *Repo) GetAllFastCalls(ctx context.Context) ([]*fast_call.Model, error) {
	const op = "storage.repo.GetAllFastCalls"

	query := selectFastCallFields + fromFastCallJoins + " ORDER BY fc.position ASC"

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query fast calls: %w", op, err)
	}
	defer rows.Close()

	var fastCalls []*fast_call.Model
	for rows.Next() {
		fc, err := scanFastCallRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan fast call: %w", op, err)
		}
		fastCalls = append(fastCalls, fc)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows error: %w", op, err)
	}

	if fastCalls == nil {
		fastCalls = make([]*fast_call.Model, 0)
	}

	return fastCalls, nil
}

// --- 3. UPDATE ---

// EditFastCall updates an existing fast call entry
func (r *Repo) EditFastCall(ctx context.Context, fastCallID int64, req dto.EditFastCallRequest) error {
	const op = "storage.repo.EditFastCall"

	var updates []string
	var args []interface{}
	argID := 1

	if req.ContactID != nil {
		updates = append(updates, fmt.Sprintf("contact_id = $%d", argID))
		args = append(args, *req.ContactID)
		argID++
	}

	if req.Position != nil {
		updates = append(updates, fmt.Sprintf("position = $%d", argID))
		args = append(args, *req.Position)
		argID++
	}

	if len(updates) == 0 {
		return fmt.Errorf("%s: no fields to update", op)
	}

	query := fmt.Sprintf("UPDATE fast_calls SET %s WHERE id = $%d",
		strings.Join(updates, ", "),
		argID,
	)
	args = append(args, fastCallID)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to update fast call: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- 4. DELETE ---

// DeleteFastCall deletes a fast call entry by its ID
func (r *Repo) DeleteFastCall(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteFastCall"

	res, err := r.db.ExecContext(ctx, "DELETE FROM fast_calls WHERE id = $1", id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to delete fast call: %w", op, err)
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
