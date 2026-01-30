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

// --- Performance Reviews ---

// AddPerformanceReview creates a new performance review
func (r *Repo) AddPerformanceReview(ctx context.Context, req hrm.AddPerformanceReviewRequest) (int64, error) {
	const op = "storage.repo.AddPerformanceReview"

	const query = `
		INSERT INTO hrm_performance_reviews (
			employee_id, review_type, review_period_start, review_period_end,
			status, reviewer_id, self_review_deadline, manager_review_deadline
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.EmployeeID, req.ReviewType, req.ReviewPeriodStart, req.ReviewPeriodEnd,
		hrmmodel.ReviewStatusPending, req.ReviewerID, req.SelfReviewDeadline, req.ManagerReviewDeadline,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("%s: failed to insert review: %w", op, err)
	}

	return id, nil
}

// GetPerformanceReviewByID retrieves performance review by ID
func (r *Repo) GetPerformanceReviewByID(ctx context.Context, id int64) (*hrmmodel.PerformanceReview, error) {
	const op = "storage.repo.GetPerformanceReviewByID"

	const query = `
		SELECT id, employee_id, review_type, review_period_start, review_period_end,
			status, self_review_deadline, manager_review_deadline,
			self_review_started_at, self_review_completed_at, self_assessment, self_rating,
			reviewer_id, manager_review_started_at, manager_review_completed_at, manager_assessment, manager_rating,
			final_rating, final_rating_label, calibrated_by, calibrated_at,
			achievements, areas_for_improvement, development_recommendations,
			completed_at, notes, created_at, updated_at
		FROM hrm_performance_reviews
		WHERE id = $1`

	rev, err := r.scanPerformanceReview(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get review: %w", op, err)
	}

	return rev, nil
}

// GetPerformanceReviews retrieves reviews with filters
func (r *Repo) GetPerformanceReviews(ctx context.Context, filter hrm.PerformanceReviewFilter) ([]*hrmmodel.PerformanceReview, error) {
	const op = "storage.repo.GetPerformanceReviews"

	var query strings.Builder
	query.WriteString(`
		SELECT id, employee_id, review_type, review_period_start, review_period_end,
			status, self_review_deadline, manager_review_deadline,
			self_review_started_at, self_review_completed_at, self_assessment, self_rating,
			reviewer_id, manager_review_started_at, manager_review_completed_at, manager_assessment, manager_rating,
			final_rating, final_rating_label, calibrated_by, calibrated_at,
			achievements, areas_for_improvement, development_recommendations,
			completed_at, notes, created_at, updated_at
		FROM hrm_performance_reviews
		WHERE 1=1
	`)

	args := []interface{}{}
	argIdx := 1

	if filter.EmployeeID != nil {
		query.WriteString(fmt.Sprintf(" AND employee_id = $%d", argIdx))
		args = append(args, *filter.EmployeeID)
		argIdx++
	}
	if filter.ReviewerID != nil {
		query.WriteString(fmt.Sprintf(" AND reviewer_id = $%d", argIdx))
		args = append(args, *filter.ReviewerID)
		argIdx++
	}
	if filter.ReviewType != nil {
		query.WriteString(fmt.Sprintf(" AND review_type = $%d", argIdx))
		args = append(args, *filter.ReviewType)
		argIdx++
	}
	if filter.Status != nil {
		query.WriteString(fmt.Sprintf(" AND status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.FromDate != nil {
		query.WriteString(fmt.Sprintf(" AND review_period_start >= $%d", argIdx))
		args = append(args, *filter.FromDate)
		argIdx++
	}
	if filter.ToDate != nil {
		query.WriteString(fmt.Sprintf(" AND review_period_end <= $%d", argIdx))
		args = append(args, *filter.ToDate)
		argIdx++
	}

	query.WriteString(" ORDER BY review_period_start DESC")

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
		return nil, fmt.Errorf("%s: failed to query reviews: %w", op, err)
	}
	defer rows.Close()

	var reviews []*hrmmodel.PerformanceReview
	for rows.Next() {
		rev, err := r.scanPerformanceReviewRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan review: %w", op, err)
		}
		reviews = append(reviews, rev)
	}

	if reviews == nil {
		reviews = make([]*hrmmodel.PerformanceReview, 0)
	}

	return reviews, nil
}

// EditPerformanceReview updates performance review
func (r *Repo) EditPerformanceReview(ctx context.Context, id int64, req hrm.EditPerformanceReviewRequest) error {
	const op = "storage.repo.EditPerformanceReview"

	var updates []string
	var args []interface{}
	argIdx := 1

	if req.ReviewType != nil {
		updates = append(updates, fmt.Sprintf("review_type = $%d", argIdx))
		args = append(args, *req.ReviewType)
		argIdx++
	}
	if req.ReviewPeriodStart != nil {
		updates = append(updates, fmt.Sprintf("review_period_start = $%d", argIdx))
		args = append(args, *req.ReviewPeriodStart)
		argIdx++
	}
	if req.ReviewPeriodEnd != nil {
		updates = append(updates, fmt.Sprintf("review_period_end = $%d", argIdx))
		args = append(args, *req.ReviewPeriodEnd)
		argIdx++
	}
	if req.ReviewerID != nil {
		updates = append(updates, fmt.Sprintf("reviewer_id = $%d", argIdx))
		args = append(args, *req.ReviewerID)
		argIdx++
	}
	if req.SelfReviewDeadline != nil {
		updates = append(updates, fmt.Sprintf("self_review_deadline = $%d", argIdx))
		args = append(args, *req.SelfReviewDeadline)
		argIdx++
	}
	if req.ManagerReviewDeadline != nil {
		updates = append(updates, fmt.Sprintf("manager_review_deadline = $%d", argIdx))
		args = append(args, *req.ManagerReviewDeadline)
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

	query := fmt.Sprintf("UPDATE hrm_performance_reviews SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to update review: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// SubmitSelfReview submits employee self-review
func (r *Repo) SubmitSelfReview(ctx context.Context, id int64, req hrm.SubmitSelfReviewRequest) error {
	const op = "storage.repo.SubmitSelfReview"

	const query = `
		UPDATE hrm_performance_reviews
		SET status = $1, self_review_completed_at = $2, self_assessment = $3, self_rating = $4,
			self_review_started_at = COALESCE(self_review_started_at, $2)
		WHERE id = $5`

	res, err := r.db.ExecContext(ctx, query,
		hrmmodel.ReviewStatusManagerReview, time.Now(), req.SelfAssessment, req.SelfRating, id,
	)
	if err != nil {
		return fmt.Errorf("%s: failed to submit self-review: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// SubmitManagerReview submits manager review
func (r *Repo) SubmitManagerReview(ctx context.Context, id int64, req hrm.SubmitManagerReviewRequest) error {
	const op = "storage.repo.SubmitManagerReview"

	const query = `
		UPDATE hrm_performance_reviews
		SET status = $1, manager_review_completed_at = $2, manager_assessment = $3, manager_rating = $4,
			achievements = $5, areas_for_improvement = $6, development_recommendations = $7,
			manager_review_started_at = COALESCE(manager_review_started_at, $2)
		WHERE id = $8`

	res, err := r.db.ExecContext(ctx, query,
		hrmmodel.ReviewStatusCalibration, time.Now(), req.ManagerAssessment, req.ManagerRating,
		req.Achievements, req.AreasForImprovement, req.DevelopmentRecommendations, id,
	)
	if err != nil {
		return fmt.Errorf("%s: failed to submit manager review: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// CalibrateReview calibrates final rating
func (r *Repo) CalibrateReview(ctx context.Context, id int64, calibratorID int64, req hrm.CalibrateReviewRequest) error {
	const op = "storage.repo.CalibrateReview"

	label := req.FinalRatingLabel
	if label == nil {
		l := hrmmodel.GetRatingLabel(req.FinalRating)
		label = &l
	}

	const query = `
		UPDATE hrm_performance_reviews
		SET status = $1, completed_at = $2, final_rating = $3, final_rating_label = $4,
			calibrated_by = $5, calibrated_at = $2, notes = COALESCE($6, notes)
		WHERE id = $7`

	res, err := r.db.ExecContext(ctx, query,
		hrmmodel.ReviewStatusCompleted, time.Now(), req.FinalRating, label,
		calibratorID, req.Notes, id,
	)
	if err != nil {
		return fmt.Errorf("%s: failed to calibrate review: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeletePerformanceReview deletes performance review
func (r *Repo) DeletePerformanceReview(ctx context.Context, id int64) error {
	const op = "storage.repo.DeletePerformanceReview"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_performance_reviews WHERE id = $1", id)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			return storage.ErrForeignKeyViolation
		}
		return fmt.Errorf("%s: failed to delete review: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- Performance Goals ---

// AddPerformanceGoal creates a new performance goal
func (r *Repo) AddPerformanceGoal(ctx context.Context, req hrm.AddPerformanceGoalRequest) (int64, error) {
	const op = "storage.repo.AddPerformanceGoal"

	weight := req.Weight
	if weight == 0 {
		weight = 1.0
	}

	const query = `
		INSERT INTO hrm_performance_goals (
			employee_id, review_id, title, description, category,
			success_criteria, metrics, aligned_to, weight,
			start_date, target_date, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.EmployeeID, req.ReviewID, req.Title, req.Description, req.Category,
		req.SuccessCriteria, req.Metrics, req.AlignedTo, weight,
		req.StartDate, req.TargetDate, hrmmodel.GoalStatusNotStarted,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("%s: failed to insert goal: %w", op, err)
	}

	return id, nil
}

