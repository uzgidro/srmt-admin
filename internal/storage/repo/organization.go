package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/lib/pq"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/contact"
	"srmt-admin/internal/lib/model/department"
	"srmt-admin/internal/lib/model/organization"
	"srmt-admin/internal/lib/model/position"
	"srmt-admin/internal/storage"
	"strings"
)

func (r *Repo) GetAllOrganizations(ctx context.Context, orgType *string) ([]*organization.Model, error) {
	const op = "storage.repo.GetAllOrganizations"

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
		ORDER BY
			o.name;
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query organizations: %w", op, err)
	}
	defer rows.Close()

	var allOrgs []*organization.Model
	for rows.Next() {
		var org organization.Model
		var typesJSON []byte
		if err := rows.Scan(&org.ID, &org.Name, &org.ParentOrganizationID, &org.ParentOrganizationName, &typesJSON); err != nil {
			return nil, fmt.Errorf("%s: failed to scan organization: %w", op, err)
		}
		if err := json.Unmarshal(typesJSON, &org.Types); err != nil {
			return nil, fmt.Errorf("%s: failed to unmarshal types: %w", op, err)
		}
		allOrgs = append(allOrgs, &org)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	orgsMap := make(map[int64]*organization.Model, len(allOrgs))
	for _, org := range allOrgs {
		orgsMap[org.ID] = org
	}

	for _, org := range allOrgs {
		if org.ParentOrganizationID != nil {
			if parent, ok := orgsMap[*org.ParentOrganizationID]; ok {
				parent.Items = append(parent.Items, org)
			}
		}
	}

	var result []*organization.Model
	if orgType == nil {
		// No type filter, return only root organizations
		for _, org := range allOrgs {
			if org.ParentOrganizationID == nil {
				result = append(result, org)
			}
		}
	} else {
		// Type filter is present, find all orgs with this type
		for _, org := range allOrgs {
			hasType := false
			for _, t := range org.Types {
				if t == *orgType {
					hasType = true
					break
				}
			}
			if hasType {
				result = append(result, org)
			}
		}
	}

	if result == nil {
		result = make([]*organization.Model, 0)
	}

	return result, nil
}

