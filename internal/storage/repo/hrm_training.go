package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/hrm/training"
	"srmt-admin/internal/storage"
	"strings"
	"time"
)

// ==================== Trainings ====================

func (r *Repo) CreateTraining(ctx context.Context, req dto.CreateTrainingRequest, createdBy int64) (int64, error) {
	const op = "repo.CreateTraining"

	deptIDs := req.DepartmentIDs
	if deptIDs == nil {
		deptIDs = json.RawMessage("[]")
	}

	maxParticipants := 0
	if req.MaxParticipants != nil {
		maxParticipants = *req.MaxParticipants
	}

	mandatory := false
	if req.Mandatory != nil {
		mandatory = *req.Mandatory
	}

	query := `
		INSERT INTO trainings (title, description, type, provider, trainer,
			start_date, end_date, location, max_participants, cost,
			mandatory, department_ids, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.Title, req.Description, req.Type, req.Provider, req.Trainer,
		req.StartDate, req.EndDate, req.Location, maxParticipants, req.Cost,
		mandatory, deptIDs, createdBy,
	).Scan(&id)
	if err != nil {
		if translated := r.translator.Translate(err, op); translated != nil {
			return 0, translated
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

func (r *Repo) GetTrainingByID(ctx context.Context, id int64) (*training.Training, error) {
	const op = "repo.GetTrainingByID"

	query := `
		SELECT t.id, t.title, t.description, t.type, t.status,
			   t.provider, t.trainer, t.start_date, t.end_date, t.location,
			   t.max_participants,
			   (SELECT COUNT(*) FROM training_participants tp WHERE tp.training_id = t.id AND tp.status != 'cancelled'),
			   t.cost, t.mandatory, t.department_ids,
			   t.created_by, t.created_at, t.updated_at
		FROM trainings t
		WHERE t.id = $1`

	tr, err := scanTraining(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrTrainingNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return tr, nil
}

func (r *Repo) GetAllTrainings(ctx context.Context, filters dto.TrainingFilters) ([]*training.Training, error) {
	const op = "repo.GetAllTrainings"

	query := `
		SELECT t.id, t.title, t.description, t.type, t.status,
			   t.provider, t.trainer, t.start_date, t.end_date, t.location,
			   t.max_participants,
			   (SELECT COUNT(*) FROM training_participants tp WHERE tp.training_id = t.id AND tp.status != 'cancelled'),
			   t.cost, t.mandatory, t.department_ids,
			   t.created_by, t.created_at, t.updated_at
		FROM trainings t`

	var conditions []string
	var args []interface{}
	argIdx := 1

	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("t.status = $%d", argIdx))
		args = append(args, *filters.Status)
		argIdx++
	}
	if filters.Type != nil {
		conditions = append(conditions, fmt.Sprintf("t.type = $%d", argIdx))
		args = append(args, *filters.Type)
		argIdx++
	}
	if filters.Search != nil {
		conditions = append(conditions, fmt.Sprintf("(t.title ILIKE $%d OR t.description ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+*filters.Search+"%")
		argIdx++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY t.created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var trainings []*training.Training
	for rows.Next() {
		tr, err := scanTraining(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		trainings = append(trainings, tr)
	}
	return trainings, nil
}

func (r *Repo) UpdateTraining(ctx context.Context, id int64, req dto.UpdateTrainingRequest) error {
	const op = "repo.UpdateTraining"

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
	if req.Type != nil {
		setClauses = append(setClauses, fmt.Sprintf("type = $%d", argIdx))
		args = append(args, *req.Type)
		argIdx++
	}
	if req.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
	}
	if req.Provider != nil {
		setClauses = append(setClauses, fmt.Sprintf("provider = $%d", argIdx))
		args = append(args, *req.Provider)
		argIdx++
	}
	if req.Trainer != nil {
		setClauses = append(setClauses, fmt.Sprintf("trainer = $%d", argIdx))
		args = append(args, *req.Trainer)
		argIdx++
	}
	if req.StartDate != nil {
		setClauses = append(setClauses, fmt.Sprintf("start_date = $%d", argIdx))
		args = append(args, *req.StartDate)
		argIdx++
	}
	if req.EndDate != nil {
		setClauses = append(setClauses, fmt.Sprintf("end_date = $%d", argIdx))
		args = append(args, *req.EndDate)
		argIdx++
	}
	if req.Location != nil {
		setClauses = append(setClauses, fmt.Sprintf("location = $%d", argIdx))
		args = append(args, *req.Location)
		argIdx++
	}
	if req.MaxParticipants != nil {
		setClauses = append(setClauses, fmt.Sprintf("max_participants = $%d", argIdx))
		args = append(args, *req.MaxParticipants)
		argIdx++
	}
	if req.Cost != nil {
		setClauses = append(setClauses, fmt.Sprintf("cost = $%d", argIdx))
		args = append(args, *req.Cost)
		argIdx++
	}
	if req.Mandatory != nil {
		setClauses = append(setClauses, fmt.Sprintf("mandatory = $%d", argIdx))
		args = append(args, *req.Mandatory)
		argIdx++
	}
	if req.DepartmentIDs != nil {
		setClauses = append(setClauses, fmt.Sprintf("department_ids = $%d", argIdx))
		args = append(args, *req.DepartmentIDs)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE trainings SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if translated := r.translator.Translate(err, op); translated != nil {
			return translated
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrTrainingNotFound
	}
	return nil
}

func (r *Repo) DeleteTraining(ctx context.Context, id int64) error {
	const op = "repo.DeleteTraining"

	result, err := r.db.ExecContext(ctx, "DELETE FROM trainings WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrTrainingNotFound
	}
	return nil
}

// ==================== Participants ====================

func (r *Repo) AddParticipant(ctx context.Context, trainingID, employeeID int64) (int64, error) {
	const op = "repo.AddParticipant"

	query := `
		INSERT INTO training_participants (training_id, employee_id)
		VALUES ($1, $2)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query, trainingID, employeeID).Scan(&id)
	if err != nil {
		if translated := r.translator.Translate(err, op); translated != nil {
			return 0, translated
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

func (r *Repo) GetParticipantByID(ctx context.Context, id int64) (*training.Participant, error) {
	const op = "repo.GetParticipantByID"

	query := `
		SELECT tp.id, tp.training_id, tp.employee_id, COALESCE(c.name, ''),
			   tp.status, tp.enrolled_at, tp.completed_at, tp.score, tp.certificate_id, tp.notes
		FROM training_participants tp
		LEFT JOIN contacts c ON tp.employee_id = c.id
		WHERE tp.id = $1`

	p, err := scanParticipant(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrParticipantNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return p, nil
}

func (r *Repo) GetTrainingParticipants(ctx context.Context, trainingID int64) ([]*training.Participant, error) {
	const op = "repo.GetTrainingParticipants"

	query := `
		SELECT tp.id, tp.training_id, tp.employee_id, COALESCE(c.name, ''),
			   tp.status, tp.enrolled_at, tp.completed_at, tp.score, tp.certificate_id, tp.notes
		FROM training_participants tp
		LEFT JOIN contacts c ON tp.employee_id = c.id
		WHERE tp.training_id = $1
		ORDER BY tp.enrolled_at DESC`

	rows, err := r.db.QueryContext(ctx, query, trainingID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var participants []*training.Participant
	for rows.Next() {
		p, err := scanParticipant(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		participants = append(participants, p)
	}
	return participants, nil
}

func (r *Repo) CompleteParticipant(ctx context.Context, id int64, score *int, certificateID *int64, notes *string) error {
	const op = "repo.CompleteParticipant"

	query := `
		UPDATE training_participants
		SET status = 'completed', completed_at = $1, score = $2, certificate_id = $3, notes = $4
		WHERE id = $5`

	result, err := r.db.ExecContext(ctx, query, time.Now(), score, certificateID, notes, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrParticipantNotFound
	}
	return nil
}

func (r *Repo) GetEmployeeTrainings(ctx context.Context, employeeID int64) ([]*training.Training, error) {
	const op = "repo.GetEmployeeTrainings"

	query := `
		SELECT t.id, t.title, t.description, t.type, t.status,
			   t.provider, t.trainer, t.start_date, t.end_date, t.location,
			   t.max_participants,
			   (SELECT COUNT(*) FROM training_participants tp2 WHERE tp2.training_id = t.id AND tp2.status != 'cancelled'),
			   t.cost, t.mandatory, t.department_ids,
			   t.created_by, t.created_at, t.updated_at
		FROM trainings t
		INNER JOIN training_participants tp ON t.id = tp.training_id
		WHERE tp.employee_id = $1
		ORDER BY t.start_date DESC`

	rows, err := r.db.QueryContext(ctx, query, employeeID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var trainings []*training.Training
	for rows.Next() {
		tr, err := scanTraining(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		trainings = append(trainings, tr)
	}
	return trainings, nil
}

// ==================== Certificates ====================

func (r *Repo) CreateCertificate(ctx context.Context, employeeID int64, trainingID *int64, title string, issuer *string, issueDate string) (int64, error) {
	const op = "repo.CreateCertificate"

	query := `
		INSERT INTO certificates (employee_id, training_id, title, issuer, issue_date)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query, employeeID, trainingID, title, issuer, issueDate).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

func (r *Repo) GetEmployeeCertificates(ctx context.Context, employeeID int64) ([]*training.Certificate, error) {
	const op = "repo.GetEmployeeCertificates"

	query := `
		SELECT cert.id, cert.employee_id, COALESCE(c.name, ''),
			   cert.training_id, t.title,
			   cert.title, cert.issuer, cert.issue_date, cert.expiry_date,
			   cert.certificate_url, cert.created_at
		FROM certificates cert
		LEFT JOIN contacts c ON cert.employee_id = c.id
		LEFT JOIN trainings t ON cert.training_id = t.id
		WHERE cert.employee_id = $1
		ORDER BY cert.issue_date DESC`

	rows, err := r.db.QueryContext(ctx, query, employeeID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var certs []*training.Certificate
	for rows.Next() {
		cert, err := scanCertificate(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		certs = append(certs, cert)
	}
	return certs, nil
}

// ==================== Development Plans ====================

func (r *Repo) CreateDevelopmentPlan(ctx context.Context, req dto.CreateDevelopmentPlanRequest, createdBy int64) (int64, error) {
	const op = "repo.CreateDevelopmentPlan"

	query := `
		INSERT INTO development_plans (employee_id, title, description, start_date, end_date, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.EmployeeID, req.Title, req.Description, req.StartDate, req.EndDate, createdBy,
	).Scan(&id)
	if err != nil {
		if translated := r.translator.Translate(err, op); translated != nil {
			return 0, translated
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

func (r *Repo) GetDevelopmentPlanByID(ctx context.Context, id int64) (*training.DevelopmentPlan, error) {
	const op = "repo.GetDevelopmentPlanByID"

	query := `
		SELECT dp.id, dp.employee_id, COALESCE(c.name, ''),
			   dp.title, dp.description, dp.status,
			   dp.start_date, dp.end_date, dp.created_by,
			   dp.created_at, dp.updated_at
		FROM development_plans dp
		LEFT JOIN contacts c ON dp.employee_id = c.id
		WHERE dp.id = $1`

	plan, err := scanDevelopmentPlan(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrDevelopmentPlanNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return plan, nil
}

func (r *Repo) GetAllDevelopmentPlans(ctx context.Context, employeeID *int64) ([]*training.DevelopmentPlan, error) {
	const op = "repo.GetAllDevelopmentPlans"

	query := `
		SELECT dp.id, dp.employee_id, COALESCE(c.name, ''),
			   dp.title, dp.description, dp.status,
			   dp.start_date, dp.end_date, dp.created_by,
			   dp.created_at, dp.updated_at
		FROM development_plans dp
		LEFT JOIN contacts c ON dp.employee_id = c.id`

	var args []interface{}
	if employeeID != nil {
		query += " WHERE dp.employee_id = $1"
		args = append(args, *employeeID)
	}
	query += " ORDER BY dp.created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var plans []*training.DevelopmentPlan
	for rows.Next() {
		plan, err := scanDevelopmentPlan(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		plans = append(plans, plan)
	}
	return plans, nil
}

// ==================== Development Goals ====================

func (r *Repo) AddDevelopmentGoal(ctx context.Context, planID int64, req dto.AddDevelopmentGoalRequest) (int64, error) {
	const op = "repo.AddDevelopmentGoal"

	query := `
		INSERT INTO development_goals (plan_id, title, description, target_date)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query, planID, req.Title, req.Description, req.TargetDate).Scan(&id)
	if err != nil {
		if translated := r.translator.Translate(err, op); translated != nil {
			return 0, translated
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

// ==================== Scanners ====================

func scanTraining(s scannable) (*training.Training, error) {
	var t training.Training
	var deptIDs []byte
	err := s.Scan(
		&t.ID, &t.Title, &t.Description, &t.Type, &t.Status,
		&t.Provider, &t.Trainer, &t.StartDate, &t.EndDate, &t.Location,
		&t.MaxParticipants, &t.CurrentParticipants,
		&t.Cost, &t.Mandatory, &deptIDs,
		&t.CreatedBy, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	t.DepartmentIDs = json.RawMessage(deptIDs)
	return &t, nil
}

func scanParticipant(s scannable) (*training.Participant, error) {
	var p training.Participant
	err := s.Scan(
		&p.ID, &p.TrainingID, &p.EmployeeID, &p.EmployeeName,
		&p.Status, &p.EnrolledAt, &p.CompletedAt, &p.Score, &p.CertificateID, &p.Notes,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func scanCertificate(s scannable) (*training.Certificate, error) {
	var c training.Certificate
	err := s.Scan(
		&c.ID, &c.EmployeeID, &c.EmployeeName,
		&c.TrainingID, &c.TrainingTitle,
		&c.Title, &c.Issuer, &c.IssueDate, &c.ExpiryDate,
		&c.CertificateURL, &c.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func scanDevelopmentPlan(s scannable) (*training.DevelopmentPlan, error) {
	var dp training.DevelopmentPlan
	err := s.Scan(
		&dp.ID, &dp.EmployeeID, &dp.EmployeeName,
		&dp.Title, &dp.Description, &dp.Status,
		&dp.StartDate, &dp.EndDate, &dp.CreatedBy,
		&dp.CreatedAt, &dp.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &dp, nil
}

func scanDevelopmentGoal(s scannable) (*training.DevelopmentGoal, error) {
	var dg training.DevelopmentGoal
	err := s.Scan(
		&dg.ID, &dg.PlanID, &dg.Title, &dg.Description, &dg.Status,
		&dg.TargetDate, &dg.CompletedAt, &dg.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &dg, nil
}
