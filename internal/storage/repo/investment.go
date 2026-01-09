package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/model/investment"
	investment_status "srmt-admin/internal/lib/model/investment-status"
	investment_type "srmt-admin/internal/lib/model/investment-type"
	"srmt-admin/internal/lib/model/user"
	"srmt-admin/internal/storage"
)

// AddInvestment creates a new investment record
func (r *Repo) AddInvestment(ctx context.Context, req dto.AddInvestmentRequest, createdByID int64) (int64, error) {
	const op = "storage.repo.AddInvestment"

	const query = `
		INSERT INTO investments (name, type_id, status_id, cost, comments, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.Name,
		req.TypeID,
		req.StatusID,
		req.Cost,
		req.Comments,
		createdByID,
	).Scan(&id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return 0, translatedErr
		}
		return 0, fmt.Errorf("%s: failed to insert investment: %w", op, err)
	}

	return id, nil
}

// GetInvestmentByID retrieves a single investment with all joined data
func (r *Repo) GetInvestmentByID(ctx context.Context, id int64) (*investment.ResponseModel, error) {
	const op = "storage.repo.GetInvestmentByID"

	query := selectInvestmentFields + fromInvestmentJoins + `WHERE i.id = $1`

	row := r.db.QueryRowContext(ctx, query, id)
	inv, err := scanInvestmentRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to scan investment row: %w", op, err)
	}

	// Load files for this investment
	files, err := r.loadInvestmentFiles(ctx, inv.ID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to load files: %w", op, err)
	}
	inv.Files = files

	return inv, nil
}

// GetAllInvestments retrieves all investments with optional filters
func (r *Repo) GetAllInvestments(ctx context.Context, filters dto.GetAllInvestmentsFilters) ([]*investment.ResponseModel, error) {
	const op = "storage.repo.GetAllInvestments"

	query := selectInvestmentFields + fromInvestmentJoins

	var whereClauses []string
	var args []interface{}
	argID := 1

	// Apply filters
	if filters.TypeID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("i.type_id = $%d", argID))
		args = append(args, *filters.TypeID)
		argID++
	}
	if filters.StatusID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("i.status_id = $%d", argID))
		args = append(args, *filters.StatusID)
		argID++
	}
	if filters.MinCost != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("i.cost >= $%d", argID))
		args = append(args, *filters.MinCost)
		argID++
	}
	if filters.MaxCost != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("i.cost <= $%d", argID))
		args = append(args, *filters.MaxCost)
		argID++
	}
	if filters.NameSearch != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("i.name ILIKE $%d", argID))
		args = append(args, "%"+*filters.NameSearch+"%")
		argID++
	}
	if filters.CreatedByUserID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("i.created_by_user_id = $%d", argID))
		args = append(args, *filters.CreatedByUserID)
		argID++
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	query += " ORDER BY i.created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query investments: %w", op, err)
	}
	defer rows.Close()

	var investments []*investment.ResponseModel
	for rows.Next() {
		inv, err := scanInvestmentRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan investment row: %w", op, err)
		}
		investments = append(investments, inv)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if investments == nil {
		investments = make([]*investment.ResponseModel, 0)
	}

	// Load files for each investment
	for _, inv := range investments {
		files, err := r.loadInvestmentFiles(ctx, inv.ID)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to load files for investment %d: %w", op, inv.ID, err)
		}
		inv.Files = files
	}

	return investments, nil
}

// EditInvestment updates an investment record
func (r *Repo) EditInvestment(ctx context.Context, id int64, req dto.EditInvestmentRequest) error {
	const op = "storage.repo.EditInvestment"

	var updates []string
	var args []interface{}
	argID := 1

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argID))
		args = append(args, *req.Name)
		argID++
	}
	if req.TypeID != nil {
		updates = append(updates, fmt.Sprintf("type_id = $%d", argID))
		args = append(args, *req.TypeID)
		argID++
	}
	if req.StatusID != nil {
		updates = append(updates, fmt.Sprintf("status_id = $%d", argID))
		args = append(args, *req.StatusID)
		argID++
	}
	if req.Cost != nil {
		updates = append(updates, fmt.Sprintf("cost = $%d", argID))
		args = append(args, *req.Cost)
		argID++
	}
	if req.Comments != nil {
		updates = append(updates, fmt.Sprintf("comments = $%d", argID))
		args = append(args, *req.Comments)
		argID++
	}

	if len(updates) == 0 {
		return nil // Nothing to update
	}

	updates = append(updates, "updated_at = NOW()")

	query := fmt.Sprintf("UPDATE investments SET %s WHERE id = $%d",
		strings.Join(updates, ", "),
		argID,
	)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to update investment: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteInvestment deletes an investment record
func (r *Repo) DeleteInvestment(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteInvestment"

	res, err := r.db.ExecContext(ctx, "DELETE FROM investments WHERE id = $1", id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to delete investment: %w", op, err)
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

// LinkInvestmentFiles links files to an investment
func (r *Repo) LinkInvestmentFiles(ctx context.Context, investmentID int64, fileIDs []int64) error {
	const op = "storage.repo.investment.LinkInvestmentFiles"

	if len(fileIDs) == 0 {
		return nil
	}

	query := `
		INSERT INTO investment_file_links (investment_id, file_id)
		VALUES ($1, unnest($2::bigint[]))
		ON CONFLICT DO NOTHING
	`

	_, err := r.db.ExecContext(ctx, query, investmentID, pq.Array(fileIDs))
	if err != nil {
		return fmt.Errorf("%s: failed to link files: %w", op, err)
	}

	return nil
}

// UnlinkInvestmentFiles removes all file links for an investment
func (r *Repo) UnlinkInvestmentFiles(ctx context.Context, investmentID int64) error {
	const op = "storage.repo.investment.UnlinkInvestmentFiles"

	query := `DELETE FROM investment_file_links WHERE investment_id = $1`
	_, err := r.db.ExecContext(ctx, query, investmentID)
	if err != nil {
		return fmt.Errorf("%s: failed to unlink files: %w", op, err)
	}

	return nil
}

// loadInvestmentFiles loads files for an investment
func (r *Repo) loadInvestmentFiles(ctx context.Context, investmentID int64) ([]file.Model, error) {
	const op = "storage.repo.investment.loadInvestmentFiles"

	query := `
		SELECT f.id, f.file_name, f.object_key, f.category_id, f.mime_type, f.size_bytes, f.created_at
		FROM files f
		INNER JOIN investment_file_links ifl ON f.id = ifl.file_id
		WHERE ifl.investment_id = $1
		ORDER BY f.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, investmentID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query files: %w", op, err)
	}
	defer rows.Close()

	var files []file.Model
	for rows.Next() {
		var f file.Model
		if err := rows.Scan(&f.ID, &f.FileName, &f.ObjectKey, &f.CategoryID, &f.MimeType, &f.SizeBytes, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("%s: failed to scan file row: %w", op, err)
		}
		files = append(files, f)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	return files, nil
}

