package repo

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/model/incident" // (Импорт ResponseModel)
	"srmt-admin/internal/lib/model/user"
	"srmt-admin/internal/storage"
	"strings"
	"time"
)

func (r *Repo) AddIncident(ctx context.Context, orgID *int64, incidentTime time.Time, description string, createdByID int64) (int64, error) {
	const op = "storage.repo.AddIncident"

	const query = `
		INSERT INTO incidents (organization_id, incident_time, description, created_by_user_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query, orgID, incidentTime, description, createdByID).Scan(&id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return 0, translatedErr
		}
		return 0, fmt.Errorf("%s: failed to insert incident: %w", op, err)
	}

	return id, nil
}

func (r *Repo) GetIncidents(ctx context.Context, day time.Time) ([]*incident.ResponseModel, error) {
	const op = "storage.repo.GetIncidents"

	// Create date range for the full day in the provided timezone
	// This handles timezone conversion properly
	startOfDay := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	query := selectIncidentFields + fromIncidentJoins +
		`WHERE i.incident_time >= $1 AND i.incident_time < $2
		 ORDER BY i.incident_time ASC`

	rows, err := r.db.QueryContext(ctx, query, startOfDay, endOfDay)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query incidents: %w", op, err)
	}
	defer rows.Close()

	var incidents []*incident.ResponseModel
	for rows.Next() {
		m, err := scanIncidentRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan incident row: %w", op, err)
		}
		incidents = append(incidents, m)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if incidents == nil {
		incidents = make([]*incident.ResponseModel, 0)
	}

	// Load files for each incident
	for _, inc := range incidents {
		files, err := r.loadIncidentFiles(ctx, inc.ID)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to load files for incident %d: %w", op, inc.ID, err)
		}
		inc.Files = files
	}

	return incidents, nil
}

func (r *Repo) EditIncident(ctx context.Context, id int64, req dto.EditIncidentRequest) error {
	const op = "storage.repo.EditIncident"

	var updates []string
	var args []interface{}
	argID := 1

	if req.OrganizationID != nil {
		updates = append(updates, fmt.Sprintf("organization_id = $%d", argID))
		args = append(args, *req.OrganizationID)
		argID++
	}
	if req.IncidentTime != nil {
		updates = append(updates, fmt.Sprintf("incident_time = $%d", argID))
		args = append(args, *req.IncidentTime)
		argID++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argID))
		args = append(args, *req.Description)
		argID++
	}

	if len(updates) == 0 {
		return nil // Нечего обновлять
	}

	updates = append(updates, "updated_at = NOW()")

	query := fmt.Sprintf("UPDATE incidents SET %s WHERE id = $%d",
		strings.Join(updates, ", "),
		argID,
	)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to update incident: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

func (r *Repo) DeleteIncident(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteIncident"

	res, err := r.db.ExecContext(ctx, "DELETE FROM incidents WHERE id = $1", id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to delete incident: %w", op, err)
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

const (
	selectIncidentFields = `
		SELECT
			i.id,
			i.incident_time,
			i.description,
			i.created_at,
			i.organization_id,
			COALESCE(o.name, '') as org_name, -- (COALESCE на случай, если org удален)
			i.created_by_user_id,
			COALESCE(c.fio, '') as user_fio -- (COALESCE на случай, если user/contact удален)
	`
	fromIncidentJoins = `
		FROM
			incidents i
		LEFT JOIN
			organizations o ON i.organization_id = o.id
		LEFT JOIN
			users u ON i.created_by_user_id = u.id
		LEFT JOIN
			contacts c ON u.contact_id = c.id
	`
)

func scanIncidentRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*incident.ResponseModel, error) {
	var m incident.ResponseModel
	var desc sql.NullString    // Для nullable description
	var orgID sql.NullInt64    // Для nullable organization_id
	var orgName sql.NullString // Для nullable organization name
	var createdByUserID int64
	var createdByUserFIO string

	err := scanner.Scan(
		&m.ID,
		&m.IncidentTime,
		&desc,
		&m.CreatedAt,
		&orgID,
		&orgName,
		&createdByUserID,
		&createdByUserFIO,
	)
	if err != nil {
		return nil, err
	}

	if desc.Valid {
		m.Description = desc.String
	}
	if orgID.Valid {
		m.OrganizationID = &orgID.Int64
	}
	if orgName.Valid {
		m.OrganizationName = &orgName.String
	}

	m.CreatedByUser = &user.ShortInfo{
		ID:   createdByUserID,
		Name: &createdByUserFIO,
	}

	return &m, nil
}

// LinkIncidentFiles links files to an incident
func (r *Repo) LinkIncidentFiles(ctx context.Context, incidentID int64, fileIDs []int64) error {
	const op = "storage.repo.incident.LinkIncidentFiles"

	if len(fileIDs) == 0 {
		return nil
	}

	query := `
		INSERT INTO incident_file_links (incident_id, file_id)
		VALUES ($1, unnest($2::bigint[]))
		ON CONFLICT DO NOTHING
	`

	_, err := r.db.ExecContext(ctx, query, incidentID, pq.Array(fileIDs))
	if err != nil {
		return fmt.Errorf("%s: failed to link files: %w", op, err)
	}

	return nil
}

// UnlinkIncidentFiles removes all file links for an incident
func (r *Repo) UnlinkIncidentFiles(ctx context.Context, incidentID int64) error {
	const op = "storage.repo.incident.UnlinkIncidentFiles"

	query := `DELETE FROM incident_file_links WHERE incident_id = $1`
	_, err := r.db.ExecContext(ctx, query, incidentID)
	if err != nil {
		return fmt.Errorf("%s: failed to unlink files: %w", op, err)
	}

	return nil
}

// loadIncidentFiles loads files for an incident
func (r *Repo) loadIncidentFiles(ctx context.Context, incidentID int64) ([]file.Model, error) {
	const op = "storage.repo.incident.loadIncidentFiles"

	query := `
		SELECT f.id, f.file_name, f.object_key, f.category_id, f.mime_type, f.size_bytes, f.created_at
		FROM files f
		INNER JOIN incident_file_links ifl ON f.id = ifl.file_id
		WHERE ifl.incident_id = $1
		ORDER BY f.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, incidentID)
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
