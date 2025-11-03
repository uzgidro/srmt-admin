package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"srmt-admin/internal/lib/model/discharge"
	"srmt-admin/internal/lib/model/organization"
	"srmt-admin/internal/lib/model/user"
	"srmt-admin/internal/storage"
	"strings"
	"time"
)

// AddDischarge создает новую запись о холостом сбросе.
func (r *Repo) AddDischarge(ctx context.Context, orgID, createdByID int64, startTime time.Time, flowRate float64, reason string) (int64, error) {
	const op = "storage.repo.AddDischarge"
	const query = `
		INSERT INTO idle_water_discharges (organization_id, start_time, flow_rate_m3_s, reason, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	var id int64
	err := r.db.QueryRowContext(ctx, query, orgID, startTime, flowRate, reason, createdByID).Scan(&id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return 0, translatedErr
		}
		return 0, fmt.Errorf("%s: failed to execute query: %w", op, err)
	}

	return id, nil
}

// GetAllDischarges получает список всех сбросов, используя VIEW для вычисления объема.
func (r *Repo) GetAllDischarges(ctx context.Context) ([]discharge.Model, error) {
	const op = "storage.repo.GetAllDischarges"
	// Мы запрашиваем данные из VIEW, чтобы получить is_ongoing и total_volume_m3
	const query = `
		SELECT
			d.id,
			d.start_time,
			d.end_time,
			d.flow_rate_m3_s,
			d.reason,
			d.is_ongoing,
			d.total_volume_m3,
			-- Organization data
			o.id as org_id,
			o.name as org_name,
			o.parent_organization_id as org_parent_id,
			COALESCE(ot.types_json, '[]'::json) as org_types,
			-- Creator data
			creator.id as creator_id,
			creator.fio as creator_fio,
			-- Updater data
			updater.id as updater_id,
			updater.fio as updater_fio
		FROM
			v_idle_water_discharges_with_volume d
		JOIN organizations o ON d.organization_id = o.id
		JOIN users creator ON d.created_by = creator.id
		LEFT JOIN users updater ON d.updated_by = updater.id
		LEFT JOIN (
			SELECT
				otl.organization_id,
				json_agg(ot.name ORDER BY ot.name) as types_json
			FROM organization_type_links otl
			JOIN organization_types ot ON otl.type_id = ot.id
			GROUP BY otl.organization_id
		) ot ON o.id = ot.organization_id
		ORDER BY d.start_time DESC;
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query discharges: %w", op, err)
	}
	defer rows.Close()

	var discharges []discharge.Model
	for rows.Next() {
		var d discharge.Model
		var org organization.Model
		var creator user.ShortInfo
		var updater user.ShortInfo
		var orgTypesJSON []byte
		var updaterID sql.NullInt64 // Используем Null-типы для LEFT JOIN полей
		var updaterFIO sql.NullString

		err := rows.Scan(
			&d.ID, &d.StartedAt, &d.EndedAt, &d.FlowRate, &d.Reason, &d.IsOngoing, &d.TotalVolume,
			&org.ID, &org.Name, &org.ParentOrganizationID, &orgTypesJSON,
			&creator.ID, &creator.FIO,
			&updaterID, &updaterFIO,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan discharge row: %w", op, err)
		}

		if err := json.Unmarshal(orgTypesJSON, &org.Types); err != nil {
			return nil, fmt.Errorf("%s: failed to unmarshal org types: %w", op, err)
		}

		d.Organization = &org
		d.CreatedBy = &creator

		if updaterID.Valid {
			updater.ID = updaterID.Int64
			updater.FIO = updaterFIO.String
			d.UpdatedBy = &updater
		}

		discharges = append(discharges, d)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if discharges == nil {
		discharges = make([]discharge.Model, 0)
	}

	return discharges, nil
}

// EditDischarge обновляет запись о сбросе.
// Обновляются только не-nil поля.
func (r *Repo) EditDischarge(ctx context.Context, id, updatedByID int64, endTime *time.Time, flowRate *float64, reason *string) error {
	const op = "storage.repo.EditDischarge"

	var query strings.Builder
	query.WriteString("UPDATE idle_water_discharges SET updated_by = $1, ")
	args := []interface{}{updatedByID}
	argID := 2

	var setClauses []string
	if endTime != nil {
		setClauses = append(setClauses, fmt.Sprintf("end_time = $%d", argID))
		args = append(args, *endTime)
		argID++
	}
	if flowRate != nil {
		setClauses = append(setClauses, fmt.Sprintf("flow_rate_m3_s = $%d", argID))
		args = append(args, *flowRate)
		argID++
	}
	if reason != nil {
		setClauses = append(setClauses, fmt.Sprintf("reason = $%d", argID))
		args = append(args, *reason)
		argID++
	}

	if len(setClauses) == 0 {
		return nil // Нечего обновлять, кроме updated_by, что уже добавлено
	}

	query.WriteString(strings.Join(setClauses, ", "))
	query.WriteString(fmt.Sprintf(" WHERE id = $%d", argID))
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query.String(), args...)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to execute statement: %w", op, err)
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

// DeleteDischarge удаляет запись о сбросе по ID.
func (r *Repo) DeleteDischarge(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteDischarge"
	const query = "DELETE FROM idle_water_discharges WHERE id = $1"

	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to execute statement: %w", op, err)
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
