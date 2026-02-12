package repo

import (
	"context"
	"database/sql"
	"fmt"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/hrm/personnel"
	"srmt-admin/internal/storage"
	"strings"
)

// --- Personnel Records ---

func (r *Repo) CreatePersonnelRecord(ctx context.Context, req dto.CreatePersonnelRecordRequest) (int64, error) {
	const op = "repo.CreatePersonnelRecord"

	query := `
		INSERT INTO personnel_records (employee_id, tab_number, hire_date, department_id, position_id, contract_type, contract_end_date, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.EmployeeID, req.TabNumber, req.HireDate, req.DepartmentID,
		req.PositionID, req.ContractType, req.ContractEndDate, req.Status,
	).Scan(&id)
	if err != nil {
		if translated := r.translator.Translate(err, op); translated != nil {
			return 0, translated
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

func (r *Repo) GetPersonnelRecordByID(ctx context.Context, id int64) (*personnel.Record, error) {
	const op = "repo.GetPersonnelRecordByID"

	query := `
		SELECT pr.id, pr.employee_id, c.fio, pr.tab_number,
			   pr.hire_date::text, pr.department_id, d.name, pr.position_id, p.name,
			   pr.contract_type, pr.contract_end_date::text, pr.status, pr.created_at, pr.updated_at
		FROM personnel_records pr
		JOIN contacts c ON pr.employee_id = c.id
		JOIN departments d ON pr.department_id = d.id
		JOIN positions p ON pr.position_id = p.id
		WHERE pr.id = $1`

	rec, err := scanPersonnelRecord(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrPersonnelRecordNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return rec, nil
}

func (r *Repo) GetPersonnelRecordByEmployeeID(ctx context.Context, employeeID int64) (*personnel.Record, error) {
	const op = "repo.GetPersonnelRecordByEmployeeID"

	query := `
		SELECT pr.id, pr.employee_id, c.fio, pr.tab_number,
			   pr.hire_date::text, pr.department_id, d.name, pr.position_id, p.name,
			   pr.contract_type, pr.contract_end_date::text, pr.status, pr.created_at, pr.updated_at
		FROM personnel_records pr
		JOIN contacts c ON pr.employee_id = c.id
		JOIN departments d ON pr.department_id = d.id
		JOIN positions p ON pr.position_id = p.id
		WHERE pr.employee_id = $1`

	rec, err := scanPersonnelRecord(r.db.QueryRowContext(ctx, query, employeeID))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrPersonnelRecordNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return rec, nil
}

func (r *Repo) GetAllPersonnelRecords(ctx context.Context, filters dto.PersonnelRecordFilters) ([]*personnel.Record, error) {
	const op = "repo.GetAllPersonnelRecords"

	query := `
		SELECT pr.id, pr.employee_id, c.fio, pr.tab_number,
			   pr.hire_date::text, pr.department_id, d.name, pr.position_id, p.name,
			   pr.contract_type, pr.contract_end_date::text, pr.status, pr.created_at, pr.updated_at
		FROM personnel_records pr
		JOIN contacts c ON pr.employee_id = c.id
		JOIN departments d ON pr.department_id = d.id
		JOIN positions p ON pr.position_id = p.id`

	var conditions []string
	var args []interface{}
	argIdx := 1

	if filters.DepartmentID != nil {
		conditions = append(conditions, fmt.Sprintf("pr.department_id = $%d", argIdx))
		args = append(args, *filters.DepartmentID)
		argIdx++
	}
	if filters.PositionID != nil {
		conditions = append(conditions, fmt.Sprintf("pr.position_id = $%d", argIdx))
		args = append(args, *filters.PositionID)
		argIdx++
	}
	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("pr.status = $%d", argIdx))
		args = append(args, *filters.Status)
		argIdx++
	}
	if filters.Search != nil {
		conditions = append(conditions, fmt.Sprintf("(c.fio ILIKE $%d OR pr.tab_number ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+*filters.Search+"%")
		argIdx++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY pr.created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var records []*personnel.Record
	for rows.Next() {
		rec, err := scanPersonnelRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows: %w", op, err)
	}
	return records, nil
}

func (r *Repo) UpdatePersonnelRecord(ctx context.Context, id int64, req dto.EditPersonnelRecordRequest) error {
	const op = "repo.UpdatePersonnelRecord"

	var setClauses []string
	var args []interface{}
	argIdx := 1

	if req.TabNumber != nil {
		setClauses = append(setClauses, fmt.Sprintf("tab_number = $%d", argIdx))
		args = append(args, *req.TabNumber)
		argIdx++
	}
	if req.HireDate != nil {
		setClauses = append(setClauses, fmt.Sprintf("hire_date = $%d", argIdx))
		args = append(args, *req.HireDate)
		argIdx++
	}
	if req.DepartmentID != nil {
		setClauses = append(setClauses, fmt.Sprintf("department_id = $%d", argIdx))
		args = append(args, *req.DepartmentID)
		argIdx++
	}
	if req.PositionID != nil {
		setClauses = append(setClauses, fmt.Sprintf("position_id = $%d", argIdx))
		args = append(args, *req.PositionID)
		argIdx++
	}
	if req.ContractType != nil {
		setClauses = append(setClauses, fmt.Sprintf("contract_type = $%d", argIdx))
		args = append(args, *req.ContractType)
		argIdx++
	}
	if req.ContractEndDate != nil {
		setClauses = append(setClauses, fmt.Sprintf("contract_end_date = $%d", argIdx))
		args = append(args, *req.ContractEndDate)
		argIdx++
	}
	if req.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE personnel_records SET %s WHERE id = $%d",
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
		return storage.ErrPersonnelRecordNotFound
	}
	return nil
}

func (r *Repo) DeletePersonnelRecord(ctx context.Context, id int64) error {
	const op = "repo.DeletePersonnelRecord"

	result, err := r.db.ExecContext(ctx, "DELETE FROM personnel_records WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrPersonnelRecordNotFound
	}
	return nil
}

// --- Personnel Documents ---

func (r *Repo) GetPersonnelDocuments(ctx context.Context, recordID int64) ([]*personnel.Document, error) {
	const op = "repo.GetPersonnelDocuments"

	query := `SELECT id, record_id, type, name, file_url, uploaded_at FROM personnel_documents WHERE record_id = $1 ORDER BY uploaded_at DESC`
	rows, err := r.db.QueryContext(ctx, query, recordID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var docs []*personnel.Document
	for rows.Next() {
		var d personnel.Document
		if err := rows.Scan(&d.ID, &d.RecordID, &d.Type, &d.Name, &d.FileURL, &d.UploadedAt); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		docs = append(docs, &d)
	}
	return docs, rows.Err()
}

// --- Personnel Transfers ---

func (r *Repo) GetPersonnelTransfers(ctx context.Context, recordID int64) ([]*personnel.Transfer, error) {
	const op = "repo.GetPersonnelTransfers"

	query := `
		SELECT pt.id, pt.record_id,
			   COALESCE(fd.name, ''), COALESCE(td.name, ''),
			   COALESCE(fp.name, ''), COALESCE(tp.name, ''),
			   pt.transfer_date::text, COALESCE(pt.order_number, ''), COALESCE(pt.reason, '')
		FROM personnel_transfers pt
		LEFT JOIN departments fd ON pt.from_department_id = fd.id
		LEFT JOIN departments td ON pt.to_department_id = td.id
		LEFT JOIN positions fp ON pt.from_position_id = fp.id
		LEFT JOIN positions tp ON pt.to_position_id = tp.id
		WHERE pt.record_id = $1
		ORDER BY pt.transfer_date DESC`

	rows, err := r.db.QueryContext(ctx, query, recordID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var transfers []*personnel.Transfer
	for rows.Next() {
		var t personnel.Transfer
		if err := rows.Scan(&t.ID, &t.RecordID, &t.FromDepartment, &t.ToDepartment,
			&t.FromPosition, &t.ToPosition, &t.TransferDate, &t.OrderNumber, &t.Reason); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		transfers = append(transfers, &t)
	}
	return transfers, rows.Err()
}

// --- Helpers ---

func scanPersonnelRecord(scanner interface {
	Scan(dest ...interface{}) error
}) (*personnel.Record, error) {
	var rec personnel.Record
	var contractEndDate sql.NullString

	err := scanner.Scan(
		&rec.ID, &rec.EmployeeID, &rec.EmployeeName, &rec.TabNumber,
		&rec.HireDate, &rec.DepartmentID, &rec.DepartmentName,
		&rec.PositionID, &rec.PositionName,
		&rec.ContractType, &contractEndDate, &rec.Status,
		&rec.CreatedAt, &rec.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if contractEndDate.Valid {
		rec.ContractEndDate = &contractEndDate.String
	}
	return &rec, nil
}
