package repo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/signature"
	"srmt-admin/internal/lib/model/user"
	"srmt-admin/internal/storage"
)

// GetStatusIDByCode returns status ID by its code
func (r *Repo) GetStatusIDByCode(ctx context.Context, code string) (int, error) {
	const op = "storage.repo.GetStatusIDByCode"
	const query = `SELECT id FROM document_status WHERE code = $1`

	var id int
	err := r.db.QueryRowContext(ctx, query, code).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("%s: status not found: %w", op, storage.ErrNotFound)
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

// GetPendingSignatureDocuments returns all documents waiting for signature across all document types
func (r *Repo) GetPendingSignatureDocuments(ctx context.Context) ([]signature.PendingDocument, error) {
	const op = "storage.repo.GetPendingSignatureDocuments"

	// Get the status ID for 'pending_signature'
	statusID, err := r.GetStatusIDByCode(ctx, "pending_signature")
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get status ID: %w", op, err)
	}

	// Union query for all document types
	const query = `
		SELECT 'decree' as document_type, d.id, d.name, d.number, d.document_date,
			   dt.id as type_id, dt.name as type_name,
			   o.name as organization, d.organization_id,
			   rc.fio as responsible_name, d.responsible_contact_id,
			   d.created_at, uc.fio as created_by
		FROM decrees d
		LEFT JOIN decree_type dt ON d.type_id = dt.id
		LEFT JOIN organizations o ON d.organization_id = o.id
		LEFT JOIN contacts rc ON d.responsible_contact_id = rc.id
		LEFT JOIN users u ON d.created_by_user_id = u.id
		LEFT JOIN contacts uc ON u.contact_id = uc.id
		WHERE d.status_id = $1

		UNION ALL

		SELECT 'report' as document_type, r.id, r.name, r.number, r.document_date,
			   rt.id as type_id, rt.name as type_name,
			   o.name as organization, r.organization_id,
			   rc.fio as responsible_name, r.responsible_contact_id,
			   r.created_at, uc.fio as created_by
		FROM reports r
		LEFT JOIN report_type rt ON r.type_id = rt.id
		LEFT JOIN organizations o ON r.organization_id = o.id
		LEFT JOIN contacts rc ON r.responsible_contact_id = rc.id
		LEFT JOIN users u ON r.created_by_user_id = u.id
		LEFT JOIN contacts uc ON u.contact_id = uc.id
		WHERE r.status_id = $1

		UNION ALL

		SELECT 'letter' as document_type, l.id, l.name, l.number, l.document_date,
			   lt.id as type_id, lt.name as type_name,
			   o.name as organization, l.organization_id,
			   rc.fio as responsible_name, l.responsible_contact_id,
			   l.created_at, uc.fio as created_by
		FROM letters l
		LEFT JOIN letter_type lt ON l.type_id = lt.id
		LEFT JOIN organizations o ON l.organization_id = o.id
		LEFT JOIN contacts rc ON l.responsible_contact_id = rc.id
		LEFT JOIN users u ON l.created_by_user_id = u.id
		LEFT JOIN contacts uc ON u.contact_id = uc.id
		WHERE l.status_id = $1

		UNION ALL

		SELECT 'instruction' as document_type, i.id, i.name, i.number, i.document_date,
			   it.id as type_id, it.name as type_name,
			   o.name as organization, i.organization_id,
			   rc.fio as responsible_name, i.responsible_contact_id,
			   i.created_at, uc.fio as created_by
		FROM instructions i
		LEFT JOIN instruction_type it ON i.type_id = it.id
		LEFT JOIN organizations o ON i.organization_id = o.id
		LEFT JOIN contacts rc ON i.responsible_contact_id = rc.id
		LEFT JOIN users u ON i.created_by_user_id = u.id
		LEFT JOIN contacts uc ON u.contact_id = uc.id
		WHERE i.status_id = $1

		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, statusID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query: %w", op, err)
	}
	defer rows.Close()

	var documents []signature.PendingDocument
	for rows.Next() {
		var doc signature.PendingDocument
		var organization, responsibleName, createdBy sql.NullString
		var organizationID, responsibleID sql.NullInt64

		err := rows.Scan(
			&doc.DocumentType,
			&doc.DocumentID,
			&doc.Name,
			&doc.Number,
			&doc.DocumentDate,
			&doc.TypeID,
			&doc.TypeName,
			&organization,
			&organizationID,
			&responsibleName,
			&responsibleID,
			&doc.CreatedAt,
			&createdBy,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan: %w", op, err)
		}

		if organization.Valid {
			doc.Organization = &organization.String
		}
		if organizationID.Valid {
			doc.OrganizationID = &organizationID.Int64
		}
		if responsibleName.Valid {
			doc.ResponsibleName = &responsibleName.String
		}
		if responsibleID.Valid {
			doc.ResponsibleID = &responsibleID.Int64
		}
		if createdBy.Valid {
			doc.CreatedBy = &createdBy.String
		}

		documents = append(documents, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows error: %w", op, err)
	}

	return documents, nil
}

// SignDocument signs a document and optionally assigns executor and due date
func (r *Repo) SignDocument(ctx context.Context, docType string, docID int64, req dto.SignDocumentRequest, userID int64) error {
	const op = "storage.repo.SignDocument"

	// Validate document type
	if !signature.IsValidDocumentType(docType) {
		return fmt.Errorf("%s: invalid document type: %s", op, docType)
	}

	// Get status IDs
	pendingStatusID, err := r.GetStatusIDByCode(ctx, "pending_signature")
	if err != nil {
		return fmt.Errorf("%s: failed to get pending_signature status: %w", op, err)
	}

	signedStatusID, err := r.GetStatusIDByCode(ctx, "signed")
	if err != nil {
		return fmt.Errorf("%s: failed to get signed status: %w", op, err)
	}

	// Start transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: failed to begin transaction: %w", op, err)
	}
	defer tx.Rollback()

	// Check document exists and is in pending_signature status
	tableName := getTableName(docType)
	checkQuery := fmt.Sprintf(`SELECT status_id FROM %s WHERE id = $1`, tableName)

	var currentStatusID int
	err = tx.QueryRowContext(ctx, checkQuery, docID).Scan(&currentStatusID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("%s: document not found: %w", op, storage.ErrNotFound)
		}
		return fmt.Errorf("%s: failed to check document: %w", op, err)
	}

	if currentStatusID != pendingStatusID {
		return fmt.Errorf("%s: document is not in pending_signature status", op)
	}

	// Parse due date if provided
	var assignedDueDate *time.Time
	if req.AssignedDueDate != nil && *req.AssignedDueDate != "" {
		parsed, err := time.Parse("2006-01-02", *req.AssignedDueDate)
		if err != nil {
			return fmt.Errorf("%s: invalid due date format: %w", op, err)
		}
		assignedDueDate = &parsed
	}

	// Insert signature record
	insertQuery := `
		INSERT INTO document_signatures (
			document_type, document_id, action, resolution_text,
			assigned_executor_id, assigned_due_date, signed_by_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err = tx.ExecContext(ctx, insertQuery,
		docType, docID, signature.ActionSigned, req.ResolutionText, req.AssignedExecutorID, assignedDueDate, userID,
	)
	if err != nil {
		return fmt.Errorf("%s: failed to insert signature: %w", op, err)
	}

	// Update document status and optionally executor/due_date
	updateQuery := fmt.Sprintf(`
		UPDATE %s SET
			status_id = $1,
			updated_by_user_id = $2,
			executor_contact_id = COALESCE($3, executor_contact_id),
			due_date = COALESCE($4, due_date)
		WHERE id = $5
	`, tableName)

	_, err = tx.ExecContext(ctx, updateQuery, signedStatusID, userID, req.AssignedExecutorID, assignedDueDate, docID)
	if err != nil {
		return fmt.Errorf("%s: failed to update document: %w", op, err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: failed to commit: %w", op, err)
	}

	return nil
}

// RejectSignature rejects a document signature
func (r *Repo) RejectSignature(ctx context.Context, docType string, docID int64, reason *string, userID int64) error {
	const op = "storage.repo.RejectSignature"

	// Validate document type
	if !signature.IsValidDocumentType(docType) {
		return fmt.Errorf("%s: invalid document type: %s", op, docType)
	}

	// Get status IDs
	pendingStatusID, err := r.GetStatusIDByCode(ctx, "pending_signature")
	if err != nil {
		return fmt.Errorf("%s: failed to get pending_signature status: %w", op, err)
	}

	rejectedStatusID, err := r.GetStatusIDByCode(ctx, "signature_rejected")
	if err != nil {
		return fmt.Errorf("%s: failed to get signature_rejected status: %w", op, err)
	}

	// Start transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: failed to begin transaction: %w", op, err)
	}
	defer tx.Rollback()

	// Check document exists and is in pending_signature status
	tableName := getTableName(docType)
	checkQuery := fmt.Sprintf(`SELECT status_id FROM %s WHERE id = $1`, tableName)

	var currentStatusID int
	err = tx.QueryRowContext(ctx, checkQuery, docID).Scan(&currentStatusID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("%s: document not found: %w", op, storage.ErrNotFound)
		}
		return fmt.Errorf("%s: failed to check document: %w", op, err)
	}

	if currentStatusID != pendingStatusID {
		return fmt.Errorf("%s: document is not in pending_signature status", op)
	}

	// Insert signature record
	insertQuery := `
		INSERT INTO document_signatures (
			document_type, document_id, action, rejection_reason, signed_by_user_id
		)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err = tx.ExecContext(ctx, insertQuery, docType, docID, signature.ActionRejected, reason, userID)
	if err != nil {
		return fmt.Errorf("%s: failed to insert signature: %w", op, err)
	}

	// Update document status
	updateQuery := fmt.Sprintf(`UPDATE %s SET status_id = $1, updated_by_user_id = $2 WHERE id = $3`, tableName)
	_, err = tx.ExecContext(ctx, updateQuery, rejectedStatusID, userID, docID)
	if err != nil {
		return fmt.Errorf("%s: failed to update document: %w", op, err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: failed to commit: %w", op, err)
	}

	return nil
}

// GetDocumentSignatures returns all signatures for a document
func (r *Repo) GetDocumentSignatures(ctx context.Context, docType string, docID int64) ([]signature.Signature, error) {
	const op = "storage.repo.GetDocumentSignatures"

	// Validate document type
	if !signature.IsValidDocumentType(docType) {
		return nil, fmt.Errorf("%s: invalid document type: %s", op, docType)
	}

	const query = `
		SELECT
			s.id, s.document_type, s.document_id, s.action,
			s.resolution_text, s.rejection_reason,
			s.assigned_executor_id, c.fio as executor_name,
			s.assigned_due_date,
			s.signed_by_user_id, uc.fio as signed_by_name,
			s.signed_at
		FROM document_signatures s
		LEFT JOIN contacts c ON s.assigned_executor_id = c.id
		LEFT JOIN users u ON s.signed_by_user_id = u.id
		LEFT JOIN contacts uc ON u.contact_id = uc.id
		WHERE s.document_type = $1 AND s.document_id = $2
		ORDER BY s.signed_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, docType, docID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query: %w", op, err)
	}
	defer rows.Close()

	var signatures []signature.Signature
	for rows.Next() {
		var sig signature.Signature
		var executorID sql.NullInt64
		var executorName sql.NullString
		var signedByID sql.NullInt64
		var signedByName sql.NullString
		var assignedDueDate sql.NullTime

		err := rows.Scan(
			&sig.ID,
			&sig.DocumentType,
			&sig.DocumentID,
			&sig.Action,
			&sig.ResolutionText,
			&sig.RejectionReason,
			&executorID,
			&executorName,
			&assignedDueDate,
			&signedByID,
			&signedByName,
			&sig.SignedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan: %w", op, err)
		}

		if executorID.Valid && executorName.Valid {
			sig.AssignedExecutor = &signature.ContactShort{
				ID:   executorID.Int64,
				Name: executorName.String,
			}
		}

		if assignedDueDate.Valid {
			sig.AssignedDueDate = &assignedDueDate.Time
		}

		if signedByID.Valid {
			name := signedByName.String
			sig.SignedBy = &user.ShortInfo{
				ID:   signedByID.Int64,
				Name: &name,
			}
		}

		signatures = append(signatures, sig)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows error: %w", op, err)
	}

	return signatures, nil
}

