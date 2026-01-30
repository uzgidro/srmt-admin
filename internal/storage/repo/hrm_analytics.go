package repo

import (
	"context"
	"fmt"
	"time"

	"srmt-admin/internal/lib/dto/hrm"
)

// --- Dashboard Stats ---

// GetDashboardStats retrieves dashboard statistics
func (r *Repo) GetDashboardStats(ctx context.Context, filter hrm.AnalyticsFilter) (*hrm.DashboardResponse, error) {
	const op = "storage.repo.GetDashboardStats"

	dashboard := &hrm.DashboardResponse{
		HeadcountByDepartment: make([]hrm.DepartmentHeadcount, 0),
		RecentActivity:        make([]hrm.ActivityItem, 0),
	}

	// Get total employees count
	totalQuery := `SELECT COUNT(*) FROM hrm_employees WHERE employment_status != 'terminated'`
	if err := r.db.QueryRowContext(ctx, totalQuery).Scan(&dashboard.TotalEmployees); err != nil {
		return nil, fmt.Errorf("%s: failed to get total employees: %w", op, err)
	}

	// Get active employees count
	activeQuery := `SELECT COUNT(*) FROM hrm_employees WHERE employment_status = 'active'`
	if err := r.db.QueryRowContext(ctx, activeQuery).Scan(&dashboard.ActiveEmployees); err != nil {
		return nil, fmt.Errorf("%s: failed to get active employees: %w", op, err)
	}

	// Get new hires this month
	now := time.Now()
	firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	newHiresQuery := `SELECT COUNT(*) FROM hrm_employees WHERE hire_date >= $1`
	if err := r.db.QueryRowContext(ctx, newHiresQuery, firstOfMonth).Scan(&dashboard.NewHiresThisMonth); err != nil {
		return nil, fmt.Errorf("%s: failed to get new hires: %w", op, err)
	}

	// Get terminations this month
	terminationsQuery := `SELECT COUNT(*) FROM hrm_employees WHERE termination_date >= $1`
	if err := r.db.QueryRowContext(ctx, terminationsQuery, firstOfMonth).Scan(&dashboard.TerminationsThisMonth); err != nil {
		return nil, fmt.Errorf("%s: failed to get terminations: %w", op, err)
	}

	// Get pending vacations
	pendingVacationsQuery := `SELECT COUNT(*) FROM hrm_vacations WHERE status = 'pending'`
	if err := r.db.QueryRowContext(ctx, pendingVacationsQuery).Scan(&dashboard.PendingVacations); err != nil {
		return nil, fmt.Errorf("%s: failed to get pending vacations: %w", op, err)
	}

	// Get headcount by department
	headcountQuery := `
		SELECT d.id, d.name, COUNT(e.id) as total,
			COUNT(CASE WHEN e.employment_status = 'active' THEN 1 END) as active
		FROM hrm_employees e
		JOIN contacts c ON e.contact_id = c.id
		JOIN departments d ON c.department_id = d.id
		WHERE e.employment_status != 'terminated'
		GROUP BY d.id, d.name
		ORDER BY total DESC
		LIMIT 10`

	rows, err := r.db.QueryContext(ctx, headcountQuery)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get headcount by department: %w", op, err)
	}
	defer rows.Close()

	for rows.Next() {
		var h hrm.DepartmentHeadcount
		if err := rows.Scan(&h.DepartmentID, &h.DepartmentName, &h.Count, &h.ActiveCount); err != nil {
			return nil, fmt.Errorf("%s: failed to scan headcount: %w", op, err)
		}
		dashboard.HeadcountByDepartment = append(dashboard.HeadcountByDepartment, h)
	}

	return dashboard, nil
}

// --- Headcount Stats ---

