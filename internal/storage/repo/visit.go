package repo

import (
	"context"
	"fmt"
	"github.com/lib/pq"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/model/user"
	"srmt-admin/internal/lib/model/visit"
	"srmt-admin/internal/storage"
	"strings"
	"time"
)

func (r *Repo) AddVisit(ctx context.Context, req dto.AddVisitRequest) (int64, error) {
	const op = "storage.repo.AddVisit"

	const query = `
		INSERT INTO visits (organization_id, visit_date, description, responsible_name, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.OrganizationID,
		req.VisitDate,
		req.Description,
		req.ResponsibleName,
		req.CreatedByUserID,
	).Scan(&id)

	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return 0, translatedErr
		}
		return 0, fmt.Errorf("%s: failed to insert visit: %w", op, err)
	}

	return id, nil
}

func (r *Repo) GetVisits(ctx context.Context, day time.Time) ([]*visit.ResponseModel, error) {
	const op = "storage.repo.GetVisits"

	// Create date range for the full day in the provided timezone
	// This handles timezone conversion properly
	startOfDay := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	query := selectVisitFields + fromVisitJoins +
		`WHERE v.visit_date >= $1 AND v.visit_date < $2
		 ORDER BY v.visit_date ASC`

	rows, err := r.db.QueryContext(ctx, query, startOfDay, endOfDay)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query visits: %w", op, err)
	}
	defer rows.Close()

	var visits []*visit.ResponseModel
	for rows.Next() {
		m, err := scanVisitRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan visit row: %w", op, err)
		}
		visits = append(visits, m)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if visits == nil {
		visits = make([]*visit.ResponseModel, 0)
	}

	// Load files for each visit
	for _, v := range visits {
		files, err := r.loadVisitFiles(ctx, v.ID)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to load files for visit %d: %w", op, v.ID, err)
		}
		v.Files = files
	}

	return visits, nil
}

func (r *Repo) EditVisit(ctx context.Context, id int64, req dto.EditVisitRequest) error {
	const op = "storage.repo.EditVisit"

	var updates []string
	var args []interface{}
	argID := 1

	if req.OrganizationID != nil {
		updates = append(updates, fmt.Sprintf("organization_id = $%d", argID))
		args = append(args, *req.OrganizationID)
		argID++
	}
	if req.VisitDate != nil {
		updates = append(updates, fmt.Sprintf("visit_date = $%d", argID))
		args = append(args, *req.VisitDate)
		argID++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argID))
		args = append(args, *req.Description)
		argID++
	}
	if req.ResponsibleName != nil {
		updates = append(updates, fmt.Sprintf("responsible_name = $%d", argID))
		args = append(args, *req.ResponsibleName)
		argID++
	}

	if len(updates) == 0 {
		return nil // Nothing to update
	}

	updates = append(updates, "updated_at = NOW()")

	query := fmt.Sprintf("UPDATE visits SET %s WHERE id = $%d",
		strings.Join(updates, ", "),
		argID,
	)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to update visit: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

func (r *Repo) DeleteVisit(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteVisit"

	res, err := r.db.ExecContext(ctx, "DELETE FROM visits WHERE id = $1", id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to delete visit: %w", op, err)
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
	selectVisitFields = `
		SELECT
			v.id,
			v.organization_id,
			COALESCE(o.name, '') as org_name,
			v.visit_date,
			v.description,
			v.responsible_name,
			v.created_at,
			v.created_by_user_id,
			COALESCE(c.fio, '') as user_fio
	`
	fromVisitJoins = `
		FROM
			visits v
		LEFT JOIN
			organizations o ON v.organization_id = o.id
		LEFT JOIN
			users u ON v.created_by_user_id = u.id
		LEFT JOIN
			contacts c ON u.contact_id = c.id
	`
)

func scanVisitRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*visit.ResponseModel, error) {
	var m visit.ResponseModel
	var createdByUserID int64
	var createdByUserFIO string

	err := scanner.Scan(
		&m.ID,
		&m.OrganizationID,
		&m.OrganizationName,
		&m.VisitDate,
		&m.Description,
		&m.ResponsibleName,
		&m.CreatedAt,
		&createdByUserID,
		&createdByUserFIO,
	)
	if err != nil {
		return nil, err
	}

	m.CreatedByUser = &user.ShortInfo{
		ID:   createdByUserID,
		Name: &createdByUserFIO,
	}

	return &m, nil
}

// LinkVisitFiles links files to a visit
func (r *Repo) LinkVisitFiles(ctx context.Context, visitID int64, fileIDs []int64) error {
	const op = "storage.repo.visit.LinkVisitFiles"

	if len(fileIDs) == 0 {
		return nil
	}

	query := `
		INSERT INTO visit_file_links (visit_id, file_id)
		VALUES ($1, unnest($2::bigint[]))
		ON CONFLICT DO NOTHING
	`

	_, err := r.db.ExecContext(ctx, query, visitID, pq.Array(fileIDs))
	if err != nil {
		return fmt.Errorf("%s: failed to link files: %w", op, err)
	}

	return nil
}

// UnlinkVisitFiles removes all file links for a visit
func (r *Repo) UnlinkVisitFiles(ctx context.Context, visitID int64) error {
	const op = "storage.repo.visit.UnlinkVisitFiles"

	query := `DELETE FROM visit_file_links WHERE visit_id = $1`
	_, err := r.db.ExecContext(ctx, query, visitID)
	if err != nil {
		return fmt.Errorf("%s: failed to unlink files: %w", op, err)
	}

	return nil
}

// loadVisitFiles loads files for a visit
func (r *Repo) loadVisitFiles(ctx context.Context, visitID int64) ([]file.Model, error) {
	const op = "storage.repo.visit.loadVisitFiles"

	query := `
		SELECT f.id, f.file_name, f.object_key, f.category_id, f.mime_type, f.size_bytes, f.created_at
		FROM files f
		INNER JOIN visit_file_links vfl ON f.id = vfl.file_id
		WHERE vfl.visit_id = $1
		ORDER BY f.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, visitID)
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
