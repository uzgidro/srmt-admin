package repo

import (
	"context"
	"fmt"

	infraeventcategory "srmt-admin/internal/lib/model/infra-event-category"
	"srmt-admin/internal/storage"
)

func (r *Repo) GetInfraEventCategories(ctx context.Context) ([]*infraeventcategory.Model, error) {
	const op = "storage.repo.GetInfraEventCategories"

	query := `
		SELECT id, slug, display_name, label, sort_order, created_at
		FROM sc_infra_event_categories
		ORDER BY sort_order ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var categories []*infraeventcategory.Model
	for rows.Next() {
		var m infraeventcategory.Model
		if err := rows.Scan(&m.ID, &m.Slug, &m.DisplayName, &m.Label, &m.SortOrder, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		categories = append(categories, &m)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows: %w", op, err)
	}

	if categories == nil {
		categories = make([]*infraeventcategory.Model, 0)
	}

	return categories, nil
}

func (r *Repo) CreateInfraEventCategory(ctx context.Context, slug, displayName, label string, sortOrder int) (int64, error) {
	const op = "storage.repo.CreateInfraEventCategory"

	query := `
		INSERT INTO sc_infra_event_categories (slug, display_name, label, sort_order)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query, slug, displayName, label, sortOrder).Scan(&id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return 0, translatedErr
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

func (r *Repo) UpdateInfraEventCategory(ctx context.Context, id int64, slug, displayName, label string, sortOrder int) error {
	const op = "storage.repo.UpdateInfraEventCategory"

	query := `
		UPDATE sc_infra_event_categories
		SET slug = $1, display_name = $2, label = $3, sort_order = $4
		WHERE id = $5`

	res, err := r.db.ExecContext(ctx, query, slug, displayName, label, sortOrder, id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

func (r *Repo) DeleteInfraEventCategory(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteInfraEventCategory"

	res, err := r.db.ExecContext(ctx, "DELETE FROM sc_infra_event_categories WHERE id = $1", id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: affected rows: %w", op, err)
	}

	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}
