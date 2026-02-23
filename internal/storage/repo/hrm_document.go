package repo

import (
	"context"
	"database/sql"
	"fmt"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/hrm/document"
	"srmt-admin/internal/storage"
	"strings"
)

// ==================== HR Documents ====================

func (r *Repo) CreateHRDocument(ctx context.Context, req dto.CreateHRDocumentRequest, createdBy int64) (int64, error) {
	const op = "repo.CreateHRDocument"

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO hr_documents (title, type, category, number, date, content, file_url,
			department_id, employee_id, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`

	var id int64
	err = tx.QueryRowContext(ctx, query,
		req.Title, req.Type, req.Category, req.Number, req.Date,
		req.Content, req.FileURL, req.DepartmentID, req.EmployeeID, createdBy,
	).Scan(&id)
	if err != nil {
		if translated := r.translator.Translate(err, op); translated != nil {
			return 0, translated
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	for _, sig := range req.Signatures {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO document_signatures (document_id, signer_id, sign_order) VALUES ($1, $2, $3)`,
			id, sig.SignerID, sig.Order,
		)
		if err != nil {
			return 0, fmt.Errorf("%s: insert signature: %w", op, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("%s: commit: %w", op, err)
	}
	return id, nil
}

func (r *Repo) GetHRDocumentByID(ctx context.Context, id int64) (*document.HRDocument, error) {
	const op = "repo.GetHRDocumentByID"

	query := `
		SELECT d.id, d.title, d.type, d.category, d.number, d.date, d.status,
			   d.content, d.file_url, d.department_id, d.employee_id, d.created_by,
			   d.version, d.created_at, d.updated_at
		FROM hr_documents d
		WHERE d.id = $1`

	doc, err := scanHRDocument(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrHRDocumentNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	sigQuery := `
		SELECT ds.id, ds.document_id, ds.signer_id,
			   COALESCE(c.fio, ''), COALESCE(p.name, ''),
			   ds.status, ds.signed_at, ds.comment, ds.sign_order, ds.created_at
		FROM document_signatures ds
		LEFT JOIN contacts c ON ds.signer_id = c.id
		LEFT JOIN positions p ON c.position_id = p.id
		WHERE ds.document_id = $1
		ORDER BY ds.sign_order`

	rows, err := r.db.QueryContext(ctx, sigQuery, id)
	if err != nil {
		return nil, fmt.Errorf("%s: signatures: %w", op, err)
	}
	defer rows.Close()

	for rows.Next() {
		sig, err := scanSignature(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: scan signature: %w", op, err)
		}
		doc.Signatures = append(doc.Signatures, *sig)
	}
	if doc.Signatures == nil {
		doc.Signatures = []document.Signature{}
	}

	return doc, nil
}

func (r *Repo) GetAllHRDocuments(ctx context.Context, filters dto.HRDocumentFilters) ([]*document.HRDocument, error) {
	const op = "repo.GetAllHRDocuments"

	query := `
		SELECT d.id, d.title, d.type, d.category, d.number, d.date, d.status,
			   d.content, d.file_url, d.department_id, d.employee_id, d.created_by,
			   d.version, d.created_at, d.updated_at
		FROM hr_documents d`

	var conditions []string
	var args []interface{}
	argIdx := 1

	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("d.status = $%d", argIdx))
		args = append(args, *filters.Status)
		argIdx++
	}
	if filters.Type != nil {
		conditions = append(conditions, fmt.Sprintf("d.type = $%d", argIdx))
		args = append(args, *filters.Type)
		argIdx++
	}
	if filters.Category != nil {
		conditions = append(conditions, fmt.Sprintf("d.category = $%d", argIdx))
		args = append(args, *filters.Category)
		argIdx++
	}
	if filters.Search != nil {
		conditions = append(conditions, fmt.Sprintf("(d.title ILIKE $%d OR d.number ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+*filters.Search+"%")
		argIdx++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY d.created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var docs []*document.HRDocument
	for rows.Next() {
		doc, err := scanHRDocument(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		doc.Signatures = []document.Signature{}
		docs = append(docs, doc)
	}
	return docs, nil
}

func (r *Repo) UpdateHRDocument(ctx context.Context, id int64, req dto.UpdateHRDocumentRequest) error {
	const op = "repo.UpdateHRDocument"

	var setClauses []string
	var args []interface{}
	argIdx := 1

	if req.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *req.Title)
		argIdx++
	}
	if req.Type != nil {
		setClauses = append(setClauses, fmt.Sprintf("type = $%d", argIdx))
		args = append(args, *req.Type)
		argIdx++
	}
	if req.Category != nil {
		setClauses = append(setClauses, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, *req.Category)
		argIdx++
	}
	if req.Number != nil {
		setClauses = append(setClauses, fmt.Sprintf("number = $%d", argIdx))
		args = append(args, *req.Number)
		argIdx++
	}
	if req.Date != nil {
		setClauses = append(setClauses, fmt.Sprintf("date = $%d", argIdx))
		args = append(args, *req.Date)
		argIdx++
	}
	if req.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
	}
	if req.Content != nil {
		setClauses = append(setClauses, fmt.Sprintf("content = $%d", argIdx))
		args = append(args, *req.Content)
		argIdx++
	}
	if req.FileURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("file_url = $%d", argIdx))
		args = append(args, *req.FileURL)
		argIdx++
	}
	if req.DepartmentID != nil {
		setClauses = append(setClauses, fmt.Sprintf("department_id = $%d", argIdx))
		args = append(args, *req.DepartmentID)
		argIdx++
	}
	if req.EmployeeID != nil {
		setClauses = append(setClauses, fmt.Sprintf("employee_id = $%d", argIdx))
		args = append(args, *req.EmployeeID)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE hr_documents SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if translated := r.translator.Translate(err, op); translated != nil {
			return translated
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrHRDocumentNotFound
	}
	return nil
}

