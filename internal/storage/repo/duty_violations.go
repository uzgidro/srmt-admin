package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"

	dvmodel "srmt-admin/internal/lib/model/duty-violations"
	"srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/storage"
)

// AddDutyViolationWithFiles inserts a new record AND its file links in
// a single transaction. If linking fails the INSERT is rolled back — no
// orphaned record without files. Returns the new id on success.
//
// Use this from the service layer instead of AddDutyViolation + Link,
// which can leave the DB in an intermediate state on the failure path.
func (r *Repo) AddDutyViolationWithFiles(ctx context.Context, req dvmodel.CreateRequest, createdByUserID int64) (int64, error) {
	const op = "storage.repo.AddDutyViolationWithFiles"

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("%s: begin tx: %w", op, err)
	}
	defer tx.Rollback()

	var id int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO duty_violations
		    (organization_id, start_time, end_time, duty_officer_name, reason, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		req.OrganizationID, req.StartTime, req.EndTime,
		req.DutyOfficerName, req.Reason, createdByUserID,
	).Scan(&id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return 0, translatedErr
		}
		return 0, fmt.Errorf("%s: insert: %w", op, err)
	}

	if len(req.FileIDs) > 0 {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO duty_violation_file_links (duty_violation_id, file_id)
			VALUES ($1, unnest($2::bigint[]))
			ON CONFLICT DO NOTHING`,
			id, pq.Array(req.FileIDs),
		); err != nil {
			if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
				return 0, translatedErr
			}
			return 0, fmt.Errorf("%s: link files: %w", op, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("%s: commit: %w", op, err)
	}
	return id, nil
}


// GetDutyViolations returns matching records grouped by organization.
// Groups are sorted by org name ASC; records within a group are newest
// first. The SQL itself orders by (o.name ASC, start_time DESC, id DESC)
// so a single linear scan is enough to build the groups in order.
//
// Files for each record are loaded with a per-row follow-up query (N+1
// by design — these reports are short-lived list views, not high-throughput).
func (r *Repo) GetDutyViolations(ctx context.Context, f dvmodel.ListFilter) ([]dvmodel.OrgGroup, error) {
	const op = "storage.repo.GetDutyViolations"

	query, args := buildDutyViolationsListQuery(f)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: query: %w", op, err)
	}
	defer rows.Close()

	// Stream-group: SQL emits rows already grouped by org (ORDER BY o.name),
	// so we just append to the trailing group while org_id is unchanged.
	groups := make([]dvmodel.OrgGroup, 0)
	for rows.Next() {
		dv, err := scanDutyViolationRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		if n := len(groups); n > 0 && groups[n-1].ID == dv.OrganizationID {
			groups[n-1].Violations = append(groups[n-1].Violations, *dv)
			continue
		}
		groups = append(groups, dvmodel.OrgGroup{
			ID:         dv.OrganizationID,
			Name:       dv.OrganizationName,
			Violations: []dvmodel.DutyViolation{*dv},
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows: %w", op, err)
	}

	// Load files per record. Indexed loop so we mutate the slice element
	// in place; ranging by value would make Files writes invisible.
	for gi := range groups {
		for vi := range groups[gi].Violations {
			files, err := r.loadDutyViolationFiles(ctx, groups[gi].Violations[vi].ID)
			if err != nil {
				return nil, fmt.Errorf("%s: load files for %d: %w",
					op, groups[gi].Violations[vi].ID, err)
			}
			groups[gi].Violations[vi].Files = files
		}
	}
	return groups, nil
}

// GetDutyViolationByID is the single-row variant used by the service after
// Create/Update to return a fully populated record (including org name and
// the resolved file list).
func (r *Repo) GetDutyViolationByID(ctx context.Context, id int64) (*dvmodel.DutyViolation, error) {
	const op = "storage.repo.GetDutyViolationByID"

	query := selectDutyViolationFields + fromDutyViolationJoins + `WHERE dv.id = $1`

	dv, err := scanDutyViolationRow(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	files, err := r.loadDutyViolationFiles(ctx, dv.ID)
	if err != nil {
		return nil, fmt.Errorf("%s: load files: %w", op, err)
	}
	dv.Files = files
	return dv, nil
}

// UpdateDutyViolationWithFiles overwrites the record AND replaces its
// file links in a single transaction. The semantics mirror the API:
// req.FileIDs is the new full list, not a delta. If any step fails the
// record's previous state is restored.
//
// Returns storage.ErrNotFound when no row matched the id.
func (r *Repo) UpdateDutyViolationWithFiles(ctx context.Context, id int64, req dvmodel.UpdateRequest) error {
	const op = "storage.repo.UpdateDutyViolationWithFiles"

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: begin tx: %w", op, err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
		UPDATE duty_violations
		SET organization_id   = $1,
		    start_time        = $2,
		    end_time          = $3,
		    duty_officer_name = $4,
		    reason            = $5
		WHERE id = $6`,
		req.OrganizationID, req.StartTime, req.EndTime,
		req.DutyOfficerName, req.Reason, id,
	)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: update: %w", op, err)
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		return storage.ErrNotFound
	}

	if _, err := tx.ExecContext(ctx,
		"DELETE FROM duty_violation_file_links WHERE duty_violation_id = $1", id,
	); err != nil {
		return fmt.Errorf("%s: unlink: %w", op, err)
	}

	if len(req.FileIDs) > 0 {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO duty_violation_file_links (duty_violation_id, file_id)
			VALUES ($1, unnest($2::bigint[]))
			ON CONFLICT DO NOTHING`,
			id, pq.Array(req.FileIDs),
		); err != nil {
			if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
				return translatedErr
			}
			return fmt.Errorf("%s: link: %w", op, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: commit: %w", op, err)
	}
	return nil
}

// DeleteDutyViolation removes one record. The junction table rows are
// removed automatically by ON DELETE CASCADE; files in the storage are
// NOT touched (they may be referenced by other records or kept for audit).
func (r *Repo) DeleteDutyViolation(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteDutyViolation"

	res, err := r.db.ExecContext(ctx, "DELETE FROM duty_violations WHERE id = $1", id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: %w", op, err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// loadDutyViolationFiles fetches the attached files for one record.
// Mirrors loadIncidentFiles — same column set, same ordering (newest
// attached first).
func (r *Repo) loadDutyViolationFiles(ctx context.Context, dvID int64) ([]file.Model, error) {
	const op = "storage.repo.loadDutyViolationFiles"

	const query = `
		SELECT f.id, f.file_name, f.object_key, f.category_id, f.mime_type, f.size_bytes, f.created_at
		FROM files f
		INNER JOIN duty_violation_file_links dvfl ON f.id = dvfl.file_id
		WHERE dvfl.duty_violation_id = $1
		ORDER BY f.created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, dvID)
	if err != nil {
		return nil, fmt.Errorf("%s: query: %w", op, err)
	}
	defer rows.Close()

	files := make([]file.Model, 0)
	for rows.Next() {
		var f file.Model
		if err := rows.Scan(&f.ID, &f.FileName, &f.ObjectKey,
			&f.CategoryID, &f.MimeType, &f.SizeBytes, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		files = append(files, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows: %w", op, err)
	}
	return files, nil
}

// --- Internal helpers (kept top-level so tests can lock the SQL shape) ---

const selectDutyViolationFields = `
	SELECT
	    dv.id,
	    dv.organization_id,
	    COALESCE(o.name, '') AS organization_name,
	    dv.start_time,
	    dv.end_time,
	    dv.duty_officer_name,
	    dv.reason,
	    dv.created_at,
	    dv.created_by_user_id,
	    dv.updated_at
`

const fromDutyViolationJoins = `
	FROM duty_violations dv
	LEFT JOIN organizations o ON o.id = dv.organization_id
`

// buildDutyViolationsListQuery composes the SQL + args for the list endpoint.
// Each filter is optional; omitted ones add neither clause nor arg. Newest
// rows first to match the typical "recent violations" view.
func buildDutyViolationsListQuery(f dvmodel.ListFilter) (string, []any) {
	q := selectDutyViolationFields + fromDutyViolationJoins
	var conds []string
	var args []any
	idx := 1
	add := func(cond string, arg any) {
		conds = append(conds, fmt.Sprintf(cond, idx))
		args = append(args, arg)
		idx++
	}
	if f.OrganizationID != nil {
		add("dv.organization_id = $%d", *f.OrganizationID)
	}
	if f.Day != nil {
		// Day is anchored at 05:00 local; the op-day window is half-open
		// [Day, Day+24h). Matches incidents/visits/shutdowns.
		start := *f.Day
		end := start.Add(24 * time.Hour)
		add("dv.start_time >= $%d", start)
		add("dv.start_time < $%d", end)
	}
	if len(conds) > 0 {
		q += "WHERE " + joinConds(conds)
	}
	// org name first → groups arrive contiguously in the row stream, so
	// GetDutyViolations can group with one linear pass. Within each org
	// newest-first matches the prior flat-list convention.
	q += " ORDER BY organization_name ASC, dv.start_time DESC, dv.id DESC"
	return q, args
}

// joinConds concatenates conditions with AND. Kept here so the SQL-shape
// tests can rebuild and inspect the query without importing strings.
func joinConds(conds []string) string {
	out := ""
	for i, c := range conds {
		if i > 0 {
			out += " AND "
		}
		out += c
	}
	return out
}

func scanDutyViolationRow(scanner interface {
	Scan(dest ...any) error
}) (*dvmodel.DutyViolation, error) {
	var dv dvmodel.DutyViolation
	var createdBy sql.NullInt64
	if err := scanner.Scan(
		&dv.ID,
		&dv.OrganizationID,
		&dv.OrganizationName,
		&dv.StartTime,
		&dv.EndTime,
		&dv.DutyOfficerName,
		&dv.Reason,
		&dv.CreatedAt,
		&createdBy,
		&dv.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if createdBy.Valid {
		dv.CreatedByUserID = &createdBy.Int64
	}
	return &dv, nil
}