// GetSignedStatusInfo returns info about 'signed' status
func (r *Repo) GetSignedStatusInfo(ctx context.Context) (*dto.StatusInfo, error) {
	const query = `SELECT id, code, name FROM document_status WHERE code = 'signed'`

	var status dto.StatusInfo
	err := r.db.QueryRowContext(ctx, query).Scan(&status.ID, &status.Code, &status.Name)
	if err != nil {
		return nil, err
	}
	return &status, nil
}

// GetSignatureRejectedStatusInfo returns info about 'signature_rejected' status
func (r *Repo) GetSignatureRejectedStatusInfo(ctx context.Context) (*dto.StatusInfo, error) {
	const query = `SELECT id, code, name FROM document_status WHERE code = 'signature_rejected'`

	var status dto.StatusInfo
	err := r.db.QueryRowContext(ctx, query).Scan(&status.ID, &status.Code, &status.Name)
	if err != nil {
		return nil, err
	}
	return &status, nil
}

// getTableName returns the table name for a document type
func getTableName(docType string) string {
	switch docType {
	case signature.DocTypeDecree:
		return "decrees"
	case signature.DocTypeReport:
		return "reports"
	case signature.DocTypeLetter:
		return "letters"
	case signature.DocTypeInstruction:
		return "instructions"
	default:
		return ""
	}
}
