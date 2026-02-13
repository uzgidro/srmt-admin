package repo

import (
	"context"
	"fmt"
	"math"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/hrm/analytics"
	"strings"
	"time"
)

func defaultDateRange() (string, string) {
	now := time.Now()
	end := now.Format("2006-01-02")
	start := now.AddDate(-1, 0, 0).Format("2006-01-02")
	return start, end
}

func filterDates(f dto.ReportFilter) (string, string) {
	start, end := defaultDateRange()
	if f.StartDate != nil {
		start = *f.StartDate
	}
	if f.EndDate != nil {
		end = *f.EndDate
	}
	return start, end
}

// ==================== Dashboard ====================

func (r *Repo) GetAnalyticsDashboard(ctx context.Context, filter dto.ReportFilter) (*analytics.Dashboard, error) {
	const op = "repo.GetAnalyticsDashboard"

	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	monthEnd := now.Format("2006-01-02")

	dash := &analytics.Dashboard{}

	// Total active employees
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM personnel_records WHERE status != 'dismissed'`).Scan(&dash.TotalEmployees)
	if err != nil {
		return nil, fmt.Errorf("%s: total: %w", op, err)
	}

	// New hires this month
	err = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM personnel_records WHERE hire_date >= $1 AND hire_date <= $2`, monthStart, monthEnd).Scan(&dash.NewHiresMonth)
	if err != nil {
		return nil, fmt.Errorf("%s: new hires: %w", op, err)
	}

	// Terminations this month
	err = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM personnel_records WHERE status = 'dismissed' AND updated_at >= $1 AND updated_at <= $2`, monthStart, monthEnd).Scan(&dash.TerminationsMonth)
	if err != nil {
		return nil, fmt.Errorf("%s: terminations: %w", op, err)
	}

	// Turnover rate
	if dash.TotalEmployees > 0 {
		dash.TurnoverRate = math.Round(float64(dash.TerminationsMonth)/float64(dash.TotalEmployees+dash.TerminationsMonth)*100*100) / 100
	}

	// Avg tenure
	err = r.db.QueryRowContext(ctx, `
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (NOW() - hire_date)) / 31557600), 0)
		FROM personnel_records WHERE status != 'dismissed'
	`).Scan(&dash.AvgTenureYears)
	if err != nil {
		return nil, fmt.Errorf("%s: avg tenure: %w", op, err)
	}
	dash.AvgTenureYears = math.Round(dash.AvgTenureYears*100) / 100

	// Avg age
	err = r.db.QueryRowContext(ctx, `
		SELECT COALESCE(AVG(EXTRACT(YEAR FROM AGE(NOW(), c.dob))), 0)
		FROM personnel_records pr
		JOIN contacts c ON c.id = pr.employee_id
		WHERE pr.status != 'dismissed' AND c.dob IS NOT NULL
	`).Scan(&dash.AvgAge)
	if err != nil {
		return nil, fmt.Errorf("%s: avg age: %w", op, err)
	}
	dash.AvgAge = math.Round(dash.AvgAge*100) / 100

	// Gender distribution (no gender field — stub)
	dash.GenderDistribution = &analytics.GenderDistribution{
		Male:   0,
		Female: 0,
		Total:  dash.TotalEmployees,
	}

	// Age distribution
	dash.AgeDistribution, err = r.queryAgeDistribution(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("%s: age dist: %w", op, err)
	}

	// Tenure distribution
	dash.TenureDistribution, err = r.queryTenureDistribution(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("%s: tenure dist: %w", op, err)
	}

	// Department headcount
	dash.DepartmentHeadcount, err = r.queryDepartmentHeadcount(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: dept headcount: %w", op, err)
	}

	// Position headcount
	dash.PositionHeadcount, err = r.queryPositionHeadcount(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: pos headcount: %w", op, err)
	}

	return dash, nil
}

// ==================== Headcount ====================

func (r *Repo) GetHeadcountReport(ctx context.Context, filter dto.ReportFilter) (*analytics.HeadcountReport, error) {
	const op = "repo.GetHeadcountReport"

	report := &analytics.HeadcountReport{}

	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, "pr.status != 'dismissed'")

	if filter.DepartmentID != nil {
		conditions = append(conditions, fmt.Sprintf("pr.department_id = $%d", argIdx))
		args = append(args, *filter.DepartmentID)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	query := `SELECT COUNT(*) FROM personnel_records pr` + where
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&report.TotalEmployees)
	if err != nil {
		return nil, fmt.Errorf("%s: total: %w", op, err)
	}

	report.ByDepartment, err = r.queryDepartmentHeadcount(ctx, filter.DepartmentID)
	if err != nil {
		return nil, fmt.Errorf("%s: by dept: %w", op, err)
	}

	report.ByPosition, err = r.queryPositionHeadcount(ctx, filter.DepartmentID)
	if err != nil {
		return nil, fmt.Errorf("%s: by pos: %w", op, err)
	}

	return report, nil
}

func (r *Repo) GetHeadcountTrend(ctx context.Context, filter dto.ReportFilter) (*analytics.HeadcountTrend, error) {
	const op = "repo.GetHeadcountTrend"

	start, end := filterDates(filter)

	query := `
		SELECT EXTRACT(YEAR FROM m)::int AS y, EXTRACT(MONTH FROM m)::int AS mo,
			(SELECT COUNT(DISTINCT pr2.employee_id) FROM personnel_records pr2
			 WHERE pr2.hire_date <= m + INTERVAL '1 month' - INTERVAL '1 day'
			   AND (pr2.status != 'dismissed' OR pr2.updated_at > m + INTERVAL '1 month' - INTERVAL '1 day')
			) AS value
		FROM generate_series($1::date, $2::date, '1 month') m
		ORDER BY m`

	rows, err := r.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	trend := &analytics.HeadcountTrend{}
	for rows.Next() {
		p := &analytics.TrendPoint{}
		if err := rows.Scan(&p.Year, &p.Month, &p.Value); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		trend.Points = append(trend.Points, p)
	}
	if trend.Points == nil {
		trend.Points = []*analytics.TrendPoint{}
	}
	return trend, rows.Err()
}

// ==================== Turnover ====================

func (r *Repo) GetTurnoverReport(ctx context.Context, filter dto.ReportFilter) (*analytics.TurnoverReport, error) {
	const op = "repo.GetTurnoverReport"

	start, end := filterDates(filter)

	report := &analytics.TurnoverReport{
		PeriodStart: start,
		PeriodEnd:   end,
		ByReason:    []*analytics.DistributionItem{}, // no termination_reason field
	}

	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("pr.status = 'dismissed' AND pr.updated_at >= $%d AND pr.updated_at <= $%d", argIdx, argIdx+1))
	args = append(args, start, end)
	argIdx += 2

	if filter.DepartmentID != nil {
		conditions = append(conditions, fmt.Sprintf("pr.department_id = $%d", argIdx))
		args = append(args, *filter.DepartmentID)
		argIdx++
	}

	where := " WHERE " + strings.Join(conditions, " AND ")

	// Total terminations
	query := `SELECT COUNT(*) FROM personnel_records pr` + where
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&report.TotalTerminations)
	if err != nil {
		return nil, fmt.Errorf("%s: total: %w", op, err)
	}

	// No voluntary/involuntary distinction available
	report.VoluntaryTerminations = 0
	report.InvoluntaryTerminations = 0

	// Avg tenure at termination
	tenureQuery := `
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (pr.updated_at - pr.hire_date)) / 31557600), 0)
		FROM personnel_records pr` + where
	err = r.db.QueryRowContext(ctx, tenureQuery, args...).Scan(&report.AvgTenureAtTermination)
	if err != nil {
		return nil, fmt.Errorf("%s: avg tenure: %w", op, err)
	}
	report.AvgTenureAtTermination = math.Round(report.AvgTenureAtTermination*100) / 100

	// Total active for rate calculation
	var totalActive int
	err = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM personnel_records WHERE status != 'dismissed'`).Scan(&totalActive)
	if err != nil {
		return nil, fmt.Errorf("%s: active: %w", op, err)
	}

	denominator := totalActive + report.TotalTerminations
	if denominator > 0 {
		report.TurnoverRate = math.Round(float64(report.TotalTerminations)/float64(denominator)*100*100) / 100
		report.RetentionRate = math.Round((100-report.TurnoverRate)*100) / 100
	} else {
		report.RetentionRate = 100
	}

	// By department
	deptQuery := `
		SELECT d.name, COUNT(*) AS terminations
		FROM personnel_records pr
		JOIN departments d ON d.id = pr.department_id` + where + `
		GROUP BY d.name ORDER BY terminations DESC`
	deptRows, err := r.db.QueryContext(ctx, deptQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: by dept: %w", op, err)
	}
	defer deptRows.Close()

	for deptRows.Next() {
		dt := &analytics.DepartmentTurnover{}
		if err := deptRows.Scan(&dt.Department, &dt.Terminations); err != nil {
			return nil, fmt.Errorf("%s: scan dept: %w", op, err)
		}
		if denominator > 0 {
			dt.TurnoverRate = math.Round(float64(dt.Terminations)/float64(denominator)*100*100) / 100
		}
		report.ByDepartment = append(report.ByDepartment, dt)
	}
	if report.ByDepartment == nil {
		report.ByDepartment = []*analytics.DepartmentTurnover{}
	}

	return report, deptRows.Err()
}

