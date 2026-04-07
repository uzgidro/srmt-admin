package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"

	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/file"
	infraevent "srmt-admin/internal/lib/model/infra-event"
	"srmt-admin/internal/lib/model/user"
	"srmt-admin/internal/storage"
)

const (
	selectInfraEventFields = `
		SELECT
			e.id,
			e.category_id,
			c.slug,
			c.display_name,
			e.organization_id,
			COALESCE(o.name, '') as org_name,
			e.occurred_at,
			e.restored_at,
			e.description,
			e.remediation,
			e.notes,
			e.created_at,
			e.created_by_user_id,
			COALESCE(ct.fio, '') as user_fio
	`
	fromInfraEventJoins = `
		FROM sc_infra_events e
		JOIN sc_infra_event_categories c ON e.category_id = c.id
		LEFT JOIN organizations o ON e.organization_id = o.id
		LEFT JOIN users u ON e.created_by_user_id = u.id
		LEFT JOIN contacts ct ON u.contact_id = ct.id
	`
)

func scanInfraEventRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*infraevent.ResponseModel, error) {
	var m infraevent.ResponseModel
	var createdByUserID sql.NullInt64
	var createdByUserFIO string

	err := scanner.Scan(
		&m.ID,
		&m.CategoryID,
		&m.CategorySlug,
		&m.CategoryName,
		&m.OrganizationID,
		&m.OrganizationName,
		&m.OccurredAt,
		&m.RestoredAt,
		&m.Description,
		&m.Remediation,
		&m.Notes,
		&m.CreatedAt,
		&createdByUserID,
		&createdByUserFIO,
	)
	if err != nil {
		return nil, err
	}

	if createdByUserID.Valid {
		m.CreatedByUser = &user.ShortInfo{
			ID:   createdByUserID.Int64,
			Name: &createdByUserFIO,
		}
	}

	return &m, nil
}

