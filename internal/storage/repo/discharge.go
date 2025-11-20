package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"srmt-admin/internal/lib/model/discharge"
	"srmt-admin/internal/lib/model/organization"
	"srmt-admin/internal/lib/model/user"
	"srmt-admin/internal/storage"
	"strings"
	"time"
)

// AddDischarge создает новую запись о холостом сбросе.
func (r *Repo) AddDischarge(ctx context.Context, orgID, createdByID int64, startTime time.Time, endTime *time.Time, flowRate float64, reason *string) (int64, error) {
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

	// ИСПРАВЛЕНИЕ: Запрос обновлен для получения ФИО из таблицы contacts
	baseQuery := `
		SELECT
			d.id, d.start_time, d.end_time, d.flow_rate_m3_s, d.reason, d.approved,
			d.is_ongoing, d.total_volume_mln_m3,
			o.id as org_id, o.name as org_name, o.parent_organization_id as org_parent_id,
			COALESCE(ot.types_json, '[]'::json) as org_types,
			creator.id as creator_id,
			creator_contact.fio as creator_fio,
			approver.id as approver_id,
			approver_contact.fio as approver_fio
		FROM
			v_idle_water_discharges_with_volume d
		JOIN
			organizations o ON d.organization_id = o.id
		JOIN
			users creator ON d.created_by = creator.id
		JOIN
			contacts creator_contact ON creator.contact_id = creator_contact.id
		LEFT JOIN
			users approver ON d.approved_by = approver.id
		LEFT JOIN
			contacts approver_contact ON approver.contact_id = approver_contact.id
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

	// Добавляем условия фильтрации (без изменений)
	if isOngoing != nil {
		conditions = append(conditions, fmt.Sprintf("d.is_ongoing = $%d", argID))
		args = append(args, *isOngoing)
		argID++
	}

	if startDate != nil && endDate != nil {
		conditions = append(conditions, fmt.Sprintf("(d.start_time >= $%d AND d.start_time < $%d)", argID, argID+1))
		args = append(args, *startDate, *endDate)
		argID += 2
	} else if startDate != nil {
		conditions = append(conditions, fmt.Sprintf("d.start_time >= $%d", argID))
		args = append(args, *startDate)
		argID++
	} else if endDate != nil {
		conditions = append(conditions, fmt.Sprintf("d.start_time < $%d", argID))
		args = append(args, *endDate)
		argID++
	}

	// Собираем финальный запрос (без изменений)
	var finalQuery strings.Builder
	finalQuery.WriteString(baseQuery)
	if len(conditions) > 0 {
		finalQuery.WriteString(" WHERE ")
		finalQuery.WriteString(strings.Join(conditions, " AND "))
	}
	finalQuery.WriteString(" ORDER BY d.start_time ASC;")

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
		var orgTypesJSON []byte

		// ИСПРАВЛЕНИЕ: Переменные для сканирования данных пользователя
		var creatorID int64
		var creatorFIO string
		var approverID sql.NullInt64
		var approverFIO sql.NullString

		err := rows.Scan(
			&d.ID, &d.StartedAt, &d.EndedAt, &d.FlowRate, &d.Reason, &d.Approved,
			&d.IsOngoing, &d.TotalVolume,
			&org.ID, &org.Name, &org.ParentOrganizationID, &orgTypesJSON,
			&creatorID, &creatorFIO,
			&approverID, &approverFIO,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan discharge row: %w", op, err)
		}

		if err := json.Unmarshal(orgTypesJSON, &org.Types); err != nil {
			return nil, fmt.Errorf("%s: failed to unmarshal org types: %w", op, err)
		}

		d.Organization = &org

		// ИСПРАВЛЕНИЕ: Собираем модель пользователя
		fioCreator := creatorFIO
		d.CreatedByUser = &user.ShortInfo{
			ID:   creatorID,
			Name: &fioCreator,
		}

		if approverID.Valid {
			approver := &user.ShortInfo{
				ID: approverID.Int64,
			}
			if approverFIO.Valid {
				fioApprover := approverFIO.String
				approver.Name = &fioApprover
			}
			d.ApprovedByUser = approver
		}

		discharges = append(discharges, d)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if discharges == nil {
		return []discharge.Model{}, nil // Возвращаем пустой слайс вместо nil
	}

	return discharges, nil
}

func (r *Repo) GetDischargesByCascades(ctx context.Context, isOngoing *bool, startDate, endDate *time.Time) ([]discharge.Cascade, error) {
	const op = "storage.repo.discharge.GetDischargesByCascades"

	// ИСПРАВЛЕНИЕ: Запрос обновлен для использования users.contact_id -> contacts.id
	const query = `
		SELECT
			cascade_org.id as cascade_id,
			cascade_org.name as cascade_name,
			hpp_org.id as hpp_id,
			hpp_org.name as hpp_name,
			d.id, d.start_time, d.end_time, d.flow_rate_m3_s, d.reason, d.approved,
			d.is_ongoing, d.total_volume_mln_m3,
			creator.id as creator_id,
			creator_contact.fio as creator_fio,
			approver.id as approver_id,
			approver_contact.fio as approver_fio
		FROM
			v_idle_water_discharges_with_volume d
		JOIN
			organizations hpp_org ON d.organization_id = hpp_org.id
		JOIN
			organizations cascade_org ON hpp_org.parent_organization_id = cascade_org.id
		JOIN
			users creator ON d.created_by = creator.id
		JOIN
			contacts creator_contact ON creator.contact_id = creator_contact.id
		LEFT JOIN
			users approver ON d.approved_by = approver.id
		LEFT JOIN
			contacts approver_contact ON approver.contact_id = approver_contact.id
	`

	var conditions []string
	var args []interface{}
	argID := 1

	// Динамическое добавление фильтров (код без изменений)
	if isOngoing != nil {
		conditions = append(conditions, fmt.Sprintf("d.is_ongoing = $%d", argID))
		args = append(args, *isOngoing)
		argID++
	}
	if startDate != nil && endDate != nil {
		conditions = append(conditions, fmt.Sprintf("(d.start_time >= $%d AND d.start_time < $%d)", argID, argID+1))
		args = append(args, *startDate, *endDate)
		argID += 2
	} else if startDate != nil {
		conditions = append(conditions, fmt.Sprintf("d.start_time >= $%d", argID))
		args = append(args, *startDate)
		argID++
	} else if endDate != nil {
		conditions = append(conditions, fmt.Sprintf("d.start_time < $%d", argID))
		args = append(args, *endDate)
		argID++
	}

	var finalQuery strings.Builder
	finalQuery.WriteString(query)
	if len(conditions) > 0 {
		finalQuery.WriteString(" WHERE " + strings.Join(conditions, " AND "))
	}
	finalQuery.WriteString(" ORDER BY cascade_name, hpp_name, d.start_time ASC;")

	rows, err := r.db.QueryContext(ctx, finalQuery.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query discharges: %w", op, err)
	}
	defer rows.Close()

	// Сборка иерархии (код без изменений)
	var result []discharge.Cascade
	var currentCascade *discharge.Cascade
	var currentHPP *discharge.HPP

	for rows.Next() {
		var cascadeID int64
		var cascadeName string
		var hppID int64
		var hppName string
		var d discharge.Model
		var creatorID int64
		var creatorFIO string
		var approverID sql.NullInt64
		var approverFIO sql.NullString

		err := rows.Scan(
			&cascadeID, &cascadeName, &hppID, &hppName,
			&d.ID, &d.StartedAt, &d.EndedAt, &d.FlowRate, &d.Reason, &d.Approved,
			&d.IsOngoing, &d.TotalVolume,
			&creatorID, &creatorFIO,
			&approverID, &approverFIO,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan row: %w", op, err)
		}

		d.Organization = &organization.Model{
			ID:   hppID,
			Name: hppName,
		}

		// Присваиваем Name создателю
		fioCreator := creatorFIO
		d.CreatedByUser = &user.ShortInfo{
			ID:   creatorID,
			Name: &fioCreator,
		}

		// Присваиваем Name утверждающему, если он есть
		if approverID.Valid {
			approver := &user.ShortInfo{
				ID: approverID.Int64,
			}
			if approverFIO.Valid {
				fioApprover := approverFIO.String
				approver.Name = &fioApprover
			}
			d.ApprovedByUser = approver
		}

		// Логика группировки по каскадам и ГЭС (без изменений)
		if currentCascade == nil || currentCascade.ID != cascadeID {
			result = append(result, discharge.Cascade{
				ID:   cascadeID,
				Name: cascadeName,
				HPPs: []discharge.HPP{},
			})
			currentCascade = &result[len(result)-1]
			currentHPP = nil
		}

		if currentHPP == nil || currentHPP.ID != hppID {
			currentCascade.HPPs = append(currentCascade.HPPs, discharge.HPP{
				ID:         hppID,
				Name:       hppName,
				Discharges: []discharge.Model{},
			})
			currentHPP = &currentCascade.HPPs[len(currentCascade.HPPs)-1]
		}

		currentHPP.TotalVolume += d.TotalVolume
		currentCascade.TotalVolume += d.TotalVolume
		d.TotalVolume = roundToThree(d.TotalVolume)
		currentHPP.Discharges = append(currentHPP.Discharges, d)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	// Округление и возврат результата (код без изменений)
	for i := range result {
		for j := range result[i].HPPs {
			result[i].HPPs[j].TotalVolume = roundToThree(result[i].HPPs[j].TotalVolume)
		}
		result[i].TotalVolume = roundToThree(result[i].TotalVolume)
	}

	if result == nil {
		return []discharge.Cascade{}, nil
	}

	return result, nil
}

// EditDischarge обновляет запись о сбросе.
// Обновляются только не-nil поля.
func (r *Repo) EditDischarge(ctx context.Context, id, approvedByID int64, startTime, endTime *time.Time, flowRate *float64, reason *string, approved *bool, organizationID *int64) error {
	const op = "storage.repo.discharge.EditDischarge"

	var query strings.Builder
	query.WriteString("UPDATE idle_water_discharges SET ")

	var args []interface{}
	var setClauses []string
	argID := 1

	if startTime != nil {
		setClauses = append(setClauses, fmt.Sprintf("start_time = $%d", argID))
		args = append(args, *startTime)
		argID++
	}
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
	if organizationID != nil {
		setClauses = append(setClauses, fmt.Sprintf("organization_id = $%d", argID))
		args = append(args, *organizationID)
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

// GetCurrentDischarges получает список текущих активных сбросов (start_time <= NOW() AND (end_time > NOW() OR end_time IS NULL))
func (r *Repo) GetCurrentDischarges(ctx context.Context) ([]discharge.Model, error) {
	const op = "storage.repo.discharge.GetCurrentDischarges"

	const query = `
		SELECT
			d.id, d.start_time, d.end_time, d.flow_rate_m3_s, d.reason, d.approved,
			d.is_ongoing, d.total_volume_mln_m3,
			o.id as org_id, o.name as org_name, o.parent_organization_id as org_parent_id,
			COALESCE(ot.types_json, '[]'::json) as org_types,
			creator.id as creator_id,
			creator_contact.fio as creator_fio,
			approver.id as approver_id,
			approver_contact.fio as approver_fio
		FROM
			v_idle_water_discharges_with_volume d
		JOIN
			organizations o ON d.organization_id = o.id
		JOIN
			users creator ON d.created_by = creator.id
		JOIN
			contacts creator_contact ON creator.contact_id = creator_contact.id
		LEFT JOIN
			users approver ON d.approved_by = approver.id
		LEFT JOIN
			contacts approver_contact ON approver.contact_id = approver_contact.id
		LEFT JOIN (
			SELECT
				otl.organization_id,
				json_agg(ot.name ORDER BY ot.name) as types_json
			FROM organization_type_links otl
			JOIN organization_types ot ON otl.type_id = ot.id
			GROUP BY otl.organization_id
		) ot ON o.id = ot.organization_id
		WHERE
			d.start_time <= NOW()
			AND (d.end_time > NOW() OR d.end_time IS NULL)
		ORDER BY d.start_time ASC;
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query current discharges: %w", op, err)
	}
	defer rows.Close()

	var discharges []discharge.Model
	for rows.Next() {
		var d discharge.Model
		var org organization.Model
		var orgTypesJSON []byte

		var creatorID int64
		var creatorFIO string
		var approverID sql.NullInt64
		var approverFIO sql.NullString

		err := rows.Scan(
			&d.ID, &d.StartedAt, &d.EndedAt, &d.FlowRate, &d.Reason, &d.Approved,
			&d.IsOngoing, &d.TotalVolume,
			&org.ID, &org.Name, &org.ParentOrganizationID, &orgTypesJSON,
			&creatorID, &creatorFIO,
			&approverID, &approverFIO,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan discharge row: %w", op, err)
		}

		if err := json.Unmarshal(orgTypesJSON, &org.Types); err != nil {
			return nil, fmt.Errorf("%s: failed to unmarshal org types: %w", op, err)
		}

		d.Organization = &org

		fioCreator := creatorFIO
		d.CreatedByUser = &user.ShortInfo{
			ID:   creatorID,
			Name: &fioCreator,
		}

		if approverID.Valid {
			approver := &user.ShortInfo{
				ID: approverID.Int64,
			}
			if approverFIO.Valid {
				fioApprover := approverFIO.String
				approver.Name = &fioApprover
			}
			d.ApprovedByUser = approver
		}

		discharges = append(discharges, d)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if discharges == nil {
		return []discharge.Model{}, nil
	}

	return discharges, nil
}

func roundToThree(val float64) float64 {
	return math.Round(val*1000) / 1000
}
