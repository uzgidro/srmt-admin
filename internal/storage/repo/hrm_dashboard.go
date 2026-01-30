package repo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"srmt-admin/internal/lib/dto/hrm"
	hrmmodel "srmt-admin/internal/lib/model/hrm"
)

// GetDashboard retrieves HRM dashboard data
func (r *Repo) GetDashboard(ctx context.Context, filter hrm.DashboardFilter) (*hrmmodel.Dashboard, error) {
	const op = "storage.repo.GetDashboard"

	dashboard := &hrmmodel.Dashboard{
		GeneratedAt: time.Now(),
	}

	// Get employee stats
	empStats, err := r.GetEmployeeStats(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get employee stats: %w", op, err)
	}
	dashboard.EmployeeStats = *empStats

	// Get vacation stats
	vacStats, err := r.GetVacationStats(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get vacation stats: %w", op, err)
	}
	dashboard.VacationStats = *vacStats

	// Get recruiting stats
	recStats, err := r.GetRecruitingStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get recruiting stats: %w", op, err)
	}
	dashboard.RecruitingStats = *recStats

	// Get training stats
	trainStats, err := r.GetTrainingStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get training stats: %w", op, err)
	}
	dashboard.TrainingStats = *trainStats

	// Get performance stats
	perfStats, err := r.GetPerformanceStats(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get performance stats: %w", op, err)
	}
	dashboard.PerformanceStats = *perfStats

	// Get quick counts
	dashboard.PendingApprovals, _ = r.CountPendingApprovals(ctx)
	dashboard.ExpiringDocuments, _ = r.CountExpiringDocuments(ctx, 30)
	dashboard.UpcomingReviews, _ = r.CountUpcomingReviews(ctx, 30)

	// Get upcoming birthdays
	dashboard.UpcomingBirthdays, _ = r.GetUpcomingBirthdays(ctx, 7)

	// Get recent hires
	recentHires, _ := r.GetRecentHires(ctx, 30, 10)
	for _, emp := range recentHires {
		if emp != nil {
			dashboard.RecentHires = append(dashboard.RecentHires, *emp)
		}
	}

	return dashboard, nil
}

// GetEmployeeStats retrieves employee statistics
func (r *Repo) GetEmployeeStats(ctx context.Context, filter hrm.DashboardFilter) (*hrmmodel.EmployeeStats, error) {
	const op = "storage.repo.GetEmployeeStats"

	stats := &hrmmodel.EmployeeStats{}

	// Total and active employees
	const countQuery = `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE e.employment_status = 'active') as active,
			COUNT(*) FILTER (WHERE e.employment_status = 'on_leave') as on_leave,
			COUNT(*) FILTER (WHERE e.employment_status = 'terminated' AND EXTRACT(YEAR FROM e.termination_date) = $1) as terminated_this_year,
			COUNT(*) FILTER (WHERE EXTRACT(YEAR FROM e.hire_date) = $1) as hired_this_year
		FROM hrm_employees e
		LEFT JOIN contacts c ON e.contact_id = c.id
		WHERE 1=1`

	year := time.Now().Year()
	if filter.Year != nil {
		year = *filter.Year
	}

	err := r.db.QueryRowContext(ctx, countQuery, year).Scan(
		&stats.TotalEmployees,
		&stats.ActiveEmployees,
		&stats.OnLeaveCount,
		&stats.TerminatedThisYear,
		&stats.HiredThisYear,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get employee counts: %w", op, err)
	}

	// Calculate turnover rate
	if stats.ActiveEmployees > 0 {
		avgEmployees := float64(stats.ActiveEmployees+stats.TerminatedThisYear) / 2
		if avgEmployees > 0 {
			stats.TurnoverRate = float64(stats.TerminatedThisYear) / avgEmployees * 100
		}
	}

	// Get distribution by department
	const deptQuery = `
		SELECT d.name, COUNT(*)
		FROM hrm_employees e
		JOIN contacts c ON e.contact_id = c.id
		JOIN departments d ON c.department_id = d.id
		WHERE e.employment_status = 'active'
		GROUP BY d.id, d.name
		ORDER BY COUNT(*) DESC`

	rows, err := r.db.QueryContext(ctx, deptQuery)
	if err != nil {
		return stats, nil // Non-critical, continue
	}
	defer rows.Close()

	stats.ByDepartment = make(map[string]int)
	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err == nil {
			stats.ByDepartment[name] = count
		}
	}

	// Get distribution by employment type
	const typeQuery = `
		SELECT employment_type, COUNT(*)
		FROM hrm_employees
		WHERE employment_status = 'active'
		GROUP BY employment_type`

	rows2, err := r.db.QueryContext(ctx, typeQuery)
	if err != nil {
		return stats, nil
	}
	defer rows2.Close()

	stats.ByEmploymentType = make(map[string]int)
	for rows2.Next() {
		var empType string
		var count int
		if err := rows2.Scan(&empType, &count); err == nil {
			stats.ByEmploymentType[empType] = count
		}
	}

	return stats, nil
}