func (r *Repo) AddOrganization(ctx context.Context, name string, parentID *int64, typeIDs []int64) (int64, error) {
	const op = "storage.repo.AddOrganization"

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("%s: failed to begin transaction: %w", op, err)
	}
	defer tx.Rollback()

	// Insert organization
	var orgID int64
	err = tx.QueryRowContext(ctx,
		"INSERT INTO organizations(name, parent_organization_id) VALUES($1, $2) RETURNING id",
		name, parentID,
	).Scan(&orgID)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code.Name() == "unique_violation" {
				return 0, storage.ErrDuplicate
			}
			if pqErr.Code.Name() == "foreign_key_violation" {
				return 0, storage.ErrForeignKeyViolation
			}
		}
		return 0, fmt.Errorf("%s: failed to insert organization: %w", op, err)
	}

	// Link types
	stmt, err := tx.PrepareContext(ctx, "INSERT INTO organization_type_links(organization_id, type_id) VALUES($1, $2)")
	if err != nil {
		return 0, fmt.Errorf("%s: failed to prepare type link statement: %w", op, err)
	}
	defer stmt.Close()

	for _, typeID := range typeIDs {
		_, err := stmt.ExecContext(ctx, orgID, typeID)
		if err != nil {
			var pqErr *pq.Error
			if errors.As(err, &pqErr) && pqErr.Code.Name() == "foreign_key_violation" {
				return 0, storage.ErrForeignKeyViolation
			}
			return 0, fmt.Errorf("%s: failed to link type id %d: %w", op, typeID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("%s: failed to commit transaction: %w", op, err)
	}

	return orgID, nil
}

func (r *Repo) EditOrganization(ctx context.Context, id int64, name *string, parentID **int64, typeIDs []int64) error {
	const op = "storage.repo.EditOrganization"

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: failed to begin transaction: %w", op, err)
	}
	defer tx.Rollback()

	// Update organization fields
	var updates []string
	var args []interface{}
	argID := 1

	if name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argID))
		args = append(args, *name)
		argID++
	}
	if parentID != nil {
		updates = append(updates, fmt.Sprintf("parent_organization_id = $%d", argID))
		args = append(args, *parentID)
		argID++
	}

	if len(updates) > 0 {
		query := fmt.Sprintf("UPDATE organizations SET %s WHERE id = $%d", strings.Join(updates, ", "), argID)
		args = append(args, id)

		res, err := tx.ExecContext(ctx, query, args...)
		if err != nil {
			var pqErr *pq.Error
			if errors.As(err, &pqErr) {
				if pqErr.Code.Name() == "unique_violation" {
					return storage.ErrDuplicate
				}
				if pqErr.Code.Name() == "foreign_key_violation" {
					return storage.ErrForeignKeyViolation
				}
			}
			return fmt.Errorf("%s: failed to update organization: %w", op, err)
		}
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 0 {
			return storage.ErrNotFound
		}
	}

	// Update types if provided
	if len(typeIDs) > 0 {
		// Delete old links
		_, err := tx.ExecContext(ctx, "DELETE FROM organization_type_links WHERE organization_id = $1", id)
		if err != nil {
			return fmt.Errorf("%s: failed to delete old type links: %w", op, err)
		}

		// Insert new links
		stmt, err := tx.PrepareContext(ctx, "INSERT INTO organization_type_links(organization_id, type_id) VALUES($1, $2)")
		if err != nil {
			return fmt.Errorf("%s: failed to prepare type link statement: %w", op, err)
		}
		defer stmt.Close()

		for _, typeID := range typeIDs {
			_, err := stmt.ExecContext(ctx, id, typeID)
			if err != nil {
				var pqErr *pq.Error
				if errors.As(err, &pqErr) && pqErr.Code.Name() == "foreign_key_violation" {
					return storage.ErrForeignKeyViolation
				}
				return fmt.Errorf("%s: failed to link type id %d: %w", op, typeID, err)
			}
		}
	}

	return tx.Commit()
}