func (r *Repo) GetTurnoverTrend(ctx context.Context, filter dto.ReportFilter) (*analytics.TurnoverTrend, error) {
	const op = "repo.GetTurnoverTrend"

	start, end := filterDates(filter)

	query := `
		SELECT EXTRACT(YEAR FROM m)::int AS y, EXTRACT(MONTH FROM m)::int AS mo,
			(SELECT COUNT(*) FROM personnel_records pr
			 WHERE pr.status = 'dismissed'
			   AND pr.updated_at >= m
			   AND pr.updated_at < m + INTERVAL '1 month'
			) AS value
		FROM generate_series($1::date, $2::date, '1 month') m
		ORDER BY m`

	rows, err := r.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	trend := &analytics.TurnoverTrend{}
	for rows.Next() {
		p := &analytics.TrendPoint{}
		if err := rows.Scan(&p.Year, &p.Month, &p.Value); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		trend.Points = append(trend.Points, p)
	}
	if trend.Points == nil {
		trend.Points = []*analytics.TrendPoint{}
	}
	return trend, rows.Err()
}

// ==================== Attendance ====================

func (r *Repo) GetAttendanceReport(ctx context.Context, filter dto.ReportFilter) (*analytics.AttendanceReport, error) {
	const op = "repo.GetAttendanceReport"

	start, end := filterDates(filter)

	report := &analytics.AttendanceReport{
		PeriodStart: start,
		PeriodEnd:   end,
	}

	var deptCond string
	var args []interface{}
	args = append(args, start, end)
	argIdx := 3

	if filter.DepartmentID != nil {
		deptCond = fmt.Sprintf(" AND pr.department_id = $%d", argIdx)
		args = append(args, *filter.DepartmentID)
		argIdx++
	}

	// Total work days & avg attendance
	summaryQuery := fmt.Sprintf(`
		SELECT
			COUNT(DISTINCT te.date) AS total_days,
			COALESCE(COUNT(CASE WHEN te.status = 'present' THEN 1 END)::float / NULLIF(COUNT(*), 0) * 100, 0),
			COALESCE(COUNT(CASE WHEN te.status NOT IN ('present','holiday','day_off') THEN 1 END)::float / NULLIF(COUNT(*), 0) * 100, 0)
		FROM timesheet_entries te
		JOIN personnel_records pr ON pr.employee_id = te.employee_id
		WHERE te.date >= $1 AND te.date <= $2%s`, deptCond)

	err := r.db.QueryRowContext(ctx, summaryQuery, args...).Scan(&report.TotalWorkDays, &report.AvgAttendance, &report.AvgAbsence)
	if err != nil {
		return nil, fmt.Errorf("%s: summary: %w", op, err)
	}
	report.AvgAttendance = math.Round(report.AvgAttendance*100) / 100
	report.AvgAbsence = math.Round(report.AvgAbsence*100) / 100

	// By status
	statusQuery := fmt.Sprintf(`
		SELECT te.status, COUNT(*) AS cnt
		FROM timesheet_entries te
		JOIN personnel_records pr ON pr.employee_id = te.employee_id
		WHERE te.date >= $1 AND te.date <= $2%s
		GROUP BY te.status ORDER BY cnt DESC`, deptCond)

	statusRows, err := r.db.QueryContext(ctx, statusQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: by status: %w", op, err)
	}
	defer statusRows.Close()

	var totalEntries int
	var statusItems []*analytics.DistributionItem
	for statusRows.Next() {
		item := &analytics.DistributionItem{}
		if err := statusRows.Scan(&item.Label, &item.Count); err != nil {
			return nil, fmt.Errorf("%s: scan status: %w", op, err)
		}
		totalEntries += item.Count
		statusItems = append(statusItems, item)
	}
	if err := statusRows.Err(); err != nil {
		return nil, fmt.Errorf("%s: status rows: %w", op, err)
	}
	for _, item := range statusItems {
		if totalEntries > 0 {
			item.Percentage = math.Round(float64(item.Count)/float64(totalEntries)*100*100) / 100
		}
	}
	report.ByStatus = statusItems
	if report.ByStatus == nil {
		report.ByStatus = []*analytics.DistributionItem{}
	}

	// By department
	deptQuery := fmt.Sprintf(`
		SELECT d.name,
			COALESCE(COUNT(CASE WHEN te.status = 'present' THEN 1 END)::float / NULLIF(COUNT(*), 0) * 100, 0) AS attendance_rate,
			COALESCE(COUNT(CASE WHEN te.status NOT IN ('present','holiday','day_off') THEN 1 END)::float / NULLIF(COUNT(*), 0) * 100, 0) AS absence_rate
		FROM timesheet_entries te
		JOIN personnel_records pr ON pr.employee_id = te.employee_id
		JOIN departments d ON d.id = pr.department_id
		WHERE te.date >= $1 AND te.date <= $2%s
		GROUP BY d.name ORDER BY attendance_rate DESC`, deptCond)

	deptRows, err := r.db.QueryContext(ctx, deptQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: by dept: %w", op, err)
	}
	defer deptRows.Close()

	for deptRows.Next() {
		da := &analytics.DepartmentAttendance{}
		if err := deptRows.Scan(&da.Department, &da.AttendanceRate, &da.AbsenceRate); err != nil {
			return nil, fmt.Errorf("%s: scan dept: %w", op, err)
		}
		da.AttendanceRate = math.Round(da.AttendanceRate*100) / 100
		da.AbsenceRate = math.Round(da.AbsenceRate*100) / 100
		report.ByDepartment = append(report.ByDepartment, da)
	}
	if report.ByDepartment == nil {
		report.ByDepartment = []*analytics.DepartmentAttendance{}
	}

	return report, deptRows.Err()
}

