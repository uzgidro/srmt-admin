package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"

	"srmt-admin/internal/lib/dto/hrm"
	hrmmodel "srmt-admin/internal/lib/model/hrm"
	"srmt-admin/internal/storage"
)

// --- Competency Categories ---

// AddCompetencyCategory creates a new competency category
func (r *Repo) AddCompetencyCategory(ctx context.Context, req hrm.AddCompetencyCategoryRequest) (int, error) {
	const op = "storage.repo.AddCompetencyCategory"

	const query = `
		INSERT INTO hrm_competency_categories (name, description, sort_order)
		VALUES ($1, $2, $3)
		RETURNING id`

	var id int
	err := r.db.QueryRowContext(ctx, query, req.Name, req.Description, req.SortOrder).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("%s: failed to insert category: %w", op, err)
	}

	return id, nil
}

// GetCompetencyCategoryByID retrieves category by ID
func (r *Repo) GetCompetencyCategoryByID(ctx context.Context, id int) (*hrmmodel.CompetencyCategory, error) {
	const op = "storage.repo.GetCompetencyCategoryByID"

	const query = `
		SELECT id, name, description, sort_order, created_at, updated_at
		FROM hrm_competency_categories
		WHERE id = $1`

	var c hrmmodel.CompetencyCategory
	var description sql.NullString
	var updatedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID, &c.Name, &description, &c.SortOrder, &c.CreatedAt, &updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get category: %w", op, err)
	}

	if description.Valid {
		c.Description = &description.String
	}
	if updatedAt.Valid {
		c.UpdatedAt = &updatedAt.Time
	}

	return &c, nil
}

// GetCompetencyCategories retrieves all categories
func (r *Repo) GetCompetencyCategories(ctx context.Context) ([]*hrmmodel.CompetencyCategory, error) {
	const op = "storage.repo.GetCompetencyCategories"

	const query = `
		SELECT id, name, description, sort_order, created_at, updated_at
		FROM hrm_competency_categories
		ORDER BY sort_order, name`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query categories: %w", op, err)
	}
	defer rows.Close()

	var categories []*hrmmodel.CompetencyCategory
	for rows.Next() {
		var c hrmmodel.CompetencyCategory
		var description sql.NullString
		var updatedAt sql.NullTime

		err := rows.Scan(&c.ID, &c.Name, &description, &c.SortOrder, &c.CreatedAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan category: %w", op, err)
		}

		if description.Valid {
			c.Description = &description.String
		}
		if updatedAt.Valid {
			c.UpdatedAt = &updatedAt.Time
		}

		categories = append(categories, &c)
	}

	if categories == nil {
		categories = make([]*hrmmodel.CompetencyCategory, 0)
	}

	return categories, nil
}

