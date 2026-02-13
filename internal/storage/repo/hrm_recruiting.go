package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/hrm/recruiting"
	"srmt-admin/internal/storage"
	"strings"
	"time"
)

// ==================== Vacancies ====================

func (r *Repo) CreateVacancy(ctx context.Context, req dto.CreateVacancyRequest, createdBy int64) (int64, error) {
	const op = "repo.CreateVacancy"

	skills := req.Skills
	if skills == nil {
		skills = json.RawMessage("[]")
	}

	query := `
		INSERT INTO vacancies (title, department_id, position_id, description, requirements,
			salary_from, salary_to, employment_type, experience_required, education_required,
			skills, priority, deadline, responsible_id, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.Title, req.DepartmentID, req.PositionID, req.Description, req.Requirements,
		req.SalaryFrom, req.SalaryTo, req.EmploymentType, req.ExperienceRequired, req.EducationRequired,
		skills, req.Priority, req.Deadline, req.ResponsibleID, createdBy,
	).Scan(&id)
	if err != nil {
		if translated := r.translator.Translate(err, op); translated != nil {
			return 0, translated
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

func (r *Repo) GetVacancyByID(ctx context.Context, id int64) (*recruiting.Vacancy, error) {
	const op = "repo.GetVacancyByID"

	query := `
		SELECT v.id, v.title, v.department_id, COALESCE(d.name, ''), v.position_id, COALESCE(p.name, ''),
			   v.description, v.requirements, v.salary_from, v.salary_to,
			   v.employment_type, v.experience_required, v.education_required,
			   v.skills, v.status, v.priority, v.published_at, v.deadline,
			   v.responsible_id, v.created_by,
			   (SELECT COUNT(*) FROM candidates c WHERE c.vacancy_id = v.id),
			   v.created_at, v.updated_at
		FROM vacancies v
		LEFT JOIN departments d ON v.department_id = d.id
		LEFT JOIN positions p ON v.position_id = p.id
		WHERE v.id = $1`

	vacancy, err := scanVacancy(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrVacancyNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return vacancy, nil
}

func (r *Repo) GetAllVacancies(ctx context.Context, filters dto.VacancyFilters) ([]*recruiting.Vacancy, error) {
	const op = "repo.GetAllVacancies"

	query := `
		SELECT v.id, v.title, v.department_id, COALESCE(d.name, ''), v.position_id, COALESCE(p.name, ''),
			   v.description, v.requirements, v.salary_from, v.salary_to,
			   v.employment_type, v.experience_required, v.education_required,
			   v.skills, v.status, v.priority, v.published_at, v.deadline,
			   v.responsible_id, v.created_by,
			   (SELECT COUNT(*) FROM candidates c WHERE c.vacancy_id = v.id),
			   v.created_at, v.updated_at
		FROM vacancies v
		LEFT JOIN departments d ON v.department_id = d.id
		LEFT JOIN positions p ON v.position_id = p.id`

	var conditions []string
	var args []interface{}
	argIdx := 1

	if filters.DepartmentID != nil {
		conditions = append(conditions, fmt.Sprintf("v.department_id = $%d", argIdx))
		args = append(args, *filters.DepartmentID)
		argIdx++
	}
	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("v.status = $%d", argIdx))
		args = append(args, *filters.Status)
		argIdx++
	}
	if filters.Priority != nil {
		conditions = append(conditions, fmt.Sprintf("v.priority = $%d", argIdx))
		args = append(args, *filters.Priority)
		argIdx++
	}
	if filters.EmploymentType != nil {
		conditions = append(conditions, fmt.Sprintf("v.employment_type = $%d", argIdx))
		args = append(args, *filters.EmploymentType)
		argIdx++
	}
	if filters.Search != nil {
		conditions = append(conditions, fmt.Sprintf("(v.title ILIKE $%d OR v.description ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+*filters.Search+"%")
		argIdx++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY v.created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var vacancies []*recruiting.Vacancy
	for rows.Next() {
		v, err := scanVacancy(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		vacancies = append(vacancies, v)
	}
	return vacancies, nil
}

func (r *Repo) UpdateVacancy(ctx context.Context, id int64, req dto.UpdateVacancyRequest) error {
	const op = "repo.UpdateVacancy"

	var setClauses []string
	var args []interface{}
	argIdx := 1

	if req.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *req.Title)
		argIdx++
	}
	if req.DepartmentID != nil {
		setClauses = append(setClauses, fmt.Sprintf("department_id = $%d", argIdx))
		args = append(args, *req.DepartmentID)
		argIdx++
	}
	if req.PositionID != nil {
		setClauses = append(setClauses, fmt.Sprintf("position_id = $%d", argIdx))
		args = append(args, *req.PositionID)
		argIdx++
	}
	if req.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.Requirements != nil {
		setClauses = append(setClauses, fmt.Sprintf("requirements = $%d", argIdx))
		args = append(args, *req.Requirements)
		argIdx++
	}
	if req.SalaryFrom != nil {
		setClauses = append(setClauses, fmt.Sprintf("salary_from = $%d", argIdx))
		args = append(args, *req.SalaryFrom)
		argIdx++
	}
	if req.SalaryTo != nil {
		setClauses = append(setClauses, fmt.Sprintf("salary_to = $%d", argIdx))
		args = append(args, *req.SalaryTo)
		argIdx++
	}
	if req.EmploymentType != nil {
		setClauses = append(setClauses, fmt.Sprintf("employment_type = $%d", argIdx))
		args = append(args, *req.EmploymentType)
		argIdx++
	}
	if req.ExperienceRequired != nil {
		setClauses = append(setClauses, fmt.Sprintf("experience_required = $%d", argIdx))
		args = append(args, *req.ExperienceRequired)
		argIdx++
	}
	if req.EducationRequired != nil {
		setClauses = append(setClauses, fmt.Sprintf("education_required = $%d", argIdx))
		args = append(args, *req.EducationRequired)
		argIdx++
	}
	if req.Skills != nil {
		setClauses = append(setClauses, fmt.Sprintf("skills = $%d", argIdx))
		args = append(args, *req.Skills)
		argIdx++
	}
	if req.Priority != nil {
		setClauses = append(setClauses, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, *req.Priority)
		argIdx++
	}
	if req.Deadline != nil {
		setClauses = append(setClauses, fmt.Sprintf("deadline = $%d", argIdx))
		args = append(args, *req.Deadline)
		argIdx++
	}
	if req.ResponsibleID != nil {
		setClauses = append(setClauses, fmt.Sprintf("responsible_id = $%d", argIdx))
		args = append(args, *req.ResponsibleID)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE vacancies SET %s WHERE id = $%d", strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return storage.ErrVacancyNotFound
	}
	return nil
}

func (r *Repo) DeleteVacancy(ctx context.Context, id int64) error {
	const op = "repo.DeleteVacancy"

	res, err := r.db.ExecContext(ctx, "DELETE FROM vacancies WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return storage.ErrVacancyNotFound
	}
	return nil
}

func (r *Repo) UpdateVacancyStatus(ctx context.Context, id int64, status string) error {
	const op = "repo.UpdateVacancyStatus"

	var query string
	if status == "published" {
		query = "UPDATE vacancies SET status = $1, published_at = $2 WHERE id = $3"
		res, err := r.db.ExecContext(ctx, query, status, time.Now(), id)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		rows, _ := res.RowsAffected()
		if rows == 0 {
			return storage.ErrVacancyNotFound
		}
		return nil
	}

	query = "UPDATE vacancies SET status = $1 WHERE id = $2"
	res, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return storage.ErrVacancyNotFound
	}
	return nil
}

// ==================== Candidates ====================

func (r *Repo) CreateCandidate(ctx context.Context, req dto.CreateCandidateRequest) (int64, error) {
	const op = "repo.CreateCandidate"

	skills := req.Skills
	if skills == nil {
		skills = json.RawMessage("[]")
	}
	languages := req.Languages
	if languages == nil {
		languages = json.RawMessage("[]")
	}

	query := `
		INSERT INTO candidates (vacancy_id, name, email, phone, source, resume_url, photo_url,
			skills, languages, salary_expectation, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.VacancyID, req.Name, req.Email, req.Phone, req.Source,
		req.ResumeURL, req.PhotoURL, skills, languages,
		req.SalaryExpectation, req.Notes,
	).Scan(&id)
	if err != nil {
		if translated := r.translator.Translate(err, op); translated != nil {
			return 0, translated
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

func (r *Repo) GetCandidateByID(ctx context.Context, id int64) (*recruiting.CandidateListItem, error) {
	const op = "repo.GetCandidateByID"

	query := `
		SELECT id, vacancy_id, name, email, phone, source, status, stage,
			   resume_url, photo_url, skills, languages, salary_expectation,
			   notes, rating, created_at, updated_at
		FROM candidates
		WHERE id = $1`

	candidate, err := scanCandidateListItem(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrCandidateNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return candidate, nil
}

func (r *Repo) GetAllCandidates(ctx context.Context, filters dto.CandidateFilters) ([]*recruiting.CandidateListItem, error) {
	const op = "repo.GetAllCandidates"

	query := `
		SELECT id, vacancy_id, name, email, phone, source, status, stage,
			   resume_url, photo_url, skills, languages, salary_expectation,
			   notes, rating, created_at, updated_at
		FROM candidates`

	var conditions []string
	var args []interface{}
	argIdx := 1

	if filters.VacancyID != nil {
		conditions = append(conditions, fmt.Sprintf("vacancy_id = $%d", argIdx))
		args = append(args, *filters.VacancyID)
		argIdx++
	}
	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filters.Status)
		argIdx++
	}
	if filters.Stage != nil {
		conditions = append(conditions, fmt.Sprintf("stage = $%d", argIdx))
		args = append(args, *filters.Stage)
		argIdx++
	}
	if filters.Source != nil {
		conditions = append(conditions, fmt.Sprintf("source = $%d", argIdx))
		args = append(args, *filters.Source)
		argIdx++
	}
	if filters.Search != nil {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR email ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+*filters.Search+"%")
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

	var candidates []*recruiting.CandidateListItem
	for rows.Next() {
		c, err := scanCandidateListItem(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		candidates = append(candidates, c)
	}
	return candidates, nil
}

func (r *Repo) UpdateCandidate(ctx context.Context, id int64, req dto.UpdateCandidateRequest) error {
	const op = "repo.UpdateCandidate"

	var setClauses []string
	var args []interface{}
	argIdx := 1

	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Email != nil {
		setClauses = append(setClauses, fmt.Sprintf("email = $%d", argIdx))
		args = append(args, *req.Email)
		argIdx++
	}
	if req.Phone != nil {
		setClauses = append(setClauses, fmt.Sprintf("phone = $%d", argIdx))
		args = append(args, *req.Phone)
		argIdx++
	}
	if req.Source != nil {
		setClauses = append(setClauses, fmt.Sprintf("source = $%d", argIdx))
		args = append(args, *req.Source)
		argIdx++
	}
	if req.ResumeURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("resume_url = $%d", argIdx))
		args = append(args, *req.ResumeURL)
		argIdx++
	}
	if req.PhotoURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("photo_url = $%d", argIdx))
		args = append(args, *req.PhotoURL)
		argIdx++
	}
	if req.Skills != nil {
		setClauses = append(setClauses, fmt.Sprintf("skills = $%d", argIdx))
		args = append(args, *req.Skills)
		argIdx++
	}
	if req.Languages != nil {
		setClauses = append(setClauses, fmt.Sprintf("languages = $%d", argIdx))
		args = append(args, *req.Languages)
		argIdx++
	}
	if req.SalaryExpectation != nil {
		setClauses = append(setClauses, fmt.Sprintf("salary_expectation = $%d", argIdx))
		args = append(args, *req.SalaryExpectation)
		argIdx++
	}
	if req.Notes != nil {
		setClauses = append(setClauses, fmt.Sprintf("notes = $%d", argIdx))
		args = append(args, *req.Notes)
		argIdx++
	}
	if req.Rating != nil {
		setClauses = append(setClauses, fmt.Sprintf("rating = $%d", argIdx))
		args = append(args, *req.Rating)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE candidates SET %s WHERE id = $%d", strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return storage.ErrCandidateNotFound
	}
	return nil
}