// ==================== Salary ====================

func (r *Repo) GetSalaryReport(ctx context.Context, filter dto.ReportFilter) (*analytics.SalaryReport, error) {
	const op = "repo.GetSalaryReport"

	start, end := filterDates(filter)

	report := &analytics.SalaryReport{
		PeriodStart: start,
		PeriodEnd:   end,
	}

	var deptCond string
	var args []interface{}
	args = append(args, start, end)
	argIdx := 3

	if filter.DepartmentID != nil {
		deptCond = fmt.Sprintf(" AND pr.department_id = $%d", argIdx)
		args = append(args, *filter.DepartmentID)
		argIdx++
	}

	summaryQuery := fmt.Sprintf(`
		SELECT
			COALESCE(SUM(s.net_salary), 0),
			COALESCE(AVG(s.net_salary), 0),
			COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY s.net_salary), 0),
			COALESCE(MIN(s.net_salary), 0),
			COALESCE(MAX(s.net_salary), 0)
		FROM salaries s
		JOIN personnel_records pr ON pr.employee_id = s.employee_id
		WHERE s.status IN ('paid', 'approved')
		  AND MAKE_DATE(s.period_year, s.period_month, 1) >= $1::date
		  AND MAKE_DATE(s.period_year, s.period_month, 1) <= $2::date%s`, deptCond)

	err := r.db.QueryRowContext(ctx, summaryQuery, args...).Scan(
		&report.TotalPayroll, &report.AvgSalary, &report.MedianSalary, &report.MinSalary, &report.MaxSalary,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: summary: %w", op, err)
	}
	report.TotalPayroll = math.Round(report.TotalPayroll*100) / 100
	report.AvgSalary = math.Round(report.AvgSalary*100) / 100
	report.MedianSalary = math.Round(report.MedianSalary*100) / 100

	// By department
	deptQuery := fmt.Sprintf(`
		SELECT d.name,
			COALESCE(AVG(s.net_salary), 0),
			COALESCE(SUM(s.net_salary), 0),
			COUNT(DISTINCT s.employee_id)
		FROM salaries s
		JOIN personnel_records pr ON pr.employee_id = s.employee_id
		JOIN departments d ON d.id = pr.department_id
		WHERE s.status IN ('paid', 'approved')
		  AND MAKE_DATE(s.period_year, s.period_month, 1) >= $1::date
		  AND MAKE_DATE(s.period_year, s.period_month, 1) <= $2::date%s
		GROUP BY d.name ORDER BY total_payroll DESC`, deptCond)

	// Fix alias — use index
	deptQuery = fmt.Sprintf(`
		SELECT d.name,
			COALESCE(AVG(s.net_salary), 0) AS avg_sal,
			COALESCE(SUM(s.net_salary), 0) AS total_pay,
			COUNT(DISTINCT s.employee_id) AS hc
		FROM salaries s
		JOIN personnel_records pr ON pr.employee_id = s.employee_id
		JOIN departments d ON d.id = pr.department_id
		WHERE s.status IN ('paid', 'approved')
		  AND MAKE_DATE(s.period_year, s.period_month, 1) >= $1::date
		  AND MAKE_DATE(s.period_year, s.period_month, 1) <= $2::date%s
		GROUP BY d.name ORDER BY total_pay DESC`, deptCond)

	deptRows, err := r.db.QueryContext(ctx, deptQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: by dept: %w", op, err)
	}
	defer deptRows.Close()

	for deptRows.Next() {
		ds := &analytics.DepartmentSalary{}
		if err := deptRows.Scan(&ds.Department, &ds.AvgSalary, &ds.TotalPayroll, &ds.Headcount); err != nil {
			return nil, fmt.Errorf("%s: scan dept: %w", op, err)
		}
		ds.AvgSalary = math.Round(ds.AvgSalary*100) / 100
		ds.TotalPayroll = math.Round(ds.TotalPayroll*100) / 100
		report.ByDepartment = append(report.ByDepartment, ds)
	}
	if report.ByDepartment == nil {
		report.ByDepartment = []*analytics.DepartmentSalary{}
	}

	return report, deptRows.Err()
}