func (r *Repo) CreateInfraEvent(ctx context.Context, req dto.AddInfraEventRequest) (int64, error) {
	const op = "storage.repo.CreateInfraEvent"

	query := `
		INSERT INTO sc_infra_events (category_id, organization_id, occurred_at, restored_at, description, remediation, notes, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.CategoryID,
		req.OrganizationID,
		req.OccurredAt,
		req.RestoredAt,
		req.Description,
		req.Remediation,
		req.Notes,
		req.CreatedByUserID,
	).Scan(&id)

	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return 0, translatedErr
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

func (r *Repo) GetInfraEvents(ctx context.Context, categoryID int64, day time.Time) ([]*infraevent.ResponseModel, error) {
	const op = "storage.repo.GetInfraEvents"

	startOfDay := time.Date(day.Year(), day.Month(), day.Day(), 5, 0, 0, 0, day.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	query := selectInfraEventFields + fromInfraEventJoins +
		`WHERE e.category_id = $1 AND e.occurred_at >= $2 AND e.occurred_at < $3
		 ORDER BY e.occurred_at ASC`

	return r.queryInfraEvents(ctx, op, query, categoryID, startOfDay, endOfDay)
}

func (r *Repo) GetInfraEventsByDate(ctx context.Context, day time.Time) ([]*infraevent.ResponseModel, error) {
	const op = "storage.repo.GetInfraEventsByDate"

	startOfDay := time.Date(day.Year(), day.Month(), day.Day(), 5, 0, 0, 0, day.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	query := selectInfraEventFields + fromInfraEventJoins +
		`WHERE e.occurred_at >= $1 AND e.occurred_at < $2
		 ORDER BY c.sort_order ASC, e.occurred_at ASC`

	return r.queryInfraEvents(ctx, op, query, startOfDay, endOfDay)
}

func (r *Repo) GetInfraEventByID(ctx context.Context, id int64) (*infraevent.ResponseModel, error) {
	const op = "storage.repo.GetInfraEventByID"

	query := selectInfraEventFields + fromInfraEventJoins + `WHERE e.id = $1`

	row := r.db.QueryRowContext(ctx, query, id)
	m, err := scanInfraEventRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	files, err := r.loadInfraEventFiles(ctx, m.ID)
	if err != nil {
		return nil, fmt.Errorf("%s: load files: %w", op, err)
	}
	if files == nil {
		files = make([]file.Model, 0)
	}
	m.Files = files

	return m, nil
}

func (r *Repo) UpdateInfraEvent(ctx context.Context, id int64, req dto.EditInfraEventRequest) error {
	const op = "storage.repo.UpdateInfraEvent"

	var updates []string
	var args []interface{}
	argID := 1

	if req.CategoryID != nil {
		updates = append(updates, fmt.Sprintf("category_id = $%d", argID))
		args = append(args, *req.CategoryID)
		argID++
	}
	if req.OrganizationID != nil {
		updates = append(updates, fmt.Sprintf("organization_id = $%d", argID))
		args = append(args, *req.OrganizationID)
		argID++
	}
	if req.OccurredAt != nil {
		updates = append(updates, fmt.Sprintf("occurred_at = $%d", argID))
		args = append(args, *req.OccurredAt)
		argID++
	}
	if req.ClearRestoredAt {
		updates = append(updates, "restored_at = NULL")
	} else if req.RestoredAt != nil {
		updates = append(updates, fmt.Sprintf("restored_at = $%d", argID))
		args = append(args, *req.RestoredAt)
		argID++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argID))
		args = append(args, *req.Description)
		argID++
	}
	if req.Remediation != nil {
		updates = append(updates, fmt.Sprintf("remediation = $%d", argID))
		args = append(args, *req.Remediation)
		argID++
	}
	if req.Notes != nil {
		updates = append(updates, fmt.Sprintf("notes = $%d", argID))
		args = append(args, *req.Notes)
		argID++
	}

	if len(updates) == 0 {
		return nil
	}

	updates = append(updates, "updated_at = NOW()")

	query := fmt.Sprintf("UPDATE sc_infra_events SET %s WHERE id = $%d",
		strings.Join(updates, ", "),
		argID,
	)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
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

func (r *Repo) DeleteInfraEvent(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteInfraEvent"

	res, err := r.db.ExecContext(ctx, "DELETE FROM sc_infra_events WHERE id = $1", id)
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

func (r *Repo) queryInfraEvents(ctx context.Context, op, query string, args ...interface{}) ([]*infraevent.ResponseModel, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var events []*infraevent.ResponseModel
	for rows.Next() {
		m, err := scanInfraEventRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		events = append(events, m)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows: %w", op, err)
	}

	if events == nil {
		events = make([]*infraevent.ResponseModel, 0)
	}

	// Batch-load files for all events
	if len(events) > 0 {
		eventIDs := make([]int64, len(events))
		for i, e := range events {
			eventIDs[i] = e.ID
		}
		filesMap, err := r.loadInfraEventFilesBatch(ctx, eventIDs)
		if err != nil {
			return nil, fmt.Errorf("%s: load files batch: %w", op, err)
		}
		for _, e := range events {
			if files, ok := filesMap[e.ID]; ok {
				e.Files = files
			} else {
				e.Files = make([]file.Model, 0)
			}
		}
	}

	return events, nil
}

// LinkInfraEventFiles links files to an infra event
func (r *Repo) LinkInfraEventFiles(ctx context.Context, eventID int64, fileIDs []int64) error {
	const op = "storage.repo.LinkInfraEventFiles"

	if len(fileIDs) == 0 {
		return nil
	}

	query := `
		INSERT INTO sc_infra_event_file_links (event_id, file_id)
		VALUES ($1, unnest($2::bigint[]))
		ON CONFLICT DO NOTHING`

	_, err := r.db.ExecContext(ctx, query, eventID, pq.Array(fileIDs))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// UnlinkInfraEventFiles removes all file links for an infra event
func (r *Repo) UnlinkInfraEventFiles(ctx context.Context, eventID int64) error {
	const op = "storage.repo.UnlinkInfraEventFiles"

	_, err := r.db.ExecContext(ctx, "DELETE FROM sc_infra_event_file_links WHERE event_id = $1", eventID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *Repo) loadInfraEventFiles(ctx context.Context, eventID int64) ([]file.Model, error) {
	const op = "storage.repo.loadInfraEventFiles"

	query := `
		SELECT f.id, f.file_name, f.object_key, f.category_id, f.mime_type, f.size_bytes, f.created_at
		FROM files f
		INNER JOIN sc_infra_event_file_links efl ON f.id = efl.file_id
		WHERE efl.event_id = $1
		ORDER BY f.created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, eventID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var files []file.Model
	for rows.Next() {
		var f file.Model
		if err := rows.Scan(&f.ID, &f.FileName, &f.ObjectKey, &f.CategoryID, &f.MimeType, &f.SizeBytes, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		files = append(files, f)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows: %w", op, err)
	}

	return files, nil
}

func (r *Repo) loadInfraEventFilesBatch(ctx context.Context, eventIDs []int64) (map[int64][]file.Model, error) {
	const op = "storage.repo.loadInfraEventFilesBatch"

	query := `
		SELECT efl.event_id, f.id, f.file_name, f.object_key, f.category_id, f.mime_type, f.size_bytes, f.created_at
		FROM files f
		INNER JOIN sc_infra_event_file_links efl ON f.id = efl.file_id
		WHERE efl.event_id = ANY($1::bigint[])
		ORDER BY efl.event_id, f.created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(eventIDs))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	result := make(map[int64][]file.Model)
	for rows.Next() {
		var eventID int64
		var f file.Model
		if err := rows.Scan(&eventID, &f.ID, &f.FileName, &f.ObjectKey, &f.CategoryID, &f.MimeType, &f.SizeBytes, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		result[eventID] = append(result[eventID], f)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows: %w", op, err)
	}

	return result, nil
}
