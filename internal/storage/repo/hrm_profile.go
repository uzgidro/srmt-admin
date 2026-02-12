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
