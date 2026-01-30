package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"

	"srmt-admin/internal/lib/dto/hrm"
	hrmmodel "srmt-admin/internal/lib/model/hrm"
	"srmt-admin/internal/storage"
)

// --- Document Types ---

// AddDocumentType creates a new document type
func (r *Repo) AddDocumentType(ctx context.Context, req hrm.AddDocumentTypeRequest) (int, error) {
	const op = "storage.repo.AddDocumentType"

	const query = `
		INSERT INTO hrm_document_types (
			name, code, description, template_id,
			requires_signature, requires_employee_signature, requires_manager_signature, requires_hr_signature,
			expiry_days, sort_order
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`

	var id int
	err := r.db.QueryRowContext(ctx, query,
		req.Name, req.Code, req.Description, req.TemplateID,
		req.RequiresSignature, req.RequiresEmployeeSignature, req.RequiresManagerSignature, req.RequiresHRSignature,
		req.ExpiryDays, req.SortOrder,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("%s: failed to insert document type: %w", op, err)
	}

	return id, nil
}

// GetDocumentTypeByID retrieves document type by ID
func (r *Repo) GetDocumentTypeByID(ctx context.Context, id int) (*hrmmodel.DocumentType, error) {
	const op = "storage.repo.GetDocumentTypeByID"

	const query = `
		SELECT id, name, code, description, template_id,
			requires_signature, requires_employee_signature, requires_manager_signature, requires_hr_signature,
			expiry_days, is_active, sort_order, created_at, updated_at
		FROM hrm_document_types
		WHERE id = $1`

	dt, err := r.scanDocumentType(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get document type: %w", op, err)
	}

	return dt, nil
}

// GetDocumentTypes retrieves all document types
func (r *Repo) GetDocumentTypes(ctx context.Context, activeOnly bool) ([]*hrmmodel.DocumentType, error) {
	const op = "storage.repo.GetDocumentTypes"

	var query strings.Builder
	query.WriteString(`
		SELECT id, name, code, description, template_id,
			requires_signature, requires_employee_signature, requires_manager_signature, requires_hr_signature,
			expiry_days, is_active, sort_order, created_at, updated_at
		FROM hrm_document_types
	`)

	if activeOnly {
		query.WriteString(" WHERE is_active = TRUE")
	}

	query.WriteString(" ORDER BY sort_order, name")

	rows, err := r.db.QueryContext(ctx, query.String())
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query document types: %w", op, err)
	}
	defer rows.Close()

	var types []*hrmmodel.DocumentType
	for rows.Next() {
		dt, err := r.scanDocumentTypeRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan document type: %w", op, err)
		}
		types = append(types, dt)
	}

	if types == nil {
		types = make([]*hrmmodel.DocumentType, 0)
	}

	return types, nil
}

