package repo

import (
	"context"
	"database/sql"
	"fmt"
	"srmt-admin/internal/lib/model/hrm/profile"
	"srmt-admin/internal/storage"
)

func (r *Repo) GetMyProfile(ctx context.Context, employeeID int64) (*profile.MyProfile, error) {
	const op = "storage.repo.GetMyProfile"

	query := `
		SELECT
			c.id,
			c.fio,
			COALESCE(p.name, ''),
			COALESCE(d.name, ''),
			COALESCE(o.name, ''),
			c.email,
			c.phone,
			c.ip_phone,
			CASE WHEN pr.hire_date IS NOT NULL THEN TO_CHAR(pr.hire_date, 'YYYY-MM-DD') END,
			pr.status,
			pr.contract_type,
			f.object_key,
			pr.tab_number
		FROM contacts c
		LEFT JOIN personnel_records pr ON pr.employee_id = c.id
		LEFT JOIN positions p ON COALESCE(pr.position_id, c.position_id) = p.id
		LEFT JOIN departments d ON COALESCE(pr.department_id, c.department_id) = d.id
		LEFT JOIN organizations o ON c.organization_id = o.id
		LEFT JOIN files f ON c.icon_id = f.id
		WHERE c.id = $1
	`

	var mp profile.MyProfile
	err := r.db.QueryRowContext(ctx, query, employeeID).Scan(
		&mp.ID,
		&mp.Name,
		&mp.Position,
		&mp.Department,
		&mp.Organization,
		&mp.Email,
		&mp.Phone,
		&mp.InternalPhone,
		&mp.HireDate,
		&mp.Status,
		&mp.ContractType,
		&mp.Avatar,
		&mp.TabNumber,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%s: %w", op, storage.ErrNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &mp, nil
}

func (r *Repo) GetMyLeaveBalance(ctx context.Context, employeeID int64, year int) (*profile.LeaveBalance, error) {
	const op = "storage.repo.GetMyLeaveBalance"

	query := `
		SELECT
			vacation_type,
			COALESCE(SUM(days) FILTER (WHERE status IN ('approved', 'active', 'completed')), 0) AS used,
			COALESCE(SUM(days) FILTER (WHERE status = 'pending'), 0) AS pending
		FROM vacations
		WHERE employee_id = $1
		  AND EXTRACT(YEAR FROM start_date) = $2
		  AND status NOT IN ('cancelled', 'rejected', 'draft')
		GROUP BY vacation_type
	`

	rows, err := r.db.QueryContext(ctx, query, employeeID, year)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	usedMap := map[string]int{}
	for rows.Next() {
		var vtype string
		var used, pending int
		if err := rows.Scan(&vtype, &used, &pending); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		usedMap[vtype] = used + pending
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows: %w", op, err)
	}

	// Get total balance from vacation_balances table
	var totalDays, carriedOver int
	balanceQuery := `SELECT COALESCE(total_days, 0), COALESCE(carried_over, 0) FROM vacation_balances WHERE employee_id = $1 AND year = $2`
	err = r.db.QueryRowContext(ctx, balanceQuery, employeeID, year).Scan(&totalDays, &carriedOver)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("%s: balance: %w", op, err)
	}

	annualUsed := usedMap["annual"]
	additionalUsed := usedMap["additional"]
	studyUsed := usedMap["study"]
	compUsed := usedMap["comp"]

	balance := &profile.LeaveBalance{
		AnnualLeave: profile.LeaveCategory{
			Total:     totalDays + carriedOver,
			Used:      annualUsed,
			Remaining: totalDays + carriedOver - annualUsed,
		},
		AdditionalLeave: profile.LeaveCategory{
			Total:     0,
			Used:      additionalUsed,
			Remaining: -additionalUsed,
		},
		StudyLeave: profile.LeaveCategory{
			Total:     0,
			Used:      studyUsed,
			Remaining: -studyUsed,
		},
		SickLeave: profile.LeaveCategory{
			Total:     0,
			Used:      usedMap["maternity"] + usedMap["unpaid"],
			Remaining: 0,
		},
		CompDays: profile.LeaveCategory{
			Total:     0,
			Used:      compUsed,
			Remaining: -compUsed,
		},
	}

	return balance, nil
}

func (r *Repo) GetMyVacations(ctx context.Context, employeeID int64, status *string, vacationType *string) ([]*profile.MyVacation, error) {
	const op = "storage.repo.GetMyVacations"

	query := `
		SELECT v.id, v.vacation_type, v.start_date::text, v.end_date::text,
			   v.days, v.status, v.reason, v.rejection_reason,
			   ac.fio, v.approved_at::text, v.created_at::text
		FROM vacations v
		LEFT JOIN contacts ac ON v.approved_by = ac.id
		WHERE v.employee_id = $1`

	args := []interface{}{employeeID}
	argIdx := 2

	if status != nil {
		query += fmt.Sprintf(" AND v.status = $%d", argIdx)
		args = append(args, *status)
		argIdx++
	}
	if vacationType != nil {
		query += fmt.Sprintf(" AND v.vacation_type = $%d", argIdx)
		args = append(args, *vacationType)
		argIdx++
	}

	query += " ORDER BY v.created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var vacations []*profile.MyVacation
	for rows.Next() {
		var v profile.MyVacation
		if err := rows.Scan(
			&v.ID, &v.Type, &v.StartDate, &v.EndDate,
			&v.Days, &v.Status, &v.Reason, &v.RejectionReason,
			&v.ApprovedBy, &v.ApprovedAt, &v.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		vacations = append(vacations, &v)
	}
	return vacations, rows.Err()
}

func (r *Repo) GetMyDocuments(ctx context.Context, employeeID int64) ([]*profile.MyDocument, error) {
	const op = "storage.repo.GetMyDocuments"

	query := `
		SELECT pd.id, pd.name, pd.type, pd.uploaded_at::text, NULL::bigint
		FROM personnel_documents pd
		JOIN personnel_records pr ON pd.record_id = pr.id
		WHERE pr.employee_id = $1
		ORDER BY pd.uploaded_at DESC`

	rows, err := r.db.QueryContext(ctx, query, employeeID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var docs []*profile.MyDocument
	for rows.Next() {
		var d profile.MyDocument
		if err := rows.Scan(&d.ID, &d.Name, &d.Type, &d.Date, &d.Size); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		docs = append(docs, &d)
	}
	return docs, rows.Err()
}