// GetPerformanceGoalByID retrieves goal by ID
func (r *Repo) GetPerformanceGoalByID(ctx context.Context, id int64) (*hrmmodel.PerformanceGoal, error) {
	const op = "storage.repo.GetPerformanceGoalByID"

	const query = `
		SELECT id, employee_id, review_id, title, description, category,
			success_criteria, metrics, aligned_to, weight,
			start_date, target_date, status, progress, completed_at,
			self_rating, self_comments, manager_rating, manager_comments,
			notes, created_at, updated_at
		FROM hrm_performance_goals
		WHERE id = $1`

	g, err := r.scanPerformanceGoal(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get goal: %w", op, err)
	}

	return g, nil
}

// GetPerformanceGoals retrieves goals with filters
func (r *Repo) GetPerformanceGoals(ctx context.Context, filter hrm.PerformanceGoalFilter) ([]*hrmmodel.PerformanceGoal, error) {
	const op = "storage.repo.GetPerformanceGoals"

	var query strings.Builder
	query.WriteString(`
		SELECT id, employee_id, review_id, title, description, category,
			success_criteria, metrics, aligned_to, weight,
			start_date, target_date, status, progress, completed_at,
			self_rating, self_comments, manager_rating, manager_comments,
			notes, created_at, updated_at
		FROM hrm_performance_goals
		WHERE 1=1
	`)

	args := []interface{}{}
	argIdx := 1

	if filter.EmployeeID != nil {
		query.WriteString(fmt.Sprintf(" AND employee_id = $%d", argIdx))
		args = append(args, *filter.EmployeeID)
		argIdx++
	}
	if filter.ReviewID != nil {
		query.WriteString(fmt.Sprintf(" AND review_id = $%d", argIdx))
		args = append(args, *filter.ReviewID)
		argIdx++
	}
	if filter.Status != nil {
		query.WriteString(fmt.Sprintf(" AND status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.Category != nil {
		query.WriteString(fmt.Sprintf(" AND category = $%d", argIdx))
		args = append(args, *filter.Category)
		argIdx++
	}
	if filter.FromDate != nil {
		query.WriteString(fmt.Sprintf(" AND start_date >= $%d", argIdx))
		args = append(args, *filter.FromDate)
		argIdx++
	}
	if filter.ToDate != nil {
		query.WriteString(fmt.Sprintf(" AND target_date <= $%d", argIdx))
		args = append(args, *filter.ToDate)
		argIdx++
	}

	query.WriteString(" ORDER BY target_date ASC NULLS LAST")

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
		return nil, fmt.Errorf("%s: failed to query goals: %w", op, err)
	}
	defer rows.Close()

	var goals []*hrmmodel.PerformanceGoal
	for rows.Next() {
		g, err := r.scanPerformanceGoalRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan goal: %w", op, err)
		}
		goals = append(goals, g)
	}

	if goals == nil {
		goals = make([]*hrmmodel.PerformanceGoal, 0)
	}

	return goals, nil
}