// GetHeadcountStats retrieves headcount statistics
func (r *Repo) GetHeadcountStats(ctx context.Context, filter hrm.AnalyticsFilter) (*hrm.HeadcountReportResponse, error) {
	const op = "storage.repo.GetHeadcountStats"

	report := &hrm.HeadcountReportResponse{
		ByDepartment:       make([]hrm.DepartmentHeadcount, 0),
		ByEmploymentType:   make([]hrm.TypeHeadcount, 0),
		ByEmploymentStatus: make([]hrm.StatusHeadcount, 0),
	}

	// Total headcount
	totalQuery := `SELECT COUNT(*) FROM hrm_employees WHERE employment_status != 'terminated'`
	if err := r.db.QueryRowContext(ctx, totalQuery).Scan(&report.TotalHeadcount); err != nil {
		return nil, fmt.Errorf("%s: failed to get total headcount: %w", op, err)
	}

	// Active headcount
	activeQuery := `SELECT COUNT(*) FROM hrm_employees WHERE employment_status = 'active'`
	if err := r.db.QueryRowContext(ctx, activeQuery).Scan(&report.ActiveHeadcount); err != nil {
		return nil, fmt.Errorf("%s: failed to get active headcount: %w", op, err)
	}

	// By department
	deptQuery := `
		SELECT d.id, d.name, COUNT(e.id) as total,
			COUNT(CASE WHEN e.employment_status = 'active' THEN 1 END) as active
		FROM hrm_employees e
		JOIN contacts c ON e.contact_id = c.id
		JOIN departments d ON c.department_id = d.id
		WHERE e.employment_status != 'terminated'
		GROUP BY d.id, d.name
		ORDER BY total DESC`

	deptRows, err := r.db.QueryContext(ctx, deptQuery)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get by department: %w", op, err)
	}
	defer deptRows.Close()

	for deptRows.Next() {
		var h hrm.DepartmentHeadcount
		if err := deptRows.Scan(&h.DepartmentID, &h.DepartmentName, &h.Count, &h.ActiveCount); err != nil {
			return nil, fmt.Errorf("%s: failed to scan department: %w", op, err)
		}
		report.ByDepartment = append(report.ByDepartment, h)
	}

	// By employment type
	typeQuery := `
		SELECT employment_type, COUNT(*)
		FROM hrm_employees
		WHERE employment_status != 'terminated'
		GROUP BY employment_type`

	typeRows, err := r.db.QueryContext(ctx, typeQuery)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get by type: %w", op, err)
	}
	defer typeRows.Close()

	for typeRows.Next() {
		var t hrm.TypeHeadcount
		if err := typeRows.Scan(&t.Type, &t.Count); err != nil {
			return nil, fmt.Errorf("%s: failed to scan type: %w", op, err)
		}
		report.ByEmploymentType = append(report.ByEmploymentType, t)
	}

	// By employment status
	statusQuery := `
		SELECT employment_status, COUNT(*)
		FROM hrm_employees
		GROUP BY employment_status`

	statusRows, err := r.db.QueryContext(ctx, statusQuery)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get by status: %w", op, err)
	}
	defer statusRows.Close()

	for statusRows.Next() {
		var s hrm.StatusHeadcount
		if err := statusRows.Scan(&s.Status, &s.Count); err != nil {
			return nil, fmt.Errorf("%s: failed to scan status: %w", op, err)
		}
		report.ByEmploymentStatus = append(report.ByEmploymentStatus, s)
	}

	return report, nil
}

// GetHeadcountTrend retrieves headcount trend data
func (r *Repo) GetHeadcountTrend(ctx context.Context, filter hrm.AnalyticsFilter) (*hrm.HeadcountTrendResponse, error) {
	const op = "storage.repo.GetHeadcountTrend"

	trend := &hrm.HeadcountTrendResponse{
		DataPoints: make([]hrm.HeadcountTrendPoint, 0),
	}

	// Get last 12 months trend
	query := `
		WITH months AS (
			SELECT generate_series(
				date_trunc('month', NOW() - INTERVAL '11 months'),
				date_trunc('month', NOW()),
				'1 month'::interval
			) AS month
		)
		SELECT
			m.month::date,
			EXTRACT(YEAR FROM m.month)::int as year,
			EXTRACT(MONTH FROM m.month)::int as month,
			COALESCE((SELECT COUNT(*) FROM hrm_employees
				WHERE hire_date <= m.month + INTERVAL '1 month' - INTERVAL '1 day'
				AND (termination_date IS NULL OR termination_date > m.month + INTERVAL '1 month' - INTERVAL '1 day')), 0) as headcount,
			COALESCE((SELECT COUNT(*) FROM hrm_employees
				WHERE date_trunc('month', hire_date) = m.month), 0) as hired,
			COALESCE((SELECT COUNT(*) FROM hrm_employees
				WHERE date_trunc('month', termination_date) = m.month), 0) as terminated
		FROM months m
		ORDER BY m.month`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get trend: %w", op, err)
	}
	defer rows.Close()

	for rows.Next() {
		var p hrm.HeadcountTrendPoint
		var date time.Time
		if err := rows.Scan(&date, &p.Year, &p.Month, &p.Headcount, &p.Hired, &p.Terminated); err != nil {
			return nil, fmt.Errorf("%s: failed to scan trend: %w", op, err)
		}
		p.Date = date.Format("2006-01")
		trend.DataPoints = append(trend.DataPoints, p)
	}

	return trend, nil
}

