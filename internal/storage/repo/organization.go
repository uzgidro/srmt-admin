package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"srmt-admin/internal/lib/model/organization"
	"srmt-admin/internal/storage"
	"strings"
)

func (r *Repo) AddOrganization(ctx context.Context, name string, parentID *int64, typeIDs []int64) (int64, error) {
	const op = "storage.repo.AddOrganization"

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("%s: failed to begin transaction: %w", op, err)
	}
	defer tx.Rollback()

	const orgQuery = "INSERT INTO organizations (name, parent_organization_id) VALUES ($1, $2) RETURNING id"
	var orgID int64
	err = tx.QueryRowContext(ctx, orgQuery, name, parentID).Scan(&orgID)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return 0, translatedErr
		}
		return 0, fmt.Errorf("%s: failed to insert organization: %w", op, err)
	}

	if len(typeIDs) > 0 {
		var valueStrings []string
		var valueArgs []interface{}
		paramIndex := 1
		for _, typeID := range typeIDs {
			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d)", paramIndex, paramIndex+1))
			valueArgs = append(valueArgs, orgID)
			valueArgs = append(valueArgs, typeID)
			paramIndex += 2
		}

		linkQuery := "INSERT INTO organization_type_links (organization_id, type_id) VALUES " + strings.Join(valueStrings, ",")
		_, err = tx.ExecContext(ctx, linkQuery, valueArgs...)
		if err != nil {
			if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
				return 0, translatedErr
			}
			return 0, fmt.Errorf("%s: failed to link organization types: %w", op, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("%s: failed to commit transaction: %w", op, err)
	}

	return orgID, nil
}

func (r *Repo) GetAllOrganizations(ctx context.Context) ([]organization.Model, error) {
	const op = "storage.repo.GetAllOrganizations"

	const query = `
		SELECT
			o.id,
			o.name,
			o.parent_organization_id,
			COALESCE(t.types_json, '[]'::json) as types
		FROM
			organizations o
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

	var orgs []organization.Model
	for rows.Next() {
		var org organization.Model
		var typesJSON []byte
		if err := rows.Scan(&org.ID, &org.Name, &org.ParentOrganizationID, &typesJSON); err != nil {
			return nil, fmt.Errorf("%s: failed to scan organization: %w", op, err)
		}
		if err := json.Unmarshal(typesJSON, &org.Types); err != nil {
			return nil, fmt.Errorf("%s: failed to unmarshal types: %w", op, err)
		}
		orgs = append(orgs, org)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if orgs == nil {
		orgs = make([]organization.Model, 0)
	}

	return orgs, nil
}

func (r *Repo) EditOrganization(ctx context.Context, id int64, name *string, parentID **int64, typeIDs []int64) error {
	const op = "storage.repo.EditOrganization"

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: failed to begin transaction: %w", op, err)
	}
	defer tx.Rollback()

	var setClauses []string
	var args []interface{}
	argID := 1

	if name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argID))
		args = append(args, *name)
		argID++
	}
	if parentID != nil {
		setClauses = append(setClauses, fmt.Sprintf("parent_organization_id = $%d", argID))
		args = append(args, *parentID)
		argID++
	}

	if len(setClauses) > 0 {
		query := "UPDATE organizations SET " + strings.Join(setClauses, ", ") + fmt.Sprintf(" WHERE id = $%d", argID)
		args = append(args, id)

		res, err := tx.ExecContext(ctx, query, args...)
		if err != nil {
			if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
				return translatedErr
			}
			return fmt.Errorf("%s: failed to update organization: %w", op, err)
		}
		if rows, _ := res.RowsAffected(); rows == 0 {
			return storage.ErrNotFound
		}
	}

	if typeIDs != nil {
		const deleteQuery = "DELETE FROM organization_type_links WHERE organization_id = $1"
		if _, err := tx.ExecContext(ctx, deleteQuery, id); err != nil {
			return fmt.Errorf("%s: failed to delete old types: %w", op, err)
		}

		if len(typeIDs) > 0 {
			var valueStrings []string
			var valueArgs []interface{}
			paramIndex := 1
			for _, typeID := range typeIDs {
				valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d)", paramIndex, paramIndex+1))
				valueArgs = append(valueArgs, id)
				valueArgs = append(valueArgs, typeID)
				paramIndex += 2
			}
			linkQuery := "INSERT INTO organization_type_links (organization_id, type_id) VALUES " + strings.Join(valueStrings, ",")
			if _, err := tx.ExecContext(ctx, linkQuery, valueArgs...); err != nil {
				if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
					return translatedErr
				}
				return fmt.Errorf("%s: failed to link new types: %w", op, err)
			}
		}
	}

	return tx.Commit()
}

func (r *Repo) DeleteOrganization(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteOrganization"
	const query = "DELETE FROM organizations WHERE id = $1"

	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("%s: failed to execute statement: %w", op, err)
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