func (r *Repo) GetSalaryTrend(ctx context.Context, filter dto.ReportFilter) (*analytics.SalaryTrend, error) {
	const op = "repo.GetSalaryTrend"

	start, end := filterDates(filter)

	query := `
		SELECT s.period_year, s.period_month, COALESCE(AVG(s.net_salary), 0)
		FROM salaries s
		WHERE s.status IN ('paid', 'approved')
		  AND MAKE_DATE(s.period_year, s.period_month, 1) >= $1::date
		  AND MAKE_DATE(s.period_year, s.period_month, 1) <= $2::date
		GROUP BY s.period_year, s.period_month
		ORDER BY s.period_year, s.period_month`

	rows, err := r.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	trend := &analytics.SalaryTrend{}
	for rows.Next() {
		p := &analytics.TrendPoint{}
		if err := rows.Scan(&p.Year, &p.Month, &p.Value); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		p.Value = math.Round(p.Value*100) / 100
		trend.Points = append(trend.Points, p)
	}
	if trend.Points == nil {
		trend.Points = []*analytics.TrendPoint{}
	}
	return trend, rows.Err()
}

// ==================== Performance ====================

func (r *Repo) GetPerformanceAnalytics(ctx context.Context, filter dto.ReportFilter) (*analytics.PerformanceAnalytics, error) {
	const op = "repo.GetPerformanceAnalytics"

	start, end := filterDates(filter)

	result := &analytics.PerformanceAnalytics{}

	var deptCond string
	var args []interface{}
	args = append(args, start, end)
	argIdx := 3

	if filter.DepartmentID != nil {
		deptCond = fmt.Sprintf(" AND pr.department_id = $%d", argIdx)
		args = append(args, *filter.DepartmentID)
		argIdx++
	}

	// Total reviews and avg rating
	summaryQuery := fmt.Sprintf(`
		SELECT COUNT(*), COALESCE(AVG(rv.final_rating), 0)
		FROM performance_reviews rv
		JOIN personnel_records pr ON pr.employee_id = rv.employee_id
		WHERE rv.period_start >= $1::date AND rv.period_end <= $2::date%s`, deptCond)

	err := r.db.QueryRowContext(ctx, summaryQuery, args...).Scan(&result.TotalReviews, &result.AvgRating)
	if err != nil {
		return nil, fmt.Errorf("%s: summary: %w", op, err)
	}
	result.AvgRating = math.Round(result.AvgRating*100) / 100

	// Rating distribution (1-5)
	ratingQuery := fmt.Sprintf(`
		SELECT rv.final_rating::text, COUNT(*)
		FROM performance_reviews rv
		JOIN personnel_records pr ON pr.employee_id = rv.employee_id
		WHERE rv.final_rating IS NOT NULL
		  AND rv.period_start >= $1::date AND rv.period_end <= $2::date%s
		GROUP BY rv.final_rating ORDER BY rv.final_rating`, deptCond)

	ratingRows, err := r.db.QueryContext(ctx, ratingQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: rating dist: %w", op, err)
	}
	defer ratingRows.Close()

	var ratedTotal int
	var ratingItems []*analytics.DistributionItem
	for ratingRows.Next() {
		item := &analytics.DistributionItem{}
		if err := ratingRows.Scan(&item.Label, &item.Count); err != nil {
			return nil, fmt.Errorf("%s: scan rating: %w", op, err)
		}
		ratedTotal += item.Count
		ratingItems = append(ratingItems, item)
	}
	if err := ratingRows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rating rows: %w", op, err)
	}
	for _, item := range ratingItems {
		if ratedTotal > 0 {
			item.Percentage = math.Round(float64(item.Count)/float64(ratedTotal)*100*100) / 100
		}
	}
	result.RatingDistribution = ratingItems
	if result.RatingDistribution == nil {
		result.RatingDistribution = []*analytics.DistributionItem{}
	}

	// Goal completion
	goalQuery := fmt.Sprintf(`
		SELECT COUNT(*), COUNT(CASE WHEN g.status = 'completed' THEN 1 END)
		FROM performance_goals g
		JOIN personnel_records pr ON pr.employee_id = g.employee_id
		WHERE g.due_date >= $1::date AND g.due_date <= $2::date%s`, deptCond)

	gc := &analytics.GoalCompletion{}
	err = r.db.QueryRowContext(ctx, goalQuery, args...).Scan(&gc.Total, &gc.Completed)
	if err != nil {
		return nil, fmt.Errorf("%s: goals: %w", op, err)
	}
	if gc.Total > 0 {
		gc.Rate = math.Round(float64(gc.Completed)/float64(gc.Total)*100*100) / 100
	}
	result.GoalCompletion = gc

	// By department
	byDeptQuery := fmt.Sprintf(`
		SELECT d.name,
			COALESCE(AVG(rv.final_rating), 0),
			COALESCE(
				COUNT(CASE WHEN g.status = 'completed' THEN 1 END)::float / NULLIF(COUNT(g.id), 0) * 100,
				0
			)
		FROM performance_reviews rv
		JOIN personnel_records pr ON pr.employee_id = rv.employee_id
		JOIN departments d ON d.id = pr.department_id
		LEFT JOIN performance_goals g ON g.employee_id = rv.employee_id
		WHERE rv.period_start >= $1::date AND rv.period_end <= $2::date%s
		GROUP BY d.name ORDER BY d.name`, deptCond)

	byDeptRows, err := r.db.QueryContext(ctx, byDeptQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: by dept: %w", op, err)
	}
	defer byDeptRows.Close()

	for byDeptRows.Next() {
		dp := &analytics.DepartmentPerformance{}
		if err := byDeptRows.Scan(&dp.Department, &dp.AvgRating, &dp.GoalRate); err != nil {
			return nil, fmt.Errorf("%s: scan dept: %w", op, err)
		}
		dp.AvgRating = math.Round(dp.AvgRating*100) / 100
		dp.GoalRate = math.Round(dp.GoalRate*100) / 100
		result.ByDepartment = append(result.ByDepartment, dp)
	}
	if result.ByDepartment == nil {
		result.ByDepartment = []*analytics.DepartmentPerformance{}
	}

	return result, byDeptRows.Err()
}