// --- Turnover Stats ---

// GetTurnoverStats retrieves turnover statistics
func (r *Repo) GetTurnoverStats(ctx context.Context, filter hrm.AnalyticsFilter) (*hrm.TurnoverReportResponse, error) {
	const op = "storage.repo.GetTurnoverStats"

	// Use filter dates, default to current year if not set
	now := time.Now()
	startDate := filter.FromDate
	endDate := filter.ToDate
	if startDate.IsZero() {
		startDate = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	}
	if endDate.IsZero() {
		endDate = time.Date(now.Year(), 12, 31, 23, 59, 59, 0, time.UTC)
	}

	report := &hrm.TurnoverReportResponse{
		Period:       fmt.Sprintf("%s - %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")),
		ByDepartment: make([]hrm.DepartmentTurnover, 0),
		ByReason:     make([]hrm.ReasonCount, 0),
	}

	// Get total, hired, terminated
	statsQuery := `
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN hire_date BETWEEN $1 AND $2 THEN 1 END) as hired,
			COUNT(CASE WHEN termination_date BETWEEN $1 AND $2 THEN 1 END) as terminated
		FROM hrm_employees
		WHERE hire_date <= $2`

	err := r.db.QueryRowContext(ctx, statsQuery, startDate, endDate).Scan(
		&report.TotalEmployees, &report.Hired, &report.Terminated)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get stats: %w", op, err)
	}

	// Calculate rates
	if report.TotalEmployees > 0 {
		report.TurnoverRate = float64(report.Terminated) / float64(report.TotalEmployees) * 100
		report.RetentionRate = 100 - report.TurnoverRate
	}

	// By department
	deptQuery := `
		SELECT d.id, d.name,
			COUNT(CASE WHEN e.hire_date BETWEEN $1 AND $2 THEN 1 END) as hired,
			COUNT(CASE WHEN e.termination_date BETWEEN $1 AND $2 THEN 1 END) as terminated,
			COUNT(*) as total
		FROM hrm_employees e
		JOIN contacts c ON e.contact_id = c.id
		JOIN departments d ON c.department_id = d.id
		WHERE e.hire_date <= $2
		GROUP BY d.id, d.name`

	deptRows, err := r.db.QueryContext(ctx, deptQuery, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get dept turnover: %w", op, err)
	}
	defer deptRows.Close()

	for deptRows.Next() {
		var d hrm.DepartmentTurnover
		var total int
		if err := deptRows.Scan(&d.DepartmentID, &d.DepartmentName, &d.Hired, &d.Terminated, &total); err != nil {
			return nil, fmt.Errorf("%s: failed to scan dept: %w", op, err)
		}
		if total > 0 {
			d.TurnoverRate = float64(d.Terminated) / float64(total) * 100
		}
		report.ByDepartment = append(report.ByDepartment, d)
	}

	return report, nil
}