// EditCompetencyCategory updates category
func (r *Repo) EditCompetencyCategory(ctx context.Context, id int, req hrm.EditCompetencyCategoryRequest) error {
	const op = "storage.repo.EditCompetencyCategory"

	var updates []string
	var args []interface{}
	argIdx := 1

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.SortOrder != nil {
		updates = append(updates, fmt.Sprintf("sort_order = $%d", argIdx))
		args = append(args, *req.SortOrder)
		argIdx++
	}

	if len(updates) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE hrm_competency_categories SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to update category: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteCompetencyCategory deletes category
func (r *Repo) DeleteCompetencyCategory(ctx context.Context, id int) error {
	const op = "storage.repo.DeleteCompetencyCategory"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_competency_categories WHERE id = $1", id)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			return storage.ErrForeignKeyViolation
		}
		return fmt.Errorf("%s: failed to delete category: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- Competencies ---

// AddCompetency creates a new competency
func (r *Repo) AddCompetency(ctx context.Context, req hrm.AddCompetencyRequest) (int, error) {
	const op = "storage.repo.AddCompetency"

	const query = `
		INSERT INTO hrm_competencies (category_id, name, code, description, behavioral_indicators)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`

	var id int
	err := r.db.QueryRowContext(ctx, query,
		req.CategoryID, req.Name, req.Code, req.Description, req.BehavioralIndicators,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("%s: failed to insert competency: %w", op, err)
	}

	return id, nil
}

// GetCompetencyByID retrieves competency by ID
func (r *Repo) GetCompetencyByID(ctx context.Context, id int) (*hrmmodel.Competency, error) {
	const op = "storage.repo.GetCompetencyByID"

	const query = `
		SELECT id, category_id, name, code, description, behavioral_indicators, is_active, created_at, updated_at
		FROM hrm_competencies
		WHERE id = $1`

	c, err := r.scanCompetency(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get competency: %w", op, err)
	}

	return c, nil
}

// GetCompetencies retrieves competencies with filters
func (r *Repo) GetCompetencies(ctx context.Context, filter hrm.CompetencyFilter) ([]*hrmmodel.Competency, error) {
	const op = "storage.repo.GetCompetencies"

	var query strings.Builder
	query.WriteString(`
		SELECT id, category_id, name, code, description, behavioral_indicators, is_active, created_at, updated_at
		FROM hrm_competencies
		WHERE 1=1
	`)

	args := []interface{}{}
	argIdx := 1

	if filter.CategoryID != nil {
		query.WriteString(fmt.Sprintf(" AND category_id = $%d", argIdx))
		args = append(args, *filter.CategoryID)
		argIdx++
	}
	if filter.IsActive != nil {
		query.WriteString(fmt.Sprintf(" AND is_active = $%d", argIdx))
		args = append(args, *filter.IsActive)
		argIdx++
	}
	if filter.Search != nil {
		query.WriteString(fmt.Sprintf(" AND (name ILIKE $%d OR code ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+*filter.Search+"%")
		argIdx++
	}

	query.WriteString(" ORDER BY category_id, name")

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query competencies: %w", op, err)
	}
	defer rows.Close()

	var competencies []*hrmmodel.Competency
	for rows.Next() {
		c, err := r.scanCompetencyRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan competency: %w", op, err)
		}
		competencies = append(competencies, c)
	}

	if competencies == nil {
		competencies = make([]*hrmmodel.Competency, 0)
	}

	return competencies, nil
}

// EditCompetency updates competency
func (r *Repo) EditCompetency(ctx context.Context, id int, req hrm.EditCompetencyRequest) error {
	const op = "storage.repo.EditCompetency"

	var updates []string
	var args []interface{}
	argIdx := 1

	if req.CategoryID != nil {
		updates = append(updates, fmt.Sprintf("category_id = $%d", argIdx))
		args = append(args, *req.CategoryID)
		argIdx++
	}
	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Code != nil {
		updates = append(updates, fmt.Sprintf("code = $%d", argIdx))
		args = append(args, *req.Code)
		argIdx++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.BehavioralIndicators != nil {
		updates = append(updates, fmt.Sprintf("behavioral_indicators = $%d", argIdx))
		args = append(args, *req.BehavioralIndicators)
		argIdx++
	}
	if req.IsActive != nil {
		updates = append(updates, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *req.IsActive)
		argIdx++
	}

	if len(updates) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE hrm_competencies SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to update competency: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteCompetency deletes competency
func (r *Repo) DeleteCompetency(ctx context.Context, id int) error {
	const op = "storage.repo.DeleteCompetency"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_competencies WHERE id = $1", id)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			return storage.ErrForeignKeyViolation
		}
		return fmt.Errorf("%s: failed to delete competency: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- Competency Levels ---

// AddCompetencyLevel adds level to competency
func (r *Repo) AddCompetencyLevel(ctx context.Context, req hrm.AddCompetencyLevelRequest) (int, error) {
	const op = "storage.repo.AddCompetencyLevel"

	const query = `
		INSERT INTO hrm_competency_levels (competency_id, level, name, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	var id int
	err := r.db.QueryRowContext(ctx, query,
		req.CompetencyID, req.Level, req.Name, req.Description,
	).Scan(&id)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return 0, storage.ErrUniqueViolation
		}
		return 0, fmt.Errorf("%s: failed to insert level: %w", op, err)
	}

	return id, nil
}

// GetCompetencyLevels retrieves levels for competency
func (r *Repo) GetCompetencyLevels(ctx context.Context, competencyID int) ([]*hrmmodel.CompetencyLevel, error) {
	const op = "storage.repo.GetCompetencyLevels"

	const query = `
		SELECT id, competency_id, level, name, description, created_at, updated_at
		FROM hrm_competency_levels
		WHERE competency_id = $1
		ORDER BY level`

	rows, err := r.db.QueryContext(ctx, query, competencyID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query levels: %w", op, err)
	}
	defer rows.Close()

	var levels []*hrmmodel.CompetencyLevel
	for rows.Next() {
		var l hrmmodel.CompetencyLevel
		var description sql.NullString
		var updatedAt sql.NullTime

		err := rows.Scan(&l.ID, &l.CompetencyID, &l.Level, &l.Name, &description, &l.CreatedAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan level: %w", op, err)
		}

		if description.Valid {
			l.Description = &description.String
		}
		if updatedAt.Valid {
			l.UpdatedAt = &updatedAt.Time
		}

		levels = append(levels, &l)
	}

	if levels == nil {
		levels = make([]*hrmmodel.CompetencyLevel, 0)
	}

	return levels, nil
}

// EditCompetencyLevel updates level
func (r *Repo) EditCompetencyLevel(ctx context.Context, id int, req hrm.EditCompetencyLevelRequest) error {
	const op = "storage.repo.EditCompetencyLevel"

	var updates []string
	var args []interface{}
	argIdx := 1

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}

	if len(updates) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE hrm_competency_levels SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to update level: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteCompetencyLevel deletes level
func (r *Repo) DeleteCompetencyLevel(ctx context.Context, id int) error {
	const op = "storage.repo.DeleteCompetencyLevel"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_competency_levels WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("%s: failed to delete level: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- Competency Matrix ---

// AddCompetencyMatrix adds matrix entry
func (r *Repo) AddCompetencyMatrix(ctx context.Context, req hrm.AddCompetencyMatrixRequest) (int64, error) {
	const op = "storage.repo.AddCompetencyMatrix"

	weight := req.Weight
	if weight == 0 {
		weight = 1.0
	}

	const query = `
		INSERT INTO hrm_competency_matrices (position_id, competency_id, required_level, is_mandatory, weight)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.PositionID, req.CompetencyID, req.RequiredLevel, req.IsMandatory, weight,
	).Scan(&id)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return 0, storage.ErrUniqueViolation
		}
		return 0, fmt.Errorf("%s: failed to insert matrix entry: %w", op, err)
	}

	return id, nil
}

// GetCompetencyMatrix retrieves matrix entries with filters
func (r *Repo) GetCompetencyMatrix(ctx context.Context, filter hrm.CompetencyMatrixFilter) ([]*hrmmodel.CompetencyMatrix, error) {
	const op = "storage.repo.GetCompetencyMatrix"

	var query strings.Builder
	query.WriteString(`
		SELECT id, position_id, competency_id, required_level, is_mandatory, weight, created_at, updated_at
		FROM hrm_competency_matrices
		WHERE 1=1
	`)

	args := []interface{}{}
	argIdx := 1

	if filter.PositionID != nil {
		query.WriteString(fmt.Sprintf(" AND position_id = $%d", argIdx))
		args = append(args, *filter.PositionID)
		argIdx++
	}
	if filter.CompetencyID != nil {
		query.WriteString(fmt.Sprintf(" AND competency_id = $%d", argIdx))
		args = append(args, *filter.CompetencyID)
		argIdx++
	}
	if filter.IsMandatory != nil {
		query.WriteString(fmt.Sprintf(" AND is_mandatory = $%d", argIdx))
		args = append(args, *filter.IsMandatory)
		argIdx++
	}

	query.WriteString(" ORDER BY position_id, competency_id")

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query matrix: %w", op, err)
	}
	defer rows.Close()

	var entries []*hrmmodel.CompetencyMatrix
	for rows.Next() {
		var m hrmmodel.CompetencyMatrix
		var updatedAt sql.NullTime

		err := rows.Scan(&m.ID, &m.PositionID, &m.CompetencyID, &m.RequiredLevel, &m.IsMandatory, &m.Weight, &m.CreatedAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan matrix entry: %w", op, err)
		}

		if updatedAt.Valid {
			m.UpdatedAt = &updatedAt.Time
		}

		entries = append(entries, &m)
	}

	if entries == nil {
		entries = make([]*hrmmodel.CompetencyMatrix, 0)
	}

	return entries, nil
}

// EditCompetencyMatrix updates matrix entry
func (r *Repo) EditCompetencyMatrix(ctx context.Context, id int64, req hrm.EditCompetencyMatrixRequest) error {
	const op = "storage.repo.EditCompetencyMatrix"

	var updates []string
	var args []interface{}
	argIdx := 1

	if req.RequiredLevel != nil {
		updates = append(updates, fmt.Sprintf("required_level = $%d", argIdx))
		args = append(args, *req.RequiredLevel)
		argIdx++
	}
	if req.IsMandatory != nil {
		updates = append(updates, fmt.Sprintf("is_mandatory = $%d", argIdx))
		args = append(args, *req.IsMandatory)
		argIdx++
	}
	if req.Weight != nil {
		updates = append(updates, fmt.Sprintf("weight = $%d", argIdx))
		args = append(args, *req.Weight)
		argIdx++
	}

	if len(updates) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE hrm_competency_matrices SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to update matrix entry: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteCompetencyMatrix deletes matrix entry
func (r *Repo) DeleteCompetencyMatrix(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteCompetencyMatrix"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_competency_matrices WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("%s: failed to delete matrix entry: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- Competency Assessments ---

// AddCompetencyAssessment creates a new assessment
func (r *Repo) AddCompetencyAssessment(ctx context.Context, req hrm.AddAssessmentRequest) (int64, error) {
	const op = "storage.repo.AddCompetencyAssessment"

	const query = `
		INSERT INTO hrm_competency_assessments (
			employee_id, assessment_type, assessment_period_start, assessment_period_end,
			status, assessor_id
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.EmployeeID, req.AssessmentType, req.AssessmentPeriodStart, req.AssessmentPeriodEnd,
		hrmmodel.AssessmentStatusPending, req.AssessorID,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("%s: failed to insert assessment: %w", op, err)
	}

	return id, nil
}

// GetCompetencyAssessmentByID retrieves assessment by ID
func (r *Repo) GetCompetencyAssessmentByID(ctx context.Context, id int64) (*hrmmodel.CompetencyAssessment, error) {
	const op = "storage.repo.GetCompetencyAssessmentByID"

	const query = `
		SELECT id, employee_id, assessment_type, assessment_period_start, assessment_period_end,
			status, started_at, completed_at, assessor_id, overall_score, notes, created_at, updated_at
		FROM hrm_competency_assessments
		WHERE id = $1`

	a, err := r.scanCompetencyAssessment(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get assessment: %w", op, err)
	}

	return a, nil
}

// GetCompetencyAssessments retrieves assessments with filters
func (r *Repo) GetCompetencyAssessments(ctx context.Context, filter hrm.AssessmentFilter) ([]*hrmmodel.CompetencyAssessment, error) {
	const op = "storage.repo.GetCompetencyAssessments"

	var query strings.Builder
	query.WriteString(`
		SELECT id, employee_id, assessment_type, assessment_period_start, assessment_period_end,
			status, started_at, completed_at, assessor_id, overall_score, notes, created_at, updated_at
		FROM hrm_competency_assessments
		WHERE 1=1
	`)

	args := []interface{}{}
	argIdx := 1

	if filter.EmployeeID != nil {
		query.WriteString(fmt.Sprintf(" AND employee_id = $%d", argIdx))
		args = append(args, *filter.EmployeeID)
		argIdx++
	}
	if filter.AssessorID != nil {
		query.WriteString(fmt.Sprintf(" AND assessor_id = $%d", argIdx))
		args = append(args, *filter.AssessorID)
		argIdx++
	}
	if filter.AssessmentType != nil {
		query.WriteString(fmt.Sprintf(" AND assessment_type = $%d", argIdx))
		args = append(args, *filter.AssessmentType)
		argIdx++
	}
	if filter.Status != nil {
		query.WriteString(fmt.Sprintf(" AND status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.FromDate != nil {
		query.WriteString(fmt.Sprintf(" AND assessment_period_start >= $%d", argIdx))
		args = append(args, *filter.FromDate)
		argIdx++
	}
	if filter.ToDate != nil {
		query.WriteString(fmt.Sprintf(" AND assessment_period_end <= $%d", argIdx))
		args = append(args, *filter.ToDate)
		argIdx++
	}

	query.WriteString(" ORDER BY created_at DESC")

	if filter.Limit > 0 {
		query.WriteString(fmt.Sprintf(" LIMIT $%d", argIdx))
		args = append(args, filter.Limit)
		argIdx++
	}
	if filter.Offset > 0 {
		query.WriteString(fmt.Sprintf(" OFFSET $%d", argIdx))
		args = append(args, filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query assessments: %w", op, err)
	}
	defer rows.Close()

	var assessments []*hrmmodel.CompetencyAssessment
	for rows.Next() {
		a, err := r.scanCompetencyAssessmentRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan assessment: %w", op, err)
		}
		assessments = append(assessments, a)
	}

	if assessments == nil {
		assessments = make([]*hrmmodel.CompetencyAssessment, 0)
	}

	return assessments, nil
}

// StartCompetencyAssessment starts an assessment
func (r *Repo) StartCompetencyAssessment(ctx context.Context, id int64) error {
	const op = "storage.repo.StartCompetencyAssessment"

	const query = `UPDATE hrm_competency_assessments SET status = $1, started_at = $2 WHERE id = $3`

	res, err := r.db.ExecContext(ctx, query, hrmmodel.AssessmentStatusInProgress, time.Now(), id)
	if err != nil {
		return fmt.Errorf("%s: failed to start assessment: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// CompleteCompetencyAssessment completes an assessment
func (r *Repo) CompleteCompetencyAssessment(ctx context.Context, id int64, req hrm.CompleteAssessmentRequest) error {
	const op = "storage.repo.CompleteCompetencyAssessment"

	const query = `
		UPDATE hrm_competency_assessments
		SET status = $1, completed_at = $2, overall_score = $3, notes = COALESCE($4, notes)
		WHERE id = $5`

	res, err := r.db.ExecContext(ctx, query,
		hrmmodel.AssessmentStatusCompleted, time.Now(), req.OverallScore, req.Notes, id,
	)
	if err != nil {
		return fmt.Errorf("%s: failed to complete assessment: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteCompetencyAssessment deletes assessment
func (r *Repo) DeleteCompetencyAssessment(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteCompetencyAssessment"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_competency_assessments WHERE id = $1", id)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			return storage.ErrForeignKeyViolation
		}
		return fmt.Errorf("%s: failed to delete assessment: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- Competency Scores ---

// AddCompetencyScore adds score to assessment
func (r *Repo) AddCompetencyScore(ctx context.Context, req hrm.AddScoreRequest) (int64, error) {
	const op = "storage.repo.AddCompetencyScore"

	const query = `
		INSERT INTO hrm_competency_scores (assessment_id, competency_id, score, evidence, notes)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.AssessmentID, req.CompetencyID, req.Score, req.Evidence, req.Notes,
	).Scan(&id)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return 0, storage.ErrUniqueViolation
		}
		return 0, fmt.Errorf("%s: failed to insert score: %w", op, err)
	}

	return id, nil
}