// GetVacationStats retrieves vacation statistics
func (r *Repo) GetVacationStats(ctx context.Context, filter hrm.DashboardFilter) (*hrmmodel.VacationStats, error) {
	const op = "storage.repo.GetVacationStats"

	stats := &hrmmodel.VacationStats{}
	today := time.Now()

	// Pending requests
	const pendingQuery = `SELECT COUNT(*) FROM hrm_vacations WHERE status = 'pending'`
	r.db.QueryRowContext(ctx, pendingQuery).Scan(&stats.PendingRequests)

	// Approved this month
	const approvedQuery = `
		SELECT COUNT(*) FROM hrm_vacations
		WHERE status = 'approved'
		AND EXTRACT(YEAR FROM approved_at) = $1
		AND EXTRACT(MONTH FROM approved_at) = $2`
	r.db.QueryRowContext(ctx, approvedQuery, today.Year(), int(today.Month())).Scan(&stats.ApprovedThisMonth)

	// On vacation today
	const todayQuery = `
		SELECT COUNT(*) FROM hrm_vacations
		WHERE status = 'approved'
		AND $1 BETWEEN start_date AND end_date`
	r.db.QueryRowContext(ctx, todayQuery, today).Scan(&stats.OnVacationToday)

	// Average days used
	const avgQuery = `
		SELECT COALESCE(AVG(used_days), 0)
		FROM hrm_vacation_balances
		WHERE year = $1`
	r.db.QueryRowContext(ctx, avgQuery, today.Year()).Scan(&stats.AverageVacationDays)

	// Employees with low balance (<5 days)
	const lowBalQuery = `
		SELECT COUNT(DISTINCT employee_id)
		FROM hrm_vacation_balances
		WHERE year = $1
		AND vacation_type_id = (SELECT id FROM hrm_vacation_types WHERE code = 'ANNUAL')
		AND (entitled_days + carried_over_days + adjustment_days - used_days) < 5`
	r.db.QueryRowContext(ctx, lowBalQuery, today.Year()).Scan(&stats.EmployeesLowBalance)

	return stats, nil
}

// GetRecruitingStats retrieves recruiting statistics
func (r *Repo) GetRecruitingStats(ctx context.Context) (*hrmmodel.RecruitingStats, error) {
	stats := &hrmmodel.RecruitingStats{}
	today := time.Now()

	const query = `
		SELECT
			(SELECT COUNT(*) FROM hrm_vacancies WHERE status = 'open') as open_vacancies,
			(SELECT COUNT(*) FROM hrm_candidates) as total_candidates,
			(SELECT COUNT(*) FROM hrm_candidates WHERE status = 'new') as new_candidates,
			(SELECT COUNT(*) FROM hrm_interviews WHERE status = 'scheduled' AND scheduled_at >= $1) as scheduled_interviews,
			(SELECT COUNT(*) FROM hrm_candidates WHERE status = 'offer') as offers,
			(SELECT COUNT(*) FROM hrm_candidates WHERE status = 'hired' AND EXTRACT(MONTH FROM updated_at) = $2 AND EXTRACT(YEAR FROM updated_at) = $3) as hired_this_month`

	err := r.db.QueryRowContext(ctx, query, today, int(today.Month()), today.Year()).Scan(
		&stats.OpenVacancies,
		&stats.TotalCandidates,
		&stats.NewCandidates,
		&stats.ScheduledInterviews,
		&stats.OffersExtended,
		&stats.HiredThisMonth,
	)
	if err != nil {
		return stats, nil // Return partial stats
	}

	return stats, nil
}