// GetTurnoverTrend retrieves turnover trend data
func (r *Repo) GetTurnoverTrend(ctx context.Context, filter hrm.AnalyticsFilter) (*hrm.TurnoverTrendResponse, error) {
	const op = "storage.repo.GetTurnoverTrend"

	trend := &hrm.TurnoverTrendResponse{
		DataPoints: make([]hrm.TurnoverTrendPoint, 0),
	}

	// Get last 12 months trend
	query := `
		WITH months AS (
			SELECT generate_series(
				date_trunc('month', NOW() - INTERVAL '11 months'),
				date_trunc('month', NOW()),
				'1 month'::interval
			) AS month
		)
		SELECT
			m.month::date,
			EXTRACT(YEAR FROM m.month)::int as year,
			EXTRACT(MONTH FROM m.month)::int as month,
			COALESCE((SELECT COUNT(*) FROM hrm_employees
				WHERE date_trunc('month', hire_date) = m.month), 0) as hired,
			COALESCE((SELECT COUNT(*) FROM hrm_employees
				WHERE date_trunc('month', termination_date) = m.month), 0) as terminated,
			COALESCE((SELECT COUNT(*) FROM hrm_employees
				WHERE hire_date <= m.month + INTERVAL '1 month' - INTERVAL '1 day'
				AND (termination_date IS NULL OR termination_date > m.month + INTERVAL '1 month' - INTERVAL '1 day')), 1) as total
		FROM months m
		ORDER BY m.month`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get turnover trend: %w", op, err)
	}
	defer rows.Close()

	for rows.Next() {
		var p hrm.TurnoverTrendPoint
		var date time.Time
		var total int
		if err := rows.Scan(&date, &p.Year, &p.Month, &p.Hired, &p.Terminated, &total); err != nil {
			return nil, fmt.Errorf("%s: failed to scan trend: %w", op, err)
		}
		p.Date = date.Format("2006-01")
		if total > 0 {
			p.TurnoverRate = float64(p.Terminated) / float64(total) * 100
		}
		trend.DataPoints = append(trend.DataPoints, p)
	}

	return trend, nil
}

// --- Salary Stats ---

// GetSalaryStats retrieves salary statistics
func (r *Repo) GetSalaryStats(ctx context.Context, filter hrm.AnalyticsFilter) (*hrm.SalaryReportResponse, error) {
	const op = "storage.repo.GetSalaryStats"

	// Use filter dates, default to current month if not set
	now := time.Now()
	year := now.Year()
	month := int(now.Month())
	if !filter.FromDate.IsZero() {
		year = filter.FromDate.Year()
		month = int(filter.FromDate.Month())
	}

	report := &hrm.SalaryReportResponse{
		Period:       fmt.Sprintf("%d-%02d", year, month),
		ByDepartment: make([]hrm.DepartmentSalary, 0),
	}

	// Get totals
	statsQuery := `
		SELECT
			COALESCE(SUM(gross_amount), 0),
			COALESCE(SUM(net_amount), 0),
			COALESCE(SUM(tax_amount), 0),
			COALESCE(SUM(bonuses_amount), 0),
			COALESCE(AVG(net_amount), 0),
			COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY net_amount), 0)
		FROM hrm_salaries
		WHERE year = $1 AND month = $2`

	err := r.db.QueryRowContext(ctx, statsQuery, year, month).Scan(
		&report.TotalGross, &report.TotalNet, &report.TotalTax,
		&report.TotalBonuses, &report.AverageSalary, &report.MedianSalary)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get salary stats: %w", op, err)
	}

	// By department
	deptQuery := `
		SELECT d.id, d.name,
			COALESCE(SUM(s.gross_amount), 0),
			COALESCE(AVG(s.net_amount), 0),
			COUNT(DISTINCT s.employee_id)
		FROM hrm_salaries s
		JOIN hrm_employees e ON s.employee_id = e.id
		JOIN contacts c ON e.contact_id = c.id
		JOIN departments d ON c.department_id = d.id
		WHERE s.year = $1 AND s.month = $2
		GROUP BY d.id, d.name
		ORDER BY SUM(s.gross_amount) DESC`

	deptRows, err := r.db.QueryContext(ctx, deptQuery, year, month)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get dept salary: %w", op, err)
	}
	defer deptRows.Close()

	for deptRows.Next() {
		var d hrm.DepartmentSalary
		if err := deptRows.Scan(&d.DepartmentID, &d.DepartmentName, &d.TotalGross, &d.AverageSalary, &d.EmployeeCount); err != nil {
			return nil, fmt.Errorf("%s: failed to scan dept: %w", op, err)
		}
		report.ByDepartment = append(report.ByDepartment, d)
	}

	return report, nil
}

