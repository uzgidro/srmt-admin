package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"srmt-admin/internal/lib/dto"
	document_status "srmt-admin/internal/lib/model/document-status"
	"srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/model/instruction"
	instruction_type "srmt-admin/internal/lib/model/instruction-type"
	"srmt-admin/internal/lib/model/user"
	"srmt-admin/internal/storage"
)

// AddInstruction creates a new instruction
func (r *Repo) AddInstruction(ctx context.Context, req dto.AddInstructionRequest, createdByID int64) (int64, error) {
	const op = "storage.repo.AddInstruction"

	statusID := 1 // Default to 'draft' status
	if req.StatusID != nil {
		statusID = *req.StatusID
	}

	const query = `
		INSERT INTO instructions (
			name, number, document_date, description, type_id, status_id,
			responsible_contact_id, organization_id, executor_contact_id,
			due_date, parent_document_id, created_by_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.Name,
		req.Number,
		req.DocumentDate,
		req.Description,
		req.TypeID,
		statusID,
		req.ResponsibleContactID,
		req.OrganizationID,
		req.ExecutorContactID,
		req.DueDate,
		req.ParentDocumentID,
		createdByID,
	).Scan(&id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return 0, translatedErr
		}
		return 0, fmt.Errorf("%s: failed to insert instruction: %w", op, err)
	}

	return id, nil
}

// GetInstructionByID retrieves a single instruction with all joined data
func (r *Repo) GetInstructionByID(ctx context.Context, id int64) (*instruction.ResponseModel, error) {
	const op = "storage.repo.GetInstructionByID"

	query := selectInstructionFields + fromInstructionJoins + ` WHERE d.id = $1`

	row := r.db.QueryRowContext(ctx, query, id)
	doc, err := scanInstructionRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to scan instruction row: %w", op, err)
	}

	files, err := r.loadInstructionFiles(ctx, doc.ID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to load files: %w", op, err)
	}
	doc.Files = files

	links, err := r.loadInstructionLinkedDocuments(ctx, doc.ID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to load linked documents: %w", op, err)
	}
	doc.LinkedDocuments = links

	return doc, nil
}

// GetAllInstructions retrieves all instructions with optional filters
func (r *Repo) GetAllInstructions(ctx context.Context, filters dto.GetAllInstructionsFilters) ([]*instruction.ResponseModel, error) {
	const op = "storage.repo.GetAllInstructions"

	query := selectInstructionFields + fromInstructionJoins

	var whereClauses []string
	var args []interface{}
	argID := 1

	if filters.TypeID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("d.type_id = $%d", argID))
		args = append(args, *filters.TypeID)
		argID++
	}
	if filters.StatusID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("d.status_id = $%d", argID))
		args = append(args, *filters.StatusID)
		argID++
	}
	if filters.OrganizationID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("d.organization_id = $%d", argID))
		args = append(args, *filters.OrganizationID)
		argID++
	}
	if filters.ResponsibleContactID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("d.responsible_contact_id = $%d", argID))
		args = append(args, *filters.ResponsibleContactID)
		argID++
	}
	if filters.ExecutorContactID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("d.executor_contact_id = $%d", argID))
		args = append(args, *filters.ExecutorContactID)
		argID++
	}
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
	if filters.DueDateFrom != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("d.due_date >= $%d", argID))
		args = append(args, *filters.DueDateFrom)
		argID++
	}
	if filters.DueDateTo != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("d.due_date <= $%d", argID))
		args = append(args, *filters.DueDateTo)
		argID++
	}
	if filters.NameSearch != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("d.name ILIKE $%d", argID))
		args = append(args, "%"+*filters.NameSearch+"%")
		argID++
	}
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
		return nil, fmt.Errorf("%s: failed to query instructions: %w", op, err)
	}
	defer rows.Close()

	var documents []*instruction.ResponseModel
	for rows.Next() {
		doc, err := scanInstructionRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan instruction row: %w", op, err)
		}
		documents = append(documents, doc)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if documents == nil {
		documents = make([]*instruction.ResponseModel, 0)
	}

	for _, doc := range documents {
		files, err := r.loadInstructionFiles(ctx, doc.ID)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to load files for instruction %d: %w", op, doc.ID, err)
		}
		doc.Files = files
	}

	return documents, nil
}

// EditInstruction updates an instruction
func (r *Repo) EditInstruction(ctx context.Context, id int64, req dto.EditInstructionRequest, updatedByID int64) error {
	const op = "storage.repo.EditInstruction"

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
	if req.StatusID != nil {
		updates = append(updates, fmt.Sprintf("status_id = $%d", argID))
		args = append(args, *req.StatusID)
		argID++
	}
	if req.ResponsibleContactID != nil {
		updates = append(updates, fmt.Sprintf("responsible_contact_id = $%d", argID))
		args = append(args, *req.ResponsibleContactID)
		argID++
	}
	if req.OrganizationID != nil {
		updates = append(updates, fmt.Sprintf("organization_id = $%d", argID))
		args = append(args, *req.OrganizationID)
		argID++
	}
	if req.ExecutorContactID != nil {
		updates = append(updates, fmt.Sprintf("executor_contact_id = $%d", argID))
		args = append(args, *req.ExecutorContactID)
		argID++
	}
	if req.DueDate != nil {
		updates = append(updates, fmt.Sprintf("due_date = $%d", argID))
		args = append(args, *req.DueDate)
		argID++
	}
	if req.ParentDocumentID != nil {
		updates = append(updates, fmt.Sprintf("parent_document_id = $%d", argID))
		args = append(args, *req.ParentDocumentID)
		argID++
	}

	updates = append(updates, fmt.Sprintf("updated_by_user_id = $%d", argID))
	args = append(args, updatedByID)
	argID++

	if len(updates) == 1 && len(req.FileIDs) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE instructions SET %s WHERE id = $%d",
		strings.Join(updates, ", "),
		argID,
	)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to update instruction: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteInstruction deletes an instruction
func (r *Repo) DeleteInstruction(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteInstruction"

	res, err := r.db.ExecContext(ctx, "DELETE FROM instructions WHERE id = $1", id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to delete instruction: %w", op, err)
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

// LinkInstructionFiles links files to an instruction
func (r *Repo) LinkInstructionFiles(ctx context.Context, instructionID int64, fileIDs []int64) error {
	const op = "storage.repo.LinkInstructionFiles"

	if len(fileIDs) == 0 {
		return nil
	}

	query := `
		INSERT INTO instruction_file_links (instruction_id, file_id)
		VALUES ($1, unnest($2::bigint[]))
		ON CONFLICT DO NOTHING
	`

	_, err := r.db.ExecContext(ctx, query, instructionID, pq.Array(fileIDs))
	if err != nil {
		return fmt.Errorf("%s: failed to link files: %w", op, err)
	}

	return nil
}

// UnlinkInstructionFiles removes all file links for an instruction
func (r *Repo) UnlinkInstructionFiles(ctx context.Context, instructionID int64) error {
	const op = "storage.repo.UnlinkInstructionFiles"

	query := `DELETE FROM instruction_file_links WHERE instruction_id = $1`
	_, err := r.db.ExecContext(ctx, query, instructionID)
	if err != nil {
		return fmt.Errorf("%s: failed to unlink files: %w", op, err)
	}

	return nil
}

// LinkInstructionDocuments creates links to other documents
func (r *Repo) LinkInstructionDocuments(ctx context.Context, instructionID int64, links []dto.LinkedDocumentRequest, userID int64) error {
	const op = "storage.repo.LinkInstructionDocuments"

	if len(links) == 0 {
		return nil
	}

	query := `
		INSERT INTO instruction_document_links (instruction_id, linked_document_type, linked_document_id, link_description, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (instruction_id, linked_document_type, linked_document_id) DO UPDATE
		SET link_description = EXCLUDED.link_description
	`

	for _, link := range links {
		_, err := r.db.ExecContext(ctx, query, instructionID, link.LinkedDocumentType, link.LinkedDocumentID, link.LinkDescription, userID)
		if err != nil {
			return fmt.Errorf("%s: failed to link document: %w", op, err)
		}
	}

	return nil
}

// UnlinkInstructionDocuments removes all document links for an instruction
func (r *Repo) UnlinkInstructionDocuments(ctx context.Context, instructionID int64) error {
	const op = "storage.repo.UnlinkInstructionDocuments"

	query := `DELETE FROM instruction_document_links WHERE instruction_id = $1`
	_, err := r.db.ExecContext(ctx, query, instructionID)
	if err != nil {
		return fmt.Errorf("%s: failed to unlink documents: %w", op, err)
	}

	return nil
}

// GetInstructionStatusHistory retrieves status change history for an instruction
func (r *Repo) GetInstructionStatusHistory(ctx context.Context, instructionID int64) ([]instruction.StatusHistory, error) {
	const op = "storage.repo.GetInstructionStatusHistory"

	query := `
		SELECT
			h.id,
			h.from_status_id,
			fs.code as from_status_code,
			fs.name as from_status_name,
			h.to_status_id,
			ts.code as to_status_code,
			ts.name as to_status_name,
			h.changed_at,
			h.changed_by_user_id,
			COALESCE(c.fio, '') as changed_by_fio,
			h.comment
		FROM instruction_status_history h
		LEFT JOIN document_status fs ON h.from_status_id = fs.id
		INNER JOIN document_status ts ON h.to_status_id = ts.id
		LEFT JOIN users u ON h.changed_by_user_id = u.id
		LEFT JOIN contacts c ON u.contact_id = c.id
		WHERE h.instruction_id = $1
		ORDER BY h.changed_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, instructionID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query status history: %w", op, err)
	}
	defer rows.Close()

	var history []instruction.StatusHistory
	for rows.Next() {
		var h instruction.StatusHistory
		var fromStatusID sql.NullInt64
		var fromStatusCode, fromStatusName sql.NullString
		var toStatusID int
		var toStatusCode, toStatusName string
		var changedByID sql.NullInt64
		var changedByFIO string
		var comment sql.NullString

		err := rows.Scan(
			&h.ID,
			&fromStatusID,
			&fromStatusCode,
			&fromStatusName,
			&toStatusID,
			&toStatusCode,
			&toStatusName,
			&h.ChangedAt,
			&changedByID,
			&changedByFIO,
			&comment,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan history row: %w", op, err)
		}

		if fromStatusID.Valid {
			h.From = &document_status.ShortModel{
				ID:   int(fromStatusID.Int64),
				Code: fromStatusCode.String,
				Name: fromStatusName.String,
			}
		}

		h.To = document_status.ShortModel{
			ID:   toStatusID,
			Code: toStatusCode,
			Name: toStatusName,
		}

		if changedByID.Valid {
			h.ChangedBy = &user.ShortInfo{
				ID:   changedByID.Int64,
				Name: &changedByFIO,
			}
		}

		if comment.Valid {
			h.Comment = &comment.String
		}

		history = append(history, h)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if history == nil {
		history = make([]instruction.StatusHistory, 0)
	}

	return history, nil
}

// AddInstructionStatusHistoryComment adds a comment to the latest status history entry
func (r *Repo) AddInstructionStatusHistoryComment(ctx context.Context, instructionID int64, comment string) error {
	const op = "storage.repo.AddInstructionStatusHistoryComment"

	query := `
		UPDATE instruction_status_history
		SET comment = $2
		WHERE id = (
			SELECT id FROM instruction_status_history
			WHERE instruction_id = $1
			ORDER BY changed_at DESC
			LIMIT 1
		)
	`

	_, err := r.db.ExecContext(ctx, query, instructionID, comment)
	if err != nil {
		return fmt.Errorf("%s: failed to add comment: %w", op, err)
	}

	return nil
}

// GetAllInstructionTypes retrieves all instruction types
func (r *Repo) GetAllInstructionTypes(ctx context.Context) ([]instruction_type.Model, error) {
	const op = "storage.repo.GetAllInstructionTypes"
	const query = "SELECT id, name, COALESCE(description, '') as description FROM instruction_type ORDER BY id"

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query instruction types: %w", op, err)
	}
	defer rows.Close()

	var types []instruction_type.Model
	for rows.Next() {
		var t instruction_type.Model
		if err := rows.Scan(&t.ID, &t.Name, &t.Description); err != nil {
			return nil, fmt.Errorf("%s: failed to scan type row: %w", op, err)
		}
		types = append(types, t)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if types == nil {
		types = make([]instruction_type.Model, 0)
	}

	return types, nil
}

// --- Query fragments and helpers ---

const (
	selectInstructionFields = `
		SELECT
			d.id,
			d.name,
			d.number,
			d.document_date,
			d.description,
			d.type_id,
			t.name as type_name,
			COALESCE(t.description, '') as type_description,
			d.status_id,
			s.code as status_code,
			s.name as status_name,
			d.due_date,
			d.responsible_contact_id,
			COALESCE(rc.fio, '') as responsible_name,
			d.organization_id,
			COALESCE(o.name, '') as organization_name,
			d.executor_contact_id,
			COALESCE(ec.fio, '') as executor_name,
			d.parent_document_id,
			pd.name as parent_name,
			pd.number as parent_number,
			pd.document_date as parent_date,
			d.created_at,
			d.created_by_user_id,
			COALESCE(cc.fio, '') as created_by_fio,
			d.updated_at,
			d.updated_by_user_id,
			COALESCE(uc.fio, '') as updated_by_fio
	`
	fromInstructionJoins = `
		FROM instructions d
		INNER JOIN instruction_type t ON d.type_id = t.id
		INNER JOIN document_status s ON d.status_id = s.id
		LEFT JOIN contacts rc ON d.responsible_contact_id = rc.id
		LEFT JOIN organizations o ON d.organization_id = o.id
		LEFT JOIN contacts ec ON d.executor_contact_id = ec.id
		LEFT JOIN instructions pd ON d.parent_document_id = pd.id
		LEFT JOIN users cu ON d.created_by_user_id = cu.id
		LEFT JOIN contacts cc ON cu.contact_id = cc.id
		LEFT JOIN users uu ON d.updated_by_user_id = uu.id
		LEFT JOIN contacts uc ON uu.contact_id = uc.id
	`
)

func (r *Repo) loadInstructionFiles(ctx context.Context, instructionID int64) ([]file.Model, error) {
	const op = "storage.repo.loadInstructionFiles"

	query := `
		SELECT f.id, f.file_name, f.object_key, f.category_id, f.mime_type, f.size_bytes, f.created_at, f.target_date
		FROM files f
		INNER JOIN instruction_file_links ifl ON f.id = ifl.file_id
		WHERE ifl.instruction_id = $1
		ORDER BY f.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, instructionID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query files: %w", op, err)
	}
	defer rows.Close()

	var files []file.Model
	for rows.Next() {
		var f file.Model
		if err := rows.Scan(&f.ID, &f.FileName, &f.ObjectKey, &f.CategoryID, &f.MimeType, &f.SizeBytes, &f.CreatedAt, &f.TargetDate); err != nil {
			return nil, fmt.Errorf("%s: failed to scan file row: %w", op, err)
		}
		files = append(files, f)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	return files, nil
}

func (r *Repo) loadInstructionLinkedDocuments(ctx context.Context, instructionID int64) ([]instruction.DocumentLink, error) {
	const op = "storage.repo.loadInstructionLinkedDocuments"

	query := `
		SELECT
			dl.id,
			dl.linked_document_type,
			dl.linked_document_id,
			dl.link_description,
			CASE dl.linked_document_type
				WHEN 'decree' THEN (SELECT name FROM decrees WHERE id = dl.linked_document_id)
				WHEN 'report' THEN (SELECT name FROM reports WHERE id = dl.linked_document_id)
				WHEN 'letter' THEN (SELECT name FROM letters WHERE id = dl.linked_document_id)
				WHEN 'instruction' THEN (SELECT name FROM instructions WHERE id = dl.linked_document_id)
				WHEN 'legal_document' THEN (SELECT name FROM legal_documents WHERE id = dl.linked_document_id)
				ELSE NULL
			END as document_name,
			CASE dl.linked_document_type
				WHEN 'decree' THEN (SELECT number FROM decrees WHERE id = dl.linked_document_id)
				WHEN 'report' THEN (SELECT number FROM reports WHERE id = dl.linked_document_id)
				WHEN 'letter' THEN (SELECT number FROM letters WHERE id = dl.linked_document_id)
				WHEN 'instruction' THEN (SELECT number FROM instructions WHERE id = dl.linked_document_id)
				WHEN 'legal_document' THEN (SELECT number FROM legal_documents WHERE id = dl.linked_document_id)
				ELSE NULL
			END as document_number
		FROM instruction_document_links dl
		WHERE dl.instruction_id = $1
		ORDER BY dl.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, instructionID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query linked documents: %w", op, err)
	}
	defer rows.Close()

	var links []instruction.DocumentLink
	for rows.Next() {
		var link instruction.DocumentLink
		var linkDesc, docName, docNumber sql.NullString

		err := rows.Scan(
			&link.ID,
			&link.DocumentType,
			&link.DocumentID,
			&linkDesc,
			&docName,
			&docNumber,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan link row: %w", op, err)
		}

		if linkDesc.Valid {
			link.LinkDescription = &linkDesc.String
		}
		if docName.Valid {
			link.DocumentName = docName.String
		}
		if docNumber.Valid {
			link.DocumentNumber = &docNumber.String
		}

		links = append(links, link)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	return links, nil
}

func scanInstructionRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*instruction.ResponseModel, error) {
	var doc instruction.ResponseModel
	var number, description sql.NullString
	var typeID int
	var typeName, typeDescription string
	var statusID int
	var statusCode, statusName string
	var dueDate sql.NullTime
	var responsibleContactID sql.NullInt64
	var responsibleName string
	var organizationID sql.NullInt64
	var organizationName string
	var executorContactID sql.NullInt64
	var executorName string
	var parentDocumentID sql.NullInt64
	var parentName, parentNumber sql.NullString
	var parentDate sql.NullTime
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
		&description,
		&typeID,
		&typeName,
		&typeDescription,
		&statusID,
		&statusCode,
		&statusName,
		&dueDate,
		&responsibleContactID,
		&responsibleName,
		&organizationID,
		&organizationName,
		&executorContactID,
		&executorName,
		&parentDocumentID,
		&parentName,
		&parentNumber,
		&parentDate,
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

	if number.Valid {
		doc.Number = &number.String
	}
	if description.Valid {
		doc.Description = &description.String
	}
	if dueDate.Valid {
		doc.DueDate = &dueDate.Time
	}
	if updatedAt.Valid {
		doc.UpdatedAt = &updatedAt.Time
	}

	doc.Type = instruction_type.Model{
		ID:          typeID,
		Name:        typeName,
		Description: typeDescription,
	}

	doc.Status = document_status.ShortModel{
		ID:   statusID,
		Code: statusCode,
		Name: statusName,
	}

	if responsibleContactID.Valid {
		doc.ResponsibleContact = &instruction.ContactShortInfo{
			ID:   responsibleContactID.Int64,
			Name: responsibleName,
		}
	}

	if organizationID.Valid {
		doc.Organization = &instruction.OrganizationShortInfo{
			ID:   organizationID.Int64,
			Name: organizationName,
		}
	}

	if executorContactID.Valid {
		doc.ExecutorContact = &instruction.ContactShortInfo{
			ID:   executorContactID.Int64,
			Name: executorName,
		}
	}

	if parentDocumentID.Valid {
		doc.ParentDocument = &instruction.ParentInstructionInfo{
			ID:   parentDocumentID.Int64,
			Name: parentName.String,
		}
		if parentNumber.Valid {
			doc.ParentDocument.Number = &parentNumber.String
		}
		if parentDate.Valid {
			doc.ParentDocument.DocumentDate = &parentDate.Time
		}
	}

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