// GetTrainingStats retrieves training statistics
func (r *Repo) GetTrainingStats(ctx context.Context) (*hrmmodel.TrainingStats, error) {
	stats := &hrmmodel.TrainingStats{}
	today := time.Now()

	const query = `
		SELECT
			(SELECT COUNT(*) FROM hrm_trainings) as total,
			(SELECT COUNT(*) FROM hrm_trainings WHERE status = 'in_progress') as active,
			(SELECT COUNT(*) FROM hrm_trainings WHERE status = 'planned' AND start_date > $1) as upcoming,
			(SELECT COUNT(*) FROM hrm_training_participants) as participants,
			(SELECT COUNT(*) FROM hrm_certificates WHERE expiry_date IS NOT NULL AND expiry_date BETWEEN $1 AND $1 + INTERVAL '30 days') as expiring`

	err := r.db.QueryRowContext(ctx, query, today).Scan(
		&stats.TotalTrainings,
		&stats.ActiveTrainings,
		&stats.UpcomingTrainings,
		&stats.TotalParticipants,
		&stats.ExpiringCertificates,
	)
	if err != nil {
		return stats, nil
	}

	// Calculate completion rate
	var completed, total int
	r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status = 'completed'),
			COUNT(*)
		FROM hrm_training_participants`).Scan(&completed, &total)

	if total > 0 {
		stats.CompletionRate = float64(completed) / float64(total) * 100
	}

	// Average feedback
	r.db.QueryRowContext(ctx, `
		SELECT COALESCE(AVG(feedback_rating), 0)
		FROM hrm_training_participants
		WHERE feedback_rating IS NOT NULL`).Scan(&stats.AverageFeedback)

	return stats, nil
}

// GetPerformanceStats retrieves performance statistics
func (r *Repo) GetPerformanceStats(ctx context.Context, filter hrm.DashboardFilter) (*hrmmodel.PerformanceStats, error) {
	stats := &hrmmodel.PerformanceStats{
		DistributionByRating: make(map[int]int),
	}

	year := time.Now().Year()
	if filter.Year != nil {
		year = *filter.Year
	}

	const query = `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status != 'completed') as pending,
			COUNT(*) FILTER (WHERE status = 'completed') as completed,
			COALESCE(AVG(final_rating) FILTER (WHERE final_rating IS NOT NULL), 0) as avg_rating
		FROM hrm_performance_reviews
		WHERE EXTRACT(YEAR FROM review_period_start) = $1`

	err := r.db.QueryRowContext(ctx, query, year).Scan(
		&stats.TotalReviews,
		&stats.PendingReviews,
		&stats.CompletedReviews,
		&stats.AverageRating,
	)
	if err != nil {
		return stats, nil
	}

	// Rating distribution
	rows, err := r.db.QueryContext(ctx, `
		SELECT final_rating, COUNT(*)
		FROM hrm_performance_reviews
		WHERE final_rating IS NOT NULL AND EXTRACT(YEAR FROM review_period_start) = $1
		GROUP BY final_rating`, year)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var rating, count int
			if err := rows.Scan(&rating, &count); err == nil {
				stats.DistributionByRating[rating] = count
			}
		}
	}

	// Goals completion rate
	var completedGoals, totalGoals int
	r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status = 'completed'),
			COUNT(*)
		FROM hrm_performance_goals
		WHERE EXTRACT(YEAR FROM target_date) = $1`, year).Scan(&completedGoals, &totalGoals)

	if totalGoals > 0 {
		stats.GoalsCompletionRate = float64(completedGoals) / float64(totalGoals) * 100
	}

	return stats, nil
}

// CountPendingApprovals counts all pending approvals
func (r *Repo) CountPendingApprovals(ctx context.Context) (int, error) {
	var count int

	// Vacations + Timesheets + Documents
	const query = `
		SELECT
			(SELECT COUNT(*) FROM hrm_vacations WHERE status = 'pending') +
			(SELECT COUNT(*) FROM hrm_timesheets WHERE status = 'submitted') +
			(SELECT COUNT(*) FROM hrm_document_signatures WHERE status = 'pending')`

	r.db.QueryRowContext(ctx, query).Scan(&count)
	return count, nil
}

// CountExpiringDocuments counts documents expiring within days
func (r *Repo) CountExpiringDocuments(ctx context.Context, days int) (int, error) {
	var count int
	const query = `
		SELECT COUNT(*) FROM hrm_documents
		WHERE expiry_date IS NOT NULL
		AND expiry_date BETWEEN CURRENT_DATE AND CURRENT_DATE + $1`

	r.db.QueryRowContext(ctx, query, days).Scan(&count)
	return count, nil
}

// CountUpcomingReviews counts reviews with upcoming deadlines
func (r *Repo) CountUpcomingReviews(ctx context.Context, days int) (int, error) {
	var count int
	const query = `
		SELECT COUNT(*) FROM hrm_performance_reviews
		WHERE status != 'completed'
		AND (
			(self_review_deadline IS NOT NULL AND self_review_deadline BETWEEN CURRENT_DATE AND CURRENT_DATE + $1)
			OR (manager_review_deadline IS NOT NULL AND manager_review_deadline BETWEEN CURRENT_DATE AND CURRENT_DATE + $1)
		)`

	r.db.QueryRowContext(ctx, query, days).Scan(&count)
	return count, nil
}