// GetSalaryTrend retrieves salary trend data
func (r *Repo) GetSalaryTrend(ctx context.Context, filter hrm.AnalyticsFilter) (*hrm.SalaryTrendResponse, error) {
	const op = "storage.repo.GetSalaryTrend"

	trend := &hrm.SalaryTrendResponse{
		DataPoints: make([]hrm.SalaryTrendPoint, 0),
	}

	// Get last 12 months trend
	query := `
		SELECT
			year, month,
			COALESCE(SUM(gross_amount), 0),
			COALESCE(SUM(net_amount), 0),
			COALESCE(AVG(net_amount), 0)
		FROM hrm_salaries
		WHERE (year * 12 + month) >= (EXTRACT(YEAR FROM NOW())::int * 12 + EXTRACT(MONTH FROM NOW())::int - 11)
		GROUP BY year, month
		ORDER BY year, month`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get salary trend: %w", op, err)
	}
	defer rows.Close()

	for rows.Next() {
		var p hrm.SalaryTrendPoint
		if err := rows.Scan(&p.Year, &p.Month, &p.TotalGross, &p.TotalNet, &p.AverageSalary); err != nil {
			return nil, fmt.Errorf("%s: failed to scan trend: %w", op, err)
		}
		p.Date = fmt.Sprintf("%d-%02d", p.Year, p.Month)
		trend.DataPoints = append(trend.DataPoints, p)
	}

	return trend, nil
}

// --- Attendance Stats ---

// GetAttendanceStats retrieves attendance statistics
func (r *Repo) GetAttendanceStats(ctx context.Context, filter hrm.AnalyticsFilter) (*hrm.AttendanceReportResponse, error) {
	const op = "storage.repo.GetAttendanceStats"

	// Use filter dates, default to current month if not set
	now := time.Now()
	year := now.Year()
	month := int(now.Month())
	if !filter.FromDate.IsZero() {
		year = filter.FromDate.Year()
		month = int(filter.FromDate.Month())
	}

	report := &hrm.AttendanceReportResponse{
		Period:       fmt.Sprintf("%d-%02d", year, month),
		ByDepartment: make([]hrm.DepartmentAttendance, 0),
	}

	// Get totals from timesheets
	statsQuery := `
		SELECT
			COALESCE(SUM(work_days), 0),
			COALESCE(AVG(CASE WHEN total_days > 0 THEN work_days::float / total_days * 100 END), 0),
			COALESCE(SUM(sick_days), 0),
			COALESCE(SUM(vacation_days), 0),
			COALESCE(SUM(overtime_hours), 0)
		FROM hrm_timesheets
		WHERE year = $1 AND month = $2`

	err := r.db.QueryRowContext(ctx, statsQuery, year, month).Scan(
		&report.TotalWorkDays, &report.AverageAttendance,
		&report.TotalSickDays, &report.TotalVacationDays, &report.TotalOvertimeHours)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get attendance stats: %w", op, err)
	}

	report.TotalAbsences = report.TotalSickDays + report.TotalVacationDays

	// By department
	deptQuery := `
		SELECT d.id, d.name,
			COALESCE(AVG(CASE WHEN t.total_days > 0 THEN t.work_days::float / t.total_days * 100 END), 0),
			COALESCE(SUM(t.sick_days + t.vacation_days), 0),
			COALESCE(SUM(t.overtime_hours), 0)
		FROM hrm_timesheets t
		JOIN hrm_employees e ON t.employee_id = e.id
		JOIN contacts c ON e.contact_id = c.id
		JOIN departments d ON c.department_id = d.id
		WHERE t.year = $1 AND t.month = $2
		GROUP BY d.id, d.name`

	deptRows, err := r.db.QueryContext(ctx, deptQuery, year, month)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get dept attendance: %w", op, err)
	}
	defer deptRows.Close()

	for deptRows.Next() {
		var d hrm.DepartmentAttendance
		if err := deptRows.Scan(&d.DepartmentID, &d.DepartmentName, &d.AttendanceRate, &d.AbsenceDays, &d.OvertimeHours); err != nil {
			return nil, fmt.Errorf("%s: failed to scan dept: %w", op, err)
		}
		report.ByDepartment = append(report.ByDepartment, d)
	}

	return report, nil
}