// BulkAddCompetencyScores adds multiple scores
func (r *Repo) BulkAddCompetencyScores(ctx context.Context, req hrm.BulkScoresRequest) error {
	const op = "storage.repo.BulkAddCompetencyScores"

	const query = `
		INSERT INTO hrm_competency_scores (assessment_id, competency_id, score, evidence, notes)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (assessment_id, competency_id) DO UPDATE SET score = $3, evidence = $4, notes = $5`

	for _, s := range req.Scores {
		_, err := r.db.ExecContext(ctx, query,
			req.AssessmentID, s.CompetencyID, s.Score, s.Evidence, s.Notes,
		)
		if err != nil {
			return fmt.Errorf("%s: failed to add score for competency %d: %w", op, s.CompetencyID, err)
		}
	}

	return nil
}

// GetCompetencyScores retrieves scores for assessment
func (r *Repo) GetCompetencyScores(ctx context.Context, filter hrm.ScoreFilter) ([]*hrmmodel.CompetencyScore, error) {
	const op = "storage.repo.GetCompetencyScores"

	var query strings.Builder
	query.WriteString(`
		SELECT id, assessment_id, competency_id, score, evidence, notes, created_at, updated_at
		FROM hrm_competency_scores
		WHERE 1=1
	`)

	args := []interface{}{}
	argIdx := 1

	if filter.AssessmentID != nil {
		query.WriteString(fmt.Sprintf(" AND assessment_id = $%d", argIdx))
		args = append(args, *filter.AssessmentID)
		argIdx++
	}
	if filter.CompetencyID != nil {
		query.WriteString(fmt.Sprintf(" AND competency_id = $%d", argIdx))
		args = append(args, *filter.CompetencyID)
		argIdx++
	}

	query.WriteString(" ORDER BY competency_id")

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query scores: %w", op, err)
	}
	defer rows.Close()

	var scores []*hrmmodel.CompetencyScore
	for rows.Next() {
		var s hrmmodel.CompetencyScore
		var evidence, notes sql.NullString
		var updatedAt sql.NullTime

		err := rows.Scan(&s.ID, &s.AssessmentID, &s.CompetencyID, &s.Score, &evidence, &notes, &s.CreatedAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan score: %w", op, err)
		}

		if evidence.Valid {
			s.Evidence = &evidence.String
		}
		if notes.Valid {
			s.Notes = &notes.String
		}
		if updatedAt.Valid {
			s.UpdatedAt = &updatedAt.Time
		}

		scores = append(scores, &s)
	}

	if scores == nil {
		scores = make([]*hrmmodel.CompetencyScore, 0)
	}

	return scores, nil
}

