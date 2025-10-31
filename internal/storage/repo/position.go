package repo

import (
	"context"
	"fmt"
	"srmt-admin/internal/lib/model/position"
	"srmt-admin/internal/storage"
	"strings"
)

func (r *Repo) AddPosition(ctx context.Context, name, description string) (int64, error) {
	const op = "storage.repo.position.AddPosition"
	const query = "INSERT INTO positions (name, description) VALUES ($1, $2) RETURNING id"

	var id int64
	// Используем QueryRowContext, так как ожидаем одну строку в ответ (RETURNING id)
	err := r.db.QueryRowContext(ctx, query, name, description).Scan(&id)
	if err != nil {
		// Проверяем на специфические ошибки БД, например, нарушение UNIQUE constraint
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return 0, translatedErr
		}
		// Возвращаем общую ошибку, если трансляция не удалась
		return 0, fmt.Errorf("%s: failed to execute query: %w", op, err)
	}

	return id, nil
}

func (r *Repo) GetAllPositions(ctx context.Context) ([]position.Model, error) {
	const op = "storage.repo.position.GetAllPositions"
	const query = "SELECT id, name, description FROM positions ORDER BY name"

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query positions: %w", op, err)
	}
	defer rows.Close()

	var positions []position.Model
	for rows.Next() {
		var p position.Model
		if err := rows.Scan(&p.ID, &p.Name, &p.Description); err != nil {
			return nil, fmt.Errorf("%s: failed to scan position row: %w", op, err)
		}
		positions = append(positions, p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	// Возвращаем пустой срез, а не nil, если должностей нет. Это лучшая практика.
	if positions == nil {
		positions = make([]position.Model, 0)
	}

	return positions, nil
}

func (r *Repo) EditPosition(ctx context.Context, id int64, name, description *string) error {
	const op = "storage.repo.position.EditPosition"

	var query strings.Builder
	query.WriteString("UPDATE positions SET ")

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

	// Если нечего обновлять, просто выходим
	if len(setClauses) == 0 {
		return nil
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
		return storage.ErrNotFound // Используем вашу стандартную ошибку "не найдено"
	}

	return nil
}

func (r *Repo) DeletePosition(ctx context.Context, id int64) error {
	const op = "storage.repo.position.DeletePosition"
	const query = "DELETE FROM positions WHERE id = $1"

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