// --- Performance Stats ---

// GetPerformanceReport retrieves performance report for analytics
func (r *Repo) GetPerformanceReport(ctx context.Context, filter hrm.AnalyticsFilter) (*hrm.PerformanceReportResponse, error) {
	const op = "storage.repo.GetPerformanceStats"

	// Use filter dates, default to current year if not set
	year := time.Now().Year()
	if !filter.FromDate.IsZero() {
		year = filter.FromDate.Year()
	}

	report := &hrm.PerformanceReportResponse{
		Period:             fmt.Sprintf("%d", year),
		ByDepartment:       make([]hrm.DepartmentPerformance, 0),
		RatingDistribution: make([]hrm.RatingCount, 0),
	}

	// Get totals
	statsQuery := `
		SELECT
			COUNT(*),
			COUNT(CASE WHEN status = 'completed' THEN 1 END),
			COALESCE(AVG(overall_rating), 0)
		FROM hrm_performance_reviews
		WHERE EXTRACT(YEAR FROM review_period_start) = $1`

	err := r.db.QueryRowContext(ctx, statsQuery, year).Scan(
		&report.TotalReviews, &report.CompletedReviews, &report.AverageRating)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get performance stats: %w", op, err)
	}

	// By department
	deptQuery := `
		SELECT d.id, d.name,
			COALESCE(AVG(pr.overall_rating), 0),
			COUNT(*)
		FROM hrm_performance_reviews pr
		JOIN hrm_employees e ON pr.employee_id = e.id
		JOIN contacts c ON e.contact_id = c.id
		JOIN departments d ON c.department_id = d.id
		WHERE EXTRACT(YEAR FROM pr.review_period_start) = $1 AND pr.status = 'completed'
		GROUP BY d.id, d.name`

	deptRows, err := r.db.QueryContext(ctx, deptQuery, year)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get dept performance: %w", op, err)
	}
	defer deptRows.Close()

	for deptRows.Next() {
		var d hrm.DepartmentPerformance
		if err := deptRows.Scan(&d.DepartmentID, &d.DepartmentName, &d.AverageRating, &d.ReviewCount); err != nil {
			return nil, fmt.Errorf("%s: failed to scan dept: %w", op, err)
		}
		report.ByDepartment = append(report.ByDepartment, d)
	}

	// Rating distribution
	ratingQuery := `
		SELECT ROUND(overall_rating)::int, COUNT(*)
		FROM hrm_performance_reviews
		WHERE EXTRACT(YEAR FROM review_period_start) = $1 AND status = 'completed'
		GROUP BY ROUND(overall_rating)
		ORDER BY ROUND(overall_rating)`

	ratingRows, err := r.db.QueryContext(ctx, ratingQuery, year)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get rating dist: %w", op, err)
	}
	defer ratingRows.Close()

	for ratingRows.Next() {
		var r hrm.RatingCount
		if err := ratingRows.Scan(&r.Rating, &r.Count); err != nil {
			continue // Skip invalid ratings
		}
		report.RatingDistribution = append(report.RatingDistribution, r)
	}

	return report, nil
}

// --- Training Stats ---

