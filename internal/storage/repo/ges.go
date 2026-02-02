package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"srmt-admin/internal/lib/model/contact"
	"srmt-admin/internal/lib/model/department"
	"srmt-admin/internal/lib/model/discharge"
	"srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/model/incident"
	"srmt-admin/internal/lib/model/organization"
	"srmt-admin/internal/lib/model/position"
	"srmt-admin/internal/lib/model/shutdown"
	"srmt-admin/internal/lib/model/user"
	"srmt-admin/internal/lib/model/visit"
	"srmt-admin/internal/storage"
	"time"
)

// GetOrganizationByID retrieves an organization by its ID
func (r *Repo) GetOrganizationByID(ctx context.Context, id int64) (*organization.Model, error) {
	const op = "storage.repo.ges.GetOrganizationByID"

	const query = `
		SELECT
			o.id,
			o.name,
			o.parent_organization_id,
			po.name as parent_organization_name,
			COALESCE(t.types_json, '[]'::json) as types
		FROM
			organizations o
		LEFT JOIN
			organizations po ON o.parent_organization_id = po.id
		LEFT JOIN (
			SELECT
				otl.organization_id,
				json_agg(ot.name ORDER BY ot.name) as types_json
			FROM
				organization_type_links otl
			JOIN
				organization_types ot ON otl.type_id = ot.id
			GROUP BY
				otl.organization_id
		) t ON o.id = t.organization_id
		WHERE o.id = $1
	`

	var org organization.Model
	var typesJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&org.ID,
		&org.Name,
		&org.ParentOrganizationID,
		&org.ParentOrganizationName,
		&typesJSON,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to query organization: %w", op, err)
	}

	if err := json.Unmarshal(typesJSON, &org.Types); err != nil {
		return nil, fmt.Errorf("%s: failed to unmarshal types: %w", op, err)
	}

	return &org, nil
}

// GetDepartmentsByOrgID retrieves all departments for a given organization
func (r *Repo) GetDepartmentsByOrgID(ctx context.Context, orgID int64) ([]*department.Model, error) {
	const op = "storage.repo.ges.GetDepartmentsByOrgID"

	const query = `
		SELECT
			d.id, d.name, d.description, d.organization_id, d.created_at, d.updated_at,
			o.id as org_id, o.name as org_name
		FROM
			departments d
		LEFT JOIN
			organizations o ON d.organization_id = o.id
		WHERE d.organization_id = $1
		ORDER BY d.name
	`

	rows, err := r.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query departments: %w", op, err)
	}
	defer rows.Close()

	var departments []*department.Model
	for rows.Next() {
		var dep department.Model
		var desc sql.NullString
		var orgDBID sql.NullInt64
		var orgName sql.NullString

		if err := rows.Scan(
			&dep.ID,
			&dep.Name,
			&desc,
			&dep.OrganizationID,
			&dep.CreatedAt,
			&dep.UpdatedAt,
			&orgDBID,
			&orgName,
		); err != nil {
			return nil, fmt.Errorf("%s: failed to scan department: %w", op, err)
		}

		if desc.Valid {
			dep.Description = &desc.String
		}

		if orgDBID.Valid && orgName.Valid {
			dep.Organization = &organization.Model{
				ID:   orgDBID.Int64,
				Name: orgName.String,
			}
		}

		departments = append(departments, &dep)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if departments == nil {
		departments = make([]*department.Model, 0)
	}

	return departments, nil
}

// GetContactsByOrgID retrieves all contacts for a given organization
func (r *Repo) GetContactsByOrgID(ctx context.Context, orgID int64) ([]*contact.Model, error) {
	const op = "storage.repo.ges.GetContactsByOrgID"

	query := selectContactFields + fromContactJoins + " WHERE c.organization_id = $1 ORDER BY c.fio"

	rows, err := r.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query contacts: %w", op, err)
	}
	defer rows.Close()

	var contacts []*contact.Model
	for rows.Next() {
		c, err := scanContactRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan contact row: %w", op, err)
		}
		contacts = append(contacts, c)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if contacts == nil {
		contacts = make([]*contact.Model, 0)
	}

	return contacts, nil
}