// EditCompetencyScore updates score
func (r *Repo) EditCompetencyScore(ctx context.Context, id int64, req hrm.EditScoreRequest) error {
	const op = "storage.repo.EditCompetencyScore"

	var updates []string
	var args []interface{}
	argIdx := 1

	if req.Score != nil {
		updates = append(updates, fmt.Sprintf("score = $%d", argIdx))
		args = append(args, *req.Score)
		argIdx++
	}
	if req.Evidence != nil {
		updates = append(updates, fmt.Sprintf("evidence = $%d", argIdx))
		args = append(args, *req.Evidence)
		argIdx++
	}
	if req.Notes != nil {
		updates = append(updates, fmt.Sprintf("notes = $%d", argIdx))
		args = append(args, *req.Notes)
		argIdx++
	}

	if len(updates) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE hrm_competency_scores SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to update score: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteCompetencyScore deletes score
func (r *Repo) DeleteCompetencyScore(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteCompetencyScore"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_competency_scores WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("%s: failed to delete score: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- Helpers ---

func (r *Repo) scanCompetency(row *sql.Row) (*hrmmodel.Competency, error) {
	var c hrmmodel.Competency
	var code, description, behavioralIndicators sql.NullString
	var updatedAt sql.NullTime

	err := row.Scan(
		&c.ID, &c.CategoryID, &c.Name, &code, &description, &behavioralIndicators, &c.IsActive, &c.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if code.Valid {
		c.Code = &code.String
	}
	if description.Valid {
		c.Description = &description.String
	}
	if behavioralIndicators.Valid {
		c.BehavioralIndicators = &behavioralIndicators.String
	}
	if updatedAt.Valid {
		c.UpdatedAt = &updatedAt.Time
	}

	return &c, nil
}

func (r *Repo) scanCompetencyRow(rows *sql.Rows) (*hrmmodel.Competency, error) {
	var c hrmmodel.Competency
	var code, description, behavioralIndicators sql.NullString
	var updatedAt sql.NullTime

	err := rows.Scan(
		&c.ID, &c.CategoryID, &c.Name, &code, &description, &behavioralIndicators, &c.IsActive, &c.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if code.Valid {
		c.Code = &code.String
	}
	if description.Valid {
		c.Description = &description.String
	}
	if behavioralIndicators.Valid {
		c.BehavioralIndicators = &behavioralIndicators.String
	}
	if updatedAt.Valid {
		c.UpdatedAt = &updatedAt.Time
	}

	return &c, nil
}

func (r *Repo) scanCompetencyAssessment(row *sql.Row) (*hrmmodel.CompetencyAssessment, error) {
	var a hrmmodel.CompetencyAssessment
	var startedAt, completedAt, updatedAt sql.NullTime
	var assessorID sql.NullInt64
	var overallScore sql.NullFloat64
	var notes sql.NullString

	err := row.Scan(
		&a.ID, &a.EmployeeID, &a.AssessmentType, &a.AssessmentPeriodStart, &a.AssessmentPeriodEnd,
		&a.Status, &startedAt, &completedAt, &assessorID, &overallScore, &notes, &a.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if startedAt.Valid {
		a.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		a.CompletedAt = &completedAt.Time
	}
	if assessorID.Valid {
		a.AssessorID = &assessorID.Int64
	}
	if overallScore.Valid {
		a.OverallScore = &overallScore.Float64
	}
	if notes.Valid {
		a.Notes = &notes.String
	}
	if updatedAt.Valid {
		a.UpdatedAt = &updatedAt.Time
	}

	return &a, nil
}

func (r *Repo) scanCompetencyAssessmentRow(rows *sql.Rows) (*hrmmodel.CompetencyAssessment, error) {
	var a hrmmodel.CompetencyAssessment
	var startedAt, completedAt, updatedAt sql.NullTime
	var assessorID sql.NullInt64
	var overallScore sql.NullFloat64
	var notes sql.NullString

	err := rows.Scan(
		&a.ID, &a.EmployeeID, &a.AssessmentType, &a.AssessmentPeriodStart, &a.AssessmentPeriodEnd,
		&a.Status, &startedAt, &completedAt, &assessorID, &overallScore, &notes, &a.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if startedAt.Valid {
		a.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		a.CompletedAt = &completedAt.Time
	}
	if assessorID.Valid {
		a.AssessorID = &assessorID.Int64
	}
	if overallScore.Valid {
		a.OverallScore = &overallScore.Float64
	}
	if notes.Valid {
		a.Notes = &notes.String
	}
	if updatedAt.Valid {
		a.UpdatedAt = &updatedAt.Time
	}

	return &a, nil
}
