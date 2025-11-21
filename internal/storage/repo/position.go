package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/lib/pq"
	"srmt-admin/internal/lib/model/position"
	"srmt-admin/internal/storage"
	"strings"
)

func (r *Repo) AddPosition(ctx context.Context, name string, description *string) (int64, error) {
	const op = "storage.repo.AddPosition"

	const query = `
		INSERT INTO positions (name, description)
		VALUES ($1, $2)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query, name, description).Scan(&id)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code.Name() == "unique_violation" { // (UNIQUE(name))
				return 0, storage.ErrDuplicate
			}
		}
		return 0, fmt.Errorf("%s: failed to insert position: %w", op, err)
	}

	return id, nil
}

// GetAllPositions реализует интерфейс handlers.position.get_all.PositionGetter
func (r *Repo) GetAllPositions(ctx context.Context) ([]*position.Model, error) {
	const op = "storage.repo.GetAllPositions"

	const query = `
		SELECT id, name, description, created_at, updated_at
		FROM positions
		ORDER BY name`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query positions: %w", op, err)
	}
	defer rows.Close()

	var positions []*position.Model
	for rows.Next() {
		var pos position.Model
		var desc sql.NullString // Для nullable description
		if err := rows.Scan(
			&pos.ID,
			&pos.Name,
			&desc, // Сначала в NullString
			&pos.CreatedAt,
			&pos.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("%s: failed to scan position: %w", op, err)
		}
		if desc.Valid {
			pos.Description = &desc.String // Потом в *string
		}
		positions = append(positions, &pos)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if positions == nil {
		positions = make([]*position.Model, 0) // Возвращаем '[]' вместо 'null'
	}

	return positions, nil
}

// EditPosition реализует интерфейс handlers.position.update.PositionUpdater
func (r *Repo) EditPosition(ctx context.Context, id int64, name *string, description *string) error {
	const op = "storage.repo.EditPosition"

	var updates []string
	var args []interface{}
	argID := 1

	if name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argID))
		args = append(args, *name)
		argID++
	}
	if description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argID))
		args = append(args, *description)
		argID++
	}

	if len(updates) == 0 {
		return nil // Нечего обновлять
	}

	// (У таблицы positions должен быть триггер на updated_at)

	query := fmt.Sprintf("UPDATE positions SET %s WHERE id = $%d",
		strings.Join(updates, ", "),
		argID,
	)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code.Name() == "unique_violation" {
				return storage.ErrDuplicate
			}
		}
		return fmt.Errorf("%s: failed to update position: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeletePosition реализует интерфейс handlers.position.delete.PositionDeleter
func (r *Repo) DeletePosition(ctx context.Context, id int64) error {
	const op = "storage.repo.DeletePosition"

	res, err := r.db.ExecContext(ctx, "DELETE FROM positions WHERE id = $1", id)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code.Name() == "foreign_key_violation" {
			// (ON DELETE RESTRICT - не даст удалить, если используется в contacts)
			return storage.ErrForeignKeyViolation
		}
		return fmt.Errorf("%s: failed to delete position: %w", op, err)
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