// GetShutdownsByOrgID retrieves shutdowns for a given organization within a date range
func (r *Repo) GetShutdownsByOrgID(ctx context.Context, orgID int64, startDate, endDate *time.Time) ([]*shutdown.ResponseModel, error) {
	const op = "storage.repo.ges.GetShutdownsByOrgID"

	query := selectShutdownFields + fromShutdownJoins + " WHERE s.organization_id = $1"
	args := []interface{}{orgID}
	argID := 2

	if startDate != nil {
		query += fmt.Sprintf(" AND s.start_time >= $%d", argID)
		args = append(args, *startDate)
		argID++
	}

	if endDate != nil {
		// Add one day to end date to include the entire end day
		endOfDay := endDate.AddDate(0, 0, 1)
		query += fmt.Sprintf(" AND s.start_time < $%d", argID)
		args = append(args, endOfDay)
	}

	query += " ORDER BY s.start_time DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query shutdowns: %w", op, err)
	}
	defer rows.Close()

	var shutdowns []*shutdown.ResponseModel
	for rows.Next() {
		m, err := scanShutdownRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan shutdown row: %w", op, err)
		}
		shutdowns = append(shutdowns, m)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if shutdowns == nil {
		shutdowns = make([]*shutdown.ResponseModel, 0)
	}

	// Load files for each shutdown
	for _, s := range shutdowns {
		files, err := r.loadShutdownFiles(ctx, s.ID)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to load files for shutdown %d: %w", op, s.ID, err)
		}
		s.Files = files
	}

	return shutdowns, nil
}

// GetDischargesByOrgID retrieves discharges for a given organization within a date range
func (r *Repo) GetDischargesByOrgID(ctx context.Context, orgID int64, startDate, endDate *time.Time) ([]discharge.Model, error) {
	const op = "storage.repo.ges.GetDischargesByOrgID"

	baseQuery := `
		SELECT
			d.id, d.start_time, d.end_time, d.flow_rate_m3_s, d.reason, d.approved,
			d.is_ongoing, d.total_volume_mln_m3,
			o.id as org_id, o.name as org_name, o.parent_organization_id as org_parent_id,
			COALESCE(ot.types_json, '[]'::json) as org_types,
			creator.id as creator_id,
			creator_contact.fio as creator_fio,
			approver.id as approver_id,
			approver_contact.fio as approver_fio
		FROM
			v_idle_water_discharges_with_volume d
		JOIN
			organizations o ON d.organization_id = o.id
		JOIN
			users creator ON d.created_by = creator.id
		JOIN
			contacts creator_contact ON creator.contact_id = creator_contact.id
		LEFT JOIN
			users approver ON d.approved_by = approver.id
		LEFT JOIN
			contacts approver_contact ON approver.contact_id = approver_contact.id
		LEFT JOIN (
			SELECT
				otl.organization_id,
				json_agg(ot.name ORDER BY ot.name) as types_json
			FROM organization_type_links otl
			JOIN organization_types ot ON otl.type_id = ot.id
			GROUP BY otl.organization_id
		) ot ON o.id = ot.organization_id
		WHERE d.organization_id = $1
	`

	args := []interface{}{orgID}
	argID := 2

	if startDate != nil {
		baseQuery += fmt.Sprintf(" AND d.start_time >= $%d", argID)
		args = append(args, *startDate)
		argID++
	}

	if endDate != nil {
		endOfDay := endDate.AddDate(0, 0, 1)
		baseQuery += fmt.Sprintf(" AND d.start_time < $%d", argID)
		args = append(args, endOfDay)
	}

	baseQuery += " ORDER BY d.start_time DESC"

	rows, err := r.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query discharges: %w", op, err)
	}
	defer rows.Close()

	var discharges []discharge.Model
	for rows.Next() {
		var d discharge.Model
		var org organization.Model
		var orgTypesJSON []byte

		var creatorID int64
		var creatorFIO string
		var approverID sql.NullInt64
		var approverFIO sql.NullString

		err := rows.Scan(
			&d.ID, &d.StartedAt, &d.EndedAt, &d.FlowRate, &d.Reason, &d.Approved,
			&d.IsOngoing, &d.TotalVolume,
			&org.ID, &org.Name, &org.ParentOrganizationID, &orgTypesJSON,
			&creatorID, &creatorFIO,
			&approverID, &approverFIO,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan discharge row: %w", op, err)
		}

		if err := json.Unmarshal(orgTypesJSON, &org.Types); err != nil {
			return nil, fmt.Errorf("%s: failed to unmarshal org types: %w", op, err)
		}

		d.Organization = &org

		fioCreator := creatorFIO
		d.CreatedByUser = &user.ShortInfo{
			ID:   creatorID,
			Name: &fioCreator,
		}

		if approverID.Valid {
			approver := &user.ShortInfo{
				ID: approverID.Int64,
			}
			if approverFIO.Valid {
				fioApprover := approverFIO.String
				approver.Name = &fioApprover
			}
			d.ApprovedByUser = approver
		}

		discharges = append(discharges, d)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if discharges == nil {
		return []discharge.Model{}, nil
	}

	// Load files for each discharge
	for i := range discharges {
		files, err := r.loadDischargeFiles(ctx, discharges[i].ID)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to load files for discharge %d: %w", op, discharges[i].ID, err)
		}
		discharges[i].Files = files
	}

	return discharges, nil
}