// ==================== Training ====================

func (r *Repo) GetTrainingAnalytics(ctx context.Context, filter dto.ReportFilter) (*analytics.TrainingAnalytics, error) {
	const op = "repo.GetTrainingAnalytics"

	start, end := filterDates(filter)

	result := &analytics.TrainingAnalytics{}

	// Summary
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM trainings WHERE start_date >= $1 AND start_date <= $2
	`, start, end).Scan(&result.TotalTrainings)
	if err != nil {
		return nil, fmt.Errorf("%s: total: %w", op, err)
	}

	err = r.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT tp.employee_id)
		FROM training_participants tp
		JOIN trainings t ON t.id = tp.training_id
		WHERE t.start_date >= $1 AND t.start_date <= $2
	`, start, end).Scan(&result.TotalParticipants)
	if err != nil {
		return nil, fmt.Errorf("%s: participants: %w", op, err)
	}

	// Completion rate
	var totalP, completedP int
	err = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COUNT(CASE WHEN tp.status = 'completed' THEN 1 END)
		FROM training_participants tp
		JOIN trainings t ON t.id = tp.training_id
		WHERE t.start_date >= $1 AND t.start_date <= $2
	`, start, end).Scan(&totalP, &completedP)
	if err != nil {
		return nil, fmt.Errorf("%s: completion: %w", op, err)
	}
	if totalP > 0 {
		result.CompletionRate = math.Round(float64(completedP)/float64(totalP)*100*100) / 100
	}

	// By status
	statusRows, err := r.db.QueryContext(ctx, `
		SELECT t.status, COUNT(*)
		FROM trainings t
		WHERE t.start_date >= $1 AND t.start_date <= $2
		GROUP BY t.status ORDER BY COUNT(*) DESC
	`, start, end)
	if err != nil {
		return nil, fmt.Errorf("%s: by status: %w", op, err)
	}
	defer statusRows.Close()

	for statusRows.Next() {
		item := &analytics.DistributionItem{}
		if err := statusRows.Scan(&item.Label, &item.Count); err != nil {
			return nil, fmt.Errorf("%s: scan status: %w", op, err)
		}
		if result.TotalTrainings > 0 {
			item.Percentage = math.Round(float64(item.Count)/float64(result.TotalTrainings)*100*100) / 100
		}
		result.ByStatus = append(result.ByStatus, item)
	}
	if err := statusRows.Err(); err != nil {
		return nil, fmt.Errorf("%s: status rows: %w", op, err)
	}
	if result.ByStatus == nil {
		result.ByStatus = []*analytics.DistributionItem{}
	}

	// By type
	typeRows, err := r.db.QueryContext(ctx, `
		SELECT t.type, COUNT(*)
		FROM trainings t
		WHERE t.start_date >= $1 AND t.start_date <= $2
		GROUP BY t.type ORDER BY COUNT(*) DESC
	`, start, end)
	if err != nil {
		return nil, fmt.Errorf("%s: by type: %w", op, err)
	}
	defer typeRows.Close()

	for typeRows.Next() {
		item := &analytics.DistributionItem{}
		if err := typeRows.Scan(&item.Label, &item.Count); err != nil {
			return nil, fmt.Errorf("%s: scan type: %w", op, err)
		}
		if result.TotalTrainings > 0 {
			item.Percentage = math.Round(float64(item.Count)/float64(result.TotalTrainings)*100*100) / 100
		}
		result.ByType = append(result.ByType, item)
	}
	if err := typeRows.Err(); err != nil {
		return nil, fmt.Errorf("%s: type rows: %w", op, err)
	}
	if result.ByType == nil {
		result.ByType = []*analytics.DistributionItem{}
	}

	return result, nil
}

// ==================== Demographics ====================

func (r *Repo) GetDemographicsReport(ctx context.Context, filter dto.ReportFilter) (*analytics.DemographicsReport, error) {
	const op = "repo.GetDemographicsReport"

	report := &analytics.DemographicsReport{}

	var deptCond string
	if filter.DepartmentID != nil {
		deptCond = fmt.Sprintf(" AND pr.department_id = %d", *filter.DepartmentID)
	}

	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM personnel_records pr WHERE pr.status != 'dismissed'`+deptCond,
	).Scan(&report.TotalEmployees)
	if err != nil {
		return nil, fmt.Errorf("%s: total: %w", op, err)
	}

	// Avg age
	err = r.db.QueryRowContext(ctx, `
		SELECT COALESCE(AVG(EXTRACT(YEAR FROM AGE(NOW(), c.dob))), 0)
		FROM personnel_records pr
		JOIN contacts c ON c.id = pr.employee_id
		WHERE pr.status != 'dismissed' AND c.dob IS NOT NULL`+deptCond,
	).Scan(&report.AvgAge)
	if err != nil {
		return nil, fmt.Errorf("%s: avg age: %w", op, err)
	}
	report.AvgAge = math.Round(report.AvgAge*100) / 100

	report.AgeDistribution, err = r.queryAgeDistribution(ctx, deptCond)
	if err != nil {
		return nil, fmt.Errorf("%s: age dist: %w", op, err)
	}

	report.TenureDistribution, err = r.queryTenureDistribution(ctx, deptCond)
	if err != nil {
		return nil, fmt.Errorf("%s: tenure dist: %w", op, err)
	}

	return report, nil
}

