package repo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lib/pq"
)

type SnowCoverItem struct {
	OrganizationID int64
	Cover          *float64
	Zones          json.RawMessage
}

type SnowCoverRow struct {
	OrganizationID   int64
	OrganizationName string
	Date             string
	Cover            *float64
	Zones            json.RawMessage
	ResourceDate     *string
}

func (r *Repo) GetSnowCoverByDates(ctx context.Context, dates []string) ([]SnowCoverRow, error) {
	const op = "storage.repo.GetSnowCoverByDates"

	const query = `
		SELECT m.organization_id, o.name, m.date, m.cover, m.zones, m.resource_date
		FROM modsnow m
		JOIN organizations o ON o.id = m.organization_id
		WHERE m.date = ANY($1)
		ORDER BY m.organization_id, m.date`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(dates))
	if err != nil {
		return nil, fmt.Errorf("%s: failed to execute query: %w", op, err)
	}
	defer rows.Close()

	var result []SnowCoverRow
	for rows.Next() {
		var row SnowCoverRow
		if err := rows.Scan(&row.OrganizationID, &row.OrganizationName, &row.Date, &row.Cover, &row.Zones, &row.ResourceDate); err != nil {
			return nil, fmt.Errorf("%s: failed to scan row: %w", op, err)
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: error during rows iteration: %w", op, err)
	}

	return result, nil
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
