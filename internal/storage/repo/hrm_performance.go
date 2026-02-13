package repo

import (
	"context"
	"database/sql"
	"fmt"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/hrm/performance"
	"srmt-admin/internal/storage"
	"strings"
)

// ==================== Reviews ====================

func (r *Repo) CreateReview(ctx context.Context, req dto.CreateReviewRequest) (int64, error) {
	const op = "repo.CreateReview"

	query := `
		INSERT INTO performance_reviews (employee_id, reviewer_id, type, period_start, period_end)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query, req.EmployeeID, req.ReviewerID, req.Type, req.PeriodStart, req.PeriodEnd).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

func (r *Repo) GetReviewByID(ctx context.Context, id int64) (*performance.PerformanceReview, error) {
	const op = "repo.GetReviewByID"

	query := `
		SELECT r.id, r.employee_id, COALESCE(ce.fio,'') as employee_name,
			   r.reviewer_id, COALESCE(cr.fio,'') as reviewer_name,
			   r.type, r.status, r.period_start, r.period_end,
			   r.self_rating, r.manager_rating, r.final_rating,
			   r.self_comment, r.manager_comment, r.strengths, r.improvements,
			   r.created_at, r.updated_at
		FROM performance_reviews r
		LEFT JOIN contacts ce ON r.employee_id = ce.id
		LEFT JOIN contacts cr ON r.reviewer_id = cr.id
		WHERE r.id = $1`

	review, err := scanReview(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrReviewNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	goals, err := r.GetGoalsByReviewID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%s: load goals: %w", op, err)
	}
	review.Goals = goals

	return review, nil
}

func (r *Repo) GetAllReviews(ctx context.Context, filters dto.ReviewFilters) ([]*performance.PerformanceReview, error) {
	const op = "repo.GetAllReviews"

	query := `
		SELECT r.id, r.employee_id, COALESCE(ce.fio,'') as employee_name,
			   r.reviewer_id, COALESCE(cr.fio,'') as reviewer_name,
			   r.type, r.status, r.period_start, r.period_end,
			   r.self_rating, r.manager_rating, r.final_rating,
			   r.self_comment, r.manager_comment, r.strengths, r.improvements,
			   r.created_at, r.updated_at
		FROM performance_reviews r
		LEFT JOIN contacts ce ON r.employee_id = ce.id
		LEFT JOIN contacts cr ON r.reviewer_id = cr.id`

	var conditions []string
	var args []interface{}
	argIdx := 1

	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("r.status = $%d", argIdx))
		args = append(args, *filters.Status)
		argIdx++
	}
	if filters.Type != nil {
		conditions = append(conditions, fmt.Sprintf("r.type = $%d", argIdx))
		args = append(args, *filters.Type)
		argIdx++
	}
	if filters.EmployeeID != nil {
		conditions = append(conditions, fmt.Sprintf("r.employee_id = $%d", argIdx))
		args = append(args, *filters.EmployeeID)
		argIdx++
	}
	if filters.Search != nil {
		conditions = append(conditions, fmt.Sprintf("ce.fio ILIKE $%d", argIdx))
		args = append(args, "%"+*filters.Search+"%")
		argIdx++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY r.created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var result []*performance.PerformanceReview
	for rows.Next() {
		review, err := scanReview(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		result = append(result, review)
	}
	return result, nil
}

func (r *Repo) UpdateReview(ctx context.Context, id int64, req dto.UpdateReviewRequest) error {
	const op = "repo.UpdateReview"

	var setClauses []string
	var args []interface{}
	argIdx := 1

	if req.Type != nil {
		setClauses = append(setClauses, fmt.Sprintf("type = $%d", argIdx))
		args = append(args, *req.Type)
		argIdx++
	}
	if req.PeriodStart != nil {
		setClauses = append(setClauses, fmt.Sprintf("period_start = $%d", argIdx))
		args = append(args, *req.PeriodStart)
		argIdx++
	}
	if req.PeriodEnd != nil {
		setClauses = append(setClauses, fmt.Sprintf("period_end = $%d", argIdx))
		args = append(args, *req.PeriodEnd)
		argIdx++
	}
	if req.ReviewerID != nil {
		setClauses = append(setClauses, fmt.Sprintf("reviewer_id = $%d", argIdx))
		args = append(args, *req.ReviewerID)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE performance_reviews SET %s WHERE id = $%d", strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return storage.ErrReviewNotFound
	}
	return nil
}

func (r *Repo) UpdateReviewFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	const op = "repo.UpdateReviewFields"

	var setClauses []string
	var args []interface{}
	argIdx := 1

	for field, value := range fields {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", field, argIdx))
		args = append(args, value)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE performance_reviews SET %s WHERE id = $%d", strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return storage.ErrReviewNotFound
	}
	return nil
}

func (r *Repo) UpdateReviewStatus(ctx context.Context, id int64, status string) error {
	const op = "repo.UpdateReviewStatus"

	res, err := r.db.ExecContext(ctx, "UPDATE performance_reviews SET status = $1 WHERE id = $2", status, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return storage.ErrReviewNotFound
	}
	return nil
}

// ==================== Goals ====================

func (r *Repo) CreateGoal(ctx context.Context, req dto.CreateGoalRequest) (int64, error) {
	const op = "repo.CreateGoal"

	targetValue := 0.0
	if req.TargetValue != nil {
		targetValue = *req.TargetValue
	}
	weight := 1.0
	if req.Weight != nil {
		weight = *req.Weight
	}

	query := `
		INSERT INTO performance_goals (review_id, employee_id, title, description, metric, target_value, weight, due_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.ReviewID, req.EmployeeID, req.Title, req.Description, req.Metric,
		targetValue, weight, req.DueDate,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

func (r *Repo) GetGoalByID(ctx context.Context, id int64) (*performance.PerformanceGoal, error) {
	const op = "repo.GetGoalByID"

	query := `
		SELECT id, review_id, employee_id, title, description, metric,
			   target_value, current_value, weight, status, due_date, progress,
			   created_at, updated_at
		FROM performance_goals
		WHERE id = $1`

	goal, err := scanGoal(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrGoalNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return goal, nil
}

func (r *Repo) GetAllGoals(ctx context.Context, filters dto.GoalFilters) ([]*performance.PerformanceGoal, error) {
	const op = "repo.GetAllGoals"

	query := `
		SELECT id, review_id, employee_id, title, description, metric,
			   target_value, current_value, weight, status, due_date, progress,
			   created_at, updated_at
		FROM performance_goals`

	var conditions []string
	var args []interface{}
	argIdx := 1

	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filters.Status)
		argIdx++
	}
	if filters.EmployeeID != nil {
		conditions = append(conditions, fmt.Sprintf("employee_id = $%d", argIdx))
		args = append(args, *filters.EmployeeID)
		argIdx++
	}
	if filters.ReviewID != nil {
		conditions = append(conditions, fmt.Sprintf("review_id = $%d", argIdx))
		args = append(args, *filters.ReviewID)
		argIdx++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var result []*performance.PerformanceGoal
	for rows.Next() {
		goal, err := scanGoal(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		result = append(result, goal)
	}
	return result, nil
}

func (r *Repo) GetGoalsByReviewID(ctx context.Context, reviewID int64) ([]*performance.PerformanceGoal, error) {
	const op = "repo.GetGoalsByReviewID"

	query := `
		SELECT id, review_id, employee_id, title, description, metric,
			   target_value, current_value, weight, status, due_date, progress,
			   created_at, updated_at
		FROM performance_goals
		WHERE review_id = $1
		ORDER BY created_at`

	rows, err := r.db.QueryContext(ctx, query, reviewID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var result []*performance.PerformanceGoal
	for rows.Next() {
		goal, err := scanGoal(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		result = append(result, goal)
	}
	return result, nil
}

func (r *Repo) UpdateGoal(ctx context.Context, id int64, req dto.UpdateGoalRequest) error {
	const op = "repo.UpdateGoal"

	var setClauses []string
	var args []interface{}
	argIdx := 1

	if req.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *req.Title)
		argIdx++
	}
	if req.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.Metric != nil {
		setClauses = append(setClauses, fmt.Sprintf("metric = $%d", argIdx))
		args = append(args, *req.Metric)
		argIdx++
	}
	if req.TargetValue != nil {
		setClauses = append(setClauses, fmt.Sprintf("target_value = $%d", argIdx))
		args = append(args, *req.TargetValue)
		argIdx++
	}
	if req.Weight != nil {
		setClauses = append(setClauses, fmt.Sprintf("weight = $%d", argIdx))
		args = append(args, *req.Weight)
		argIdx++
	}
	if req.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
	}
	if req.DueDate != nil {
		setClauses = append(setClauses, fmt.Sprintf("due_date = $%d", argIdx))
		args = append(args, *req.DueDate)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE performance_goals SET %s WHERE id = $%d", strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return storage.ErrGoalNotFound
	}
	return nil
}

func (r *Repo) UpdateGoalProgress(ctx context.Context, id int64, currentValue float64, progress int, status string) error {
	const op = "repo.UpdateGoalProgress"

	res, err := r.db.ExecContext(ctx,
		"UPDATE performance_goals SET current_value = $1, progress = $2, status = $3 WHERE id = $4",
		currentValue, progress, status, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return storage.ErrGoalNotFound
	}
	return nil
}

func (r *Repo) DeleteGoal(ctx context.Context, id int64) error {
	const op = "repo.DeleteGoal"

	res, err := r.db.ExecContext(ctx, "DELETE FROM performance_goals WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return storage.ErrGoalNotFound
	}
	return nil
}

// ==================== Analytics ====================

func (r *Repo) GetKPIs(ctx context.Context) ([]*performance.KPI, error) {
	const op = "repo.GetKPIs"

	query := `
		SELECT g.employee_id, COALESCE(c.fio,'') as employee_name,
			   COALESCE(d.name,'') as department, COALESCE(p.name,'') as position,
			   COUNT(g.id) as goals_total,
			   COUNT(CASE WHEN g.status='completed' THEN 1 END) as goals_done,
			   COALESCE(AVG(g.progress),0) as avg_progress,
			   COALESCE((SELECT AVG(r.final_rating) FROM performance_reviews r WHERE r.employee_id = g.employee_id AND r.status = 'completed'),0) as avg_rating
		FROM performance_goals g
		JOIN contacts c ON g.employee_id = c.id
		LEFT JOIN personnel_records pr ON pr.contact_id = c.id
		LEFT JOIN departments d ON pr.department_id = d.id
		LEFT JOIN positions p ON pr.position_id = p.id
		GROUP BY g.employee_id, c.fio, d.name, p.name
		ORDER BY avg_progress DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var result []*performance.KPI
	for rows.Next() {
		var kpi performance.KPI
		if err := rows.Scan(&kpi.EmployeeID, &kpi.EmployeeName, &kpi.Department, &kpi.Position,
			&kpi.GoalsTotal, &kpi.GoalsDone, &kpi.AvgProgress, &kpi.AvgRating); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		result = append(result, &kpi)
	}
	return result, nil
}

func (r *Repo) GetAllRatings(ctx context.Context) ([]*performance.EmployeeRating, error) {
	const op = "repo.GetAllRatings"

	query := `
		SELECT r.employee_id, COALESCE(c.fio,'') as employee_name,
			   COALESCE(d.name,'') as department, COALESCE(p.name,'') as position,
			   COUNT(r.id) as reviews_count,
			   COALESCE(AVG(r.final_rating),0) as avg_final_rating
		FROM performance_reviews r
		JOIN contacts c ON r.employee_id = c.id
		LEFT JOIN personnel_records pr ON pr.contact_id = c.id
		LEFT JOIN departments d ON pr.department_id = d.id
		LEFT JOIN positions p ON pr.position_id = p.id
		WHERE r.status = 'completed'
		GROUP BY r.employee_id, c.fio, d.name, p.name
		ORDER BY avg_final_rating DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var result []*performance.EmployeeRating
	for rows.Next() {
		var rating performance.EmployeeRating
		if err := rows.Scan(&rating.EmployeeID, &rating.EmployeeName, &rating.Department, &rating.Position,
			&rating.ReviewsCount, &rating.AvgFinalRating); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		result = append(result, &rating)
	}
	return result, nil
}

func (r *Repo) GetEmployeeRating(ctx context.Context, employeeID int64) (*performance.EmployeeRating, error) {
	const op = "repo.GetEmployeeRating"

	var rating performance.EmployeeRating
	err := r.db.QueryRowContext(ctx,
		`SELECT r.employee_id, COALESCE(c.fio,'') as employee_name,
				COALESCE(d.name,'') as department, COALESCE(p.name,'') as position,
				COUNT(r.id) as reviews_count,
				COALESCE(AVG(r.final_rating),0) as avg_final_rating
		 FROM performance_reviews r
		 JOIN contacts c ON r.employee_id = c.id
		 LEFT JOIN personnel_records pr ON pr.contact_id = c.id
		 LEFT JOIN departments d ON pr.department_id = d.id
		 LEFT JOIN positions p ON pr.position_id = p.id
		 WHERE r.employee_id = $1 AND r.status = 'completed'
		 GROUP BY r.employee_id, c.fio, d.name, p.name`, employeeID,
	).Scan(&rating.EmployeeID, &rating.EmployeeName, &rating.Department, &rating.Position,
		&rating.ReviewsCount, &rating.AvgFinalRating)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return empty rating for employee with no reviews
			var name string
			err2 := r.db.QueryRowContext(ctx, "SELECT COALESCE(fio,'') FROM contacts WHERE id = $1", employeeID).Scan(&name)
			if err2 != nil {
				return nil, storage.ErrReviewNotFound
			}
			return &performance.EmployeeRating{
				EmployeeID:   employeeID,
				EmployeeName: name,
				Details:      []*performance.EmployeeRatingDetail{},
			}, nil
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Load details
	detailRows, err := r.db.QueryContext(ctx,
		`SELECT id, type, period_start, period_end, final_rating, status
		 FROM performance_reviews
		 WHERE employee_id = $1 AND status = 'completed'
		 ORDER BY period_end DESC`, employeeID)
	if err != nil {
		return nil, fmt.Errorf("%s: load details: %w", op, err)
	}
	defer detailRows.Close()

	for detailRows.Next() {
		var d performance.EmployeeRatingDetail
		if err := detailRows.Scan(&d.ReviewID, &d.Type, &d.PeriodStart, &d.PeriodEnd, &d.FinalRating, &d.Status); err != nil {
			return nil, fmt.Errorf("%s: scan detail: %w", op, err)
		}
		rating.Details = append(rating.Details, &d)
	}

	return &rating, nil
}

func (r *Repo) GetPerformanceDashboard(ctx context.Context) (*performance.PerformanceDashboard, error) {
	const op = "repo.GetPerformanceDashboard"

	dash := &performance.PerformanceDashboard{}

	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) as total_reviews,
			COUNT(CASE WHEN status='completed' THEN 1 END) as completed,
			COUNT(CASE WHEN status NOT IN ('completed','acknowledged') THEN 1 END) as pending,
			COALESCE(AVG(final_rating),0) as avg_rating
		FROM performance_reviews`).Scan(&dash.TotalReviews, &dash.CompletedReviews, &dash.PendingReviews, &dash.AvgRating)
	if err != nil {
		return nil, fmt.Errorf("%s: reviews: %w", op, err)
	}

	gs := &performance.GoalStats{}
	err = r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN status='completed' THEN 1 END) as completed,
			COUNT(CASE WHEN status='in_progress' THEN 1 END) as in_progress,
			COUNT(CASE WHEN status='overdue' THEN 1 END) as overdue,
			COALESCE(AVG(progress),0) as avg_progress
		FROM performance_goals`).Scan(&gs.Total, &gs.Completed, &gs.InProgress, &gs.Overdue, &gs.AvgProgress)
	if err != nil {
		return nil, fmt.Errorf("%s: goals: %w", op, err)
	}
	dash.GoalStats = gs

	return dash, nil
}

// ==================== Scanners ====================

func scanReview(s scannable) (*performance.PerformanceReview, error) {
	var rv performance.PerformanceReview
	var periodStart, periodEnd string
	err := s.Scan(
		&rv.ID, &rv.EmployeeID, &rv.EmployeeName,
		&rv.ReviewerID, &rv.ReviewerName,
		&rv.Type, &rv.Status, &periodStart, &periodEnd,
		&rv.SelfRating, &rv.ManagerRating, &rv.FinalRating,
		&rv.SelfComment, &rv.ManagerComment, &rv.Strengths, &rv.Improvements,
		&rv.CreatedAt, &rv.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	rv.PeriodStart = periodStart
	rv.PeriodEnd = periodEnd
	return &rv, nil
}

func scanGoal(s scannable) (*performance.PerformanceGoal, error) {
	var g performance.PerformanceGoal
	var dueDate string
	err := s.Scan(
		&g.ID, &g.ReviewID, &g.EmployeeID, &g.Title, &g.Description, &g.Metric,
		&g.TargetValue, &g.CurrentValue, &g.Weight, &g.Status, &dueDate, &g.Progress,
		&g.CreatedAt, &g.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	g.DueDate = dueDate
	return &g, nil
}