// GetAllInvestmentStatuses retrieves all investment statuses
func (r *Repo) GetAllInvestmentStatuses(ctx context.Context) ([]investment_status.Model, error) {
	const op = "storage.repo.GetAllInvestmentStatuses"
	const query = "SELECT id, name, description, type_id, display_order FROM investment_status ORDER BY display_order, id"

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query investment statuses: %w", op, err)
	}
	defer rows.Close()

	var statuses []investment_status.Model
	for rows.Next() {
		var s investment_status.Model
		var typeID sql.NullInt32
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &typeID, &s.DisplayOrder); err != nil {
			return nil, fmt.Errorf("%s: failed to scan status row: %w", op, err)
		}
		if typeID.Valid {
			typeIDInt := int(typeID.Int32)
			s.TypeID = &typeIDInt
		}
		statuses = append(statuses, s)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if statuses == nil {
		statuses = make([]investment_status.Model, 0)
	}

	return statuses, nil
}

// GetAllInvestmentTypes retrieves all investment types
func (r *Repo) GetAllInvestmentTypes(ctx context.Context) ([]investment_type.Model, error) {
	const op = "storage.repo.GetAllInvestmentTypes"
	const query = "SELECT id, name, description FROM investment_type ORDER BY id"

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query investment types: %w", op, err)
	}
	defer rows.Close()

	var types []investment_type.Model
	for rows.Next() {
		var t investment_type.Model
		if err := rows.Scan(&t.ID, &t.Name, &t.Description); err != nil {
			return nil, fmt.Errorf("%s: failed to scan type row: %w", op, err)
		}
		types = append(types, t)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if types == nil {
		types = make([]investment_type.Model, 0)
	}

	return types, nil
}