// GetIncidentsByOrgID retrieves incidents for a given organization within a date range
func (r *Repo) GetIncidentsByOrgID(ctx context.Context, orgID int64, startDate, endDate *time.Time) ([]*incident.ResponseModel, error) {
	const op = "storage.repo.ges.GetIncidentsByOrgID"

	query := selectIncidentFields + fromIncidentJoins + " WHERE i.organization_id = $1"
	args := []interface{}{orgID}
	argID := 2

	if startDate != nil {
		query += fmt.Sprintf(" AND i.incident_time >= $%d", argID)
		args = append(args, *startDate)
		argID++
	}

	if endDate != nil {
		endOfDay := endDate.AddDate(0, 0, 1)
		query += fmt.Sprintf(" AND i.incident_time < $%d", argID)
		args = append(args, endOfDay)
	}

	query += " ORDER BY i.incident_time DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query incidents: %w", op, err)
	}
	defer rows.Close()

	var incidents []*incident.ResponseModel
	for rows.Next() {
		m, err := scanIncidentRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan incident row: %w", op, err)
		}
		incidents = append(incidents, m)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if incidents == nil {
		incidents = make([]*incident.ResponseModel, 0)
	}

	// Load files for each incident
	for _, inc := range incidents {
		files, err := r.loadIncidentFiles(ctx, inc.ID)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to load files for incident %d: %w", op, inc.ID, err)
		}
		inc.Files = files
	}

	return incidents, nil
}

// GetVisitsByOrgID retrieves visits for a given organization within a date range
func (r *Repo) GetVisitsByOrgID(ctx context.Context, orgID int64, startDate, endDate *time.Time) ([]*visit.ResponseModel, error) {
	const op = "storage.repo.ges.GetVisitsByOrgID"

	query := selectVisitFields + fromVisitJoins + " WHERE v.organization_id = $1"
	args := []interface{}{orgID}
	argID := 2

	if startDate != nil {
		query += fmt.Sprintf(" AND v.visit_date >= $%d", argID)
		args = append(args, *startDate)
		argID++
	}

	if endDate != nil {
		endOfDay := endDate.AddDate(0, 0, 1)
		query += fmt.Sprintf(" AND v.visit_date < $%d", argID)
		args = append(args, endOfDay)
	}

	query += " ORDER BY v.visit_date DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query visits: %w", op, err)
	}
	defer rows.Close()

	var visits []*visit.ResponseModel
	for rows.Next() {
		m, err := scanVisitRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan visit row: %w", op, err)
		}
		visits = append(visits, m)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if visits == nil {
		visits = make([]*visit.ResponseModel, 0)
	}

	// Load files for each visit
	for _, v := range visits {
		files, err := r.loadVisitFiles(ctx, v.ID)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to load files for visit %d: %w", op, v.ID, err)
		}
		v.Files = files
	}

	return visits, nil
}

