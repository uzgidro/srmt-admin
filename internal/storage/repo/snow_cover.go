package repo

import (
	"context"
	"encoding/json"
	"fmt"
)

type SnowCoverItem struct {
	OrganizationID int64
	Cover          *float64
	Zones          json.RawMessage
}

func (r *Repo) UpsertSnowCoverBatch(ctx context.Context, date string, resourceDate string, items []SnowCoverItem) error {
	const op = "storage.repo.UpsertSnowCoverBatch"

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: begin tx: %w", op, err)
	}
	defer tx.Rollback()

	const query = `
		INSERT INTO modsnow (organization_id, date, cover, zones, resource_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (organization_id, date)
		DO UPDATE SET cover = EXCLUDED.cover, zones = EXCLUDED.zones, resource_date = EXCLUDED.resource_date, updated_at = NOW()`

	for _, item := range items {
		if _, err := tx.ExecContext(ctx, query, item.OrganizationID, date, item.Cover, item.Zones, resourceDate); err != nil {
			return fmt.Errorf("%s: upsert org_id=%d: %w", op, item.OrganizationID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: commit: %w", op, err)
	}

	return nil
}