// AddInvestmentType creates a new investment type
func (r *Repo) AddInvestmentType(ctx context.Context, req dto.AddInvestmentTypeRequest) (int, error) {
	const op = "storage.repo.AddInvestmentType"

	const query = `
		INSERT INTO investment_type (name, description)
		VALUES ($1, $2)
		RETURNING id`

	var id int
	err := r.db.QueryRowContext(ctx, query, req.Name, req.Description).Scan(&id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return 0, translatedErr
		}
		return 0, fmt.Errorf("%s: failed to insert investment type: %w", op, err)
	}

	return id, nil
}

// EditInvestmentType updates an investment type
func (r *Repo) EditInvestmentType(ctx context.Context, id int, req dto.EditInvestmentTypeRequest) error {
	const op = "storage.repo.EditInvestmentType"

	var updates []string
	var args []interface{}
	argID := 1

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argID))
		args = append(args, *req.Name)
		argID++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argID))
		args = append(args, *req.Description)
		argID++
	}

	if len(updates) == 0 {
		return nil // Nothing to update
	}

	query := fmt.Sprintf("UPDATE investment_type SET %s WHERE id = $%d",
		strings.Join(updates, ", "),
		argID,
	)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to update investment type: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteInvestmentType deletes an investment type
func (r *Repo) DeleteInvestmentType(ctx context.Context, id int) error {
	const op = "storage.repo.DeleteInvestmentType"

	res, err := r.db.ExecContext(ctx, "DELETE FROM investment_type WHERE id = $1", id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to delete investment type: %w", op, err)
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

// GetInvestmentStatusesByType returns statuses available for a specific type
// Returns shared statuses (type_id IS NULL) + type-specific statuses
func (r *Repo) GetInvestmentStatusesByType(ctx context.Context, typeID int) ([]investment_status.Model, error) {
	const op = "storage.repo.GetInvestmentStatusesByType"
	const query = `
		SELECT id, name, description, type_id, display_order
		FROM investment_status
		WHERE type_id IS NULL OR type_id = $1
		ORDER BY display_order`

	rows, err := r.db.QueryContext(ctx, query, typeID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query investment statuses: %w", op, err)
	}
	defer rows.Close()

	var statuses []investment_status.Model
	for rows.Next() {
		var s investment_status.Model
		var typeIDNullable sql.NullInt32
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &typeIDNullable, &s.DisplayOrder); err != nil {
			return nil, fmt.Errorf("%s: failed to scan status row: %w", op, err)
		}
		if typeIDNullable.Valid {
			tid := int(typeIDNullable.Int32)
			s.TypeID = &tid
		}
		statuses = append(statuses, s)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if statuses == nil {
		statuses = make([]investment_status.Model, 0)
	}

	return statuses, nil
}

// AddInvestmentStatus creates a new investment status
func (r *Repo) AddInvestmentStatus(ctx context.Context, req dto.AddInvestmentStatusRequest) (int, error) {
	const op = "storage.repo.AddInvestmentStatus"

	const query = `
		INSERT INTO investment_status (name, description, type_id, display_order)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	var id int
	err := r.db.QueryRowContext(ctx, query, req.Name, req.Description, req.TypeID, req.DisplayOrder).Scan(&id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return 0, translatedErr
		}
		return 0, fmt.Errorf("%s: failed to insert investment status: %w", op, err)
	}

	return id, nil
}

// EditInvestmentStatus updates an investment status
func (r *Repo) EditInvestmentStatus(ctx context.Context, id int, req dto.EditInvestmentStatusRequest) error {
	const op = "storage.repo.EditInvestmentStatus"

	var updates []string
	var args []interface{}
	argID := 1

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argID))
		args = append(args, *req.Name)
		argID++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argID))
		args = append(args, *req.Description)
		argID++
	}
	if req.TypeID != nil {
		updates = append(updates, fmt.Sprintf("type_id = $%d", argID))
		args = append(args, *req.TypeID)
		argID++
	}
	if req.DisplayOrder != nil {
		updates = append(updates, fmt.Sprintf("display_order = $%d", argID))
		args = append(args, *req.DisplayOrder)
		argID++
	}

	if len(updates) == 0 {
		return nil // Nothing to update
	}

	query := fmt.Sprintf("UPDATE investment_status SET %s WHERE id = $%d",
		strings.Join(updates, ", "),
		argID,
	)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to update investment status: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteInvestmentStatus deletes an investment status
func (r *Repo) DeleteInvestmentStatus(ctx context.Context, id int) error {
	const op = "storage.repo.DeleteInvestmentStatus"

	res, err := r.db.ExecContext(ctx, "DELETE FROM investment_status WHERE id = $1", id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to delete investment status: %w", op, err)
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

// --- Query fragments and helpers ---

const (
	selectInvestmentFields = `
		SELECT
			i.id,
			i.name,
			i.type_id,
			it.name as type_name,
			COALESCE(it.description, '') as type_description,
			i.status_id,
			ist.name as status_name,
			COALESCE(ist.description, '') as status_description,
			COALESCE(ist.type_id, 0) as status_type_id,
			ist.display_order as status_display_order,
			i.cost,
			i.comments,
			i.created_at,
			i.created_by_user_id,
			COALESCE(c.fio, '') as created_by_user_fio,
			i.updated_at
	`
	fromInvestmentJoins = `
		FROM
			investments i
		INNER JOIN
			investment_type it ON i.type_id = it.id
		INNER JOIN
			investment_status ist ON i.status_id = ist.id
		LEFT JOIN
			users u ON i.created_by_user_id = u.id
		LEFT JOIN
			contacts c ON u.contact_id = c.id
	`
)

func scanInvestmentRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*investment.ResponseModel, error) {
	var inv investment.ResponseModel
	var typeID int
	var typeName string
	var typeDescription string
	var statusID int
	var statusName string
	var statusDescription string
	var statusTypeID int
	var statusDisplayOrder int
	var comments sql.NullString
	var createdByUserID sql.NullInt64
	var createdByUserFIO string
	var updatedAt sql.NullTime

	err := scanner.Scan(
		&inv.ID,
		&inv.Name,
		&typeID,
		&typeName,
		&typeDescription,
		&statusID,
		&statusName,
		&statusDescription,
		&statusTypeID,
		&statusDisplayOrder,
		&inv.Cost,
		&comments,
		&inv.CreatedAt,
		&createdByUserID,
		&createdByUserFIO,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Build type object
	inv.Type = investment_type.Model{
		ID:          typeID,
		Name:        typeName,
		Description: typeDescription,
	}

	// Build status object
	var statusTypeIDPtr *int
	if statusTypeID != 0 {
		statusTypeIDPtr = &statusTypeID
	}
	inv.Status = investment_status.Model{
		ID:           statusID,
		Name:         statusName,
		Description:  statusDescription,
		TypeID:       statusTypeIDPtr,
		DisplayOrder: statusDisplayOrder,
	}

	// Handle nullable fields
	if comments.Valid {
		inv.Comments = &comments.String
	}
	if createdByUserID.Valid {
		inv.CreatedByUser = &user.ShortInfo{
			ID:   createdByUserID.Int64,
			Name: &createdByUserFIO,
		}
	}
	if updatedAt.Valid {
		inv.UpdatedAt = &updatedAt.Time
	}

	return &inv, nil
}
