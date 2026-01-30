package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"

	"srmt-admin/internal/lib/dto/hrm"
	hrmmodel "srmt-admin/internal/lib/model/hrm"
	"srmt-admin/internal/storage"
)

// --- Vacancy ---

// AddVacancy creates a new vacancy
func (r *Repo) AddVacancy(ctx context.Context, req hrm.AddVacancyRequest) (int64, error) {
	const op = "storage.repo.AddVacancy"

	status := hrmmodel.VacancyStatusDraft
	priority := req.Priority
	if priority == "" {
		priority = hrmmodel.VacancyPriorityNormal
	}

	const query = `
		INSERT INTO hrm_vacancies (
			title, position_id, department_id, organization_id,
			description, requirements, responsibilities, benefits,
			employment_type, work_format, experience_level,
			salary_min, salary_max, currency, salary_visible,
			status, priority, headcount, deadline,
			hiring_manager_id, recruiter_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.Title, req.PositionID, req.DepartmentID, req.OrganizationID,
		req.Description, req.Requirements, req.Responsibilities, req.Benefits,
		req.EmploymentType, req.WorkFormat, req.ExperienceLevel,
		req.SalaryMin, req.SalaryMax, req.Currency, req.SalaryVisible,
		status, priority, req.Headcount, req.Deadline,
		req.HiringManagerID, req.RecruiterID,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("%s: failed to insert vacancy: %w", op, err)
	}

	return id, nil
}

// GetVacancyByID retrieves vacancy by ID
func (r *Repo) GetVacancyByID(ctx context.Context, id int64) (*hrmmodel.Vacancy, error) {
	const op = "storage.repo.GetVacancyByID"

	const query = `
		SELECT id, title, position_id, department_id, organization_id,
			description, requirements, responsibilities, benefits,
			employment_type, work_format, experience_level,
			salary_min, salary_max, currency, salary_visible,
			status, priority, headcount, filled_count,
			published_at, deadline, closed_at,
			hiring_manager_id, recruiter_id, created_at, updated_at
		FROM hrm_vacancies
		WHERE id = $1`

	v, err := r.scanVacancy(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get vacancy: %w", op, err)
	}

	return v, nil
}

// GetVacancies retrieves vacancies with filters
func (r *Repo) GetVacancies(ctx context.Context, filter hrm.VacancyFilter) ([]*hrmmodel.Vacancy, error) {
	const op = "storage.repo.GetVacancies"

	var query strings.Builder
	query.WriteString(`
		SELECT id, title, position_id, department_id, organization_id,
			description, requirements, responsibilities, benefits,
			employment_type, work_format, experience_level,
			salary_min, salary_max, currency, salary_visible,
			status, priority, headcount, filled_count,
			published_at, deadline, closed_at,
			hiring_manager_id, recruiter_id, created_at, updated_at
		FROM hrm_vacancies
		WHERE 1=1
	`)

	args := []interface{}{}
	argIdx := 1

	if filter.Status != nil {
		query.WriteString(fmt.Sprintf(" AND status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.DepartmentID != nil {
		query.WriteString(fmt.Sprintf(" AND department_id = $%d", argIdx))
		args = append(args, *filter.DepartmentID)
		argIdx++
	}
	if filter.OrganizationID != nil {
		query.WriteString(fmt.Sprintf(" AND organization_id = $%d", argIdx))
		args = append(args, *filter.OrganizationID)
		argIdx++
	}
	if filter.PositionID != nil {
		query.WriteString(fmt.Sprintf(" AND position_id = $%d", argIdx))
		args = append(args, *filter.PositionID)
		argIdx++
	}
	if filter.EmploymentType != nil {
		query.WriteString(fmt.Sprintf(" AND employment_type = $%d", argIdx))
		args = append(args, *filter.EmploymentType)
		argIdx++
	}
	if filter.WorkFormat != nil {
		query.WriteString(fmt.Sprintf(" AND work_format = $%d", argIdx))
		args = append(args, *filter.WorkFormat)
		argIdx++
	}
	if filter.Priority != nil {
		query.WriteString(fmt.Sprintf(" AND priority = $%d", argIdx))
		args = append(args, *filter.Priority)
		argIdx++
	}
	if filter.HiringManagerID != nil {
		query.WriteString(fmt.Sprintf(" AND hiring_manager_id = $%d", argIdx))
		args = append(args, *filter.HiringManagerID)
		argIdx++
	}
	if filter.RecruiterID != nil {
		query.WriteString(fmt.Sprintf(" AND recruiter_id = $%d", argIdx))
		args = append(args, *filter.RecruiterID)
		argIdx++
	}
	if filter.Search != nil {
		query.WriteString(fmt.Sprintf(" AND title ILIKE $%d", argIdx))
		args = append(args, "%"+*filter.Search+"%")
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
		return nil, fmt.Errorf("%s: failed to query vacancies: %w", op, err)
	}
	defer rows.Close()

	var vacancies []*hrmmodel.Vacancy
	for rows.Next() {
		v, err := r.scanVacancyRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan vacancy: %w", op, err)
		}
		vacancies = append(vacancies, v)
	}

	if vacancies == nil {
		vacancies = make([]*hrmmodel.Vacancy, 0)
	}

	return vacancies, nil
}

// EditVacancy updates vacancy
func (r *Repo) EditVacancy(ctx context.Context, id int64, req hrm.EditVacancyRequest) error {
	const op = "storage.repo.EditVacancy"

	var updates []string
	var args []interface{}
	argIdx := 1

	if req.Title != nil {
		updates = append(updates, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *req.Title)
		argIdx++
	}
	if req.PositionID != nil {
		updates = append(updates, fmt.Sprintf("position_id = $%d", argIdx))
		args = append(args, *req.PositionID)
		argIdx++
	}
	if req.DepartmentID != nil {
		updates = append(updates, fmt.Sprintf("department_id = $%d", argIdx))
		args = append(args, *req.DepartmentID)
		argIdx++
	}
	if req.OrganizationID != nil {
		updates = append(updates, fmt.Sprintf("organization_id = $%d", argIdx))
		args = append(args, *req.OrganizationID)
		argIdx++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.Requirements != nil {
		updates = append(updates, fmt.Sprintf("requirements = $%d", argIdx))
		args = append(args, *req.Requirements)
		argIdx++
	}
	if req.Responsibilities != nil {
		updates = append(updates, fmt.Sprintf("responsibilities = $%d", argIdx))
		args = append(args, *req.Responsibilities)
		argIdx++
	}
	if req.Benefits != nil {
		updates = append(updates, fmt.Sprintf("benefits = $%d", argIdx))
		args = append(args, *req.Benefits)
		argIdx++
	}
	if req.EmploymentType != nil {
		updates = append(updates, fmt.Sprintf("employment_type = $%d", argIdx))
		args = append(args, *req.EmploymentType)
		argIdx++
	}
	if req.WorkFormat != nil {
		updates = append(updates, fmt.Sprintf("work_format = $%d", argIdx))
		args = append(args, *req.WorkFormat)
		argIdx++
	}
	if req.ExperienceLevel != nil {
		updates = append(updates, fmt.Sprintf("experience_level = $%d", argIdx))
		args = append(args, *req.ExperienceLevel)
		argIdx++
	}
	if req.SalaryMin != nil {
		updates = append(updates, fmt.Sprintf("salary_min = $%d", argIdx))
		args = append(args, *req.SalaryMin)
		argIdx++
	}
	if req.SalaryMax != nil {
		updates = append(updates, fmt.Sprintf("salary_max = $%d", argIdx))
		args = append(args, *req.SalaryMax)
		argIdx++
	}
	if req.Currency != nil {
		updates = append(updates, fmt.Sprintf("currency = $%d", argIdx))
		args = append(args, *req.Currency)
		argIdx++
	}
	if req.SalaryVisible != nil {
		updates = append(updates, fmt.Sprintf("salary_visible = $%d", argIdx))
		args = append(args, *req.SalaryVisible)
		argIdx++
	}
	if req.Status != nil {
		updates = append(updates, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
	}
	if req.Priority != nil {
		updates = append(updates, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, *req.Priority)
		argIdx++
	}
	if req.Headcount != nil {
		updates = append(updates, fmt.Sprintf("headcount = $%d", argIdx))
		args = append(args, *req.Headcount)
		argIdx++
	}
	if req.Deadline != nil {
		updates = append(updates, fmt.Sprintf("deadline = $%d", argIdx))
		args = append(args, *req.Deadline)
		argIdx++
	}
	if req.HiringManagerID != nil {
		updates = append(updates, fmt.Sprintf("hiring_manager_id = $%d", argIdx))
		args = append(args, *req.HiringManagerID)
		argIdx++
	}
	if req.RecruiterID != nil {
		updates = append(updates, fmt.Sprintf("recruiter_id = $%d", argIdx))
		args = append(args, *req.RecruiterID)
		argIdx++
	}

	if len(updates) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE hrm_vacancies SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to update vacancy: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// PublishVacancy publishes or unpublishes vacancy
func (r *Repo) PublishVacancy(ctx context.Context, id int64, publish bool) error {
	const op = "storage.repo.PublishVacancy"

	var query string
	if publish {
		query = `UPDATE hrm_vacancies SET status = $1, published_at = $2 WHERE id = $3`
		_, err := r.db.ExecContext(ctx, query, hrmmodel.VacancyStatusOpen, time.Now(), id)
		if err != nil {
			return fmt.Errorf("%s: failed to publish vacancy: %w", op, err)
		}
	} else {
		query = `UPDATE hrm_vacancies SET status = $1 WHERE id = $2`
		_, err := r.db.ExecContext(ctx, query, hrmmodel.VacancyStatusPaused, id)
		if err != nil {
			return fmt.Errorf("%s: failed to unpublish vacancy: %w", op, err)
		}
	}

	return nil
}

// CloseVacancy closes a vacancy
func (r *Repo) CloseVacancy(ctx context.Context, id int64, status string) error {
	const op = "storage.repo.CloseVacancy"

	const query = `UPDATE hrm_vacancies SET status = $1, closed_at = $2 WHERE id = $3`

	_, err := r.db.ExecContext(ctx, query, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("%s: failed to close vacancy: %w", op, err)
	}

	return nil
}

// DeleteVacancy deletes vacancy
func (r *Repo) DeleteVacancy(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteVacancy"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_vacancies WHERE id = $1", id)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			return storage.ErrForeignKeyViolation
		}
		return fmt.Errorf("%s: failed to delete vacancy: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// IncrementFilledCount increments filled count for vacancy
func (r *Repo) IncrementFilledCount(ctx context.Context, id int64) error {
	const op = "storage.repo.IncrementFilledCount"

	const query = `UPDATE hrm_vacancies SET filled_count = filled_count + 1 WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("%s: failed to increment filled count: %w", op, err)
	}

	return nil
}

