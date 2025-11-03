package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/lib/pq"
	"srmt-admin/internal/lib/model/organization"
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
				parent.Children = append(parent.Children, org)
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
