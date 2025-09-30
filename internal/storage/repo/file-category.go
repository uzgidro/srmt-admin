package repo

import (
	"context"
	"fmt"
	"srmt-admin/internal/lib/model/category"
)

func (r *Repo) AddCategory(ctx context.Context, cat category.Model) (int64, error) {
	const op = "repo.file-category.AddCategory"

	const query = `
		INSERT INTO categories (name, display_name, description, parent_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		cat.Name,
		cat.DisplayName,
		cat.Description,
		cat.ParentID,
	).Scan(&id)

	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return 0, translatedErr
		}
		return 0, fmt.Errorf("%s: failed to execute query: %w", op, err)
	}

	return id, nil
}