// GetUpcomingBirthdays retrieves upcoming birthdays
func (r *Repo) GetUpcomingBirthdays(ctx context.Context, days int) ([]hrmmodel.EmployeeBirthday, error) {
	const query = `
		SELECT e.id, c.fio, COALESCE(d.name, ''), c.dob
		FROM hrm_employees e
		JOIN contacts c ON e.contact_id = c.id
		LEFT JOIN departments d ON c.department_id = d.id
		WHERE e.employment_status = 'active'
		AND c.dob IS NOT NULL
		AND (
			(EXTRACT(MONTH FROM c.dob) = EXTRACT(MONTH FROM CURRENT_DATE)
			 AND EXTRACT(DAY FROM c.dob) >= EXTRACT(DAY FROM CURRENT_DATE)
			 AND EXTRACT(DAY FROM c.dob) <= EXTRACT(DAY FROM CURRENT_DATE) + $1)
			OR
			(EXTRACT(MONTH FROM c.dob) = EXTRACT(MONTH FROM CURRENT_DATE + $1)
			 AND EXTRACT(DAY FROM c.dob) <= EXTRACT(DAY FROM CURRENT_DATE + $1))
		)
		ORDER BY EXTRACT(MONTH FROM c.dob), EXTRACT(DAY FROM c.dob)
		LIMIT 10`

	rows, err := r.db.QueryContext(ctx, query, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var birthdays []hrmmodel.EmployeeBirthday
	today := time.Now()
	for rows.Next() {
		var b hrmmodel.EmployeeBirthday
		var dob sql.NullTime
		err := rows.Scan(&b.EmployeeID, &b.EmployeeName, &b.Department, &dob)
		if err != nil {
			continue
		}
		if dob.Valid {
			b.BirthDate = dob.Time
			// Calculate days until
			thisYearBirthday := time.Date(today.Year(), b.BirthDate.Month(), b.BirthDate.Day(), 0, 0, 0, 0, time.UTC)
			if thisYearBirthday.Before(today) {
				thisYearBirthday = thisYearBirthday.AddDate(1, 0, 0)
			}
			b.DaysUntil = int(thisYearBirthday.Sub(today).Hours() / 24)
		}
		birthdays = append(birthdays, b)
	}

	return birthdays, nil
}

// GetRecentHires retrieves recently hired employees
func (r *Repo) GetRecentHires(ctx context.Context, days int, limit int) ([]*hrmmodel.Employee, error) {
	const query = `
		SELECT e.id, e.contact_id, e.employee_number, e.hire_date, e.employment_type, e.employment_status,
			c.fio, d.name as dept_name, p.name as pos_name
		FROM hrm_employees e
		JOIN contacts c ON e.contact_id = c.id
		LEFT JOIN departments d ON c.department_id = d.id
		LEFT JOIN positions p ON c.position_id = p.id
		WHERE e.hire_date >= CURRENT_DATE - $1
		ORDER BY e.hire_date DESC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, days, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var employees []*hrmmodel.Employee
	for rows.Next() {
		var e hrmmodel.Employee
		var empNumber, deptName, posName sql.NullString
		err := rows.Scan(&e.ID, &e.ContactID, &empNumber, &e.HireDate, &e.EmploymentType, &e.EmploymentStatus,
			&e.Contact, &deptName, &posName)
		if err != nil {
			continue
		}
		if empNumber.Valid {
			e.EmployeeNumber = &empNumber.String
		}
		employees = append(employees, &e)
	}

	return employees, nil
}

// GetMyDashboard retrieves personal dashboard for employee
func (r *Repo) GetMyDashboard(ctx context.Context, employeeID int64) (*hrmmodel.MyDashboard, error) {
	const op = "storage.repo.GetMyDashboard"

	dashboard := &hrmmodel.MyDashboard{
		GeneratedAt: time.Now(),
	}

	// Get employee info
	emp, err := r.GetEmployeeByID(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get employee: %w", op, err)
	}
	dashboard.Employee = emp

	// Calculate years worked
	dashboard.YearsWorked = int(time.Since(emp.HireDate).Hours() / 24 / 365)

	// Get vacation balances
	currentYear := time.Now().Year()
	balances, _ := r.GetVacationBalances(ctx, hrm.VacationBalanceFilter{
		EmployeeID: &employeeID,
		Year:       &currentYear,
	})
	for _, bal := range balances {
		if bal != nil {
			dashboard.VacationBalances = append(dashboard.VacationBalances, *bal)
		}
	}

	// Get pending vacations
	pendingStatus := hrmmodel.VacationStatusPending
	pendingVacations, _ := r.GetVacations(ctx, hrm.VacationFilter{
		EmployeeID: &employeeID,
		Status:     &pendingStatus,
		Limit:      5,
	})
	for _, vac := range pendingVacations {
		if vac != nil {
			dashboard.PendingVacations = append(dashboard.PendingVacations, *vac)
		}
	}

	// Get current timesheet
	currentMonth := time.Now().Month()
	ts, _ := r.GetTimesheetByPeriod(ctx, employeeID, currentYear, int(currentMonth))
	dashboard.CurrentTimesheet = ts

	// Get unread notifications count
	if emp.UserID != nil {
		dashboard.UnreadNotifications, _ = r.CountUnreadNotifications(ctx, *emp.UserID)
	}

	return dashboard, nil
}
