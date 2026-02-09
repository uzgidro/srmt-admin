package repo

import (
	"context"
	"fmt"
)

type SnowCoverItem struct {
	OrganizationID int64
	Cover          *float64
}

func (r *Repo) UpsertSnowCoverBatch(ctx context.Context, date string, items []SnowCoverItem) error {
	const op = "storage.repo.UpsertSnowCoverBatch"

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: begin tx: %w", op, err)
	}
	defer tx.Rollback()

	const query = `
		INSERT INTO modsnow (organization_id, date, cover, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (organization_id, date)
		DO UPDATE SET cover = EXCLUDED.cover, updated_at = NOW()`

	for _, item := range items {
		if _, err := tx.ExecContext(ctx, query, item.OrganizationID, date, item.Cover); err != nil {
			return fmt.Errorf("%s: upsert org_id=%d: %w", op, item.OrganizationID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: commit: %w", op, err)
	}

	return nil
}