func (r *Repo) DeleteOrganization(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteOrganization"

	res, err := r.db.ExecContext(ctx, "DELETE FROM organizations WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("%s: failed to delete organization: %w", op, err)
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

// GetFlatOrganizations returns a flat list of all organizations without hierarchical nesting
func (r *Repo) GetFlatOrganizations(ctx context.Context, orgType *string) ([]*organization.Model, error) {
	const op = "storage.repo.GetFlatOrganizations"

	baseQuery := `
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
	`

	var query string
	var args []interface{}

	if orgType != nil {
		// Filter by organization type
		query = baseQuery + `
		WHERE EXISTS (
			SELECT 1
			FROM organization_type_links otl
			JOIN organization_types ot ON otl.type_id = ot.id
			WHERE otl.organization_id = o.id AND ot.name = $1
		)
		ORDER BY o.name;`
		args = append(args, *orgType)
	} else {
		query = baseQuery + " ORDER BY o.name;"
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query organizations: %w", op, err)
	}
	defer rows.Close()

	var result []*organization.Model
	for rows.Next() {
		var org organization.Model
		var typesJSON []byte
		if err := rows.Scan(&org.ID, &org.Name, &org.ParentOrganizationID, &org.ParentOrganizationName, &typesJSON); err != nil {
			return nil, fmt.Errorf("%s: failed to scan organization: %w", op, err)
		}
		if err := json.Unmarshal(typesJSON, &org.Types); err != nil {
			return nil, fmt.Errorf("%s: failed to unmarshal types: %w", op, err)
		}
		result = append(result, &org)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if result == nil {
		result = make([]*organization.Model, 0)
	}

	return result, nil
}

// GetCascadesWithDetails returns cascades with their HPPs, including contacts with specific positions and current discharges
// If ascueFetcher is provided, it enriches the data with ASCUE metrics
func (r *Repo) GetCascadesWithDetails(ctx context.Context, ascueFetcher dto.ASCUEFetcher) ([]*dto.CascadeWithDetails, error) {
	const op = "storage.repo.GetCascadesWithDetails"

	// Get all organizations with cascade type
	const orgQuery = `
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
		WHERE EXISTS (
			SELECT 1
			FROM organization_type_links otl
			JOIN organization_types ot ON otl.type_id = ot.id
			WHERE otl.organization_id = o.id
			AND ot.name IN ('cascade', 'ges', 'mini', 'micro')
		)
		ORDER BY o.name;
	`

	rows, err := r.db.QueryContext(ctx, orgQuery)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query organizations: %w", op, err)
	}
	defer rows.Close()

	var allOrgs []*dto.CascadeWithDetails
	orgMap := make(map[int64]*dto.CascadeWithDetails)

	for rows.Next() {
		var org dto.CascadeWithDetails
		var typesJSON []byte
		if err := rows.Scan(&org.ID, &org.Name, &org.ParentOrganizationID, &org.ParentOrganizationName, &typesJSON); err != nil {
			return nil, fmt.Errorf("%s: failed to scan organization: %w", op, err)
		}
		if err := json.Unmarshal(typesJSON, &org.Types); err != nil {
			return nil, fmt.Errorf("%s: failed to unmarshal types: %w", op, err)
		}
		org.Contacts = make([]*contact.Model, 0)
		allOrgs = append(allOrgs, &org)
		orgMap[org.ID] = &org
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	// Get contacts with specific positions for all organizations
	const contactQuery = `
		SELECT
			c.id, c.fio, c.email, c.phone, c.ip_phone, c.dob,
			c.external_organization_name, c.created_at, c.updated_at,
			c.organization_id,
			o.id as org_id, o.name as org_name,
			d.id as dept_id, d.name as dept_name,
			p.id as pos_id, p.name as pos_name, p.description as pos_description
		FROM
			contacts c
		LEFT JOIN
			organizations o ON c.organization_id = o.id
		LEFT JOIN
			departments d ON c.department_id = d.id
		LEFT JOIN
			positions p ON c.position_id = p.id
		WHERE
			c.organization_id = ANY($1)
			AND p.name IN ('direktor', 'bosh muhandis', 'ges boshlig''i')
		ORDER BY c.organization_id, p.name;
	`

	orgIDs := make([]int64, 0, len(allOrgs))
	for _, org := range allOrgs {
		orgIDs = append(orgIDs, org.ID)
	}

	if len(orgIDs) > 0 {
		contactRows, err := r.db.QueryContext(ctx, contactQuery, pq.Array(orgIDs))
		if err != nil {
			return nil, fmt.Errorf("%s: failed to query contacts: %w", op, err)
		}
		defer contactRows.Close()

		for contactRows.Next() {
			var c contact.Model
			var orgID int64
			var (
				email, phone, ipPhone, extOrg sql.NullString
				dob                           sql.NullTime
				orgDBID, deptID, posID        sql.NullInt64
				orgName, deptName, posName    sql.NullString
				posDescription                sql.NullString
			)

			err := contactRows.Scan(
				&c.ID, &c.Name, &email, &phone, &ipPhone, &dob, &extOrg,
				&c.CreatedAt, &c.UpdatedAt,
				&orgID,
				&orgDBID, &orgName,
				&deptID, &deptName,
				&posID, &posName, &posDescription,
			)
			if err != nil {
				return nil, fmt.Errorf("%s: failed to scan contact: %w", op, err)
			}

			// Set nullable fields
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
			if dob.Valid {
				c.DOB = &dob.Time
			}
			if orgDBID.Valid && orgName.Valid {
				c.Organization = &organization.Model{ID: orgDBID.Int64, Name: orgName.String}
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

			if org, ok := orgMap[orgID]; ok {
				org.Contacts = append(org.Contacts, &c)
			}
		}

		if err = contactRows.Err(); err != nil {
			return nil, fmt.Errorf("%s: contact rows iteration error: %w", op, err)
		}
	}

	// Get current discharges for all organizations
	const dischargeQuery = `
		SELECT
			d.organization_id,
			SUM(d.flow_rate_m3_s) as total_flow_rate
		FROM
			idle_water_discharges d
		WHERE
			d.organization_id = ANY($1)
			AND d.start_time <= NOW()
			AND (d.end_time > NOW() OR d.end_time IS NULL)
		GROUP BY d.organization_id;
	`

	if len(orgIDs) > 0 {
		dischargeRows, err := r.db.QueryContext(ctx, dischargeQuery, pq.Array(orgIDs))
		if err != nil {
			return nil, fmt.Errorf("%s: failed to query discharges: %w", op, err)
		}
		defer dischargeRows.Close()

		for dischargeRows.Next() {
			var orgID int64
			var totalFlow float64
			if err := dischargeRows.Scan(&orgID, &totalFlow); err != nil {
				return nil, fmt.Errorf("%s: failed to scan discharge: %w", op, err)
			}

			if org, ok := orgMap[orgID]; ok {
				org.CurrentDischarge = &totalFlow
			}
		}

		if err = dischargeRows.Err(); err != nil {
			return nil, fmt.Errorf("%s: discharge rows iteration error: %w", op, err)
		}
	}

	// Build hierarchy: add child organizations to their parents (HPPs to cascades, cascades to parent org)
	for _, org := range allOrgs {
		if org.ParentOrganizationID != nil {
			if parent, ok := orgMap[*org.ParentOrganizationID]; ok {
				parent.Items = append(parent.Items, org)
			}
		}
	}

	// Calculate total discharge for organizations with children (cascades)
	// We need to do this recursively from bottom to top
	var calculateDischarge func(org *dto.CascadeWithDetails)
	calculateDischarge = func(org *dto.CascadeWithDetails) {
		if len(org.Items) > 0 {
			var totalDischarge float64
			for _, child := range org.Items {
				// First calculate discharge for the child
				calculateDischarge(child)
				// Then add it to parent's total
				if child.CurrentDischarge != nil {
					totalDischarge += *child.CurrentDischarge
				}
			}
			// If this org has its own discharge or children with discharge, set the total
			if org.CurrentDischarge != nil {
				totalDischarge += *org.CurrentDischarge
			}
			if totalDischarge > 0 {
				org.CurrentDischarge = &totalDischarge
			}
		}
	}

	// Calculate discharges for all orgs
	for _, org := range allOrgs {
		calculateDischarge(org)
	}

	// Enrich with ASCUE metrics if fetcher is provided
	if ascueFetcher != nil {
		ascueMetrics, err := ascueFetcher.FetchAll(ctx)
		if err != nil {
			// Error fetching ASCUE metrics - graceful degradation
			// Just continue without ASCUE metrics (silently)
		} else {
			// Apply ASCUE metrics to organizations
			for _, org := range allOrgs {
				if metrics, ok := ascueMetrics[org.ID]; ok {
					org.ASCUEMetrics = metrics
				}
			}
		}
	}

	// Return only cascades (organizations with type 'kaskad' that don't have a kaskad parent)
	var result []*dto.CascadeWithDetails
	for _, org := range allOrgs {
		// Check if this org has type 'kaskad'
		hasKaskadType := false
		for _, t := range org.Types {
			if t == "cascade" {
				hasKaskadType = true
				break
			}
		}

		// Include if it's a kaskad and either has no parent OR parent is not a kaskad
		if hasKaskadType {
			includeOrg := true
			if org.ParentOrganizationID != nil {
				if parent, ok := orgMap[*org.ParentOrganizationID]; ok {
					// Check if parent is also a kaskad
					for _, t := range parent.Types {
						if t == "cascade" {
							includeOrg = false
							break
						}
					}
				}
			}
			if includeOrg {
				result = append(result, org)
			}
		}
	}

	if result == nil {
		result = make([]*dto.CascadeWithDetails, 0)
	}

	return result, nil
}

// GetOrganizationsWithReservoir gets specific organizations with reservoir metrics
func (r *Repo) GetOrganizationsWithReservoir(ctx context.Context, orgIDs []int64, reservoirFetcher dto.ReservoirFetcher, date string) ([]*dto.OrganizationWithReservoir, error) {
	const op = "storage.repo.GetOrganizationsWithReservoir"

	if len(orgIDs) == 0 {
		return []*dto.OrganizationWithReservoir{}, nil
	}

	// Get specific organizations by ID
	const orgQuery = `
		SELECT id, name
		FROM organizations
		WHERE id = ANY($1)
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, orgQuery, pq.Array(orgIDs))
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query organizations: %w", op, err)
	}
	defer rows.Close()

	var result []*dto.OrganizationWithReservoir
	orgMap := make(map[int64]*dto.OrganizationWithReservoir)

	for rows.Next() {
		var org dto.OrganizationWithReservoir
		if err := rows.Scan(&org.OrganizationID, &org.OrganizationName); err != nil {
			return nil, fmt.Errorf("%s: failed to scan organization: %w", op, err)
		}
		org.Contacts = make([]*contact.Model, 0)
		result = append(result, &org)
		orgMap[org.OrganizationID] = &org
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	// Get contacts for the organizations
	const contactQuery = `
		SELECT
			c.id, c.fio, c.email, c.phone, c.ip_phone, c.dob,
			c.external_organization_name, c.created_at, c.updated_at,
			c.organization_id,
			o.id as org_id, o.name as org_name,
			d.id as dept_id, d.name as dept_name,
			p.id as pos_id, p.name as pos_name, p.description as pos_description
		FROM
			contacts c
		LEFT JOIN
			organizations o ON c.organization_id = o.id
		LEFT JOIN
			departments d ON c.department_id = d.id
		LEFT JOIN
			positions p ON c.position_id = p.id
		WHERE
			c.organization_id = ANY($1)
		ORDER BY c.organization_id
	`

	contactRows, err := r.db.QueryContext(ctx, contactQuery, pq.Array(orgIDs))
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query contacts: %w", op, err)
	}
	defer contactRows.Close()

	for contactRows.Next() {
		var c contact.Model
		var orgID int64
		var (
			email, phone, ipPhone, extOrg sql.NullString
			dob                           sql.NullTime
			orgDBID, deptID, posID        sql.NullInt64
			orgName, deptName, posName    sql.NullString
			posDescription                sql.NullString
		)

		err := contactRows.Scan(
			&c.ID, &c.Name, &email, &phone, &ipPhone, &dob, &extOrg,
			&c.CreatedAt, &c.UpdatedAt,
			&orgID,
			&orgDBID, &orgName,
			&deptID, &deptName,
			&posID, &posName, &posDescription,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan contact: %w", op, err)
		}

		// Set nullable fields
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
		if dob.Valid {
			c.DOB = &dob.Time
		}

		if orgData, ok := orgMap[orgID]; ok {
			orgData.Contacts = append(orgData.Contacts, &c)
		}
	}

	if err = contactRows.Err(); err != nil {
		return nil, fmt.Errorf("%s: contact rows iteration error: %w", op, err)
	}

	// Enrich with reservoir metrics if fetcher is provided
	if reservoirFetcher != nil {
		reservoirMetrics, err := reservoirFetcher.FetchAll(ctx, date)
		if err != nil {
			// Error fetching reservoir metrics - graceful degradation
			// Just continue without reservoir metrics (silently)
		} else {
			// Apply reservoir metrics to organizations
			for _, org := range result {
				if metrics, ok := reservoirMetrics[org.OrganizationID]; ok {
					org.ReservoirMetrics = metrics
				}
			}
		}
	}

	return result, nil
}