func (r *Repo) DeleteHRDocument(ctx context.Context, id int64) error {
	const op = "repo.DeleteHRDocument"

	result, err := r.db.ExecContext(ctx, "DELETE FROM hr_documents WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrHRDocumentNotFound
	}
	return nil
}

func (r *Repo) GetHRDocumentSignatures(ctx context.Context, documentID int64) ([]document.Signature, error) {
	const op = "repo.GetHRDocumentSignatures"

	query := `
		SELECT ds.id, ds.document_id, ds.signer_id,
			   COALESCE(c.fio, ''), COALESCE(p.name, ''),
			   ds.status, ds.signed_at, ds.comment, ds.sign_order, ds.created_at
		FROM document_signatures ds
		LEFT JOIN contacts c ON ds.signer_id = c.id
		LEFT JOIN positions p ON c.position_id = p.id
		WHERE ds.document_id = $1
		ORDER BY ds.sign_order`

	rows, err := r.db.QueryContext(ctx, query, documentID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var sigs []document.Signature
	for rows.Next() {
		sig, err := scanSignature(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		sigs = append(sigs, *sig)
	}
	if sigs == nil {
		sigs = []document.Signature{}
	}
	return sigs, nil
}

// ==================== Document Requests ====================

func (r *Repo) CreateDocumentRequest(ctx context.Context, employeeID int64, req dto.CreateDocumentRequestReq) (int64, error) {
	const op = "repo.CreateDocumentRequest"

	query := `
		INSERT INTO document_requests (employee_id, document_type, purpose)
		VALUES ($1, $2, $3)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query, employeeID, req.DocumentType, req.Purpose).Scan(&id)
	if err != nil {
		if translated := r.translator.Translate(err, op); translated != nil {
			return 0, translated
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

func (r *Repo) GetAllDocumentRequests(ctx context.Context, employeeID *int64) ([]*document.DocumentRequest, error) {
	const op = "repo.GetAllDocumentRequests"

	query := `
		SELECT dr.id, dr.employee_id, COALESCE(c.fio, ''),
			   dr.document_type, dr.purpose, dr.status,
			   dr.rejection_reason, dr.completed_at, dr.created_at, dr.updated_at
		FROM document_requests dr
		LEFT JOIN contacts c ON dr.employee_id = c.id`

	var args []interface{}
	if employeeID != nil {
		query += " WHERE dr.employee_id = $1"
		args = append(args, *employeeID)
	}
	query += " ORDER BY dr.created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var requests []*document.DocumentRequest
	for rows.Next() {
		req, err := scanDocumentRequest(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		requests = append(requests, req)
	}
	return requests, nil
}

func (r *Repo) GetDocumentRequestByID(ctx context.Context, id int64) (*document.DocumentRequest, error) {
	const op = "repo.GetDocumentRequestByID"

	query := `
		SELECT dr.id, dr.employee_id, COALESCE(c.fio, ''),
			   dr.document_type, dr.purpose, dr.status,
			   dr.rejection_reason, dr.completed_at, dr.created_at, dr.updated_at
		FROM document_requests dr
		LEFT JOIN contacts c ON dr.employee_id = c.id
		WHERE dr.id = $1`

	req, err := scanDocumentRequest(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrDocumentRequestNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return req, nil
}

func (r *Repo) UpdateDocumentRequestStatus(ctx context.Context, id int64, status string, rejectionReason *string) error {
	const op = "repo.UpdateDocumentRequestStatus"

	query := `UPDATE document_requests SET status = $1, rejection_reason = $2 WHERE id = $3`
	result, err := r.db.ExecContext(ctx, query, status, rejectionReason, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrDocumentRequestNotFound
	}
	return nil
}

// ==================== Scanners ====================

func scanHRDocument(s scannable) (*document.HRDocument, error) {
	var d document.HRDocument
	err := s.Scan(
		&d.ID, &d.Title, &d.Type, &d.Category, &d.Number, &d.Date, &d.Status,
		&d.Content, &d.FileURL, &d.DepartmentID, &d.EmployeeID, &d.CreatedBy,
		&d.Version, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func scanSignature(s scannable) (*document.Signature, error) {
	var sig document.Signature
	err := s.Scan(
		&sig.ID, &sig.DocumentID, &sig.SignerID,
		&sig.SignerName, &sig.SignerPosition,
		&sig.Status, &sig.SignedAt, &sig.Comment, &sig.Order, &sig.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &sig, nil
}

func scanDocumentRequest(s scannable) (*document.DocumentRequest, error) {
	var dr document.DocumentRequest
	err := s.Scan(
		&dr.ID, &dr.EmployeeID, &dr.EmployeeName,
		&dr.DocumentType, &dr.Purpose, &dr.Status,
		&dr.RejectionReason, &dr.CompletedAt, &dr.CreatedAt, &dr.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &dr, nil
}