// --- Candidate ---

// AddCandidate creates a new candidate
func (r *Repo) AddCandidate(ctx context.Context, req hrm.AddCandidateRequest) (int64, error) {
	const op = "storage.repo.AddCandidate"

	const query = `
		INSERT INTO hrm_candidates (
			vacancy_id, first_name, last_name, middle_name,
			email, phone, current_position, current_company,
			experience_years, expected_salary, currency,
			resume_file_id, cover_letter, source, referrer_employee_id,
			status, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.VacancyID, req.FirstName, req.LastName, req.MiddleName,
		req.Email, req.Phone, req.CurrentPosition, req.CurrentCompany,
		req.ExperienceYears, req.ExpectedSalary, req.Currency,
		req.ResumeFileID, req.CoverLetter, req.Source, req.ReferrerEmployeeID,
		hrmmodel.CandidateStatusNew, req.Notes,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("%s: failed to insert candidate: %w", op, err)
	}

	return id, nil
}

// GetCandidateByID retrieves candidate by ID
func (r *Repo) GetCandidateByID(ctx context.Context, id int64) (*hrmmodel.Candidate, error) {
	const op = "storage.repo.GetCandidateByID"

	const query = `
		SELECT id, vacancy_id, first_name, last_name, middle_name,
			email, phone, current_position, current_company,
			experience_years, expected_salary, currency,
			resume_file_id, cover_letter, source, referrer_employee_id,
			status, rejection_reason, rating, notes, created_at, updated_at
		FROM hrm_candidates
		WHERE id = $1`

	c, err := r.scanCandidate(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get candidate: %w", op, err)
	}

	return c, nil
}

// GetCandidates retrieves candidates with filters
func (r *Repo) GetCandidates(ctx context.Context, filter hrm.CandidateFilter) ([]*hrmmodel.Candidate, error) {
	const op = "storage.repo.GetCandidates"

	var query strings.Builder
	query.WriteString(`
		SELECT id, vacancy_id, first_name, last_name, middle_name,
			email, phone, current_position, current_company,
			experience_years, expected_salary, currency,
			resume_file_id, cover_letter, source, referrer_employee_id,
			status, rejection_reason, rating, notes, created_at, updated_at
		FROM hrm_candidates
		WHERE 1=1
	`)

	args := []interface{}{}
	argIdx := 1

	if filter.VacancyID != nil {
		query.WriteString(fmt.Sprintf(" AND vacancy_id = $%d", argIdx))
		args = append(args, *filter.VacancyID)
		argIdx++
	}
	if filter.Status != nil {
		query.WriteString(fmt.Sprintf(" AND status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.Source != nil {
		query.WriteString(fmt.Sprintf(" AND source = $%d", argIdx))
		args = append(args, *filter.Source)
		argIdx++
	}
	if filter.ReferrerEmployeeID != nil {
		query.WriteString(fmt.Sprintf(" AND referrer_employee_id = $%d", argIdx))
		args = append(args, *filter.ReferrerEmployeeID)
		argIdx++
	}
	if filter.RatingMin != nil {
		query.WriteString(fmt.Sprintf(" AND rating >= $%d", argIdx))
		args = append(args, *filter.RatingMin)
		argIdx++
	}
	if filter.Search != nil {
		query.WriteString(fmt.Sprintf(" AND (first_name ILIKE $%d OR last_name ILIKE $%d OR email ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+*filter.Search+"%")
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
		return nil, fmt.Errorf("%s: failed to query candidates: %w", op, err)
	}
	defer rows.Close()

	var candidates []*hrmmodel.Candidate
	for rows.Next() {
		c, err := r.scanCandidateRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan candidate: %w", op, err)
		}
		candidates = append(candidates, c)
	}

	if candidates == nil {
		candidates = make([]*hrmmodel.Candidate, 0)
	}

	return candidates, nil
}

// EditCandidate updates candidate
func (r *Repo) EditCandidate(ctx context.Context, id int64, req hrm.EditCandidateRequest) error {
	const op = "storage.repo.EditCandidate"

	var updates []string
	var args []interface{}
	argIdx := 1

	if req.VacancyID != nil {
		updates = append(updates, fmt.Sprintf("vacancy_id = $%d", argIdx))
		args = append(args, *req.VacancyID)
		argIdx++
	}
	if req.FirstName != nil {
		updates = append(updates, fmt.Sprintf("first_name = $%d", argIdx))
		args = append(args, *req.FirstName)
		argIdx++
	}
	if req.LastName != nil {
		updates = append(updates, fmt.Sprintf("last_name = $%d", argIdx))
		args = append(args, *req.LastName)
		argIdx++
	}
	if req.MiddleName != nil {
		updates = append(updates, fmt.Sprintf("middle_name = $%d", argIdx))
		args = append(args, *req.MiddleName)
		argIdx++
	}
	if req.Email != nil {
		updates = append(updates, fmt.Sprintf("email = $%d", argIdx))
		args = append(args, *req.Email)
		argIdx++
	}
	if req.Phone != nil {
		updates = append(updates, fmt.Sprintf("phone = $%d", argIdx))
		args = append(args, *req.Phone)
		argIdx++
	}
	if req.CurrentPosition != nil {
		updates = append(updates, fmt.Sprintf("current_position = $%d", argIdx))
		args = append(args, *req.CurrentPosition)
		argIdx++
	}
	if req.CurrentCompany != nil {
		updates = append(updates, fmt.Sprintf("current_company = $%d", argIdx))
		args = append(args, *req.CurrentCompany)
		argIdx++
	}
	if req.ExperienceYears != nil {
		updates = append(updates, fmt.Sprintf("experience_years = $%d", argIdx))
		args = append(args, *req.ExperienceYears)
		argIdx++
	}
	if req.ExpectedSalary != nil {
		updates = append(updates, fmt.Sprintf("expected_salary = $%d", argIdx))
		args = append(args, *req.ExpectedSalary)
		argIdx++
	}
	if req.Currency != nil {
		updates = append(updates, fmt.Sprintf("currency = $%d", argIdx))
		args = append(args, *req.Currency)
		argIdx++
	}
	if req.ResumeFileID != nil {
		updates = append(updates, fmt.Sprintf("resume_file_id = $%d", argIdx))
		args = append(args, *req.ResumeFileID)
		argIdx++
	}
	if req.CoverLetter != nil {
		updates = append(updates, fmt.Sprintf("cover_letter = $%d", argIdx))
		args = append(args, *req.CoverLetter)
		argIdx++
	}
	if req.Source != nil {
		updates = append(updates, fmt.Sprintf("source = $%d", argIdx))
		args = append(args, *req.Source)
		argIdx++
	}
	if req.ReferrerEmployeeID != nil {
		updates = append(updates, fmt.Sprintf("referrer_employee_id = $%d", argIdx))
		args = append(args, *req.ReferrerEmployeeID)
		argIdx++
	}
	if req.Status != nil {
		updates = append(updates, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
	}
	if req.RejectionReason != nil {
		updates = append(updates, fmt.Sprintf("rejection_reason = $%d", argIdx))
		args = append(args, *req.RejectionReason)
		argIdx++
	}
	if req.Rating != nil {
		updates = append(updates, fmt.Sprintf("rating = $%d", argIdx))
		args = append(args, *req.Rating)
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

	query := fmt.Sprintf("UPDATE hrm_candidates SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to update candidate: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// MoveCandidateStatus moves candidate through recruitment pipeline
func (r *Repo) MoveCandidateStatus(ctx context.Context, id int64, req hrm.MoveCandidateRequest) error {
	const op = "storage.repo.MoveCandidateStatus"

	var query string
	var args []interface{}

	if req.RejectionReason != nil && req.Status == hrmmodel.CandidateStatusRejected {
		query = `UPDATE hrm_candidates SET status = $1, rejection_reason = $2 WHERE id = $3`
		args = []interface{}{req.Status, *req.RejectionReason, id}
	} else {
		query = `UPDATE hrm_candidates SET status = $1 WHERE id = $2`
		args = []interface{}{req.Status, id}
	}

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to move candidate: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteCandidate deletes candidate
func (r *Repo) DeleteCandidate(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteCandidate"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_candidates WHERE id = $1", id)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			return storage.ErrForeignKeyViolation
		}
		return fmt.Errorf("%s: failed to delete candidate: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- Interview ---

// AddInterview schedules a new interview
func (r *Repo) AddInterview(ctx context.Context, req hrm.AddInterviewRequest) (int64, error) {
	const op = "storage.repo.AddInterview"

	interviewerIDsJSON, err := json.Marshal(req.InterviewerIDs)
	if err != nil {
		return 0, fmt.Errorf("%s: failed to marshal interviewer IDs: %w", op, err)
	}

	const query = `
		INSERT INTO hrm_interviews (
			candidate_id, vacancy_id, interview_type,
			scheduled_at, duration_minutes, location,
			interviewer_ids, status, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`

	var id int64
	err = r.db.QueryRowContext(ctx, query,
		req.CandidateID, req.VacancyID, req.InterviewType,
		req.ScheduledAt, req.DurationMinutes, req.Location,
		interviewerIDsJSON, hrmmodel.InterviewStatusScheduled, req.Notes,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("%s: failed to insert interview: %w", op, err)
	}

	return id, nil
}

// GetInterviewByID retrieves interview by ID
func (r *Repo) GetInterviewByID(ctx context.Context, id int64) (*hrmmodel.Interview, error) {
	const op = "storage.repo.GetInterviewByID"

	const query = `
		SELECT id, candidate_id, vacancy_id, interview_type,
			scheduled_at, duration_minutes, location,
			interviewer_ids, status, completed_at,
			overall_rating, feedback, recommendation, notes, created_at, updated_at
		FROM hrm_interviews
		WHERE id = $1`

	i, err := r.scanInterview(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get interview: %w", op, err)
	}

	return i, nil
}

// GetInterviews retrieves interviews with filters
func (r *Repo) GetInterviews(ctx context.Context, filter hrm.InterviewFilter) ([]*hrmmodel.Interview, error) {
	const op = "storage.repo.GetInterviews"

	var query strings.Builder
	query.WriteString(`
		SELECT id, candidate_id, vacancy_id, interview_type,
			scheduled_at, duration_minutes, location,
			interviewer_ids, status, completed_at,
			overall_rating, feedback, recommendation, notes, created_at, updated_at
		FROM hrm_interviews
		WHERE 1=1
	`)

	args := []interface{}{}
	argIdx := 1

	if filter.CandidateID != nil {
		query.WriteString(fmt.Sprintf(" AND candidate_id = $%d", argIdx))
		args = append(args, *filter.CandidateID)
		argIdx++
	}
	if filter.VacancyID != nil {
		query.WriteString(fmt.Sprintf(" AND vacancy_id = $%d", argIdx))
		args = append(args, *filter.VacancyID)
		argIdx++
	}
	if filter.InterviewerID != nil {
		query.WriteString(fmt.Sprintf(" AND interviewer_ids @> $%d", argIdx))
		interviewerJSON, _ := json.Marshal([]int64{*filter.InterviewerID})
		args = append(args, interviewerJSON)
		argIdx++
	}
	if filter.InterviewType != nil {
		query.WriteString(fmt.Sprintf(" AND interview_type = $%d", argIdx))
		args = append(args, *filter.InterviewType)
		argIdx++
	}
	if filter.Status != nil {
		query.WriteString(fmt.Sprintf(" AND status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.FromDate != nil {
		query.WriteString(fmt.Sprintf(" AND scheduled_at >= $%d", argIdx))
		args = append(args, *filter.FromDate)
		argIdx++
	}
	if filter.ToDate != nil {
		query.WriteString(fmt.Sprintf(" AND scheduled_at <= $%d", argIdx))
		args = append(args, *filter.ToDate)
		argIdx++
	}

	query.WriteString(" ORDER BY scheduled_at DESC")

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
		return nil, fmt.Errorf("%s: failed to query interviews: %w", op, err)
	}
	defer rows.Close()

	var interviews []*hrmmodel.Interview
	for rows.Next() {
		i, err := r.scanInterviewRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan interview: %w", op, err)
		}
		interviews = append(interviews, i)
	}

	if interviews == nil {
		interviews = make([]*hrmmodel.Interview, 0)
	}

	return interviews, nil
}

// EditInterview updates interview
func (r *Repo) EditInterview(ctx context.Context, id int64, req hrm.EditInterviewRequest) error {
	const op = "storage.repo.EditInterview"

	var updates []string
	var args []interface{}
	argIdx := 1

	if req.InterviewType != nil {
		updates = append(updates, fmt.Sprintf("interview_type = $%d", argIdx))
		args = append(args, *req.InterviewType)
		argIdx++
	}
	if req.ScheduledAt != nil {
		updates = append(updates, fmt.Sprintf("scheduled_at = $%d", argIdx))
		args = append(args, *req.ScheduledAt)
		argIdx++
	}
	if req.DurationMinutes != nil {
		updates = append(updates, fmt.Sprintf("duration_minutes = $%d", argIdx))
		args = append(args, *req.DurationMinutes)
		argIdx++
	}
	if req.Location != nil {
		updates = append(updates, fmt.Sprintf("location = $%d", argIdx))
		args = append(args, *req.Location)
		argIdx++
	}
	if req.InterviewerIDs != nil {
		interviewerIDsJSON, _ := json.Marshal(req.InterviewerIDs)
		updates = append(updates, fmt.Sprintf("interviewer_ids = $%d", argIdx))
		args = append(args, interviewerIDsJSON)
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

	query := fmt.Sprintf("UPDATE hrm_interviews SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to update interview: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// CompleteInterview completes interview with results
func (r *Repo) CompleteInterview(ctx context.Context, id int64, req hrm.CompleteInterviewRequest) error {
	const op = "storage.repo.CompleteInterview"

	const query = `
		UPDATE hrm_interviews
		SET status = $1, completed_at = $2, overall_rating = $3, feedback = $4, recommendation = $5, notes = COALESCE($6, notes)
		WHERE id = $7`

	res, err := r.db.ExecContext(ctx, query,
		hrmmodel.InterviewStatusCompleted, time.Now(),
		req.OverallRating, req.Feedback, req.Recommendation, req.Notes, id,
	)
	if err != nil {
		return fmt.Errorf("%s: failed to complete interview: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// CancelInterview cancels interview
func (r *Repo) CancelInterview(ctx context.Context, id int64, reason string) error {
	const op = "storage.repo.CancelInterview"

	const query = `UPDATE hrm_interviews SET status = $1, notes = $2 WHERE id = $3`

	res, err := r.db.ExecContext(ctx, query, hrmmodel.InterviewStatusCancelled, reason, id)
	if err != nil {
		return fmt.Errorf("%s: failed to cancel interview: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteInterview deletes interview
func (r *Repo) DeleteInterview(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteInterview"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_interviews WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("%s: failed to delete interview: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- Helpers ---

func (r *Repo) scanVacancy(row *sql.Row) (*hrmmodel.Vacancy, error) {
	var v hrmmodel.Vacancy
	var positionID, departmentID, organizationID, hiringManagerID, recruiterID sql.NullInt64
	var description, requirements, responsibilities, benefits, experienceLevel sql.NullString
	var salaryMin, salaryMax sql.NullFloat64
	var publishedAt, deadline, closedAt, updatedAt sql.NullTime

	err := row.Scan(
		&v.ID, &v.Title, &positionID, &departmentID, &organizationID,
		&description, &requirements, &responsibilities, &benefits,
		&v.EmploymentType, &v.WorkFormat, &experienceLevel,
		&salaryMin, &salaryMax, &v.Currency, &v.SalaryVisible,
		&v.Status, &v.Priority, &v.Headcount, &v.FilledCount,
		&publishedAt, &deadline, &closedAt,
		&hiringManagerID, &recruiterID, &v.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if positionID.Valid {
		v.PositionID = &positionID.Int64
	}
	if departmentID.Valid {
		v.DepartmentID = &departmentID.Int64
	}
	if organizationID.Valid {
		v.OrganizationID = &organizationID.Int64
	}
	if description.Valid {
		v.Description = &description.String
	}
	if requirements.Valid {
		v.Requirements = &requirements.String
	}
	if responsibilities.Valid {
		v.Responsibilities = &responsibilities.String
	}
	if benefits.Valid {
		v.Benefits = &benefits.String
	}
	if experienceLevel.Valid {
		v.ExperienceLevel = &experienceLevel.String
	}
	if salaryMin.Valid {
		v.SalaryMin = &salaryMin.Float64
	}
	if salaryMax.Valid {
		v.SalaryMax = &salaryMax.Float64
	}
	if publishedAt.Valid {
		v.PublishedAt = &publishedAt.Time
	}
	if deadline.Valid {
		v.Deadline = &deadline.Time
	}
	if closedAt.Valid {
		v.ClosedAt = &closedAt.Time
	}
	if hiringManagerID.Valid {
		v.HiringManagerID = &hiringManagerID.Int64
	}
	if recruiterID.Valid {
		v.RecruiterID = &recruiterID.Int64
	}
	if updatedAt.Valid {
		v.UpdatedAt = &updatedAt.Time
	}

	return &v, nil
}

func (r *Repo) scanVacancyRow(rows *sql.Rows) (*hrmmodel.Vacancy, error) {
	var v hrmmodel.Vacancy
	var positionID, departmentID, organizationID, hiringManagerID, recruiterID sql.NullInt64
	var description, requirements, responsibilities, benefits, experienceLevel sql.NullString
	var salaryMin, salaryMax sql.NullFloat64
	var publishedAt, deadline, closedAt, updatedAt sql.NullTime

	err := rows.Scan(
		&v.ID, &v.Title, &positionID, &departmentID, &organizationID,
		&description, &requirements, &responsibilities, &benefits,
		&v.EmploymentType, &v.WorkFormat, &experienceLevel,
		&salaryMin, &salaryMax, &v.Currency, &v.SalaryVisible,
		&v.Status, &v.Priority, &v.Headcount, &v.FilledCount,
		&publishedAt, &deadline, &closedAt,
		&hiringManagerID, &recruiterID, &v.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if positionID.Valid {
		v.PositionID = &positionID.Int64
	}
	if departmentID.Valid {
		v.DepartmentID = &departmentID.Int64
	}
	if organizationID.Valid {
		v.OrganizationID = &organizationID.Int64
	}
	if description.Valid {
		v.Description = &description.String
	}
	if requirements.Valid {
		v.Requirements = &requirements.String
	}
	if responsibilities.Valid {
		v.Responsibilities = &responsibilities.String
	}
	if benefits.Valid {
		v.Benefits = &benefits.String
	}
	if experienceLevel.Valid {
		v.ExperienceLevel = &experienceLevel.String
	}
	if salaryMin.Valid {
		v.SalaryMin = &salaryMin.Float64
	}
	if salaryMax.Valid {
		v.SalaryMax = &salaryMax.Float64
	}
	if publishedAt.Valid {
		v.PublishedAt = &publishedAt.Time
	}
	if deadline.Valid {
		v.Deadline = &deadline.Time
	}
	if closedAt.Valid {
		v.ClosedAt = &closedAt.Time
	}
	if hiringManagerID.Valid {
		v.HiringManagerID = &hiringManagerID.Int64
	}
	if recruiterID.Valid {
		v.RecruiterID = &recruiterID.Int64
	}
	if updatedAt.Valid {
		v.UpdatedAt = &updatedAt.Time
	}

	return &v, nil
}

func (r *Repo) scanCandidate(row *sql.Row) (*hrmmodel.Candidate, error) {
	var c hrmmodel.Candidate
	var middleName, email, phone, currentPosition, currentCompany, coverLetter, source, rejectionReason, notes sql.NullString
	var experienceYears, rating sql.NullInt64
	var expectedSalary sql.NullFloat64
	var resumeFileID, referrerEmployeeID sql.NullInt64
	var updatedAt sql.NullTime

	err := row.Scan(
		&c.ID, &c.VacancyID, &c.FirstName, &c.LastName, &middleName,
		&email, &phone, &currentPosition, &currentCompany,
		&experienceYears, &expectedSalary, &c.Currency,
		&resumeFileID, &coverLetter, &source, &referrerEmployeeID,
		&c.Status, &rejectionReason, &rating, &notes, &c.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if middleName.Valid {
		c.MiddleName = &middleName.String
	}
	if email.Valid {
		c.Email = &email.String
	}
	if phone.Valid {
		c.Phone = &phone.String
	}
	if currentPosition.Valid {
		c.CurrentPosition = &currentPosition.String
	}
	if currentCompany.Valid {
		c.CurrentCompany = &currentCompany.String
	}
	if experienceYears.Valid {
		years := int(experienceYears.Int64)
		c.ExperienceYears = &years
	}
	if expectedSalary.Valid {
		c.ExpectedSalary = &expectedSalary.Float64
	}
	if resumeFileID.Valid {
		c.ResumeFileID = &resumeFileID.Int64
	}
	if coverLetter.Valid {
		c.CoverLetter = &coverLetter.String
	}
	if source.Valid {
		c.Source = &source.String
	}
	if referrerEmployeeID.Valid {
		c.ReferrerEmployeeID = &referrerEmployeeID.Int64
	}
	if rejectionReason.Valid {
		c.RejectionReason = &rejectionReason.String
	}
	if rating.Valid {
		r := int(rating.Int64)
		c.Rating = &r
	}
	if notes.Valid {
		c.Notes = &notes.String
	}
	if updatedAt.Valid {
		c.UpdatedAt = &updatedAt.Time
	}

	return &c, nil
}

func (r *Repo) scanCandidateRow(rows *sql.Rows) (*hrmmodel.Candidate, error) {
	var c hrmmodel.Candidate
	var middleName, email, phone, currentPosition, currentCompany, coverLetter, source, rejectionReason, notes sql.NullString
	var experienceYears, rating sql.NullInt64
	var expectedSalary sql.NullFloat64
	var resumeFileID, referrerEmployeeID sql.NullInt64
	var updatedAt sql.NullTime

	err := rows.Scan(
		&c.ID, &c.VacancyID, &c.FirstName, &c.LastName, &middleName,
		&email, &phone, &currentPosition, &currentCompany,
		&experienceYears, &expectedSalary, &c.Currency,
		&resumeFileID, &coverLetter, &source, &referrerEmployeeID,
		&c.Status, &rejectionReason, &rating, &notes, &c.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if middleName.Valid {
		c.MiddleName = &middleName.String
	}
	if email.Valid {
		c.Email = &email.String
	}
	if phone.Valid {
		c.Phone = &phone.String
	}
	if currentPosition.Valid {
		c.CurrentPosition = &currentPosition.String
	}
	if currentCompany.Valid {
		c.CurrentCompany = &currentCompany.String
	}
	if experienceYears.Valid {
		years := int(experienceYears.Int64)
		c.ExperienceYears = &years
	}
	if expectedSalary.Valid {
		c.ExpectedSalary = &expectedSalary.Float64
	}
	if resumeFileID.Valid {
		c.ResumeFileID = &resumeFileID.Int64
	}
	if coverLetter.Valid {
		c.CoverLetter = &coverLetter.String
	}
	if source.Valid {
		c.Source = &source.String
	}
	if referrerEmployeeID.Valid {
		c.ReferrerEmployeeID = &referrerEmployeeID.Int64
	}
	if rejectionReason.Valid {
		c.RejectionReason = &rejectionReason.String
	}
	if rating.Valid {
		r := int(rating.Int64)
		c.Rating = &r
	}
	if notes.Valid {
		c.Notes = &notes.String
	}
	if updatedAt.Valid {
		c.UpdatedAt = &updatedAt.Time
	}

	return &c, nil
}

func (r *Repo) scanInterview(row *sql.Row) (*hrmmodel.Interview, error) {
	var i hrmmodel.Interview
	var location, feedback, recommendation, notes sql.NullString
	var overallRating sql.NullInt64
	var completedAt, updatedAt sql.NullTime
	var interviewerIDsJSON []byte

	err := row.Scan(
		&i.ID, &i.CandidateID, &i.VacancyID, &i.InterviewType,
		&i.ScheduledAt, &i.DurationMinutes, &location,
		&interviewerIDsJSON, &i.Status, &completedAt,
		&overallRating, &feedback, &recommendation, &notes, &i.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if location.Valid {
		i.Location = &location.String
	}
	if completedAt.Valid {
		i.CompletedAt = &completedAt.Time
	}
	if overallRating.Valid {
		r := int(overallRating.Int64)
		i.OverallRating = &r
	}
	if feedback.Valid {
		i.Feedback = &feedback.String
	}
	if recommendation.Valid {
		i.Recommendation = &recommendation.String
	}
	if notes.Valid {
		i.Notes = &notes.String
	}
	if updatedAt.Valid {
		i.UpdatedAt = &updatedAt.Time
	}

	if interviewerIDsJSON != nil {
		json.Unmarshal(interviewerIDsJSON, &i.InterviewerIDs)
	}

	return &i, nil
}

func (r *Repo) scanInterviewRow(rows *sql.Rows) (*hrmmodel.Interview, error) {
	var i hrmmodel.Interview
	var location, feedback, recommendation, notes sql.NullString
	var overallRating sql.NullInt64
	var completedAt, updatedAt sql.NullTime
	var interviewerIDsJSON []byte

	err := rows.Scan(
		&i.ID, &i.CandidateID, &i.VacancyID, &i.InterviewType,
		&i.ScheduledAt, &i.DurationMinutes, &location,
		&interviewerIDsJSON, &i.Status, &completedAt,
		&overallRating, &feedback, &recommendation, &notes, &i.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if location.Valid {
		i.Location = &location.String
	}
	if completedAt.Valid {
		i.CompletedAt = &completedAt.Time
	}
	if overallRating.Valid {
		r := int(overallRating.Int64)
		i.OverallRating = &r
	}
	if feedback.Valid {
		i.Feedback = &feedback.String
	}
	if recommendation.Valid {
		i.Recommendation = &recommendation.String
	}
	if notes.Valid {
		i.Notes = &notes.String
	}
	if updatedAt.Valid {
		i.UpdatedAt = &updatedAt.Time
	}

	if interviewerIDsJSON != nil {
		json.Unmarshal(interviewerIDsJSON, &i.InterviewerIDs)
	}

	return &i, nil
}
