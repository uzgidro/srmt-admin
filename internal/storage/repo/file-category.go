package repo

import (
	"context"
	"fmt"
)

func (r *Repo) AddCategory(ctx context.Context, parentID *int64, name, displayName, description string) (int64, error) {
	const op = "repo.file-category.AddCategory"

	const query = `
		INSERT INTO categories(parent_id, name, display_name, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	stmt, err := r.db.Prepare(query)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	var id int64
	err = stmt.QueryRowContext(ctx, parentID, name, displayName, description).Scan(&id)
	if err != nil {
		return 0, r.translator.Translate(err, op) // Делегируем перевод ошибки
	}

	return id, nil
}
