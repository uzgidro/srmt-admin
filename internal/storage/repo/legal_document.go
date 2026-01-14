package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/file"
	legal_document "srmt-admin/internal/lib/model/legal-document"
	legal_document_type "srmt-admin/internal/lib/model/legal-document-type"
	"srmt-admin/internal/lib/model/user"
	"srmt-admin/internal/storage"
)

// AddLegalDocument creates a new legal document
func (r *Repo) AddLegalDocument(ctx context.Context, req dto.AddLegalDocumentRequest, createdByID int64) (int64, error) {
	const op = "storage.repo.AddLegalDocument"

	const query = `
		INSERT INTO legal_documents (name, number, document_date, type_id, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.Name,
		req.Number,
		req.DocumentDate,
		req.TypeID,
		createdByID,
	).Scan(&id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return 0, translatedErr
		}
		return 0, fmt.Errorf("%s: failed to insert legal document: %w", op, err)
	}

	return id, nil
}

// GetLegalDocumentByID retrieves a single legal document with all joined data
func (r *Repo) GetLegalDocumentByID(ctx context.Context, id int64) (*legal_document.ResponseModel, error) {
	const op = "storage.repo.GetLegalDocumentByID"

	query := selectLegalDocumentFields + fromLegalDocumentJoins + `WHERE d.id = $1`

	row := r.db.QueryRowContext(ctx, query, id)
	doc, err := scanLegalDocumentRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to scan legal document row: %w", op, err)
	}

	// Load files for this document
	files, err := r.loadLegalDocumentFiles(ctx, doc.ID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to load files: %w", op, err)
	}
	doc.Files = files

	return doc, nil
}

// GetAllLegalDocuments retrieves all legal documents with optional filters
func (r *Repo) GetAllLegalDocuments(ctx context.Context, filters dto.GetAllLegalDocumentsFilters) ([]*legal_document.ResponseModel, error) {
	const op = "storage.repo.GetAllLegalDocuments"

	query := selectLegalDocumentFields + fromLegalDocumentJoins

	var whereClauses []string
	var args []interface{}
	argID := 1

	// Filter by type_id
	if filters.TypeID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("d.type_id = $%d", argID))
		args = append(args, *filters.TypeID)
		argID++
	}

	// Filter by date range
	if filters.StartDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("d.document_date >= $%d", argID))
		args = append(args, *filters.StartDate)
		argID++
	}
	if filters.EndDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("d.document_date <= $%d", argID))
		args = append(args, *filters.EndDate)
		argID++
	}

	// Search by name
	if filters.NameSearch != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("d.name ILIKE $%d", argID))
		args = append(args, "%"+*filters.NameSearch+"%")
		argID++
	}

	// Search by number
	if filters.NumberSearch != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("d.number ILIKE $%d", argID))
		args = append(args, "%"+*filters.NumberSearch+"%")
		argID++
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	query += " ORDER BY d.document_date DESC, d.created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query legal documents: %w", op, err)
	}
	defer rows.Close()

	var documents []*legal_document.ResponseModel
	for rows.Next() {
		doc, err := scanLegalDocumentRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan legal document row: %w", op, err)
		}
		documents = append(documents, doc)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if documents == nil {
		documents = make([]*legal_document.ResponseModel, 0)
	}

	// Load files for each document
	for _, doc := range documents {
		files, err := r.loadLegalDocumentFiles(ctx, doc.ID)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to load files for document %d: %w", op, doc.ID, err)
		}
		doc.Files = files
	}

	return documents, nil
}

// EditLegalDocument updates a legal document
func (r *Repo) EditLegalDocument(ctx context.Context, id int64, req dto.EditLegalDocumentRequest, updatedByID int64) error {
	const op = "storage.repo.EditLegalDocument"

	var updates []string
	var args []interface{}
	argID := 1

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argID))
		args = append(args, *req.Name)
		argID++
	}
	if req.Number != nil {
		updates = append(updates, fmt.Sprintf("number = $%d", argID))
		args = append(args, *req.Number)
		argID++
	}
	if req.DocumentDate != nil {
		updates = append(updates, fmt.Sprintf("document_date = $%d", argID))
		args = append(args, *req.DocumentDate)
		argID++
	}
	if req.TypeID != nil {
		updates = append(updates, fmt.Sprintf("type_id = $%d", argID))
		args = append(args, *req.TypeID)
		argID++
	}

	// Always update updated_by_user_id
	updates = append(updates, fmt.Sprintf("updated_by_user_id = $%d", argID))
	args = append(args, updatedByID)
	argID++

	if len(updates) == 1 && len(req.FileIDs) == 0 {
		return nil // Only updated_by_user_id, nothing meaningful to update
	}

	query := fmt.Sprintf("UPDATE legal_documents SET %s WHERE id = $%d",
		strings.Join(updates, ", "),
		argID,
	)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to update legal document: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteLegalDocument deletes a legal document
func (r *Repo) DeleteLegalDocument(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteLegalDocument"

	res, err := r.db.ExecContext(ctx, "DELETE FROM legal_documents WHERE id = $1", id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to delete legal document: %w", op, err)
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

// LinkLegalDocumentFiles links files to a legal document
func (r *Repo) LinkLegalDocumentFiles(ctx context.Context, documentID int64, fileIDs []int64) error {
	const op = "storage.repo.LinkLegalDocumentFiles"

	if len(fileIDs) == 0 {
		return nil
	}

	query := `
		INSERT INTO legal_document_file_links (document_id, file_id)
		VALUES ($1, unnest($2::bigint[]))
		ON CONFLICT DO NOTHING
	`

	_, err := r.db.ExecContext(ctx, query, documentID, pq.Array(fileIDs))
	if err != nil {
		return fmt.Errorf("%s: failed to link files: %w", op, err)
	}

	return nil
}

// UnlinkLegalDocumentFiles removes all file links for a legal document
func (r *Repo) UnlinkLegalDocumentFiles(ctx context.Context, documentID int64) error {
	const op = "storage.repo.UnlinkLegalDocumentFiles"

	query := `DELETE FROM legal_document_file_links WHERE document_id = $1`
	_, err := r.db.ExecContext(ctx, query, documentID)
	if err != nil {
		return fmt.Errorf("%s: failed to unlink files: %w", op, err)
	}

	return nil
}

// loadLegalDocumentFiles loads files for a legal document
func (r *Repo) loadLegalDocumentFiles(ctx context.Context, documentID int64) ([]file.Model, error) {
	const op = "storage.repo.loadLegalDocumentFiles"

	query := `
		SELECT f.id, f.file_name, f.object_key, f.category_id, f.mime_type, f.size_bytes, f.created_at
		FROM files f
		INNER JOIN legal_document_file_links dfl ON f.id = dfl.file_id
		WHERE dfl.document_id = $1
		ORDER BY f.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, documentID)
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

