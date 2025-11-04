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
			d.id, d.start_time, d.end_time, d.flow_rate_m3_s, d.reason, d.approved, -- <<< ДОБАВЛЕНО d.approved
			d.is_ongoing, d.total_volume_mln_m3,
			o.id as org_id, o.name as org_name, o.parent_organization_id as org_parent_id,
			COALESCE(ot.types_json, '[]'::json) as org_types,
			creator.id as creator_id, creator.fio as creator_fio,
			approver.id as approver_id, approver.fio as approver_fio -- <<< ИЗМЕНЕНЫ АЛИАСЫ
		FROM
			v_idle_water_discharges_with_volume d
		JOIN organizations o ON d.organization_id = o.id
		JOIN users creator ON d.created_by = creator.id
		LEFT JOIN users approver ON d.approved_by = approver.id -- <<< ИЗМЕНЕН АЛИАС
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
		conditions = append(conditions, fmt.Sprintf("(d.start_time <= $%d AND COALESCE(d.end_time, NOW()) >= $%d)", argID, argID+1))
		args = append(args, *endDate, *startDate)
		argID += 2
	} else if startDate != nil {
		conditions = append(conditions, fmt.Sprintf("COALESCE(d.end_time, NOW()) >= $%d", argID))
		args = append(args, *startDate)
		argID++
	} else if endDate != nil {
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

	var discharges []discharge.Model
	for rows.Next() {
		var d discharge.Model
		var org organization.Model
		var creator user.ShortInfo
		var approver user.ShortInfo
		var orgTypesJSON []byte
		var approverID sql.NullInt64 // Используем Null-типы для LEFT JOIN полей
		var approverFIO sql.NullString

		err := rows.Scan(
			&d.ID, &d.StartedAt, &d.EndedAt, &d.FlowRate, &d.Reason, &d.Approved,
			&d.IsOngoing, &d.TotalVolume,
			&org.ID, &org.Name, &org.ParentOrganizationID, &orgTypesJSON,
			&creator.ID, &creator.FIO,
			&approverID, &approverFIO, // <<< ИЗМЕНЕНЫ ПЕРЕМЕННЫЕ
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan discharge row: %w", op, err)
		}

		if err := json.Unmarshal(orgTypesJSON, &org.Types); err != nil {
			return nil, fmt.Errorf("%s: failed to unmarshal org types: %w", op, err)
		}

		d.Organization = &org
		d.CreatedByUser = &creator // Предполагается, что в модели поле называется CreatedBy

		if approverID.Valid {
			approver.ID = approverID.Int64
			approver.FIO = approverFIO.String
			d.ApprovedByUser = &approver // Предполагается, что в модели поле называется ApprovedBy
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

func (r *Repo) GetDischargesByCascades(ctx context.Context, isOngoing *bool, startDate, endDate *time.Time) ([]discharge.Cascade, error) {
	const op = "storage.repo.discharge.GetDischargesByCascades"

	// 1. SQL-запрос остается таким же, он эффективно получает все нужные данные в плоском виде.
	// Важнейшая часть - ORDER BY, который группирует строки по каскадам и ГЭС.
	const query = `
		SELECT
			cascade_org.id as cascade_id,
			cascade_org.name as cascade_name,
			hpp_org.id as hpp_id,
			hpp_org.name as hpp_name,
			d.id, d.start_time, d.end_time, d.flow_rate_m3_s, d.reason, d.approved,
			d.is_ongoing, d.total_volume_mln_m3,
			creator.id as creator_id, creator.fio as creator_fio,
			approver.id as approver_id, approver.fio as approver_fio
		FROM
			v_idle_water_discharges_with_volume d
		JOIN
			organizations hpp_org ON d.organization_id = hpp_org.id
		JOIN
			organizations cascade_org ON hpp_org.parent_organization_id = cascade_org.id
		JOIN
			users creator ON d.created_by = creator.id
		LEFT JOIN
			users approver ON d.approved_by = approver.id
	`

	var conditions []string
	var args []interface{}
	argID := 1

	// 2. Динамическое добавление фильтров (код без изменений)
	if isOngoing != nil {
		conditions = append(conditions, fmt.Sprintf("d.is_ongoing = $%d", argID))
		args = append(args, *isOngoing)
		argID++
	}
	if startDate != nil && endDate != nil {
		conditions = append(conditions, fmt.Sprintf("(d.start_time <= $%d AND COALESCE(d.end_time, NOW()) >= $%d)", argID, argID+1))
		args = append(args, *endDate, *startDate)
		argID += 2
	} else if startDate != nil {
		conditions = append(conditions, fmt.Sprintf("COALESCE(d.end_time, NOW()) >= $%d", argID))
		args = append(args, *startDate)
		argID++
	} else if endDate != nil {
		conditions = append(conditions, fmt.Sprintf("d.start_time <= $%d", argID))
		args = append(args, *endDate)
		argID++
	}

	var finalQuery strings.Builder
	finalQuery.WriteString(query)
	if len(conditions) > 0 {
		finalQuery.WriteString(" WHERE " + strings.Join(conditions, " AND "))
	}
	finalQuery.WriteString(" ORDER BY cascade_name, hpp_name, d.start_time DESC;")

	rows, err := r.db.QueryContext(ctx, finalQuery.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query discharges: %w", op, err)
	}
	defer rows.Close()

	// 3. Упрощенная и корректная сборка иерархии
	var result []discharge.Cascade
	var currentCascade *discharge.Cascade
	var currentHPP *discharge.HPP

	for rows.Next() {
		var cascadeID int64
		var cascadeName string
		var hppID int64
		var hppName string
		var d discharge.Model
		var creator user.ShortInfo
		var approver user.ShortInfo
		var approverID sql.NullInt64
		var approverFIO sql.NullString

		err := rows.Scan(
			&cascadeID, &cascadeName, &hppID, &hppName,
			&d.ID, &d.StartedAt, &d.EndedAt, &d.FlowRate, &d.Reason, &d.Approved,
			&d.IsOngoing, &d.TotalVolume,
			&creator.ID, &creator.FIO,
			&approverID, &approverFIO,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan row: %w", op, err)
		}

		// Заполняем информацию о пользователях
		d.CreatedByUser = &creator
		if approverID.Valid {
			approver.ID = approverID.Int64
			approver.FIO = approverFIO.String
			d.ApprovedByUser = &approver
		}

		// Если это новый каскад, создаем его и добавляем в результат
		if currentCascade == nil || currentCascade.ID != cascadeID {
			result = append(result, discharge.Cascade{
				ID:   cascadeID,
				Name: cascadeName,
				HPPs: []discharge.HPP{},
			})
			currentCascade = &result[len(result)-1]
			currentHPP = nil // Сбрасываем текущую ГЭС при смене каскада
		}

		// Если это новая ГЭС (в рамках текущего каскада), создаем ее
		if currentHPP == nil || currentHPP.ID != hppID {
			currentCascade.HPPs = append(currentCascade.HPPs, discharge.HPP{
				ID:         hppID,
				Name:       hppName,
				Discharges: []discharge.Model{},
			})
			currentHPP = &currentCascade.HPPs[len(currentCascade.HPPs)-1]
		}

		// Добавляем сброс к текущей ГЭС
		currentHPP.Discharges = append(currentHPP.Discharges, d)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if result == nil {
		return []discharge.Cascade{}, nil // Возвращаем пустой срез, а не nil
	}

	return result, nil
}

// EditDischarge обновляет запись о сбросе.
// Обновляются только не-nil поля.
func (r *Repo) EditDischarge(ctx context.Context, id, approvedByID int64, endTime *time.Time, flowRate *float64, reason *string, approved *bool) error {
	const op = "storage.repo.discharge.EditDischarge"

	var query strings.Builder
	query.WriteString("UPDATE idle_water_discharges SET ")

	var args []interface{}
	var setClauses []string
	argID := 1

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
	// Если меняется статус 'approved', также записываем, кто это сделал
	if approved != nil {
		setClauses = append(setClauses, fmt.Sprintf("approved = $%d", argID))
		args = append(args, *approved)
		argID++

		setClauses = append(setClauses, fmt.Sprintf("approved_by = $%d", argID))
		args = append(args, approvedByID) // Тот, кто обновляет, тот и утверждает
		argID++
	}

	// Если нечего обновлять, выходим
	if len(setClauses) == 0 {
		return nil
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