// EditDocumentType updates document type
func (r *Repo) EditDocumentType(ctx context.Context, id int, req hrm.EditDocumentTypeRequest) error {
	const op = "storage.repo.EditDocumentType"

	var updates []string
	var args []interface{}
	argIdx := 1

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Code != nil {
		updates = append(updates, fmt.Sprintf("code = $%d", argIdx))
		args = append(args, *req.Code)
		argIdx++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.TemplateID != nil {
		updates = append(updates, fmt.Sprintf("template_id = $%d", argIdx))
		args = append(args, *req.TemplateID)
		argIdx++
	}
	if req.RequiresSignature != nil {
		updates = append(updates, fmt.Sprintf("requires_signature = $%d", argIdx))
		args = append(args, *req.RequiresSignature)
		argIdx++
	}
	if req.RequiresEmployeeSignature != nil {
		updates = append(updates, fmt.Sprintf("requires_employee_signature = $%d", argIdx))
		args = append(args, *req.RequiresEmployeeSignature)
		argIdx++
	}
	if req.RequiresManagerSignature != nil {
		updates = append(updates, fmt.Sprintf("requires_manager_signature = $%d", argIdx))
		args = append(args, *req.RequiresManagerSignature)
		argIdx++
	}
	if req.RequiresHRSignature != nil {
		updates = append(updates, fmt.Sprintf("requires_hr_signature = $%d", argIdx))
		args = append(args, *req.RequiresHRSignature)
		argIdx++
	}
	if req.ExpiryDays != nil {
		updates = append(updates, fmt.Sprintf("expiry_days = $%d", argIdx))
		args = append(args, *req.ExpiryDays)
		argIdx++
	}
	if req.IsActive != nil {
		updates = append(updates, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *req.IsActive)
		argIdx++
	}
	if req.SortOrder != nil {
		updates = append(updates, fmt.Sprintf("sort_order = $%d", argIdx))
		args = append(args, *req.SortOrder)
		argIdx++
	}

	if len(updates) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE hrm_document_types SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to update document type: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteDocumentType deletes document type
func (r *Repo) DeleteDocumentType(ctx context.Context, id int) error {
	const op = "storage.repo.DeleteDocumentType"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_document_types WHERE id = $1", id)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			return storage.ErrForeignKeyViolation
		}
		return fmt.Errorf("%s: failed to delete document type: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- Documents ---

// AddDocument creates a new document
func (r *Repo) AddDocument(ctx context.Context, req hrm.AddDocumentRequest, createdBy *int64) (int64, error) {
	const op = "storage.repo.AddDocument"

	const query = `
		INSERT INTO hrm_documents (
			employee_id, document_type_id, title, document_number, description,
			file_id, issue_date, effective_date, expiry_date, status, created_by, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.EmployeeID, req.DocumentTypeID, req.Title, req.DocumentNumber, req.Description,
		req.FileID, req.IssueDate, req.EffectiveDate, req.ExpiryDate, hrmmodel.DocumentStatusDraft, createdBy, req.Notes,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("%s: failed to insert document: %w", op, err)
	}

	return id, nil
}

// GetDocumentByID retrieves document by ID
func (r *Repo) GetDocumentByID(ctx context.Context, id int64) (*hrmmodel.Document, error) {
	const op = "storage.repo.GetDocumentByID"

	const query = `
		SELECT id, employee_id, document_type_id, title, document_number, description,
			file_id, issue_date, effective_date, expiry_date, status, created_by, notes, created_at, updated_at
		FROM hrm_documents
		WHERE id = $1`

	d, err := r.scanDocument(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get document: %w", op, err)
	}

	return d, nil
}

// GetDocuments retrieves documents with filters
func (r *Repo) GetDocuments(ctx context.Context, filter hrm.DocumentFilter) ([]*hrmmodel.Document, error) {
	const op = "storage.repo.GetDocuments"

	var query strings.Builder
	query.WriteString(`
		SELECT id, employee_id, document_type_id, title, document_number, description,
			file_id, issue_date, effective_date, expiry_date, status, created_by, notes, created_at, updated_at
		FROM hrm_documents
		WHERE 1=1
	`)

	args := []interface{}{}
	argIdx := 1

	if filter.EmployeeID != nil {
		query.WriteString(fmt.Sprintf(" AND employee_id = $%d", argIdx))
		args = append(args, *filter.EmployeeID)
		argIdx++
	}
	if filter.DocumentTypeID != nil {
		query.WriteString(fmt.Sprintf(" AND document_type_id = $%d", argIdx))
		args = append(args, *filter.DocumentTypeID)
		argIdx++
	}
	if filter.Status != nil {
		query.WriteString(fmt.Sprintf(" AND status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.ExpiringDays != nil {
		query.WriteString(fmt.Sprintf(" AND expiry_date IS NOT NULL AND expiry_date BETWEEN NOW() AND NOW() + INTERVAL '%d days'", *filter.ExpiringDays))
	}
	if filter.Expired != nil && *filter.Expired {
		query.WriteString(" AND expiry_date IS NOT NULL AND expiry_date < NOW()")
	}
	if filter.Search != nil {
		query.WriteString(fmt.Sprintf(" AND (title ILIKE $%d OR document_number ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+*filter.Search+"%")
		argIdx++
	}

	query.WriteString(" ORDER BY created_at DESC")

	if filter.Limit > 0 {
		query.WriteString(fmt.Sprintf(" LIMIT $%d", argIdx))
		args = append(args, filter.Limit)
		argIdx++
	}
	if filter.Offset > 0 {
		query.WriteString(fmt.Sprintf(" OFFSET $%d", argIdx))
		args = append(args, filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query documents: %w", op, err)
	}
	defer rows.Close()

	var documents []*hrmmodel.Document
	for rows.Next() {
		d, err := r.scanDocumentRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan document: %w", op, err)
		}
		documents = append(documents, d)
	}

	if documents == nil {
		documents = make([]*hrmmodel.Document, 0)
	}

	return documents, nil
}

// EditDocument updates document
func (r *Repo) EditDocument(ctx context.Context, id int64, req hrm.EditDocumentRequest) error {
	const op = "storage.repo.EditDocument"

	var updates []string
	var args []interface{}
	argIdx := 1

	if req.Title != nil {
		updates = append(updates, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *req.Title)
		argIdx++
	}
	if req.DocumentNumber != nil {
		updates = append(updates, fmt.Sprintf("document_number = $%d", argIdx))
		args = append(args, *req.DocumentNumber)
		argIdx++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.FileID != nil {
		updates = append(updates, fmt.Sprintf("file_id = $%d", argIdx))
		args = append(args, *req.FileID)
		argIdx++
	}
	if req.IssueDate != nil {
		updates = append(updates, fmt.Sprintf("issue_date = $%d", argIdx))
		args = append(args, *req.IssueDate)
		argIdx++
	}
	if req.EffectiveDate != nil {
		updates = append(updates, fmt.Sprintf("effective_date = $%d", argIdx))
		args = append(args, *req.EffectiveDate)
		argIdx++
	}
	if req.ExpiryDate != nil {
		updates = append(updates, fmt.Sprintf("expiry_date = $%d", argIdx))
		args = append(args, *req.ExpiryDate)
		argIdx++
	}
	if req.Status != nil {
		updates = append(updates, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
	}
	if req.Notes != nil {
		updates = append(updates, fmt.Sprintf("notes = $%d", argIdx))
		args = append(args, *req.Notes)
		argIdx++
	}

	if len(updates) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE hrm_documents SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to update document: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteDocument deletes document
func (r *Repo) DeleteDocument(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteDocument"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_documents WHERE id = $1", id)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			return storage.ErrForeignKeyViolation
		}
		return fmt.Errorf("%s: failed to delete document: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- Document Signatures ---

// AddDocumentSignature adds signature requirement
func (r *Repo) AddDocumentSignature(ctx context.Context, req hrm.AddSignatureRequest) (int64, error) {
	const op = "storage.repo.AddDocumentSignature"

	const query = `
		INSERT INTO hrm_document_signatures (
			document_id, signer_user_id, signer_role, status, sign_order, notes
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.DocumentID, req.SignerUserID, req.SignerRole, hrmmodel.SignatureStatusPending, req.SignOrder, req.Notes,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("%s: failed to insert signature: %w", op, err)
	}

	return id, nil
}

// GetHRMDocumentSignatures retrieves signatures for document
func (r *Repo) GetHRMDocumentSignatures(ctx context.Context, filter hrm.SignatureFilter) ([]*hrmmodel.DocumentSignature, error) {
	const op = "storage.repo.GetHRMDocumentSignatures"

	var query strings.Builder
	query.WriteString(`
		SELECT id, document_id, signer_user_id, signer_role, status, signed_at, signature_ip,
			rejection_reason, sign_order, notes, created_at, updated_at
		FROM hrm_document_signatures
		WHERE 1=1
	`)

	args := []interface{}{}
	argIdx := 1

	if filter.DocumentID != nil {
		query.WriteString(fmt.Sprintf(" AND document_id = $%d", argIdx))
		args = append(args, *filter.DocumentID)
		argIdx++
	}
	if filter.SignerUserID != nil {
		query.WriteString(fmt.Sprintf(" AND signer_user_id = $%d", argIdx))
		args = append(args, *filter.SignerUserID)
		argIdx++
	}
	if filter.SignerRole != nil {
		query.WriteString(fmt.Sprintf(" AND signer_role = $%d", argIdx))
		args = append(args, *filter.SignerRole)
		argIdx++
	}
	if filter.Status != nil {
		query.WriteString(fmt.Sprintf(" AND status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}

	query.WriteString(" ORDER BY sign_order")

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query signatures: %w", op, err)
	}
	defer rows.Close()

	var signatures []*hrmmodel.DocumentSignature
	for rows.Next() {
		var s hrmmodel.DocumentSignature
		var signedAt, updatedAt sql.NullTime
		var signatureIP, rejectionReason, notes sql.NullString

		err := rows.Scan(
			&s.ID, &s.DocumentID, &s.SignerUserID, &s.SignerRole, &s.Status, &signedAt, &signatureIP,
			&rejectionReason, &s.SignOrder, &notes, &s.CreatedAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan signature: %w", op, err)
		}

		if signedAt.Valid {
			s.SignedAt = &signedAt.Time
		}
		if signatureIP.Valid {
			s.SignatureIP = &signatureIP.String
		}
		if rejectionReason.Valid {
			s.RejectionReason = &rejectionReason.String
		}
		if notes.Valid {
			s.Notes = &notes.String
		}
		if updatedAt.Valid {
			s.UpdatedAt = &updatedAt.Time
		}

		signatures = append(signatures, &s)
	}

	if signatures == nil {
		signatures = make([]*hrmmodel.DocumentSignature, 0)
	}

	return signatures, nil
}

// SignHRMDocument signs or rejects document
func (r *Repo) SignHRMDocument(ctx context.Context, signatureID int64, signed bool, reason *string, ip string) error {
	const op = "storage.repo.SignHRMDocument"

	var query string
	var args []interface{}

	if signed {
		query = `UPDATE hrm_document_signatures SET status = $1, signed_at = $2, signature_ip = $3 WHERE id = $4`
		args = []interface{}{hrmmodel.SignatureStatusSigned, time.Now(), ip, signatureID}
	} else {
		query = `UPDATE hrm_document_signatures SET status = $1, rejection_reason = $2 WHERE id = $3`
		args = []interface{}{hrmmodel.SignatureStatusRejected, reason, signatureID}
	}

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to sign document: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- Document Templates ---

// AddDocumentTemplate creates a new template
func (r *Repo) AddDocumentTemplate(ctx context.Context, req hrm.AddDocumentTemplateRequest, createdBy *int64) (int64, error) {
	const op = "storage.repo.AddDocumentTemplate"

	var placeholdersJSON []byte
	if req.Placeholders != nil {
		var err error
		placeholdersJSON, err = json.Marshal(req.Placeholders)
		if err != nil {
			return 0, fmt.Errorf("%s: failed to marshal placeholders: %w", op, err)
		}
	}

	const query = `
		INSERT INTO hrm_document_templates (
			document_type_id, name, description, content, file_id, placeholders, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.DocumentTypeID, req.Name, req.Description, req.Content, req.FileID, placeholdersJSON, createdBy,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("%s: failed to insert template: %w", op, err)
	}

	return id, nil
}

// GetDocumentTemplateByID retrieves template by ID
func (r *Repo) GetDocumentTemplateByID(ctx context.Context, id int64) (*hrmmodel.DocumentTemplate, error) {
	const op = "storage.repo.GetDocumentTemplateByID"

	const query = `
		SELECT id, document_type_id, name, description, content, file_id, placeholders,
			is_active, version, created_by, created_at, updated_at
		FROM hrm_document_templates
		WHERE id = $1`

	t, err := r.scanDocumentTemplate(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get template: %w", op, err)
	}

	return t, nil
}

// GetDocumentTemplates retrieves templates with filters
func (r *Repo) GetDocumentTemplates(ctx context.Context, filter hrm.DocumentTemplateFilter) ([]*hrmmodel.DocumentTemplate, error) {
	const op = "storage.repo.GetDocumentTemplates"

	var query strings.Builder
	query.WriteString(`
		SELECT id, document_type_id, name, description, content, file_id, placeholders,
			is_active, version, created_by, created_at, updated_at
		FROM hrm_document_templates
		WHERE 1=1
	`)

	args := []interface{}{}
	argIdx := 1

	if filter.DocumentTypeID != nil {
		query.WriteString(fmt.Sprintf(" AND document_type_id = $%d", argIdx))
		args = append(args, *filter.DocumentTypeID)
		argIdx++
	}
	if filter.IsActive != nil {
		query.WriteString(fmt.Sprintf(" AND is_active = $%d", argIdx))
		args = append(args, *filter.IsActive)
		argIdx++
	}
	if filter.Search != nil {
		query.WriteString(fmt.Sprintf(" AND name ILIKE $%d", argIdx))
		args = append(args, "%"+*filter.Search+"%")
		argIdx++
	}

	query.WriteString(" ORDER BY name")

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query templates: %w", op, err)
	}
	defer rows.Close()

	var templates []*hrmmodel.DocumentTemplate
	for rows.Next() {
		t, err := r.scanDocumentTemplateRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan template: %w", op, err)
		}
		templates = append(templates, t)
	}

	if templates == nil {
		templates = make([]*hrmmodel.DocumentTemplate, 0)
	}

	return templates, nil
}

// EditDocumentTemplate updates template
func (r *Repo) EditDocumentTemplate(ctx context.Context, id int64, req hrm.EditDocumentTemplateRequest) error {
	const op = "storage.repo.EditDocumentTemplate"

	var updates []string
	var args []interface{}
	argIdx := 1

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.Content != nil {
		updates = append(updates, fmt.Sprintf("content = $%d", argIdx))
		args = append(args, *req.Content)
		argIdx++
	}
	if req.FileID != nil {
		updates = append(updates, fmt.Sprintf("file_id = $%d", argIdx))
		args = append(args, *req.FileID)
		argIdx++
	}
	if req.Placeholders != nil {
		placeholdersJSON, _ := json.Marshal(req.Placeholders)
		updates = append(updates, fmt.Sprintf("placeholders = $%d", argIdx))
		args = append(args, placeholdersJSON)
		argIdx++
	}
	if req.IsActive != nil {
		updates = append(updates, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *req.IsActive)
		argIdx++
	}

	if len(updates) == 0 {
		return nil
	}

	// Increment version
	updates = append(updates, "version = version + 1")

	query := fmt.Sprintf("UPDATE hrm_document_templates SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to update template: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteDocumentTemplate deletes template
func (r *Repo) DeleteDocumentTemplate(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteDocumentTemplate"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_document_templates WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("%s: failed to delete template: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- Helpers ---

func (r *Repo) scanDocumentType(row *sql.Row) (*hrmmodel.DocumentType, error) {
	var dt hrmmodel.DocumentType
	var code, description sql.NullString
	var templateID sql.NullInt64
	var expiryDays sql.NullInt64
	var updatedAt sql.NullTime

	err := row.Scan(
		&dt.ID, &dt.Name, &code, &description, &templateID,
		&dt.RequiresSignature, &dt.RequiresEmployeeSignature, &dt.RequiresManagerSignature, &dt.RequiresHRSignature,
		&expiryDays, &dt.IsActive, &dt.SortOrder, &dt.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if code.Valid {
		dt.Code = &code.String
	}
	if description.Valid {
		dt.Description = &description.String
	}
	if templateID.Valid {
		dt.TemplateID = &templateID.Int64
	}
	if expiryDays.Valid {
		d := int(expiryDays.Int64)
		dt.ExpiryDays = &d
	}
	if updatedAt.Valid {
		dt.UpdatedAt = &updatedAt.Time
	}

	return &dt, nil
}

func (r *Repo) scanDocumentTypeRow(rows *sql.Rows) (*hrmmodel.DocumentType, error) {
	var dt hrmmodel.DocumentType
	var code, description sql.NullString
	var templateID sql.NullInt64
	var expiryDays sql.NullInt64
	var updatedAt sql.NullTime

	err := rows.Scan(
		&dt.ID, &dt.Name, &code, &description, &templateID,
		&dt.RequiresSignature, &dt.RequiresEmployeeSignature, &dt.RequiresManagerSignature, &dt.RequiresHRSignature,
		&expiryDays, &dt.IsActive, &dt.SortOrder, &dt.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if code.Valid {
		dt.Code = &code.String
	}
	if description.Valid {
		dt.Description = &description.String
	}
	if templateID.Valid {
		dt.TemplateID = &templateID.Int64
	}
	if expiryDays.Valid {
		d := int(expiryDays.Int64)
		dt.ExpiryDays = &d
	}
	if updatedAt.Valid {
		dt.UpdatedAt = &updatedAt.Time
	}

	return &dt, nil
}

func (r *Repo) scanDocument(row *sql.Row) (*hrmmodel.Document, error) {
	var d hrmmodel.Document
	var documentNumber, description, notes sql.NullString
	var fileID, createdBy sql.NullInt64
	var issueDate, effectiveDate, expiryDate, updatedAt sql.NullTime

	err := row.Scan(
		&d.ID, &d.EmployeeID, &d.DocumentTypeID, &d.Title, &documentNumber, &description,
		&fileID, &issueDate, &effectiveDate, &expiryDate, &d.Status, &createdBy, &notes, &d.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if documentNumber.Valid {
		d.DocumentNumber = &documentNumber.String
	}
	if description.Valid {
		d.Description = &description.String
	}
	if fileID.Valid {
		d.FileID = &fileID.Int64
	}
	if issueDate.Valid {
		d.IssueDate = &issueDate.Time
	}
	if effectiveDate.Valid {
		d.EffectiveDate = &effectiveDate.Time
	}
	if expiryDate.Valid {
		d.ExpiryDate = &expiryDate.Time
	}
	if createdBy.Valid {
		d.CreatedBy = &createdBy.Int64
	}
	if notes.Valid {
		d.Notes = &notes.String
	}
	if updatedAt.Valid {
		d.UpdatedAt = &updatedAt.Time
	}

	return &d, nil
}

func (r *Repo) scanDocumentRow(rows *sql.Rows) (*hrmmodel.Document, error) {
	var d hrmmodel.Document
	var documentNumber, description, notes sql.NullString
	var fileID, createdBy sql.NullInt64
	var issueDate, effectiveDate, expiryDate, updatedAt sql.NullTime

	err := rows.Scan(
		&d.ID, &d.EmployeeID, &d.DocumentTypeID, &d.Title, &documentNumber, &description,
		&fileID, &issueDate, &effectiveDate, &expiryDate, &d.Status, &createdBy, &notes, &d.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if documentNumber.Valid {
		d.DocumentNumber = &documentNumber.String
	}
	if description.Valid {
		d.Description = &description.String
	}
	if fileID.Valid {
		d.FileID = &fileID.Int64
	}
	if issueDate.Valid {
		d.IssueDate = &issueDate.Time
	}
	if effectiveDate.Valid {
		d.EffectiveDate = &effectiveDate.Time
	}
	if expiryDate.Valid {
		d.ExpiryDate = &expiryDate.Time
	}
	if createdBy.Valid {
		d.CreatedBy = &createdBy.Int64
	}
	if notes.Valid {
		d.Notes = &notes.String
	}
	if updatedAt.Valid {
		d.UpdatedAt = &updatedAt.Time
	}

	return &d, nil
}

func (r *Repo) scanDocumentTemplate(row *sql.Row) (*hrmmodel.DocumentTemplate, error) {
	var t hrmmodel.DocumentTemplate
	var description, content sql.NullString
	var fileID, createdBy sql.NullInt64
	var placeholdersJSON []byte
	var updatedAt sql.NullTime

	err := row.Scan(
		&t.ID, &t.DocumentTypeID, &t.Name, &description, &content, &fileID, &placeholdersJSON,
		&t.IsActive, &t.Version, &createdBy, &t.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if description.Valid {
		t.Description = &description.String
	}
	if content.Valid {
		t.Content = &content.String
	}
	if fileID.Valid {
		t.FileID = &fileID.Int64
	}
	if placeholdersJSON != nil {
		json.Unmarshal(placeholdersJSON, &t.Placeholders)
	}
	if createdBy.Valid {
		t.CreatedBy = &createdBy.Int64
	}
	if updatedAt.Valid {
		t.UpdatedAt = &updatedAt.Time
	}

	return &t, nil
}

func (r *Repo) scanDocumentTemplateRow(rows *sql.Rows) (*hrmmodel.DocumentTemplate, error) {
	var t hrmmodel.DocumentTemplate
	var description, content sql.NullString
	var fileID, createdBy sql.NullInt64
	var placeholdersJSON []byte
	var updatedAt sql.NullTime

	err := rows.Scan(
		&t.ID, &t.DocumentTypeID, &t.Name, &description, &content, &fileID, &placeholdersJSON,
		&t.IsActive, &t.Version, &createdBy, &t.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if description.Valid {
		t.Description = &description.String
	}
	if content.Valid {
		t.Content = &content.String
	}
	if fileID.Valid {
		t.FileID = &fileID.Int64
	}
	if placeholdersJSON != nil {
		json.Unmarshal(placeholdersJSON, &t.Placeholders)
	}
	if createdBy.Valid {
		t.CreatedBy = &createdBy.Int64
	}
	if updatedAt.Valid {
		t.UpdatedAt = &updatedAt.Time
	}

	return &t, nil
}
