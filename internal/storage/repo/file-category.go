package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"srmt-admin/internal/lib/model/category"
	"srmt-admin/internal/lib/service/fileupload"
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

// GetCategoryByName finds a category by name, creating it if it doesn't exist
func (r *Repo) GetCategoryByName(ctx context.Context, categoryName string) (fileupload.CategoryModel, error) {
	const op = "repo.file-category.GetCategoryByName"

	// Attempt to find the category first
	const findQuery = `
		SELECT id, name, display_name, description, parent_id
		FROM categories
		WHERE name = $1
	`
	var cat category.Model
	err := r.db.QueryRowContext(ctx, findQuery, categoryName).Scan(
		&cat.ID,
		&cat.Name,
		&cat.DisplayName,
		&cat.Description,
		&cat.ParentID,
	)

	if err == nil {
		// Found it, return
		return cat, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		// A different error occurred during find
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return category.Model{}, translatedErr
		}
		return category.Model{}, fmt.Errorf("%s: failed to find category: %w", op, err)
	}

	// Not found (sql.ErrNoRows), so create it
	const createQuery = `
		INSERT INTO categories (name, display_name, description)
		VALUES ($1, $2, $3)
		RETURNING id, name, display_name, description, parent_id
	`
	var newCat category.Model
	err = r.db.QueryRowContext(ctx, createQuery, categoryName, categoryName, categoryName).Scan(
		&newCat.ID,
		&newCat.Name,
		&newCat.DisplayName,
		&newCat.Description,
		&newCat.ParentID,
	)

	if err != nil {
		// Error occurred during creation
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return category.Model{}, translatedErr
		}
		return category.Model{}, fmt.Errorf("%s: failed to create category: %w", op, err)
	}

	// Return the newly created category
	return newCat, nil
}

func (r *Repo) GetEventsCategory(ctx context.Context) (category.Model, error) {
	catModel, err := r.GetCategoryByName(ctx, "events")
	if err != nil {
		return category.Model{}, err
	}
	return catModel.(category.Model), nil
}

func (r *Repo) GetIncidentsCategory(ctx context.Context) (category.Model, error) {
	catModel, err := r.GetCategoryByName(ctx, "incidents")
	if err != nil {
		return category.Model{}, err
	}
	return catModel.(category.Model), nil
}

func (r *Repo) GetShutdownsCategory(ctx context.Context) (category.Model, error) {
	catModel, err := r.GetCategoryByName(ctx, "ges-shutdowns")
	if err != nil {
		return category.Model{}, err
	}
	return catModel.(category.Model), nil
}

func (r *Repo) GetDischargesCategory(ctx context.Context) (category.Model, error) {
	catModel, err := r.GetCategoryByName(ctx, "discharges")
	if err != nil {
		return category.Model{}, err
	}
	return catModel.(category.Model), nil
}

func (r *Repo) GetVisitsCategory(ctx context.Context) (category.Model, error) {
	catModel, err := r.GetCategoryByName(ctx, "visits")
	if err != nil {
		return category.Model{}, err
	}
	return catModel.(category.Model), nil
}
