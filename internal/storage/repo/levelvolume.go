package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"srmt-admin/internal/lib/model/levelvolume"
)

// GetLevelVolume retrieves a level volume record by organization_id and level
// If not found, returns a zero-valued Model (all fields set to 0) without error
func (r *Repo) GetLevelVolume(ctx context.Context, organizationID int64, level float64) (*levelvolume.Model, error) {
	const op = "storage.repo.GetLevelVolume"

	const query = `
		SELECT id, level, volume, organization_id, created_at, updated_at
		FROM level_volume
		WHERE organization_id = $1 AND level = $2
	`

	var lv levelvolume.Model
	var updatedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, organizationID, level).Scan(
		&lv.ID,
		&lv.Level,
		&lv.Volume,
		&lv.OrganizationID,
		&lv.CreatedAt,
		&updatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Return zero-valued struct if not found
			return &levelvolume.Model{
				ID:             0,
				Level:          0,
				Volume:         0,
				OrganizationID: 0,
			}, nil
		}
		return nil, fmt.Errorf("%s: failed to query: %w", op, err)
	}

	if updatedAt.Valid {
		lv.UpdatedAt = &updatedAt.Time
	}

	return &lv, nil
}
