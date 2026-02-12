package repo

import (
	"context"
	"database/sql"
	"fmt"
	"srmt-admin/internal/lib/model/hrm/dashboard"
)

func (r *Repo) GetHRMDashboardWidgets(ctx context.Context) (*dashboard.Widgets, error) {
	const op = "repo.GetHRMDashboardWidgets"

	w := &dashboard.Widgets{}

	// Total employees
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM contacts").Scan(&w.TotalEmployees)
	if err != nil {
		return nil, fmt.Errorf("%s: total_employees: %w", op, err)
	}

	// On vacation
	err = r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM vacations WHERE status = 'active'").Scan(&w.OnVacation)
	if err != nil {
		return nil, fmt.Errorf("%s: on_vacation: %w", op, err)
	}

	// Pending approvals
	err = r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM vacations WHERE status = 'pending'").Scan(&w.PendingApprovals)
	if err != nil {
		return nil, fmt.Errorf("%s: pending_approvals: %w", op, err)
	}

	// New employees this month
	err = r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM personnel_records
		 WHERE hire_date >= date_trunc('month', CURRENT_DATE)
		   AND hire_date < date_trunc('month', CURRENT_DATE) + interval '1 month'`).Scan(&w.NewEmployeesMonth)
	if err != nil {
		return nil, fmt.Errorf("%s: new_employees: %w", op, err)
	}

	// Dismissed this month
	err = r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM personnel_records
		 WHERE status = 'dismissed'
		   AND updated_at >= date_trunc('month', CURRENT_DATE)
		   AND updated_at < date_trunc('month', CURRENT_DATE) + interval '1 month'`).Scan(&w.DismissedMonth)
	if err != nil {
		return nil, fmt.Errorf("%s: dismissed: %w", op, err)
	}

	return w, nil
}

func (r *Repo) GetHRMDashboardTasks(ctx context.Context, userID int64) ([]dashboard.Task, error) {
	const op = "repo.GetHRMDashboardTasks"

	// Tasks are derived from pending vacation approvals for managers
	query := `
		SELECT v.id, 'Vacation approval: ' || c.fio, 'Vacation request from ' || c.fio || ' (' || v.vacation_type || ')',
			   'approval', 'medium', v.start_date::text
		FROM vacations v
		JOIN contacts c ON v.employee_id = c.id
		WHERE v.status = 'pending'
		ORDER BY v.created_at ASC
		LIMIT 20`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var tasks []dashboard.Task
	for rows.Next() {
		var t dashboard.Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Type, &t.Priority, &t.DueDate); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (r *Repo) GetHRMDashboardEvents(ctx context.Context) ([]dashboard.Event, error) {
	// Return empty for MVP — events module not yet implemented
	return []dashboard.Event{}, nil
}

func (r *Repo) GetHRMDashboardActivity(ctx context.Context) ([]dashboard.Activity, error) {
	const op = "repo.GetHRMDashboardActivity"

	query := `
		(SELECT pr.id, 'hire' AS type, 'Hired: ' || c.fio AS description, c.fio, pr.created_at::text
		 FROM personnel_records pr
		 JOIN contacts c ON pr.employee_id = c.id
		 ORDER BY pr.created_at DESC LIMIT 5)
		UNION ALL
		(SELECT v.id, 'vacation' AS type, c.fio || ' - ' || v.vacation_type || ' vacation' AS description, c.fio, v.created_at::text
		 FROM vacations v
		 JOIN contacts c ON v.employee_id = c.id
		 WHERE v.status IN ('approved', 'active')
		 ORDER BY v.created_at DESC LIMIT 5)
		ORDER BY 5 DESC LIMIT 10`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var activities []dashboard.Activity
	for rows.Next() {
		var a dashboard.Activity
		var name sql.NullString
		if err := rows.Scan(&a.ID, &a.Type, &a.Description, &name, &a.Timestamp); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		if name.Valid {
			a.EmployeeName = &name.String
		}
		activities = append(activities, a)
	}
	return activities, rows.Err()
}

func (r *Repo) GetHRMUpcomingBirthdays(ctx context.Context) ([]dashboard.Birthday, error) {
	const op = "repo.GetHRMUpcomingBirthdays"

	query := `
		SELECT c.id, c.fio, COALESCE(p.name, ''), COALESCE(d.name, ''), c.dob::text
		FROM contacts c
		LEFT JOIN personnel_records pr ON c.id = pr.employee_id
		LEFT JOIN positions p ON pr.position_id = p.id
		LEFT JOIN departments d ON pr.department_id = d.id
		WHERE c.dob IS NOT NULL
		  AND (
		    (EXTRACT(MONTH FROM c.dob) = EXTRACT(MONTH FROM CURRENT_DATE)
		     AND EXTRACT(DAY FROM c.dob) >= EXTRACT(DAY FROM CURRENT_DATE))
		    OR
		    (EXTRACT(MONTH FROM c.dob) = EXTRACT(MONTH FROM CURRENT_DATE + interval '1 month'))
		  )
		ORDER BY EXTRACT(MONTH FROM c.dob), EXTRACT(DAY FROM c.dob)
		LIMIT 10`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var birthdays []dashboard.Birthday
	for rows.Next() {
		var b dashboard.Birthday
		if err := rows.Scan(&b.ID, &b.Name, &b.Position, &b.Department, &b.Date); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		birthdays = append(birthdays, b)
	}
	return birthdays, rows.Err()
}

func (r *Repo) GetHRMProbationEmployees(ctx context.Context) ([]dashboard.ProbationEmployee, error) {
	const op = "repo.GetHRMProbationEmployees"

	// Probation: employees hired within last 3 months with active status
	query := `
		SELECT pr.employee_id, c.fio, COALESCE(p.name, ''), COALESCE(d.name, ''),
			   pr.hire_date::text,
			   (pr.hire_date + interval '3 months')::date::text,
			   LEAST(100, GREATEST(0,
				 EXTRACT(EPOCH FROM (CURRENT_DATE - pr.hire_date)) /
				 EXTRACT(EPOCH FROM interval '3 months') * 100
			   ))::int,
			   'on_track'
		FROM personnel_records pr
		JOIN contacts c ON pr.employee_id = c.id
		LEFT JOIN positions p ON pr.position_id = p.id
		LEFT JOIN departments d ON pr.department_id = d.id
		WHERE pr.status = 'active'
		  AND pr.hire_date >= CURRENT_DATE - interval '3 months'
		ORDER BY pr.hire_date DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var employees []dashboard.ProbationEmployee
	for rows.Next() {
		var e dashboard.ProbationEmployee
		if err := rows.Scan(&e.ID, &e.Name, &e.Position, &e.Department,
			&e.StartDate, &e.EndDate, &e.Progress, &e.Status); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		employees = append(employees, e)
	}
	return employees, rows.Err()
}

// GetEmployeeDepartmentID returns the department_id for a given employee from personnel_records
func (r *Repo) GetEmployeeDepartmentID(ctx context.Context, employeeID int64) (int64, error) {
	const op = "repo.GetEmployeeDepartmentID"

	var deptID int64
	err := r.db.QueryRowContext(ctx,
		"SELECT department_id FROM personnel_records WHERE employee_id = $1", employeeID).Scan(&deptID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil // No personnel record, skip blocked period check
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return deptID, nil
}
