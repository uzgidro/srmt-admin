package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"srmt-admin/internal/lib/model/category"
	"srmt-admin/internal/storage"
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

func (r *Repo) GetCategoryByID(ctx context.Context, id int64) (category.Model, error) {
	const op = "repo.file-category.GetCategoryByID"

	const query = `
		SELECT id, name, display_name, description, parent_id
		FROM categories
		WHERE id = $1
	`

	var cat category.Model
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&cat.ID,
		&cat.Name,
		&cat.DisplayName,
		&cat.Description,
		&cat.ParentID,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return category.Model{}, storage.ErrNotFound
		}
		return category.Model{}, fmt.Errorf("%s: failed to scan row: %w", op, err)
	}

	return cat, nil
}

func (r *Repo) GetAllCategories(ctx context.Context) ([]category.Model, error) {
	const op = "repo.file-category.GetAllCategories"

	const query = `
		SELECT id, name, display_name, description, parent_id
		FROM categories
		ORDER BY display_name
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query categories: %w", op, err)
	}
	defer rows.Close()

	var categories []category.Model
	for rows.Next() {
		var cat category.Model
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.DisplayName, &cat.Description, &cat.ParentID); err != nil {
			return nil, fmt.Errorf("%s: failed to scan category row: %w", op, err)
		}
		categories = append(categories, cat)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if categories == nil {
		categories = make([]category.Model, 0)
	}

	return categories, nil
}