// GetTrainingReport retrieves training report for analytics
func (r *Repo) GetTrainingReport(ctx context.Context, filter hrm.AnalyticsFilter) (*hrm.TrainingReportResponse, error) {
	const op = "storage.repo.GetTrainingStats"

	// Use filter dates, default to current year if not set
	now := time.Now()
	startDate := filter.FromDate
	endDate := filter.ToDate
	if startDate.IsZero() {
		startDate = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	}
	if endDate.IsZero() {
		endDate = time.Date(now.Year(), 12, 31, 23, 59, 59, 0, time.UTC)
	}

	report := &hrm.TrainingReportResponse{
		Period:     fmt.Sprintf("%s - %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")),
		ByCategory: make([]hrm.CategoryTraining, 0),
	}

	// Get totals
	statsQuery := `
		SELECT
			COUNT(*),
			COUNT(CASE WHEN status = 'completed' THEN 1 END),
			(SELECT COUNT(DISTINCT employee_id) FROM hrm_training_participants tp
				JOIN hrm_trainings t ON tp.training_id = t.id
				WHERE t.start_date BETWEEN $1 AND $2),
			(SELECT COALESCE(AVG(
				CASE WHEN (SELECT COUNT(*) FROM hrm_training_participants WHERE training_id = t.id) > 0
				THEN (SELECT COUNT(*) FROM hrm_training_participants WHERE training_id = t.id AND status = 'completed')::float /
					 (SELECT COUNT(*) FROM hrm_training_participants WHERE training_id = t.id) * 100
				END), 0)
				FROM hrm_trainings t WHERE t.start_date BETWEEN $1 AND $2),
			COALESCE(SUM(duration_hours), 0),
			COALESCE(SUM(cost_per_participant * (SELECT COUNT(*) FROM hrm_training_participants WHERE training_id = hrm_trainings.id)), 0)
		FROM hrm_trainings
		WHERE start_date BETWEEN $1 AND $2`

	err := r.db.QueryRowContext(ctx, statsQuery, startDate, endDate).Scan(
		&report.TotalTrainings, &report.CompletedTrainings,
		&report.TotalParticipants, &report.AverageCompletionRate,
		&report.TotalTrainingHours, &report.TotalCost)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get training stats: %w", op, err)
	}

	// By category
	categoryQuery := `
		SELECT
			COALESCE(category, 'uncategorized'),
			COUNT(*),
			(SELECT COUNT(DISTINCT tp.employee_id) FROM hrm_training_participants tp
				JOIN hrm_trainings t2 ON tp.training_id = t2.id
				WHERE COALESCE(t2.category, 'uncategorized') = COALESCE(t.category, 'uncategorized')
				AND t2.start_date BETWEEN $1 AND $2),
			COALESCE((SELECT AVG(
				CASE WHEN (SELECT COUNT(*) FROM hrm_training_participants WHERE training_id = t2.id) > 0
				THEN (SELECT COUNT(*) FROM hrm_training_participants WHERE training_id = t2.id AND status = 'completed')::float /
					 (SELECT COUNT(*) FROM hrm_training_participants WHERE training_id = t2.id) * 100
				END)
				FROM hrm_trainings t2
				WHERE COALESCE(t2.category, 'uncategorized') = COALESCE(t.category, 'uncategorized')
				AND t2.start_date BETWEEN $1 AND $2), 0)
		FROM hrm_trainings t
		WHERE start_date BETWEEN $1 AND $2
		GROUP BY COALESCE(category, 'uncategorized')`

	catRows, err := r.db.QueryContext(ctx, categoryQuery, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get category stats: %w", op, err)
	}
	defer catRows.Close()

	for catRows.Next() {
		var c hrm.CategoryTraining
		if err := catRows.Scan(&c.Category, &c.TrainingCount, &c.Participants, &c.CompletionRate); err != nil {
			return nil, fmt.Errorf("%s: failed to scan category: %w", op, err)
		}
		report.ByCategory = append(report.ByCategory, c)
	}

	return report, nil
}

// --- Demographics Stats ---

