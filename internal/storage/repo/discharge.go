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
func (r *Repo) AddDischarge(ctx context.Context, orgID, createdByID int64, startTime time.Time, endTime *time.Time, flowRate float64, reason string) (int64, error) {
	const op = "storage.repo.discharge.AddDischarge"
	// Обновляем запрос, добавляя end_time
	const query = `
		INSERT INTO idle_water_discharges (organization_id, start_time, end_time, flow_rate_m3_s, reason, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var id int64
	// Обновляем параметры для запроса
	err := r.db.QueryRowContext(ctx, query, orgID, startTime, endTime, flowRate, reason, createdByID).Scan(&id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return 0, translatedErr
		}
		return 0, fmt.Errorf("%s: failed to execute query: %w", op, err)
	}

	return id, nil
}

// GetAllDischarges получает список всех сбросов, используя VIEW для вычисления объема.
func (r *Repo) GetAllDischarges(ctx context.Context, isOngoing *bool, startDate, endDate *time.Time) ([]discharge.Model, error) {
	const op = "storage.repo.discharge.GetAllDischarges"

	// Базовый запрос с JOIN'ами
	baseQuery := `
		SELECT
			d.id, d.start_time, d.end_time, d.flow_rate_m3_s, d.reason,
			d.is_ongoing, d.total_volume_m3,
			o.id as org_id, o.name as org_name, o.parent_organization_id as org_parent_id,
			COALESCE(ot.types_json, '[]'::json) as org_types,
			creator.id as creator_id, creator.fio as creator_fio,
			updater.id as updater_id, updater.fio as updater_fio
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
	`

	var conditions []string
	var args []interface{}
	argID := 1

	// Добавляем условия фильтрации
	if isOngoing != nil {
		conditions = append(conditions, fmt.Sprintf("d.is_ongoing = $%d", argID))
		args = append(args, *isOngoing)
		argID++
	}

	// Фильтр по временному диапазону. Ищем пересечения.
	if startDate != nil && endDate != nil {
		// Период сброса [start_time, COALESCE(end_time, NOW())]
		// Период фильтра [startDate, endDate]
		// Условие пересечения: start1 <= end2 AND end1 >= start2
		conditions = append(conditions, fmt.Sprintf("(d.start_time <= $%d AND COALESCE(d.end_time, NOW()) >= $%d)", argID, argID+1))
		args = append(args, *endDate, *startDate)
		argID += 2
	} else if startDate != nil {
		// Если задана только начальная дата, ищем все, что началось или продолжалось после нее
		conditions = append(conditions, fmt.Sprintf("COALESCE(d.end_time, NOW()) >= $%d", argID))
		args = append(args, *startDate)
		argID++
	} else if endDate != nil {
		// Если задана только конечная дата, ищем все, что началось до нее
		conditions = append(conditions, fmt.Sprintf("d.start_time <= $%d", argID))
		args = append(args, *endDate)
		argID++
	}

	// Собираем финальный запрос
	var finalQuery strings.Builder
	finalQuery.WriteString(baseQuery)
	if len(conditions) > 0 {
		finalQuery.WriteString(" WHERE ")
		finalQuery.WriteString(strings.Join(conditions, " AND "))
	}
	finalQuery.WriteString(" ORDER BY d.start_time DESC;")

	// Выполняем запрос
	rows, err := r.db.QueryContext(ctx, finalQuery.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query discharges: %w", op, err)
	}
	defer rows.Close()

	// Сканирование результатов (код остается таким же, как у вас был)
	var discharges []discharge.Model
	for rows.Next() {
		var d discharge.Model
		var org organization.Model
		var creator user.ShortInfo
		var updater user.ShortInfo
		var orgTypesJSON []byte
		var updaterID sql.NullInt64
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
	const op = "storage.repo.discharge.EditDischarge"

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
	const op = "storage.repo.discharge.DeleteDischarge"
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
