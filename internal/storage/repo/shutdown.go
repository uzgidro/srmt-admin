package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/model/shutdown"
	"srmt-admin/internal/lib/model/user"
	"srmt-admin/internal/storage"
	"strings"
	"time"

	"github.com/lib/pq"
)

func (r *Repo) AddShutdown(ctx context.Context, req dto.AddShutdownRequest) (int64, error) {
	const op = "storage.repo.AddShutdown"

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("%s: failed to begin transaction: %w", op, err)
	}
	defer tx.Rollback()

	var idleDischargeID *int64

	if req.IdleDischargeVolumeThousandM3 != nil {
		flowRate, err := calculateFlowRate(req.StartTime, req.EndTime, *req.IdleDischargeVolumeThousandM3)
		if err != nil {
			return 0, fmt.Errorf("%s: failed to calculate flow rate: %w", op, err)
		}

		const idleQuery = `
			INSERT INTO idle_water_discharges (
				organization_id, start_time, end_time, flow_rate_m3_s, 
				reason, created_by
			)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id`

		var newID int64
		err = tx.QueryRowContext(ctx, idleQuery,
			req.OrganizationID, req.StartTime, req.EndTime,
			flowRate, req.Reason, req.CreatedByUserID,
		).Scan(&newID)

		if err != nil {
			if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
				return 0, translatedErr // (ErrForeignKeyViolation)
			}
			return 0, fmt.Errorf("%s: failed to insert idle_discharge: %w", op, err)
		}
		idleDischargeID = &newID
	}

	const query = `
		INSERT INTO shutdowns (
			organization_id, start_time, end_time, reason, 
			generation_loss_mwh, reported_by_contact_id, 
			idle_discharge_id, created_by_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

	var id int64
	err = tx.QueryRowContext(ctx, query,
		req.OrganizationID, req.StartTime, req.EndTime, req.Reason,
		req.GenerationLossMwh, req.ReportedByContactID,
		idleDischargeID, // (Подставляем ID)
		req.CreatedByUserID,
	).Scan(&id)

	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return 0, translatedErr
		}
		return 0, fmt.Errorf("%s: failed to insert shutdown: %w", op, err)
	}

	return id, tx.Commit()
}

// GetShutdowns (GET)
func (r *Repo) GetShutdowns(ctx context.Context, day time.Time) ([]*shutdown.ResponseModel, error) {
	const op = "storage.repo.GetShutdowns"

	// День начинается в 07:00 местного времени
	startOfDay := time.Date(day.Year(), day.Month(), day.Day(), 7, 0, 0, 0, day.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	query := selectShutdownFields + fromShutdownJoins +
		`WHERE (s.end_time > $1 OR s.end_time IS NULL) AND s.start_time < $2
		 ORDER BY s.start_time ASC`

	rows, err := r.db.QueryContext(ctx, query, startOfDay, endOfDay)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query shutdowns: %w", op, err)
	}
	defer rows.Close()

	var shutdowns []*shutdown.ResponseModel
	for rows.Next() {
		m, err := scanShutdownRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan shutdown row: %w", op, err)
		}
		shutdowns = append(shutdowns, m)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}
	if shutdowns == nil {
		shutdowns = make([]*shutdown.ResponseModel, 0)
	}

	// Load files for each shutdown
	for _, s := range shutdowns {
		files, err := r.loadShutdownFiles(ctx, s.ID)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to load files for shutdown %d: %w", op, s.ID, err)
		}
		s.Files = files
	}

	return shutdowns, nil
}

func (r *Repo) EditShutdown(ctx context.Context, id int64, req dto.EditShutdownRequest) error {
	const op = "storage.repo.EditShutdown"

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: failed to begin transaction: %w", op, err)
	}
	defer tx.Rollback()

	// 1. Получаем текущий Shutdown (organization_id, idle_discharge_id, start/end)
	var currentIdleID sql.NullInt64
	var currentStart time.Time
	var currentEnd sql.NullTime
	var currentOrgID int64 // (Нужен, если создаем новый ХС)

	err = tx.QueryRowContext(ctx,
		"SELECT organization_id, idle_discharge_id, start_time, end_time FROM shutdowns WHERE id = $1 FOR UPDATE",
		id,
	).Scan(&currentOrgID, &currentIdleID, &currentStart, &currentEnd)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ErrNotFound
		}
		return fmt.Errorf("%s: failed to get current shutdown: %w", op, err)
	}

	// --- (ИСПРАВЛЕННАЯ ЛОГИКА) ---

	// (Эта переменная будет хранить ID для Итогового UPDATE.)
	// (sql.NullInt64 позволяет нам различать NULL и 0.)
	var newIdleID sql.NullInt64
	// (Флаг, что мы *хотим* обновить поле `idle_discharge_id` в `shutdowns`)
	var updateIdleLink bool = false

	// 2. Логика обновления IdleDischarge

	// (Сценарий 1: Пользователь передал ОБЪЕМ)
	if req.IdleDischargeVolumeThousandM3 != nil {
		volume := *req.IdleDischargeVolumeThousandM3

		// (Определяем Start/End для расчета потока)
		start := currentStart
		if req.StartTime != nil {
			start = *req.StartTime
		}

		// (Определяем End)
		var end time.Time
		var hasEndTime bool
		if req.EndTime != nil {
			// Пользователь передал новый EndTime
			end = *req.EndTime
			hasEndTime = true
		} else if currentEnd.Valid {
			// Используем существующий EndTime
			end = currentEnd.Time
			hasEndTime = true
		}

		// (Проверяем, что EndTime есть для расчета)
		if !hasEndTime {
			return fmt.Errorf("%s: end_time is required to calculate idle discharge flow rate", op)
		}

		// Рассчитываем поток
		flowRate, err := calculateFlowRate(start, &end, volume)
		if err != nil {
			return fmt.Errorf("%s: failed to calculate flow rate on update: %w", op, err)
		}

		if currentIdleID.Valid {
			// (Уже был - ОБНОВЛЯЕМ)
			_, err = tx.ExecContext(ctx,
				"UPDATE idle_water_discharges SET start_time = $1, end_time = $2, flow_rate_m3_s = $3 WHERE id = $4",
				start, end, flowRate, currentIdleID.Int64,
			)
			if err != nil {
				return fmt.Errorf("%s: failed to update idle_discharge: %w", op, err)
			}
			// (Мы не меняем ссылку в `shutdowns`, она уже есть)

		} else if volume > 0 {
			// (Не было, но объем > 0 - СОЗДАЕМ)
			var createdIdleID int64
			err = tx.QueryRowContext(ctx,
				"INSERT INTO idle_water_discharges (organization_id, start_time, end_time, flow_rate_m3_s, created_by) VALUES ($1, $2, $3, $4, $5) RETURNING id",
				currentOrgID, start, end, flowRate, req.CreatedByUserID,
			).Scan(&createdIdleID)
			if err != nil {
				return fmt.Errorf("%s: failed to insert idle_discharge: %w", op, err)
			}

			// (Говорим Итоговому UPDATE, что нужно привязать этот ID)
			newIdleID.Valid = true
			newIdleID.Int64 = createdIdleID
			updateIdleLink = true
		}

		// (Сценарий 2: Объем НЕ пришел (nil), но ссылка БЫЛА)
	} else if currentIdleID.Valid {
		// (Пользователь не передал volume, значит, он хочет удалить ХС)
		_, err = tx.ExecContext(ctx, "DELETE FROM idle_water_discharges WHERE id = $1", currentIdleID.Int64)
		if err != nil {
			return fmt.Errorf("%s: failed to delete old idle_discharge: %w", op, err)
		}

		// (Говорим Итоговому UPDATE, что нужно установить NULL)
		newIdleID.Valid = false // (NULL)
		updateIdleLink = true
	}

	// 3. Динамический UPDATE самого Shutdowns
	var updates []string
	var args []interface{}
	argID := 1

	if req.OrganizationID != nil {
		updates = append(updates, fmt.Sprintf("organization_id = $%d", argID))
		args = append(args, *req.OrganizationID)
		argID++
	}
	if req.StartTime != nil {
		updates = append(updates, fmt.Sprintf("start_time = $%d", argID))
		args = append(args, *req.StartTime)
		argID++
	}
	// end_time is always updated: nil = NULL, value = value
	updates = append(updates, fmt.Sprintf("end_time = $%d", argID))
	args = append(args, req.EndTime)
	argID++
	if req.Reason != nil {
		updates = append(updates, fmt.Sprintf("reason = $%d", argID))
		args = append(args, *req.Reason)
		argID++
	}
	// generation_loss_mwh is always updated: nil = NULL, value = value
	updates = append(updates, fmt.Sprintf("generation_loss_mwh = $%d", argID))
	args = append(args, req.GenerationLossMwh) // nil → NULL
	argID++
	if req.ReportedByContactID != nil {
		updates = append(updates, fmt.Sprintf("reported_by_contact_id = $%d", argID))
		args = append(args, *req.ReportedByContactID)
		argID++
	}

	// (ИСПРАВЛЕНО: Проверяем наш флаг)
	if updateIdleLink {
		updates = append(updates, fmt.Sprintf("idle_discharge_id = $%d", argID))
		if newIdleID.Valid {
			args = append(args, newIdleID.Int64)
		} else {
			args = append(args, nil)
		}
		argID++
	}

	if len(updates) == 0 {
		return tx.Commit() // Нечего обновлять
	}
	updates = append(updates, "updated_at = NOW()")

	query := fmt.Sprintf("UPDATE shutdowns SET %s WHERE id = $%d",
		strings.Join(updates, ", "),
		argID,
	)
	args = append(args, id)

	res, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to update shutdown: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	} // (На всякий случай)

	return tx.Commit() // (Коммит!)
}

func (r *Repo) DeleteShutdown(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteShutdown"

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: failed to begin transaction: %w", op, err)
	}
	defer tx.Rollback()

	// 1. Получаем ID холостого сброса (если он есть)
	var idleDischargeID sql.NullInt64
	err = tx.QueryRowContext(ctx, "SELECT idle_discharge_id FROM shutdowns WHERE id = $1", id).Scan(&idleDischargeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ErrNotFound
		}
		return fmt.Errorf("%s: failed to get shutdown: %w", op, err)
	}

	// 2. Удаляем Shutdown
	res, err := tx.ExecContext(ctx, "DELETE FROM shutdowns WHERE id = $1", id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to delete shutdown: %w", op, err)
	}
	if rowsAffected, _ := res.RowsAffected(); rowsAffected == 0 {
		return storage.ErrNotFound
	}

	// 3. (НОВАЯ ЛОГИКА) Если ссылка была, удаляем и ХС
	if idleDischargeID.Valid {
		_, err = tx.ExecContext(ctx, "DELETE FROM idle_water_discharges WHERE id = $1", idleDischargeID.Int64)
		if err != nil {
			// (Ошибка здесь откатит всю транзакцию)
			return fmt.Errorf("%s: failed to delete associated idle_discharge: %w", op, err)
		}
	}

	return tx.Commit() // (Коммит!)
}

func calculateFlowRate(start time.Time, end *time.Time, volumeThousandM3 float64) (float64, error) {
	if end == nil {
		return 0, errors.New("end_time is nil, cannot calculate flow rate")
	}
	durationSeconds := end.Sub(start).Seconds()
	if durationSeconds <= 0 {
		return 0, errors.New("duration is zero or negative")
	}
	volumeM3 := volumeThousandM3 * 1000
	return volumeM3 / durationSeconds, nil
}

func scanShutdownRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*shutdown.ResponseModel, error) {
	var m shutdown.ResponseModel
	var (
		reason, createdByFIO sql.NullString
		genLoss, dVolumeThM3 sql.NullFloat64
		createdByUserID      int64
	)

	err := scanner.Scan(
		&m.ID, &m.StartedAt, &m.EndedAt, &reason, &genLoss, &m.CreatedAt,
		&m.OrganizationID,
		&m.OrganizationName,
		&createdByFIO,
		&createdByUserID,
		&dVolumeThM3, // (Объем)
	)
	if err != nil {
		return nil, err
	}

	if reason.Valid {
		m.Reason = &reason.String
	}
	if genLoss.Valid {
		m.GenerationLossMwh = &genLoss.Float64
	}
	if dVolumeThM3.Valid {
		m.IdleDischargeVolumeThousandM3 = &dVolumeThM3.Float64
	}

	// Create user model
	m.CreatedByUser = &user.ShortInfo{
		ID: createdByUserID,
	}
	if createdByFIO.Valid {
		m.CreatedByUser.Name = &createdByFIO.String
	}

	return &m, nil
}

const (
	selectShutdownFields = `
		SELECT
			s.id, s.start_time, s.end_time, s.reason, s.generation_loss_mwh, s.created_at,
			s.organization_id,
			COALESCE(o.name, '') as org_name,
			COALESCE(uc.fio, '') as created_by_fio,
			s.created_by_user_id,
			
			(v_idw.total_volume_mln_m3 * 1000.0) as discharge_volume_thousand_m3
	`
	fromShutdownJoins = `
		FROM
			shutdowns s
		LEFT JOIN
			organizations o ON s.organization_id = o.id
		LEFT JOIN
			users u ON s.created_by_user_id = u.id
		LEFT JOIN
			contacts uc ON u.contact_id = uc.id
		LEFT JOIN
			v_idle_water_discharges_with_volume v_idw ON s.idle_discharge_id = v_idw.id
	`
)

// LinkShutdownFiles links files to a shutdown
func (r *Repo) LinkShutdownFiles(ctx context.Context, shutdownID int64, fileIDs []int64) error {
	const op = "storage.repo.shutdown.LinkShutdownFiles"

	if len(fileIDs) == 0 {
		return nil
	}

	query := `
		INSERT INTO shutdown_file_links (shutdown_id, file_id)
		VALUES ($1, unnest($2::bigint[]))
		ON CONFLICT DO NOTHING
	`

	_, err := r.db.ExecContext(ctx, query, shutdownID, pq.Array(fileIDs))
	if err != nil {
		return fmt.Errorf("%s: failed to link files: %w", op, err)
	}

	return nil
}

// UnlinkShutdownFiles removes all file links for a shutdown
func (r *Repo) UnlinkShutdownFiles(ctx context.Context, shutdownID int64) error {
	const op = "storage.repo.shutdown.UnlinkShutdownFiles"

	query := `DELETE FROM shutdown_file_links WHERE shutdown_id = $1`
	_, err := r.db.ExecContext(ctx, query, shutdownID)
	if err != nil {
		return fmt.Errorf("%s: failed to unlink files: %w", op, err)
	}

	return nil
}

// loadShutdownFiles loads files for a shutdown
func (r *Repo) loadShutdownFiles(ctx context.Context, shutdownID int64) ([]file.Model, error) {
	const op = "storage.repo.shutdown.loadShutdownFiles"

	query := `
		SELECT f.id, f.file_name, f.object_key, f.category_id, f.mime_type, f.size_bytes, f.created_at
		FROM files f
		INNER JOIN shutdown_file_links sfl ON f.id = sfl.file_id
		WHERE sfl.shutdown_id = $1
		ORDER BY f.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, shutdownID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query files: %w", op, err)
	}
	defer rows.Close()

	var files []file.Model
	for rows.Next() {
		var f file.Model
		if err := rows.Scan(&f.ID, &f.FileName, &f.ObjectKey, &f.CategoryID, &f.MimeType, &f.SizeBytes, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("%s: failed to scan file row: %w", op, err)
		}
		files = append(files, f)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	return files, nil
}