// GetDemographicsStats retrieves demographics statistics
func (r *Repo) GetDemographicsStats(ctx context.Context) (*hrm.DemographicsReportResponse, error) {
	const op = "storage.repo.GetDemographicsStats"

	report := &hrm.DemographicsReportResponse{
		AgeDistribution:    make([]hrm.AgeGroup, 0),
		GenderDistribution: make([]hrm.GenderCount, 0),
		TenureDistribution: make([]hrm.TenureGroup, 0),
	}

	// Get totals
	statsQuery := `
		SELECT
			COUNT(*),
			COALESCE(AVG(EXTRACT(YEAR FROM AGE(c.birth_date))), 0),
			COALESCE(AVG(EXTRACT(YEAR FROM AGE(e.hire_date))), 0)
		FROM hrm_employees e
		JOIN contacts c ON e.contact_id = c.id
		WHERE e.employment_status = 'active'`

	err := r.db.QueryRowContext(ctx, statsQuery).Scan(
		&report.TotalEmployees, &report.AverageAge, &report.AverageTenure)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get demographics stats: %w", op, err)
	}

	// Age distribution
	ageQuery := `
		WITH ages AS (
			SELECT EXTRACT(YEAR FROM AGE(c.birth_date)) as age
			FROM hrm_employees e
			JOIN contacts c ON e.contact_id = c.id
			WHERE e.employment_status = 'active' AND c.birth_date IS NOT NULL
		)
		SELECT
			CASE
				WHEN age < 25 THEN 'Under 25'
				WHEN age BETWEEN 25 AND 34 THEN '25-34'
				WHEN age BETWEEN 35 AND 44 THEN '35-44'
				WHEN age BETWEEN 45 AND 54 THEN '45-54'
				ELSE '55+'
			END as age_range,
			COUNT(*),
			ROUND(COUNT(*)::numeric / (SELECT COUNT(*) FROM ages) * 100, 1)
		FROM ages
		GROUP BY
			CASE
				WHEN age < 25 THEN 'Under 25'
				WHEN age BETWEEN 25 AND 34 THEN '25-34'
				WHEN age BETWEEN 35 AND 44 THEN '35-44'
				WHEN age BETWEEN 45 AND 54 THEN '45-54'
				ELSE '55+'
			END
		ORDER BY MIN(age)`

	ageRows, err := r.db.QueryContext(ctx, ageQuery)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get age dist: %w", op, err)
	}
	defer ageRows.Close()

	for ageRows.Next() {
		var a hrm.AgeGroup
		if err := ageRows.Scan(&a.AgeRange, &a.Count, &a.Percent); err != nil {
			return nil, fmt.Errorf("%s: failed to scan age: %w", op, err)
		}
		report.AgeDistribution = append(report.AgeDistribution, a)
	}

	// Gender distribution
	genderQuery := `
		SELECT
			COALESCE(c.gender, 'unknown'),
			COUNT(*),
			ROUND(COUNT(*)::numeric / (SELECT COUNT(*) FROM hrm_employees WHERE employment_status = 'active') * 100, 1)
		FROM hrm_employees e
		JOIN contacts c ON e.contact_id = c.id
		WHERE e.employment_status = 'active'
		GROUP BY COALESCE(c.gender, 'unknown')`

	genderRows, err := r.db.QueryContext(ctx, genderQuery)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get gender dist: %w", op, err)
	}
	defer genderRows.Close()

	for genderRows.Next() {
		var g hrm.GenderCount
		if err := genderRows.Scan(&g.Gender, &g.Count, &g.Percent); err != nil {
			return nil, fmt.Errorf("%s: failed to scan gender: %w", op, err)
		}
		report.GenderDistribution = append(report.GenderDistribution, g)
	}

	// Tenure distribution
	tenureQuery := `
		WITH tenures AS (
			SELECT EXTRACT(YEAR FROM AGE(hire_date)) as tenure
			FROM hrm_employees
			WHERE employment_status = 'active'
		)
		SELECT
			CASE
				WHEN tenure < 1 THEN 'Less than 1 year'
				WHEN tenure BETWEEN 1 AND 2 THEN '1-2 years'
				WHEN tenure BETWEEN 3 AND 5 THEN '3-5 years'
				WHEN tenure BETWEEN 6 AND 10 THEN '6-10 years'
				ELSE 'More than 10 years'
			END as tenure_range,
			COUNT(*),
			ROUND(COUNT(*)::numeric / (SELECT COUNT(*) FROM tenures) * 100, 1)
		FROM tenures
		GROUP BY
			CASE
				WHEN tenure < 1 THEN 'Less than 1 year'
				WHEN tenure BETWEEN 1 AND 2 THEN '1-2 years'
				WHEN tenure BETWEEN 3 AND 5 THEN '3-5 years'
				WHEN tenure BETWEEN 6 AND 10 THEN '6-10 years'
				ELSE 'More than 10 years'
			END
		ORDER BY MIN(tenure)`

	tenureRows, err := r.db.QueryContext(ctx, tenureQuery)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get tenure dist: %w", op, err)
	}
	defer tenureRows.Close()

	for tenureRows.Next() {
		var t hrm.TenureGroup
		if err := tenureRows.Scan(&t.TenureRange, &t.Count, &t.Percent); err != nil {
			return nil, fmt.Errorf("%s: failed to scan tenure: %w", op, err)
		}
		report.TenureDistribution = append(report.TenureDistribution, t)
	}

	return report, nil
}