func (r *Repo) DeleteCandidate(ctx context.Context, id int64) error {
	const op = "repo.DeleteCandidate"

	res, err := r.db.ExecContext(ctx, "DELETE FROM candidates WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return storage.ErrCandidateNotFound
	}
	return nil
}

func (r *Repo) UpdateCandidateStatus(ctx context.Context, id int64, status, stage string) error {
	const op = "repo.UpdateCandidateStatus"

	query := "UPDATE candidates SET status = $1, stage = $2 WHERE id = $3"
	res, err := r.db.ExecContext(ctx, query, status, stage, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return storage.ErrCandidateNotFound
	}
	return nil
}

// ==================== Education ====================

func (r *Repo) CreateCandidateEducation(ctx context.Context, candidateID int64, items []dto.EducationInput) error {
	const op = "repo.CreateCandidateEducation"

	if len(items) == 0 {
		return nil
	}

	var valueParts []string
	var args []interface{}
	argIdx := 1

	for _, item := range items {
		valueParts = append(valueParts, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			argIdx, argIdx+1, argIdx+2, argIdx+3, argIdx+4, argIdx+5, argIdx+6))
		args = append(args, candidateID, item.Institution, item.Degree,
			item.FieldOfStudy, item.StartDate, item.EndDate, item.Description)
		argIdx += 7
	}

	query := fmt.Sprintf(`INSERT INTO candidate_education (candidate_id, institution, degree, field_of_study, start_date, end_date, description) VALUES %s`,
		strings.Join(valueParts, ", "))

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *Repo) GetCandidateEducation(ctx context.Context, candidateID int64) ([]recruiting.Education, error) {
	const op = "repo.GetCandidateEducation"

	query := `SELECT id, candidate_id, institution, degree, field_of_study, start_date, end_date, description
		FROM candidate_education WHERE candidate_id = $1 ORDER BY start_date DESC`

	rows, err := r.db.QueryContext(ctx, query, candidateID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var result []recruiting.Education
	for rows.Next() {
		e, err := scanEducation(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		result = append(result, *e)
	}
	return result, nil
}

func (r *Repo) DeleteCandidateEducation(ctx context.Context, candidateID int64) error {
	const op = "repo.DeleteCandidateEducation"

	_, err := r.db.ExecContext(ctx, "DELETE FROM candidate_education WHERE candidate_id = $1", candidateID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

// ==================== Experience ====================

func (r *Repo) CreateCandidateExperience(ctx context.Context, candidateID int64, items []dto.ExperienceInput) error {
	const op = "repo.CreateCandidateExperience"

	if len(items) == 0 {
		return nil
	}

	var valueParts []string
	var args []interface{}
	argIdx := 1

	for _, item := range items {
		valueParts = append(valueParts, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d)",
			argIdx, argIdx+1, argIdx+2, argIdx+3, argIdx+4, argIdx+5))
		args = append(args, candidateID, item.Company, item.Position,
			item.StartDate, item.EndDate, item.Description)
		argIdx += 6
	}

	query := fmt.Sprintf(`INSERT INTO candidate_experience (candidate_id, company, position, start_date, end_date, description) VALUES %s`,
		strings.Join(valueParts, ", "))

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *Repo) GetCandidateExperience(ctx context.Context, candidateID int64) ([]recruiting.Experience, error) {
	const op = "repo.GetCandidateExperience"

	query := `SELECT id, candidate_id, company, position, start_date, end_date, description
		FROM candidate_experience WHERE candidate_id = $1 ORDER BY start_date DESC`

	rows, err := r.db.QueryContext(ctx, query, candidateID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var result []recruiting.Experience
	for rows.Next() {
		e, err := scanExperience(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		result = append(result, *e)
	}
	return result, nil
}

func (r *Repo) DeleteCandidateExperience(ctx context.Context, candidateID int64) error {
	const op = "repo.DeleteCandidateExperience"

	_, err := r.db.ExecContext(ctx, "DELETE FROM candidate_experience WHERE candidate_id = $1", candidateID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

// ==================== Interviews ====================

func (r *Repo) CreateInterview(ctx context.Context, req dto.CreateInterviewRequest, scheduledAt time.Time) (int64, error) {
	const op = "repo.CreateInterview"

	interviewers := req.Interviewers
	if interviewers == nil {
		interviewers = json.RawMessage("[]")
	}

	query := `
		INSERT INTO interviews (candidate_id, vacancy_id, type, scheduled_at, duration_minutes, location, interviewers)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.CandidateID, req.VacancyID, req.Type, scheduledAt,
		req.DurationMinutes, req.Location, interviewers,
	).Scan(&id)
	if err != nil {
		if translated := r.translator.Translate(err, op); translated != nil {
			return 0, translated
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

func (r *Repo) GetInterviewByID(ctx context.Context, id int64) (*recruiting.Interview, error) {
	const op = "repo.GetInterviewByID"

	query := `
		SELECT id, candidate_id, vacancy_id, type, scheduled_at, duration_minutes,
			   location, interviewers, status, overall_rating, recommendation,
			   feedback, scores, created_at, updated_at
		FROM interviews
		WHERE id = $1`

	interview, err := scanInterview(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrInterviewNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return interview, nil
}

func (r *Repo) GetAllInterviews(ctx context.Context, filters dto.InterviewFilters) ([]*recruiting.Interview, error) {
	const op = "repo.GetAllInterviews"

	query := `
		SELECT id, candidate_id, vacancy_id, type, scheduled_at, duration_minutes,
			   location, interviewers, status, overall_rating, recommendation,
			   feedback, scores, created_at, updated_at
		FROM interviews`

	var conditions []string
	var args []interface{}
	argIdx := 1

	if filters.CandidateID != nil {
		conditions = append(conditions, fmt.Sprintf("candidate_id = $%d", argIdx))
		args = append(args, *filters.CandidateID)
		argIdx++
	}
	if filters.VacancyID != nil {
		conditions = append(conditions, fmt.Sprintf("vacancy_id = $%d", argIdx))
		args = append(args, *filters.VacancyID)
		argIdx++
	}
	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filters.Status)
		argIdx++
	}
	if filters.Type != nil {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argIdx))
		args = append(args, *filters.Type)
		argIdx++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY scheduled_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var interviews []*recruiting.Interview
	for rows.Next() {
		i, err := scanInterview(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		interviews = append(interviews, i)
	}
	return interviews, nil
}

func (r *Repo) UpdateInterview(ctx context.Context, id int64, req dto.UpdateInterviewRequest) error {
	const op = "repo.UpdateInterview"

	var setClauses []string
	var args []interface{}
	argIdx := 1

	if req.Type != nil {
		setClauses = append(setClauses, fmt.Sprintf("type = $%d", argIdx))
		args = append(args, *req.Type)
		argIdx++
	}
	if req.ScheduledAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ScheduledAt)
		if err != nil {
			return fmt.Errorf("%s: invalid scheduled_at: %w", op, err)
		}
		setClauses = append(setClauses, fmt.Sprintf("scheduled_at = $%d", argIdx))
		args = append(args, t)
		argIdx++
	}
	if req.DurationMinutes != nil {
		setClauses = append(setClauses, fmt.Sprintf("duration_minutes = $%d", argIdx))
		args = append(args, *req.DurationMinutes)
		argIdx++
	}
	if req.Location != nil {
		setClauses = append(setClauses, fmt.Sprintf("location = $%d", argIdx))
		args = append(args, *req.Location)
		argIdx++
	}
	if req.Interviewers != nil {
		setClauses = append(setClauses, fmt.Sprintf("interviewers = $%d", argIdx))
		args = append(args, *req.Interviewers)
		argIdx++
	}
	if req.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
	}
	if req.OverallRating != nil {
		setClauses = append(setClauses, fmt.Sprintf("overall_rating = $%d", argIdx))
		args = append(args, *req.OverallRating)
		argIdx++
	}
	if req.Recommendation != nil {
		setClauses = append(setClauses, fmt.Sprintf("recommendation = $%d", argIdx))
		args = append(args, *req.Recommendation)
		argIdx++
	}
	if req.Feedback != nil {
		setClauses = append(setClauses, fmt.Sprintf("feedback = $%d", argIdx))
		args = append(args, *req.Feedback)
		argIdx++
	}
	if req.Scores != nil {
		setClauses = append(setClauses, fmt.Sprintf("scores = $%d", argIdx))
		args = append(args, *req.Scores)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE interviews SET %s WHERE id = $%d", strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return storage.ErrInterviewNotFound
	}
	return nil
}

func (r *Repo) GetInterviewsByCandidate(ctx context.Context, candidateID int64) ([]*recruiting.Interview, error) {
	return r.GetAllInterviews(ctx, dto.InterviewFilters{CandidateID: &candidateID})
}

// ==================== Scanners ====================

type scannable interface {
	Scan(dest ...interface{}) error
}

func scanVacancy(s scannable) (*recruiting.Vacancy, error) {
	var v recruiting.Vacancy
	var skills []byte
	err := s.Scan(
		&v.ID, &v.Title, &v.DepartmentID, &v.DepartmentName, &v.PositionID, &v.PositionName,
		&v.Description, &v.Requirements, &v.SalaryFrom, &v.SalaryTo,
		&v.EmploymentType, &v.ExperienceRequired, &v.EducationRequired,
		&skills, &v.Status, &v.Priority, &v.PublishedAt, &v.Deadline,
		&v.ResponsibleID, &v.CreatedBy, &v.CandidatesCount,
		&v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	v.Skills = json.RawMessage(skills)
	return &v, nil
}

func scanCandidateListItem(s scannable) (*recruiting.CandidateListItem, error) {
	var c recruiting.CandidateListItem
	var skills, languages []byte
	err := s.Scan(
		&c.ID, &c.VacancyID, &c.Name, &c.Email, &c.Phone,
		&c.Source, &c.Status, &c.Stage,
		&c.ResumeURL, &c.PhotoURL, &skills, &languages,
		&c.SalaryExpectation, &c.Notes, &c.Rating,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	c.Skills = json.RawMessage(skills)
	c.Languages = json.RawMessage(languages)
	return &c, nil
}

func scanInterview(s scannable) (*recruiting.Interview, error) {
	var i recruiting.Interview
	var interviewers, scores []byte
	err := s.Scan(
		&i.ID, &i.CandidateID, &i.VacancyID, &i.Type, &i.ScheduledAt, &i.DurationMinutes,
		&i.Location, &interviewers, &i.Status, &i.OverallRating, &i.Recommendation,
		&i.Feedback, &scores, &i.CreatedAt, &i.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	i.Interviewers = json.RawMessage(interviewers)
	i.Scores = json.RawMessage(scores)
	return &i, nil
}

func scanEducation(s scannable) (*recruiting.Education, error) {
	var e recruiting.Education
	err := s.Scan(&e.ID, &e.CandidateID, &e.Institution, &e.Degree,
		&e.FieldOfStudy, &e.StartDate, &e.EndDate, &e.Description)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func scanExperience(s scannable) (*recruiting.Experience, error) {
	var e recruiting.Experience
	err := s.Scan(&e.ID, &e.CandidateID, &e.Company, &e.Position,
		&e.StartDate, &e.EndDate, &e.Description)
	if err != nil {
		return nil, err
	}
	return &e, nil
}