// ==================== Helpers ====================

func (r *Repo) queryAgeDistribution(ctx context.Context, extraCond string) ([]*analytics.DistributionItem, error) {
	query := `
		SELECT bucket, COUNT(*) AS cnt FROM (
			SELECT CASE
				WHEN EXTRACT(YEAR FROM AGE(NOW(), c.dob)) < 25 THEN '< 25'
				WHEN EXTRACT(YEAR FROM AGE(NOW(), c.dob)) < 35 THEN '25-34'
				WHEN EXTRACT(YEAR FROM AGE(NOW(), c.dob)) < 45 THEN '35-44'
				WHEN EXTRACT(YEAR FROM AGE(NOW(), c.dob)) < 55 THEN '45-54'
				ELSE '55+'
			END AS bucket
			FROM personnel_records pr
			JOIN contacts c ON c.id = pr.employee_id
			WHERE pr.status != 'dismissed' AND c.dob IS NOT NULL` + extraCond + `
		) sub
		GROUP BY bucket ORDER BY bucket`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var total int
	var items []*analytics.DistributionItem
	for rows.Next() {
		item := &analytics.DistributionItem{}
		if err := rows.Scan(&item.Label, &item.Count); err != nil {
			return nil, err
		}
		total += item.Count
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, item := range items {
		if total > 0 {
			item.Percentage = math.Round(float64(item.Count)/float64(total)*100*100) / 100
		}
	}
	if items == nil {
		items = []*analytics.DistributionItem{}
	}
	return items, nil
}

func (r *Repo) queryTenureDistribution(ctx context.Context, extraCond string) ([]*analytics.DistributionItem, error) {
	query := `
		SELECT bucket, COUNT(*) AS cnt FROM (
			SELECT CASE
				WHEN EXTRACT(EPOCH FROM (NOW() - pr.hire_date)) / 31557600 < 1 THEN '< 1 year'
				WHEN EXTRACT(EPOCH FROM (NOW() - pr.hire_date)) / 31557600 < 3 THEN '1-3 years'
				WHEN EXTRACT(EPOCH FROM (NOW() - pr.hire_date)) / 31557600 < 5 THEN '3-5 years'
				WHEN EXTRACT(EPOCH FROM (NOW() - pr.hire_date)) / 31557600 < 10 THEN '5-10 years'
				ELSE '10+ years'
			END AS bucket
			FROM personnel_records pr
			WHERE pr.status != 'dismissed'` + extraCond + `
		) sub
		GROUP BY bucket ORDER BY bucket`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var total int
	var items []*analytics.DistributionItem
	for rows.Next() {
		item := &analytics.DistributionItem{}
		if err := rows.Scan(&item.Label, &item.Count); err != nil {
			return nil, err
		}
		total += item.Count
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, item := range items {
		if total > 0 {
			item.Percentage = math.Round(float64(item.Count)/float64(total)*100*100) / 100
		}
	}
	if items == nil {
		items = []*analytics.DistributionItem{}
	}
	return items, nil
}

func (r *Repo) queryDepartmentHeadcount(ctx context.Context, deptID *int64) ([]*analytics.DepartmentHeadcount, error) {
	query := `
		SELECT d.id, d.name, COUNT(*) AS headcount
		FROM personnel_records pr
		JOIN departments d ON d.id = pr.department_id
		WHERE pr.status != 'dismissed'`

	var args []interface{}
	if deptID != nil {
		query += " AND pr.department_id = $1"
		args = append(args, *deptID)
	}
	query += " GROUP BY d.id, d.name ORDER BY headcount DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*analytics.DepartmentHeadcount
	for rows.Next() {
		item := &analytics.DepartmentHeadcount{}
		if err := rows.Scan(&item.DepartmentID, &item.DepartmentName, &item.Headcount); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if items == nil {
		items = []*analytics.DepartmentHeadcount{}
	}
	return items, rows.Err()
}

func (r *Repo) queryPositionHeadcount(ctx context.Context, deptID *int64) ([]*analytics.PositionHeadcount, error) {
	query := `
		SELECT p.id, p.name, COUNT(*) AS cnt
		FROM personnel_records pr
		JOIN positions p ON p.id = pr.position_id
		WHERE pr.status != 'dismissed'`

	var args []interface{}
	if deptID != nil {
		query += " AND pr.department_id = $1"
		args = append(args, *deptID)
	}
	query += " GROUP BY p.id, p.name ORDER BY cnt DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*analytics.PositionHeadcount
	for rows.Next() {
		item := &analytics.PositionHeadcount{}
		if err := rows.Scan(&item.PositionID, &item.PositionName, &item.Count); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if items == nil {
		items = []*analytics.PositionHeadcount{}
	}
	return items, rows.Err()
}