// GetAllLegalDocumentTypes retrieves all legal document types
func (r *Repo) GetAllLegalDocumentTypes(ctx context.Context) ([]legal_document_type.Model, error) {
	const op = "storage.repo.GetAllLegalDocumentTypes"
	const query = "SELECT id, name, COALESCE(description, '') as description FROM legal_document_type ORDER BY id"

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query legal document types: %w", op, err)
	}
	defer rows.Close()

	var types []legal_document_type.Model
	for rows.Next() {
		var t legal_document_type.Model
		if err := rows.Scan(&t.ID, &t.Name, &t.Description); err != nil {
			return nil, fmt.Errorf("%s: failed to scan type row: %w", op, err)
		}
		types = append(types, t)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if types == nil {
		types = make([]legal_document_type.Model, 0)
	}

	return types, nil
}

// --- Query fragments and helpers ---

const (
	selectLegalDocumentFields = `
		SELECT
			d.id,
			d.name,
			d.number,
			d.document_date,
			d.type_id,
			t.name as type_name,
			COALESCE(t.description, '') as type_description,
			d.created_at,
			d.created_by_user_id,
			COALESCE(cc.fio, '') as created_by_fio,
			d.updated_at,
			d.updated_by_user_id,
			COALESCE(uc.fio, '') as updated_by_fio
	`
	fromLegalDocumentJoins = `
		FROM legal_documents d
		INNER JOIN legal_document_type t ON d.type_id = t.id
		LEFT JOIN users cu ON d.created_by_user_id = cu.id
		LEFT JOIN contacts cc ON cu.contact_id = cc.id
		LEFT JOIN users uu ON d.updated_by_user_id = uu.id
		LEFT JOIN contacts uc ON uu.contact_id = uc.id
	`
)

func scanLegalDocumentRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*legal_document.ResponseModel, error) {
	var doc legal_document.ResponseModel
	var number sql.NullString
	var typeID int
	var typeName, typeDescription string
	var createdByID sql.NullInt64
	var createdByFIO string
	var updatedAt sql.NullTime
	var updatedByID sql.NullInt64
	var updatedByFIO string

	err := scanner.Scan(
		&doc.ID,
		&doc.Name,
		&number,
		&doc.DocumentDate,
		&typeID,
		&typeName,
		&typeDescription,
		&doc.CreatedAt,
		&createdByID,
		&createdByFIO,
		&updatedAt,
		&updatedByID,
		&updatedByFIO,
	)
	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if number.Valid {
		doc.Number = &number.String
	}
	if updatedAt.Valid {
		doc.UpdatedAt = &updatedAt.Time
	}

	// Build type object
	doc.Type = legal_document_type.Model{
		ID:          typeID,
		Name:        typeName,
		Description: typeDescription,
	}

	// Build user objects
	if createdByID.Valid {
		doc.CreatedBy = &user.ShortInfo{
			ID:   createdByID.Int64,
			Name: &createdByFIO,
		}
	}
	if updatedByID.Valid {
		doc.UpdatedBy = &user.ShortInfo{
			ID:   updatedByID.Int64,
			Name: &updatedByFIO,
		}
	}

	return &doc, nil
}