// EditPerformanceGoal updates goal
func (r *Repo) EditPerformanceGoal(ctx context.Context, id int64, req hrm.EditPerformanceGoalRequest) error {
	const op = "storage.repo.EditPerformanceGoal"

	var updates []string
	var args []interface{}
	argIdx := 1

	if req.Title != nil {
		updates = append(updates, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *req.Title)
		argIdx++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.Category != nil {
		updates = append(updates, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, *req.Category)
		argIdx++
	}
	if req.SuccessCriteria != nil {
		updates = append(updates, fmt.Sprintf("success_criteria = $%d", argIdx))
		args = append(args, *req.SuccessCriteria)
		argIdx++
	}
	if req.Metrics != nil {
		updates = append(updates, fmt.Sprintf("metrics = $%d", argIdx))
		args = append(args, *req.Metrics)
		argIdx++
	}
	if req.AlignedTo != nil {
		updates = append(updates, fmt.Sprintf("aligned_to = $%d", argIdx))
		args = append(args, *req.AlignedTo)
		argIdx++
	}
	if req.Weight != nil {
		updates = append(updates, fmt.Sprintf("weight = $%d", argIdx))
		args = append(args, *req.Weight)
		argIdx++
	}
	if req.StartDate != nil {
		updates = append(updates, fmt.Sprintf("start_date = $%d", argIdx))
		args = append(args, *req.StartDate)
		argIdx++
	}
	if req.TargetDate != nil {
		updates = append(updates, fmt.Sprintf("target_date = $%d", argIdx))
		args = append(args, *req.TargetDate)
		argIdx++
	}
	if req.Status != nil {
		updates = append(updates, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
		if *req.Status == hrmmodel.GoalStatusCompleted {
			updates = append(updates, fmt.Sprintf("completed_at = $%d", argIdx))
			args = append(args, time.Now())
			argIdx++
		}
	}
	if req.Progress != nil {
		updates = append(updates, fmt.Sprintf("progress = $%d", argIdx))
		args = append(args, *req.Progress)
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

	query := fmt.Sprintf("UPDATE hrm_performance_goals SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to update goal: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// UpdateGoalProgress updates goal progress
func (r *Repo) UpdateGoalProgress(ctx context.Context, id int64, req hrm.UpdateGoalProgressRequest) error {
	const op = "storage.repo.UpdateGoalProgress"

	status := hrmmodel.GoalStatusInProgress
	if req.Progress >= 100 {
		status = hrmmodel.GoalStatusCompleted
	}

	const query = `
		UPDATE hrm_performance_goals
		SET progress = $1, status = $2, notes = COALESCE($3, notes),
			completed_at = CASE WHEN $2 = 'completed' THEN NOW() ELSE completed_at END
		WHERE id = $4`

	res, err := r.db.ExecContext(ctx, query, req.Progress, status, req.Notes, id)
	if err != nil {
		return fmt.Errorf("%s: failed to update progress: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// RateGoal adds ratings to goal
func (r *Repo) RateGoal(ctx context.Context, id int64, req hrm.RateGoalRequest) error {
	const op = "storage.repo.RateGoal"

	var updates []string
	var args []interface{}
	argIdx := 1

	if req.SelfRating != nil {
		updates = append(updates, fmt.Sprintf("self_rating = $%d", argIdx))
		args = append(args, *req.SelfRating)
		argIdx++
	}
	if req.SelfComments != nil {
		updates = append(updates, fmt.Sprintf("self_comments = $%d", argIdx))
		args = append(args, *req.SelfComments)
		argIdx++
	}
	if req.ManagerRating != nil {
		updates = append(updates, fmt.Sprintf("manager_rating = $%d", argIdx))
		args = append(args, *req.ManagerRating)
		argIdx++
	}
	if req.ManagerComments != nil {
		updates = append(updates, fmt.Sprintf("manager_comments = $%d", argIdx))
		args = append(args, *req.ManagerComments)
		argIdx++
	}

	if len(updates) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE hrm_performance_goals SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to rate goal: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeletePerformanceGoal deletes goal
func (r *Repo) DeletePerformanceGoal(ctx context.Context, id int64) error {
	const op = "storage.repo.DeletePerformanceGoal"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_performance_goals WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("%s: failed to delete goal: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- KPIs ---

// AddKPI creates a new KPI
func (r *Repo) AddKPI(ctx context.Context, req hrm.AddKPIRequest) (int64, error) {
	const op = "storage.repo.AddKPI"

	weight := req.Weight
	if weight == 0 {
		weight = 1.0
	}

	const query = `
		INSERT INTO hrm_kpis (
			employee_id, name, description, category,
			measurement_unit, target_value, min_threshold, max_threshold,
			year, month, quarter, weight
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.EmployeeID, req.Name, req.Description, req.Category,
		req.MeasurementUnit, req.TargetValue, req.MinThreshold, req.MaxThreshold,
		req.Year, req.Month, req.Quarter, weight,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("%s: failed to insert KPI: %w", op, err)
	}

	return id, nil
}

// GetKPIByID retrieves KPI by ID
func (r *Repo) GetKPIByID(ctx context.Context, id int64) (*hrmmodel.KPI, error) {
	const op = "storage.repo.GetKPIByID"

	const query = `
		SELECT id, employee_id, name, description, category,
			measurement_unit, target_value, min_threshold, max_threshold,
			year, month, quarter, actual_value, achievement_percent,
			rating, weight, notes, created_at, updated_at
		FROM hrm_kpis
		WHERE id = $1`

	k, err := r.scanKPI(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get KPI: %w", op, err)
	}

	return k, nil
}

// GetKPIs retrieves KPIs with filters
func (r *Repo) GetKPIs(ctx context.Context, filter hrm.KPIFilter) ([]*hrmmodel.KPI, error) {
	const op = "storage.repo.GetKPIs"

	var query strings.Builder
	query.WriteString(`
		SELECT id, employee_id, name, description, category,
			measurement_unit, target_value, min_threshold, max_threshold,
			year, month, quarter, actual_value, achievement_percent,
			rating, weight, notes, created_at, updated_at
		FROM hrm_kpis
		WHERE 1=1
	`)

	args := []interface{}{}
	argIdx := 1

	if filter.EmployeeID != nil {
		query.WriteString(fmt.Sprintf(" AND employee_id = $%d", argIdx))
		args = append(args, *filter.EmployeeID)
		argIdx++
	}
	if filter.Year != nil {
		query.WriteString(fmt.Sprintf(" AND year = $%d", argIdx))
		args = append(args, *filter.Year)
		argIdx++
	}
	if filter.Month != nil {
		query.WriteString(fmt.Sprintf(" AND month = $%d", argIdx))
		args = append(args, *filter.Month)
		argIdx++
	}
	if filter.Quarter != nil {
		query.WriteString(fmt.Sprintf(" AND quarter = $%d", argIdx))
		args = append(args, *filter.Quarter)
		argIdx++
	}
	if filter.Category != nil {
		query.WriteString(fmt.Sprintf(" AND category = $%d", argIdx))
		args = append(args, *filter.Category)
		argIdx++
	}

	query.WriteString(" ORDER BY year DESC, month DESC NULLS LAST, quarter DESC NULLS LAST")

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
		return nil, fmt.Errorf("%s: failed to query KPIs: %w", op, err)
	}
	defer rows.Close()

	var kpis []*hrmmodel.KPI
	for rows.Next() {
		k, err := r.scanKPIRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan KPI: %w", op, err)
		}
		kpis = append(kpis, k)
	}

	if kpis == nil {
		kpis = make([]*hrmmodel.KPI, 0)
	}

	return kpis, nil
}

// EditKPI updates KPI
func (r *Repo) EditKPI(ctx context.Context, id int64, req hrm.EditKPIRequest) error {
	const op = "storage.repo.EditKPI"

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
	if req.Category != nil {
		updates = append(updates, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, *req.Category)
		argIdx++
	}
	if req.MeasurementUnit != nil {
		updates = append(updates, fmt.Sprintf("measurement_unit = $%d", argIdx))
		args = append(args, *req.MeasurementUnit)
		argIdx++
	}
	if req.TargetValue != nil {
		updates = append(updates, fmt.Sprintf("target_value = $%d", argIdx))
		args = append(args, *req.TargetValue)
		argIdx++
	}
	if req.MinThreshold != nil {
		updates = append(updates, fmt.Sprintf("min_threshold = $%d", argIdx))
		args = append(args, *req.MinThreshold)
		argIdx++
	}
	if req.MaxThreshold != nil {
		updates = append(updates, fmt.Sprintf("max_threshold = $%d", argIdx))
		args = append(args, *req.MaxThreshold)
		argIdx++
	}
	if req.Weight != nil {
		updates = append(updates, fmt.Sprintf("weight = $%d", argIdx))
		args = append(args, *req.Weight)
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

	query := fmt.Sprintf("UPDATE hrm_kpis SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to update KPI: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// UpdateKPIValue updates KPI actual value and calculates achievement
func (r *Repo) UpdateKPIValue(ctx context.Context, id int64, req hrm.UpdateKPIValueRequest) error {
	const op = "storage.repo.UpdateKPIValue"

	// First get target value to calculate achievement
	var targetValue float64
	err := r.db.QueryRowContext(ctx, "SELECT target_value FROM hrm_kpis WHERE id = $1", id).Scan(&targetValue)
	if err != nil {
		if err == sql.ErrNoRows {
			return storage.ErrNotFound
		}
		return fmt.Errorf("%s: failed to get KPI: %w", op, err)
	}

	var achievementPercent float64
	if targetValue > 0 {
		achievementPercent = (req.ActualValue / targetValue) * 100
	}

	const query = `
		UPDATE hrm_kpis
		SET actual_value = $1, achievement_percent = $2, notes = COALESCE($3, notes)
		WHERE id = $4`

	res, err := r.db.ExecContext(ctx, query, req.ActualValue, achievementPercent, req.Notes, id)
	if err != nil {
		return fmt.Errorf("%s: failed to update KPI value: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// RateKPI adds rating to KPI
func (r *Repo) RateKPI(ctx context.Context, id int64, req hrm.RateKPIRequest) error {
	const op = "storage.repo.RateKPI"

	const query = `UPDATE hrm_kpis SET rating = $1, notes = COALESCE($2, notes) WHERE id = $3`

	res, err := r.db.ExecContext(ctx, query, req.Rating, req.Notes, id)
	if err != nil {
		return fmt.Errorf("%s: failed to rate KPI: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteKPI deletes KPI
func (r *Repo) DeleteKPI(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteKPI"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_kpis WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("%s: failed to delete KPI: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- Helpers ---

func (r *Repo) scanPerformanceReview(row *sql.Row) (*hrmmodel.PerformanceReview, error) {
	var rev hrmmodel.PerformanceReview
	var selfReviewDeadline, managerReviewDeadline, selfReviewStartedAt, selfReviewCompletedAt sql.NullTime
	var selfAssessment sql.NullString
	var selfRating sql.NullInt64
	var reviewerID, calibratedBy sql.NullInt64
	var managerReviewStartedAt, managerReviewCompletedAt sql.NullTime
	var managerAssessment sql.NullString
	var managerRating sql.NullInt64
	var finalRating sql.NullInt64
	var finalRatingLabel sql.NullString
	var calibratedAt, completedAt, updatedAt sql.NullTime
	var achievements, areasForImprovement, developmentRecommendations, notes sql.NullString

	err := row.Scan(
		&rev.ID, &rev.EmployeeID, &rev.ReviewType, &rev.ReviewPeriodStart, &rev.ReviewPeriodEnd,
		&rev.Status, &selfReviewDeadline, &managerReviewDeadline,
		&selfReviewStartedAt, &selfReviewCompletedAt, &selfAssessment, &selfRating,
		&reviewerID, &managerReviewStartedAt, &managerReviewCompletedAt, &managerAssessment, &managerRating,
		&finalRating, &finalRatingLabel, &calibratedBy, &calibratedAt,
		&achievements, &areasForImprovement, &developmentRecommendations,
		&completedAt, &notes, &rev.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if selfReviewDeadline.Valid {
		rev.SelfReviewDeadline = &selfReviewDeadline.Time
	}
	if managerReviewDeadline.Valid {
		rev.ManagerReviewDeadline = &managerReviewDeadline.Time
	}
	if selfReviewStartedAt.Valid {
		rev.SelfReviewStartedAt = &selfReviewStartedAt.Time
	}
	if selfReviewCompletedAt.Valid {
		rev.SelfReviewCompletedAt = &selfReviewCompletedAt.Time
	}
	if selfAssessment.Valid {
		rev.SelfAssessment = &selfAssessment.String
	}
	if selfRating.Valid {
		r := int(selfRating.Int64)
		rev.SelfRating = &r
	}
	if reviewerID.Valid {
		rev.ReviewerID = &reviewerID.Int64
	}
	if managerReviewStartedAt.Valid {
		rev.ManagerReviewStartedAt = &managerReviewStartedAt.Time
	}
	if managerReviewCompletedAt.Valid {
		rev.ManagerReviewCompletedAt = &managerReviewCompletedAt.Time
	}
	if managerAssessment.Valid {
		rev.ManagerAssessment = &managerAssessment.String
	}
	if managerRating.Valid {
		r := int(managerRating.Int64)
		rev.ManagerRating = &r
	}
	if finalRating.Valid {
		r := int(finalRating.Int64)
		rev.FinalRating = &r
	}
	if finalRatingLabel.Valid {
		rev.FinalRatingLabel = &finalRatingLabel.String
	}
	if calibratedBy.Valid {
		rev.CalibratedBy = &calibratedBy.Int64
	}
	if calibratedAt.Valid {
		rev.CalibratedAt = &calibratedAt.Time
	}
	if achievements.Valid {
		rev.Achievements = &achievements.String
	}
	if areasForImprovement.Valid {
		rev.AreasForImprovement = &areasForImprovement.String
	}
	if developmentRecommendations.Valid {
		rev.DevelopmentRecommendations = &developmentRecommendations.String
	}
	if completedAt.Valid {
		rev.CompletedAt = &completedAt.Time
	}
	if notes.Valid {
		rev.Notes = &notes.String
	}
	if updatedAt.Valid {
		rev.UpdatedAt = &updatedAt.Time
	}

	return &rev, nil
}

func (r *Repo) scanPerformanceReviewRow(rows *sql.Rows) (*hrmmodel.PerformanceReview, error) {
	var rev hrmmodel.PerformanceReview
	var selfReviewDeadline, managerReviewDeadline, selfReviewStartedAt, selfReviewCompletedAt sql.NullTime
	var selfAssessment sql.NullString
	var selfRating sql.NullInt64
	var reviewerID, calibratedBy sql.NullInt64
	var managerReviewStartedAt, managerReviewCompletedAt sql.NullTime
	var managerAssessment sql.NullString
	var managerRating sql.NullInt64
	var finalRating sql.NullInt64
	var finalRatingLabel sql.NullString
	var calibratedAt, completedAt, updatedAt sql.NullTime
	var achievements, areasForImprovement, developmentRecommendations, notes sql.NullString

	err := rows.Scan(
		&rev.ID, &rev.EmployeeID, &rev.ReviewType, &rev.ReviewPeriodStart, &rev.ReviewPeriodEnd,
		&rev.Status, &selfReviewDeadline, &managerReviewDeadline,
		&selfReviewStartedAt, &selfReviewCompletedAt, &selfAssessment, &selfRating,
		&reviewerID, &managerReviewStartedAt, &managerReviewCompletedAt, &managerAssessment, &managerRating,
		&finalRating, &finalRatingLabel, &calibratedBy, &calibratedAt,
		&achievements, &areasForImprovement, &developmentRecommendations,
		&completedAt, &notes, &rev.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if selfReviewDeadline.Valid {
		rev.SelfReviewDeadline = &selfReviewDeadline.Time
	}
	if managerReviewDeadline.Valid {
		rev.ManagerReviewDeadline = &managerReviewDeadline.Time
	}
	if selfReviewStartedAt.Valid {
		rev.SelfReviewStartedAt = &selfReviewStartedAt.Time
	}
	if selfReviewCompletedAt.Valid {
		rev.SelfReviewCompletedAt = &selfReviewCompletedAt.Time
	}
	if selfAssessment.Valid {
		rev.SelfAssessment = &selfAssessment.String
	}
	if selfRating.Valid {
		r := int(selfRating.Int64)
		rev.SelfRating = &r
	}
	if reviewerID.Valid {
		rev.ReviewerID = &reviewerID.Int64
	}
	if managerReviewStartedAt.Valid {
		rev.ManagerReviewStartedAt = &managerReviewStartedAt.Time
	}
	if managerReviewCompletedAt.Valid {
		rev.ManagerReviewCompletedAt = &managerReviewCompletedAt.Time
	}
	if managerAssessment.Valid {
		rev.ManagerAssessment = &managerAssessment.String
	}
	if managerRating.Valid {
		r := int(managerRating.Int64)
		rev.ManagerRating = &r
	}
	if finalRating.Valid {
		r := int(finalRating.Int64)
		rev.FinalRating = &r
	}
	if finalRatingLabel.Valid {
		rev.FinalRatingLabel = &finalRatingLabel.String
	}
	if calibratedBy.Valid {
		rev.CalibratedBy = &calibratedBy.Int64
	}
	if calibratedAt.Valid {
		rev.CalibratedAt = &calibratedAt.Time
	}
	if achievements.Valid {
		rev.Achievements = &achievements.String
	}
	if areasForImprovement.Valid {
		rev.AreasForImprovement = &areasForImprovement.String
	}
	if developmentRecommendations.Valid {
		rev.DevelopmentRecommendations = &developmentRecommendations.String
	}
	if completedAt.Valid {
		rev.CompletedAt = &completedAt.Time
	}
	if notes.Valid {
		rev.Notes = &notes.String
	}
	if updatedAt.Valid {
		rev.UpdatedAt = &updatedAt.Time
	}

	return &rev, nil
}

func (r *Repo) scanPerformanceGoal(row *sql.Row) (*hrmmodel.PerformanceGoal, error) {
	var g hrmmodel.PerformanceGoal
	var reviewID sql.NullInt64
	var description, category, successCriteria, metrics, alignedTo sql.NullString
	var startDate, targetDate, completedAt, updatedAt sql.NullTime
	var selfRating, managerRating sql.NullInt64
	var selfComments, managerComments, notes sql.NullString

	err := row.Scan(
		&g.ID, &g.EmployeeID, &reviewID, &g.Title, &description, &category,
		&successCriteria, &metrics, &alignedTo, &g.Weight,
		&startDate, &targetDate, &g.Status, &g.Progress, &completedAt,
		&selfRating, &selfComments, &managerRating, &managerComments,
		&notes, &g.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if reviewID.Valid {
		g.ReviewID = &reviewID.Int64
	}
	if description.Valid {
		g.Description = &description.String
	}
	if category.Valid {
		g.Category = &category.String
	}
	if successCriteria.Valid {
		g.SuccessCriteria = &successCriteria.String
	}
	if metrics.Valid {
		g.Metrics = &metrics.String
	}
	if alignedTo.Valid {
		g.AlignedTo = &alignedTo.String
	}
	if startDate.Valid {
		g.StartDate = &startDate.Time
	}
	if targetDate.Valid {
		g.TargetDate = &targetDate.Time
	}
	if completedAt.Valid {
		g.CompletedAt = &completedAt.Time
	}
	if selfRating.Valid {
		r := int(selfRating.Int64)
		g.SelfRating = &r
	}
	if selfComments.Valid {
		g.SelfComments = &selfComments.String
	}
	if managerRating.Valid {
		r := int(managerRating.Int64)
		g.ManagerRating = &r
	}
	if managerComments.Valid {
		g.ManagerComments = &managerComments.String
	}
	if notes.Valid {
		g.Notes = &notes.String
	}
	if updatedAt.Valid {
		g.UpdatedAt = &updatedAt.Time
	}

	return &g, nil
}

func (r *Repo) scanPerformanceGoalRow(rows *sql.Rows) (*hrmmodel.PerformanceGoal, error) {
	var g hrmmodel.PerformanceGoal
	var reviewID sql.NullInt64
	var description, category, successCriteria, metrics, alignedTo sql.NullString
	var startDate, targetDate, completedAt, updatedAt sql.NullTime
	var selfRating, managerRating sql.NullInt64
	var selfComments, managerComments, notes sql.NullString

	err := rows.Scan(
		&g.ID, &g.EmployeeID, &reviewID, &g.Title, &description, &category,
		&successCriteria, &metrics, &alignedTo, &g.Weight,
		&startDate, &targetDate, &g.Status, &g.Progress, &completedAt,
		&selfRating, &selfComments, &managerRating, &managerComments,
		&notes, &g.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if reviewID.Valid {
		g.ReviewID = &reviewID.Int64
	}
	if description.Valid {
		g.Description = &description.String
	}
	if category.Valid {
		g.Category = &category.String
	}
	if successCriteria.Valid {
		g.SuccessCriteria = &successCriteria.String
	}
	if metrics.Valid {
		g.Metrics = &metrics.String
	}
	if alignedTo.Valid {
		g.AlignedTo = &alignedTo.String
	}
	if startDate.Valid {
		g.StartDate = &startDate.Time
	}
	if targetDate.Valid {
		g.TargetDate = &targetDate.Time
	}
	if completedAt.Valid {
		g.CompletedAt = &completedAt.Time
	}
	if selfRating.Valid {
		r := int(selfRating.Int64)
		g.SelfRating = &r
	}
	if selfComments.Valid {
		g.SelfComments = &selfComments.String
	}
	if managerRating.Valid {
		r := int(managerRating.Int64)
		g.ManagerRating = &r
	}
	if managerComments.Valid {
		g.ManagerComments = &managerComments.String
	}
	if notes.Valid {
		g.Notes = &notes.String
	}
	if updatedAt.Valid {
		g.UpdatedAt = &updatedAt.Time
	}

	return &g, nil
}

func (r *Repo) scanKPI(row *sql.Row) (*hrmmodel.KPI, error) {
	var k hrmmodel.KPI
	var description, category, measurementUnit, notes sql.NullString
	var minThreshold, maxThreshold, actualValue, achievementPercent sql.NullFloat64
	var month, quarter, rating sql.NullInt64
	var updatedAt sql.NullTime

	err := row.Scan(
		&k.ID, &k.EmployeeID, &k.Name, &description, &category,
		&measurementUnit, &k.TargetValue, &minThreshold, &maxThreshold,
		&k.Year, &month, &quarter, &actualValue, &achievementPercent,
		&rating, &k.Weight, &notes, &k.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if description.Valid {
		k.Description = &description.String
	}
	if category.Valid {
		k.Category = &category.String
	}
	if measurementUnit.Valid {
		k.MeasurementUnit = &measurementUnit.String
	}
	if minThreshold.Valid {
		k.MinThreshold = &minThreshold.Float64
	}
	if maxThreshold.Valid {
		k.MaxThreshold = &maxThreshold.Float64
	}
	if month.Valid {
		m := int(month.Int64)
		k.Month = &m
	}
	if quarter.Valid {
		q := int(quarter.Int64)
		k.Quarter = &q
	}
	if actualValue.Valid {
		k.ActualValue = &actualValue.Float64
	}
	if achievementPercent.Valid {
		k.AchievementPercent = &achievementPercent.Float64
	}
	if rating.Valid {
		r := int(rating.Int64)
		k.Rating = &r
	}
	if notes.Valid {
		k.Notes = &notes.String
	}
	if updatedAt.Valid {
		k.UpdatedAt = &updatedAt.Time
	}

	return &k, nil
}

func (r *Repo) scanKPIRow(rows *sql.Rows) (*hrmmodel.KPI, error) {
	var k hrmmodel.KPI
	var description, category, measurementUnit, notes sql.NullString
	var minThreshold, maxThreshold, actualValue, achievementPercent sql.NullFloat64
	var month, quarter, rating sql.NullInt64
	var updatedAt sql.NullTime

	err := rows.Scan(
		&k.ID, &k.EmployeeID, &k.Name, &description, &category,
		&measurementUnit, &k.TargetValue, &minThreshold, &maxThreshold,
		&k.Year, &month, &quarter, &actualValue, &achievementPercent,
		&rating, &k.Weight, &notes, &k.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if description.Valid {
		k.Description = &description.String
	}
	if category.Valid {
		k.Category = &category.String
	}
	if measurementUnit.Valid {
		k.MeasurementUnit = &measurementUnit.String
	}
	if minThreshold.Valid {
		k.MinThreshold = &minThreshold.Float64
	}
	if maxThreshold.Valid {
		k.MaxThreshold = &maxThreshold.Float64
	}
	if month.Valid {
		m := int(month.Int64)
		k.Month = &m
	}
	if quarter.Valid {
		q := int(quarter.Int64)
		k.Quarter = &q
	}
	if actualValue.Valid {
		k.ActualValue = &actualValue.Float64
	}
	if achievementPercent.Valid {
		k.AchievementPercent = &achievementPercent.Float64
	}
	if rating.Valid {
		r := int(rating.Int64)
		k.Rating = &r
	}
	if notes.Valid {
		k.Notes = &notes.String
	}
	if updatedAt.Valid {
		k.UpdatedAt = &updatedAt.Time
	}

	return &k, nil
}
