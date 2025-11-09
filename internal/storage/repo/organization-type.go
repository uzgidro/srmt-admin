package repo

import (
	"context"
	"fmt"
	"github.com/lib/pq"
	"srmt-admin/internal/lib/model/organization-type"
	"srmt-admin/internal/storage"
	"strings"
)

// AddOrganizationType добавляет новый тип организации в базу данных.
func (r *Repo) AddOrganizationType(ctx context.Context, name string, description *string) (int64, error) {
	const op = "storage.repo.AddOrganizationType"
	const query = "INSERT INTO organization_types (name, description) VALUES ($1, $2) RETURNING id"

	var id int64
	err := r.db.QueryRowContext(ctx, query, name, description).Scan(&id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return 0, translatedErr
		}
		return 0, fmt.Errorf("%s: failed to execute query: %w", op, err)
	}

	return id, nil
}

// GetAllOrganizationTypes получает список всех типов организаций.
func (r *Repo) GetAllOrganizationTypes(ctx context.Context) ([]organization_type.Model, error) {
	const op = "storage.repo.GetAllOrganizationTypes"
	const query = "SELECT id, name, description FROM organization_types ORDER BY name"

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query organization types: %w", op, err)
	}
	defer rows.Close()

	var types []organization_type.Model
	for rows.Next() {
		var ot organization_type.Model
		if err := rows.Scan(&ot.ID, &ot.Name, &ot.Description); err != nil {
			return nil, fmt.Errorf("%s: failed to scan organization type row: %w", op, err)
		}
		types = append(types, ot)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if types == nil {
		types = make([]organization_type.Model, 0)
	}

	return types, nil
}

func (r *Repo) GetOrganizationTypesMap(ctx context.Context) (map[int64][]string, error) {
	const op = "storage.repo.GetOrganizationTypesMap"
	const query = `
		SELECT
			otl.organization_id,
			array_agg(ot.name) as types
		FROM
			organization_type_links otl
		JOIN
			organization_types ot ON otl.type_id = ot.id
		GROUP BY
			otl.organization_id;
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query org types: %w", op, err)
	}
	defer rows.Close()

	typesMap := make(map[int64][]string)
	for rows.Next() {
		var orgID int64
		var types []string
		if err := rows.Scan(&orgID, pq.Array(&types)); err != nil {
			return nil, fmt.Errorf("%s: failed to scan org type row: %w", op, err)
		}
		typesMap[orgID] = types
	}
	return typesMap, rows.Err()
}

// EditOrganizationType обновляет данные типа организации по его ID.
// Обновляются только не-nil поля.
func (r *Repo) EditOrganizationType(ctx context.Context, id int64, name, description *string) error {
	const op = "storage.repo.EditOrganizationType"

	var query strings.Builder
	query.WriteString("UPDATE organization_types SET ")

	var args []interface{}
	var setClauses []string
	argID := 1

	if name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argID))
		args = append(args, *name)
		argID++
	}
	if description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argID))
		args = append(args, *description)
		argID++
	}

	if len(setClauses) == 0 {
		return nil // Нечего обновлять
	}

	query.WriteString(strings.Join(setClauses, ", "))
	query.WriteString(fmt.Sprintf(" WHERE id = $%d", argID))
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query.String(), args...)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
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

// DeleteOrganizationType удаляет тип организации по его ID.
func (r *Repo) DeleteOrganizationType(ctx context.Context, id string) error {
	const op = "storage.repo.DeleteOrganizationType"
	const query = "DELETE FROM organization_types WHERE id = $1"

	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		// Здесь может быть ошибка внешнего ключа, если тип используется
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
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