// scanGesContactRow scans a contact row for GES handlers (without icon file info)
func scanGesContactRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*contact.Model, error) {
	var c contact.Model
	var (
		email, phone, ipPhone, extOrg sql.NullString
		dob                           sql.NullTime
		iconID                        sql.NullInt64
		orgID, deptID, posID          sql.NullInt64
		orgName, deptName, posName    sql.NullString
		posDescription                sql.NullString
		fileID                        sql.NullInt64
		fileName, objectKey, mimeType sql.NullString
		sizeBytes                     sql.NullInt64
	)

	err := scanner.Scan(
		&c.ID, &c.Name, &email, &phone, &ipPhone, &dob, &extOrg, &iconID,
		&c.CreatedAt, &c.UpdatedAt,
		&orgID, &orgName,
		&deptID, &deptName,
		&posID, &posName, &posDescription,
		&fileID, &fileName, &objectKey, &mimeType, &sizeBytes,
	)
	if err != nil {
		return nil, err
	}

	if email.Valid {
		c.Email = &email.String
	}
	if phone.Valid {
		c.Phone = &phone.String
	}
	if ipPhone.Valid {
		c.IPPhone = &ipPhone.String
	}
	if extOrg.Valid {
		c.ExternalOrgName = &extOrg.String
	}
	if iconID.Valid {
		c.IconID = &iconID.Int64
	}
	if dob.Valid {
		c.DOB = &dob.Time
	}

	if orgID.Valid && orgName.Valid {
		c.Organization = &organization.Model{ID: orgID.Int64, Name: orgName.String}
	}
	if deptID.Valid && deptName.Valid {
		c.Department = &department.Model{ID: deptID.Int64, Name: deptName.String}
	}
	if posID.Valid && posName.Valid {
		var desc *string
		if posDescription.Valid {
			desc = &posDescription.String
		}
		c.Position = &position.Model{ID: posID.Int64, Name: posName.String, Description: desc}
	}

	if fileID.Valid && fileName.Valid && objectKey.Valid {
		c.Icon = &contact.IconFile{
			ID:        fileID.Int64,
			FileName:  fileName.String,
			URL:       objectKey.String,
			MimeType:  mimeType.String,
			SizeBytes: sizeBytes.Int64,
		}
	}

	return &c, nil
}

// loadFilesForEntity is a generic helper to load files for an entity
func (r *Repo) loadFilesForEntity(ctx context.Context, entityID int64, tableName, linkColumn string) ([]file.Model, error) {
	query := fmt.Sprintf(`
		SELECT f.id, f.file_name, f.object_key, f.category_id, f.mime_type, f.size_bytes, f.created_at
		FROM files f
		INNER JOIN %s fl ON f.id = fl.file_id
		WHERE fl.%s = $1
		ORDER BY f.created_at DESC
	`, tableName, linkColumn)

	rows, err := r.db.QueryContext(ctx, query, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []file.Model
	for rows.Next() {
		var f file.Model
		if err := rows.Scan(&f.ID, &f.FileName, &f.ObjectKey, &f.CategoryID, &f.MimeType, &f.SizeBytes, &f.CreatedAt); err != nil {
			return nil, err
		}
		files = append(files, f)
	}

	return files, rows.Err()
}
