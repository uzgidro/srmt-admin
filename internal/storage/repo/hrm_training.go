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

// --- Training ---

// AddTraining creates a new training
func (r *Repo) AddTraining(ctx context.Context, req hrm.AddTrainingRequest) (int64, error) {
	const op = "storage.repo.AddTraining"

	const query = `
		INSERT INTO hrm_trainings (
			title, description, training_type, category, provider, instructor,
			start_date, end_date, duration_hours, location,
			max_participants, min_participants,
			cost_per_participant, currency, status, is_mandatory,
			materials_file_id, organizer_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.Title, req.Description, req.TrainingType, req.Category, req.Provider, req.Instructor,
		req.StartDate, req.EndDate, req.DurationHours, req.Location,
		req.MaxParticipants, req.MinParticipants,
		req.CostPerParticipant, req.Currency, hrmmodel.TrainingStatusPlanned, req.IsMandatory,
		req.MaterialsFileID, req.OrganizerID,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("%s: failed to insert training: %w", op, err)
	}

	return id, nil
}

// GetTrainingByID retrieves training by ID
func (r *Repo) GetTrainingByID(ctx context.Context, id int64) (*hrmmodel.Training, error) {
	const op = "storage.repo.GetTrainingByID"

	const query = `
		SELECT id, title, description, training_type, category, provider, instructor,
			start_date, end_date, duration_hours, location,
			max_participants, min_participants,
			cost_per_participant, currency, status, is_mandatory,
			materials_file_id, organizer_id, created_at, updated_at
		FROM hrm_trainings
		WHERE id = $1`

	t, err := r.scanTraining(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get training: %w", op, err)
	}

	return t, nil
}

// GetTrainings retrieves trainings with filters
func (r *Repo) GetTrainings(ctx context.Context, filter hrm.TrainingFilter) ([]*hrmmodel.Training, error) {
	const op = "storage.repo.GetTrainings"

	var query strings.Builder
	query.WriteString(`
		SELECT id, title, description, training_type, category, provider, instructor,
			start_date, end_date, duration_hours, location,
			max_participants, min_participants,
			cost_per_participant, currency, status, is_mandatory,
			materials_file_id, organizer_id, created_at, updated_at
		FROM hrm_trainings
		WHERE 1=1
	`)

	args := []interface{}{}
	argIdx := 1

	if filter.TrainingType != nil {
		query.WriteString(fmt.Sprintf(" AND training_type = $%d", argIdx))
		args = append(args, *filter.TrainingType)
		argIdx++
	}
	if filter.Category != nil {
		query.WriteString(fmt.Sprintf(" AND category = $%d", argIdx))
		args = append(args, *filter.Category)
		argIdx++
	}
	if filter.Status != nil {
		query.WriteString(fmt.Sprintf(" AND status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.IsMandatory != nil {
		query.WriteString(fmt.Sprintf(" AND is_mandatory = $%d", argIdx))
		args = append(args, *filter.IsMandatory)
		argIdx++
	}
	if filter.FromDate != nil {
		query.WriteString(fmt.Sprintf(" AND start_date >= $%d", argIdx))
		args = append(args, *filter.FromDate)
		argIdx++
	}
	if filter.ToDate != nil {
		query.WriteString(fmt.Sprintf(" AND start_date <= $%d", argIdx))
		args = append(args, *filter.ToDate)
		argIdx++
	}
	if filter.OrganizerID != nil {
		query.WriteString(fmt.Sprintf(" AND organizer_id = $%d", argIdx))
		args = append(args, *filter.OrganizerID)
		argIdx++
	}
	if filter.Search != nil {
		query.WriteString(fmt.Sprintf(" AND title ILIKE $%d", argIdx))
		args = append(args, "%"+*filter.Search+"%")
		argIdx++
	}

	query.WriteString(" ORDER BY start_date DESC NULLS LAST")

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
		return nil, fmt.Errorf("%s: failed to query trainings: %w", op, err)
	}
	defer rows.Close()

	var trainings []*hrmmodel.Training
	for rows.Next() {
		t, err := r.scanTrainingRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan training: %w", op, err)
		}
		trainings = append(trainings, t)
	}

	if trainings == nil {
		trainings = make([]*hrmmodel.Training, 0)
	}

	return trainings, nil
}

// EditTraining updates training
func (r *Repo) EditTraining(ctx context.Context, id int64, req hrm.EditTrainingRequest) error {
	const op = "storage.repo.EditTraining"

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
	if req.TrainingType != nil {
		updates = append(updates, fmt.Sprintf("training_type = $%d", argIdx))
		args = append(args, *req.TrainingType)
		argIdx++
	}
	if req.Category != nil {
		updates = append(updates, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, *req.Category)
		argIdx++
	}
	if req.Provider != nil {
		updates = append(updates, fmt.Sprintf("provider = $%d", argIdx))
		args = append(args, *req.Provider)
		argIdx++
	}
	if req.Instructor != nil {
		updates = append(updates, fmt.Sprintf("instructor = $%d", argIdx))
		args = append(args, *req.Instructor)
		argIdx++
	}
	if req.StartDate != nil {
		updates = append(updates, fmt.Sprintf("start_date = $%d", argIdx))
		args = append(args, *req.StartDate)
		argIdx++
	}
	if req.EndDate != nil {
		updates = append(updates, fmt.Sprintf("end_date = $%d", argIdx))
		args = append(args, *req.EndDate)
		argIdx++
	}
	if req.DurationHours != nil {
		updates = append(updates, fmt.Sprintf("duration_hours = $%d", argIdx))
		args = append(args, *req.DurationHours)
		argIdx++
	}
	if req.Location != nil {
		updates = append(updates, fmt.Sprintf("location = $%d", argIdx))
		args = append(args, *req.Location)
		argIdx++
	}
	if req.MaxParticipants != nil {
		updates = append(updates, fmt.Sprintf("max_participants = $%d", argIdx))
		args = append(args, *req.MaxParticipants)
		argIdx++
	}
	if req.MinParticipants != nil {
		updates = append(updates, fmt.Sprintf("min_participants = $%d", argIdx))
		args = append(args, *req.MinParticipants)
		argIdx++
	}
	if req.CostPerParticipant != nil {
		updates = append(updates, fmt.Sprintf("cost_per_participant = $%d", argIdx))
		args = append(args, *req.CostPerParticipant)
		argIdx++
	}
	if req.Currency != nil {
		updates = append(updates, fmt.Sprintf("currency = $%d", argIdx))
		args = append(args, *req.Currency)
		argIdx++
	}
	if req.Status != nil {
		updates = append(updates, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
	}
	if req.IsMandatory != nil {
		updates = append(updates, fmt.Sprintf("is_mandatory = $%d", argIdx))
		args = append(args, *req.IsMandatory)
		argIdx++
	}
	if req.MaterialsFileID != nil {
		updates = append(updates, fmt.Sprintf("materials_file_id = $%d", argIdx))
		args = append(args, *req.MaterialsFileID)
		argIdx++
	}
	if req.OrganizerID != nil {
		updates = append(updates, fmt.Sprintf("organizer_id = $%d", argIdx))
		args = append(args, *req.OrganizerID)
		argIdx++
	}

	if len(updates) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE hrm_trainings SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to update training: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteTraining deletes training
func (r *Repo) DeleteTraining(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteTraining"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_trainings WHERE id = $1", id)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			return storage.ErrForeignKeyViolation
		}
		return fmt.Errorf("%s: failed to delete training: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- Training Participants ---

// EnrollParticipant enrolls employee in training
func (r *Repo) EnrollParticipant(ctx context.Context, req hrm.EnrollParticipantRequest, enrolledBy *int64) (int64, error) {
	const op = "storage.repo.EnrollParticipant"

	const query = `
		INSERT INTO hrm_training_participants (
			training_id, employee_id, enrolled_at, enrolled_by, status
		) VALUES ($1, $2, $3, $4, $5)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.TrainingID, req.EmployeeID, time.Now(), enrolledBy, hrmmodel.ParticipantStatusEnrolled,
	).Scan(&id)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return 0, storage.ErrUniqueViolation
		}
		return 0, fmt.Errorf("%s: failed to enroll participant: %w", op, err)
	}

	return id, nil
}

// BulkEnrollParticipants enrolls multiple employees in training
func (r *Repo) BulkEnrollParticipants(ctx context.Context, req hrm.BulkEnrollRequest, enrolledBy *int64) error {
	const op = "storage.repo.BulkEnrollParticipants"

	const query = `
		INSERT INTO hrm_training_participants (
			training_id, employee_id, enrolled_at, enrolled_by, status
		) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (training_id, employee_id) DO NOTHING`

	for _, employeeID := range req.EmployeeIDs {
		_, err := r.db.ExecContext(ctx, query,
			req.TrainingID, employeeID, time.Now(), enrolledBy, hrmmodel.ParticipantStatusEnrolled,
		)
		if err != nil {
			return fmt.Errorf("%s: failed to enroll employee %d: %w", op, employeeID, err)
		}
	}

	return nil
}

// GetParticipantByID retrieves participant by ID
func (r *Repo) GetParticipantByID(ctx context.Context, id int64) (*hrmmodel.TrainingParticipant, error) {
	const op = "storage.repo.GetParticipantByID"

	const query = `
		SELECT id, training_id, employee_id, enrolled_at, enrolled_by,
			status, attendance_percent, score, passed, completed_at,
			feedback_rating, feedback_text, notes, created_at, updated_at
		FROM hrm_training_participants
		WHERE id = $1`

	p, err := r.scanTrainingParticipant(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get participant: %w", op, err)
	}

	return p, nil
}

// GetParticipants retrieves participants with filters
func (r *Repo) GetParticipants(ctx context.Context, filter hrm.ParticipantFilter) ([]*hrmmodel.TrainingParticipant, error) {
	const op = "storage.repo.GetParticipants"

	var query strings.Builder
	query.WriteString(`
		SELECT id, training_id, employee_id, enrolled_at, enrolled_by,
			status, attendance_percent, score, passed, completed_at,
			feedback_rating, feedback_text, notes, created_at, updated_at
		FROM hrm_training_participants
		WHERE 1=1
	`)

	args := []interface{}{}
	argIdx := 1

	if filter.TrainingID != nil {
		query.WriteString(fmt.Sprintf(" AND training_id = $%d", argIdx))
		args = append(args, *filter.TrainingID)
		argIdx++
	}
	if filter.EmployeeID != nil {
		query.WriteString(fmt.Sprintf(" AND employee_id = $%d", argIdx))
		args = append(args, *filter.EmployeeID)
		argIdx++
	}
	if filter.Status != nil {
		query.WriteString(fmt.Sprintf(" AND status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.Passed != nil {
		query.WriteString(fmt.Sprintf(" AND passed = $%d", argIdx))
		args = append(args, *filter.Passed)
		argIdx++
	}

	query.WriteString(" ORDER BY enrolled_at DESC")

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query participants: %w", op, err)
	}
	defer rows.Close()

	var participants []*hrmmodel.TrainingParticipant
	for rows.Next() {
		p, err := r.scanTrainingParticipantRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan participant: %w", op, err)
		}
		participants = append(participants, p)
	}

	if participants == nil {
		participants = make([]*hrmmodel.TrainingParticipant, 0)
	}

	return participants, nil
}

// UpdateParticipant updates participant
func (r *Repo) UpdateParticipant(ctx context.Context, id int64, req hrm.UpdateParticipantRequest) error {
	const op = "storage.repo.UpdateParticipant"

	var updates []string
	var args []interface{}
	argIdx := 1

	if req.Status != nil {
		updates = append(updates, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
		// If completed, set completed_at
		if *req.Status == hrmmodel.ParticipantStatusCompleted {
			updates = append(updates, fmt.Sprintf("completed_at = $%d", argIdx))
			args = append(args, time.Now())
			argIdx++
		}
	}
	if req.AttendancePercent != nil {
		updates = append(updates, fmt.Sprintf("attendance_percent = $%d", argIdx))
		args = append(args, *req.AttendancePercent)
		argIdx++
	}
	if req.Score != nil {
		updates = append(updates, fmt.Sprintf("score = $%d", argIdx))
		args = append(args, *req.Score)
		argIdx++
	}
	if req.Passed != nil {
		updates = append(updates, fmt.Sprintf("passed = $%d", argIdx))
		args = append(args, *req.Passed)
		argIdx++
	}
	if req.FeedbackRating != nil {
		updates = append(updates, fmt.Sprintf("feedback_rating = $%d", argIdx))
		args = append(args, *req.FeedbackRating)
		argIdx++
	}
	if req.FeedbackText != nil {
		updates = append(updates, fmt.Sprintf("feedback_text = $%d", argIdx))
		args = append(args, *req.FeedbackText)
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

	query := fmt.Sprintf("UPDATE hrm_training_participants SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to update participant: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// CompleteParticipantTraining marks training as completed for participant
func (r *Repo) CompleteParticipantTraining(ctx context.Context, id int64, req hrm.CompleteTrainingRequest) error {
	const op = "storage.repo.CompleteParticipantTraining"

	const query = `
		UPDATE hrm_training_participants
		SET status = $1, completed_at = $2, score = $3, passed = $4
		WHERE id = $5`

	res, err := r.db.ExecContext(ctx, query,
		hrmmodel.ParticipantStatusCompleted, time.Now(), req.Score, req.Passed, id,
	)
	if err != nil {
		return fmt.Errorf("%s: failed to complete training: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// CancelParticipant cancels participant enrollment
func (r *Repo) CancelParticipant(ctx context.Context, id int64) error {
	const op = "storage.repo.CancelParticipant"

	const query = `UPDATE hrm_training_participants SET status = $1 WHERE id = $2`

	res, err := r.db.ExecContext(ctx, query, hrmmodel.ParticipantStatusCancelled, id)
	if err != nil {
		return fmt.Errorf("%s: failed to cancel participant: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- Certificates ---

// AddCertificate creates a new certificate
func (r *Repo) AddCertificate(ctx context.Context, req hrm.AddCertificateRequest) (int64, error) {
	const op = "storage.repo.AddCertificate"

	const query = `
		INSERT INTO hrm_certificates (
			employee_id, training_id, name, issuer,
			certificate_number, issued_date, expiry_date,
			file_id, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.EmployeeID, req.TrainingID, req.Name, req.Issuer,
		req.CertificateNumber, req.IssuedDate, req.ExpiryDate,
		req.FileID, req.Notes,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("%s: failed to insert certificate: %w", op, err)
	}

	return id, nil
}

// GetCertificateByID retrieves certificate by ID
func (r *Repo) GetCertificateByID(ctx context.Context, id int64) (*hrmmodel.Certificate, error) {
	const op = "storage.repo.GetCertificateByID"

	const query = `
		SELECT id, employee_id, training_id, name, issuer,
			certificate_number, issued_date, expiry_date,
			file_id, is_verified, verified_by, verified_at, notes, created_at, updated_at
		FROM hrm_certificates
		WHERE id = $1`

	c, err := r.scanCertificate(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get certificate: %w", op, err)
	}

	return c, nil
}

// GetCertificates retrieves certificates with filters
func (r *Repo) GetCertificates(ctx context.Context, filter hrm.CertificateFilter) ([]*hrmmodel.Certificate, error) {
	const op = "storage.repo.GetCertificates"

	var query strings.Builder
	query.WriteString(`
		SELECT id, employee_id, training_id, name, issuer,
			certificate_number, issued_date, expiry_date,
			file_id, is_verified, verified_by, verified_at, notes, created_at, updated_at
		FROM hrm_certificates
		WHERE 1=1
	`)

	args := []interface{}{}
	argIdx := 1

	if filter.EmployeeID != nil {
		query.WriteString(fmt.Sprintf(" AND employee_id = $%d", argIdx))
		args = append(args, *filter.EmployeeID)
		argIdx++
	}
	if filter.TrainingID != nil {
		query.WriteString(fmt.Sprintf(" AND training_id = $%d", argIdx))
		args = append(args, *filter.TrainingID)
		argIdx++
	}
	if filter.IsVerified != nil {
		query.WriteString(fmt.Sprintf(" AND is_verified = $%d", argIdx))
		args = append(args, *filter.IsVerified)
		argIdx++
	}
	if filter.ExpiringDays != nil {
		query.WriteString(fmt.Sprintf(" AND expiry_date IS NOT NULL AND expiry_date BETWEEN NOW() AND NOW() + INTERVAL '%d days'", *filter.ExpiringDays))
	}
	if filter.Expired != nil && *filter.Expired {
		query.WriteString(" AND expiry_date IS NOT NULL AND expiry_date < NOW()")
	}
	if filter.Search != nil {
		query.WriteString(fmt.Sprintf(" AND (name ILIKE $%d OR issuer ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+*filter.Search+"%")
		argIdx++
	}

	query.WriteString(" ORDER BY issued_date DESC")

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
		return nil, fmt.Errorf("%s: failed to query certificates: %w", op, err)
	}
	defer rows.Close()

	var certificates []*hrmmodel.Certificate
	for rows.Next() {
		c, err := r.scanCertificateRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan certificate: %w", op, err)
		}
		certificates = append(certificates, c)
	}

	if certificates == nil {
		certificates = make([]*hrmmodel.Certificate, 0)
	}

	return certificates, nil
}

// EditCertificate updates certificate
func (r *Repo) EditCertificate(ctx context.Context, id int64, req hrm.EditCertificateRequest) error {
	const op = "storage.repo.EditCertificate"

	var updates []string
	var args []interface{}
	argIdx := 1

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Issuer != nil {
		updates = append(updates, fmt.Sprintf("issuer = $%d", argIdx))
		args = append(args, *req.Issuer)
		argIdx++
	}
	if req.CertificateNumber != nil {
		updates = append(updates, fmt.Sprintf("certificate_number = $%d", argIdx))
		args = append(args, *req.CertificateNumber)
		argIdx++
	}
	if req.IssuedDate != nil {
		updates = append(updates, fmt.Sprintf("issued_date = $%d", argIdx))
		args = append(args, *req.IssuedDate)
		argIdx++
	}
	if req.ExpiryDate != nil {
		updates = append(updates, fmt.Sprintf("expiry_date = $%d", argIdx))
		args = append(args, *req.ExpiryDate)
		argIdx++
	}
	if req.FileID != nil {
		updates = append(updates, fmt.Sprintf("file_id = $%d", argIdx))
		args = append(args, *req.FileID)
		argIdx++
	}
	if req.IsVerified != nil {
		updates = append(updates, fmt.Sprintf("is_verified = $%d", argIdx))
		args = append(args, *req.IsVerified)
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

	query := fmt.Sprintf("UPDATE hrm_certificates SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to update certificate: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// VerifyCertificate marks certificate as verified
func (r *Repo) VerifyCertificate(ctx context.Context, id int64, verifiedBy int64) error {
	const op = "storage.repo.VerifyCertificate"

	const query = `
		UPDATE hrm_certificates
		SET is_verified = TRUE, verified_by = $1, verified_at = $2
		WHERE id = $3`

	res, err := r.db.ExecContext(ctx, query, verifiedBy, time.Now(), id)
	if err != nil {
		return fmt.Errorf("%s: failed to verify certificate: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteCertificate deletes certificate
func (r *Repo) DeleteCertificate(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteCertificate"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_certificates WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("%s: failed to delete certificate: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- Development Plans ---

// AddDevelopmentPlan creates a new development plan
func (r *Repo) AddDevelopmentPlan(ctx context.Context, req hrm.AddDevelopmentPlanRequest) (int64, error) {
	const op = "storage.repo.AddDevelopmentPlan"

	const query = `
		INSERT INTO hrm_development_plans (
			employee_id, title, start_date, end_date, status, manager_id, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.EmployeeID, req.Title, req.StartDate, req.EndDate,
		hrmmodel.DevelopmentPlanStatusDraft, req.ManagerID, req.Notes,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("%s: failed to insert development plan: %w", op, err)
	}

	return id, nil
}

// GetDevelopmentPlanByID retrieves development plan by ID
func (r *Repo) GetDevelopmentPlanByID(ctx context.Context, id int64) (*hrmmodel.DevelopmentPlan, error) {
	const op = "storage.repo.GetDevelopmentPlanByID"

	const query = `
		SELECT id, employee_id, title, start_date, end_date, status,
			manager_id, approved_at, completed_at, overall_progress, notes, created_at, updated_at
		FROM hrm_development_plans
		WHERE id = $1`

	p, err := r.scanDevelopmentPlan(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get development plan: %w", op, err)
	}

	return p, nil
}

// GetDevelopmentPlans retrieves development plans with filters
func (r *Repo) GetDevelopmentPlans(ctx context.Context, filter hrm.DevelopmentPlanFilter) ([]*hrmmodel.DevelopmentPlan, error) {
	const op = "storage.repo.GetDevelopmentPlans"

	var query strings.Builder
	query.WriteString(`
		SELECT id, employee_id, title, start_date, end_date, status,
			manager_id, approved_at, completed_at, overall_progress, notes, created_at, updated_at
		FROM hrm_development_plans
		WHERE 1=1
	`)

	args := []interface{}{}
	argIdx := 1

	if filter.EmployeeID != nil {
		query.WriteString(fmt.Sprintf(" AND employee_id = $%d", argIdx))
		args = append(args, *filter.EmployeeID)
		argIdx++
	}
	if filter.ManagerID != nil {
		query.WriteString(fmt.Sprintf(" AND manager_id = $%d", argIdx))
		args = append(args, *filter.ManagerID)
		argIdx++
	}
	if filter.Status != nil {
		query.WriteString(fmt.Sprintf(" AND status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}

	query.WriteString(" ORDER BY start_date DESC")

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
		return nil, fmt.Errorf("%s: failed to query development plans: %w", op, err)
	}
	defer rows.Close()

	var plans []*hrmmodel.DevelopmentPlan
	for rows.Next() {
		p, err := r.scanDevelopmentPlanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan development plan: %w", op, err)
		}
		plans = append(plans, p)
	}

	if plans == nil {
		plans = make([]*hrmmodel.DevelopmentPlan, 0)
	}

	return plans, nil
}

// EditDevelopmentPlan updates development plan
func (r *Repo) EditDevelopmentPlan(ctx context.Context, id int64, req hrm.EditDevelopmentPlanRequest) error {
	const op = "storage.repo.EditDevelopmentPlan"

	var updates []string
	var args []interface{}
	argIdx := 1

	if req.Title != nil {
		updates = append(updates, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *req.Title)
		argIdx++
	}
	if req.StartDate != nil {
		updates = append(updates, fmt.Sprintf("start_date = $%d", argIdx))
		args = append(args, *req.StartDate)
		argIdx++
	}
	if req.EndDate != nil {
		updates = append(updates, fmt.Sprintf("end_date = $%d", argIdx))
		args = append(args, *req.EndDate)
		argIdx++
	}
	if req.Status != nil {
		updates = append(updates, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
		// If completed, set completed_at
		if *req.Status == hrmmodel.DevelopmentPlanStatusCompleted {
			updates = append(updates, fmt.Sprintf("completed_at = $%d", argIdx))
			args = append(args, time.Now())
			argIdx++
		}
		// If active, set approved_at
		if *req.Status == hrmmodel.DevelopmentPlanStatusActive {
			updates = append(updates, fmt.Sprintf("approved_at = $%d", argIdx))
			args = append(args, time.Now())
			argIdx++
		}
	}
	if req.ManagerID != nil {
		updates = append(updates, fmt.Sprintf("manager_id = $%d", argIdx))
		args = append(args, *req.ManagerID)
		argIdx++
	}
	if req.OverallProgress != nil {
		updates = append(updates, fmt.Sprintf("overall_progress = $%d", argIdx))
		args = append(args, *req.OverallProgress)
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

	query := fmt.Sprintf("UPDATE hrm_development_plans SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to update development plan: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteDevelopmentPlan deletes development plan
func (r *Repo) DeleteDevelopmentPlan(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteDevelopmentPlan"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_development_plans WHERE id = $1", id)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			return storage.ErrForeignKeyViolation
		}
		return fmt.Errorf("%s: failed to delete development plan: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- Development Goals ---

// AddDevelopmentGoal creates a new development goal
func (r *Repo) AddDevelopmentGoal(ctx context.Context, req hrm.AddDevelopmentGoalRequest) (int64, error) {
	const op = "storage.repo.AddDevelopmentGoal"

	priority := req.Priority
	if priority == "" {
		priority = hrmmodel.PriorityNormal
	}

	const query = `
		INSERT INTO hrm_development_goals (
			plan_id, employee_id, title, description, category,
			target_date, priority, status, training_id, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.PlanID, req.EmployeeID, req.Title, req.Description, req.Category,
		req.TargetDate, priority, hrmmodel.DevelopmentGoalStatusNotStarted, req.TrainingID, req.Notes,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("%s: failed to insert development goal: %w", op, err)
	}

	return id, nil
}

// GetDevelopmentGoalByID retrieves development goal by ID
func (r *Repo) GetDevelopmentGoalByID(ctx context.Context, id int64) (*hrmmodel.DevelopmentGoal, error) {
	const op = "storage.repo.GetDevelopmentGoalByID"

	const query = `
		SELECT id, plan_id, employee_id, title, description, category,
			target_date, priority, status, progress, completed_at,
			training_id, notes, created_at, updated_at
		FROM hrm_development_goals
		WHERE id = $1`

	g, err := r.scanDevelopmentGoal(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get development goal: %w", op, err)
	}

	return g, nil
}

// GetDevelopmentGoals retrieves development goals with filters
func (r *Repo) GetDevelopmentGoals(ctx context.Context, filter hrm.DevelopmentGoalFilter) ([]*hrmmodel.DevelopmentGoal, error) {
	const op = "storage.repo.GetDevelopmentGoals"

	var query strings.Builder
	query.WriteString(`
		SELECT id, plan_id, employee_id, title, description, category,
			target_date, priority, status, progress, completed_at,
			training_id, notes, created_at, updated_at
		FROM hrm_development_goals
		WHERE 1=1
	`)

	args := []interface{}{}
	argIdx := 1

	if filter.PlanID != nil {
		query.WriteString(fmt.Sprintf(" AND plan_id = $%d", argIdx))
		args = append(args, *filter.PlanID)
		argIdx++
	}
	if filter.EmployeeID != nil {
		query.WriteString(fmt.Sprintf(" AND employee_id = $%d", argIdx))
		args = append(args, *filter.EmployeeID)
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
	if filter.Priority != nil {
		query.WriteString(fmt.Sprintf(" AND priority = $%d", argIdx))
		args = append(args, *filter.Priority)
		argIdx++
	}

	query.WriteString(" ORDER BY target_date ASC NULLS LAST")

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query development goals: %w", op, err)
	}
	defer rows.Close()

	var goals []*hrmmodel.DevelopmentGoal
	for rows.Next() {
		g, err := r.scanDevelopmentGoalRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan development goal: %w", op, err)
		}
		goals = append(goals, g)
	}

	if goals == nil {
		goals = make([]*hrmmodel.DevelopmentGoal, 0)
	}

	return goals, nil
}

// EditDevelopmentGoal updates development goal
func (r *Repo) EditDevelopmentGoal(ctx context.Context, id int64, req hrm.EditDevelopmentGoalRequest) error {
	const op = "storage.repo.EditDevelopmentGoal"

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
	if req.TargetDate != nil {
		updates = append(updates, fmt.Sprintf("target_date = $%d", argIdx))
		args = append(args, *req.TargetDate)
		argIdx++
	}
	if req.Priority != nil {
		updates = append(updates, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, *req.Priority)
		argIdx++
	}
	if req.Status != nil {
		updates = append(updates, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
		// If completed, set completed_at
		if *req.Status == hrmmodel.DevelopmentGoalStatusCompleted {
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
	if req.TrainingID != nil {
		updates = append(updates, fmt.Sprintf("training_id = $%d", argIdx))
		args = append(args, *req.TrainingID)
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

	query := fmt.Sprintf("UPDATE hrm_development_goals SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to update development goal: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteDevelopmentGoal deletes development goal
func (r *Repo) DeleteDevelopmentGoal(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteDevelopmentGoal"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_development_goals WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("%s: failed to delete development goal: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- Helpers ---

func (r *Repo) scanTraining(row *sql.Row) (*hrmmodel.Training, error) {
	var t hrmmodel.Training
	var description, category, provider, instructor, location sql.NullString
	var startDate, endDate, updatedAt sql.NullTime
	var durationHours, maxParticipants, minParticipants sql.NullInt64
	var costPerParticipant sql.NullFloat64
	var materialsFileID, organizerID sql.NullInt64

	err := row.Scan(
		&t.ID, &t.Title, &description, &t.TrainingType, &category, &provider, &instructor,
		&startDate, &endDate, &durationHours, &location,
		&maxParticipants, &minParticipants,
		&costPerParticipant, &t.Currency, &t.Status, &t.IsMandatory,
		&materialsFileID, &organizerID, &t.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if description.Valid {
		t.Description = &description.String
	}
	if category.Valid {
		t.Category = &category.String
	}
	if provider.Valid {
		t.Provider = &provider.String
	}
	if instructor.Valid {
		t.Instructor = &instructor.String
	}
	if startDate.Valid {
		t.StartDate = &startDate.Time
	}
	if endDate.Valid {
		t.EndDate = &endDate.Time
	}
	if durationHours.Valid {
		h := int(durationHours.Int64)
		t.DurationHours = &h
	}
	if location.Valid {
		t.Location = &location.String
	}
	if maxParticipants.Valid {
		m := int(maxParticipants.Int64)
		t.MaxParticipants = &m
	}
	if minParticipants.Valid {
		m := int(minParticipants.Int64)
		t.MinParticipants = &m
	}
	if costPerParticipant.Valid {
		t.CostPerParticipant = &costPerParticipant.Float64
	}
	if materialsFileID.Valid {
		t.MaterialsFileID = &materialsFileID.Int64
	}
	if organizerID.Valid {
		t.OrganizerID = &organizerID.Int64
	}
	if updatedAt.Valid {
		t.UpdatedAt = &updatedAt.Time
	}

	return &t, nil
}

func (r *Repo) scanTrainingRow(rows *sql.Rows) (*hrmmodel.Training, error) {
	var t hrmmodel.Training
	var description, category, provider, instructor, location sql.NullString
	var startDate, endDate, updatedAt sql.NullTime
	var durationHours, maxParticipants, minParticipants sql.NullInt64
	var costPerParticipant sql.NullFloat64
	var materialsFileID, organizerID sql.NullInt64

	err := rows.Scan(
		&t.ID, &t.Title, &description, &t.TrainingType, &category, &provider, &instructor,
		&startDate, &endDate, &durationHours, &location,
		&maxParticipants, &minParticipants,
		&costPerParticipant, &t.Currency, &t.Status, &t.IsMandatory,
		&materialsFileID, &organizerID, &t.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if description.Valid {
		t.Description = &description.String
	}
	if category.Valid {
		t.Category = &category.String
	}
	if provider.Valid {
		t.Provider = &provider.String
	}
	if instructor.Valid {
		t.Instructor = &instructor.String
	}
	if startDate.Valid {
		t.StartDate = &startDate.Time
	}
	if endDate.Valid {
		t.EndDate = &endDate.Time
	}
	if durationHours.Valid {
		h := int(durationHours.Int64)
		t.DurationHours = &h
	}
	if location.Valid {
		t.Location = &location.String
	}
	if maxParticipants.Valid {
		m := int(maxParticipants.Int64)
		t.MaxParticipants = &m
	}
	if minParticipants.Valid {
		m := int(minParticipants.Int64)
		t.MinParticipants = &m
	}
	if costPerParticipant.Valid {
		t.CostPerParticipant = &costPerParticipant.Float64
	}
	if materialsFileID.Valid {
		t.MaterialsFileID = &materialsFileID.Int64
	}
	if organizerID.Valid {
		t.OrganizerID = &organizerID.Int64
	}
	if updatedAt.Valid {
		t.UpdatedAt = &updatedAt.Time
	}

	return &t, nil
}

func (r *Repo) scanTrainingParticipant(row *sql.Row) (*hrmmodel.TrainingParticipant, error) {
	var p hrmmodel.TrainingParticipant
	var enrolledBy sql.NullInt64
	var attendancePercent, feedbackRating sql.NullInt64
	var score sql.NullFloat64
	var passed sql.NullBool
	var completedAt, updatedAt sql.NullTime
	var feedbackText, notes sql.NullString

	err := row.Scan(
		&p.ID, &p.TrainingID, &p.EmployeeID, &p.EnrolledAt, &enrolledBy,
		&p.Status, &attendancePercent, &score, &passed, &completedAt,
		&feedbackRating, &feedbackText, &notes, &p.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if enrolledBy.Valid {
		p.EnrolledBy = &enrolledBy.Int64
	}
	if attendancePercent.Valid {
		a := int(attendancePercent.Int64)
		p.AttendancePercent = &a
	}
	if score.Valid {
		p.Score = &score.Float64
	}
	if passed.Valid {
		p.Passed = &passed.Bool
	}
	if completedAt.Valid {
		p.CompletedAt = &completedAt.Time
	}
	if feedbackRating.Valid {
		f := int(feedbackRating.Int64)
		p.FeedbackRating = &f
	}
	if feedbackText.Valid {
		p.FeedbackText = &feedbackText.String
	}
	if notes.Valid {
		p.Notes = &notes.String
	}
	if updatedAt.Valid {
		p.UpdatedAt = &updatedAt.Time
	}

	return &p, nil
}

func (r *Repo) scanTrainingParticipantRow(rows *sql.Rows) (*hrmmodel.TrainingParticipant, error) {
	var p hrmmodel.TrainingParticipant
	var enrolledBy sql.NullInt64
	var attendancePercent, feedbackRating sql.NullInt64
	var score sql.NullFloat64
	var passed sql.NullBool
	var completedAt, updatedAt sql.NullTime
	var feedbackText, notes sql.NullString

	err := rows.Scan(
		&p.ID, &p.TrainingID, &p.EmployeeID, &p.EnrolledAt, &enrolledBy,
		&p.Status, &attendancePercent, &score, &passed, &completedAt,
		&feedbackRating, &feedbackText, &notes, &p.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if enrolledBy.Valid {
		p.EnrolledBy = &enrolledBy.Int64
	}
	if attendancePercent.Valid {
		a := int(attendancePercent.Int64)
		p.AttendancePercent = &a
	}
	if score.Valid {
		p.Score = &score.Float64
	}
	if passed.Valid {
		p.Passed = &passed.Bool
	}
	if completedAt.Valid {
		p.CompletedAt = &completedAt.Time
	}
	if feedbackRating.Valid {
		f := int(feedbackRating.Int64)
		p.FeedbackRating = &f
	}
	if feedbackText.Valid {
		p.FeedbackText = &feedbackText.String
	}
	if notes.Valid {
		p.Notes = &notes.String
	}
	if updatedAt.Valid {
		p.UpdatedAt = &updatedAt.Time
	}

	return &p, nil
}

func (r *Repo) scanCertificate(row *sql.Row) (*hrmmodel.Certificate, error) {
	var c hrmmodel.Certificate
	var trainingID, fileID, verifiedBy sql.NullInt64
	var certificateNumber, notes sql.NullString
	var expiryDate, verifiedAt, updatedAt sql.NullTime

	err := row.Scan(
		&c.ID, &c.EmployeeID, &trainingID, &c.Name, &c.Issuer,
		&certificateNumber, &c.IssuedDate, &expiryDate,
		&fileID, &c.IsVerified, &verifiedBy, &verifiedAt, &notes, &c.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if trainingID.Valid {
		c.TrainingID = &trainingID.Int64
	}
	if certificateNumber.Valid {
		c.CertificateNumber = &certificateNumber.String
	}
	if expiryDate.Valid {
		c.ExpiryDate = &expiryDate.Time
		c.IsExpired = expiryDate.Time.Before(time.Now())
	}
	if fileID.Valid {
		c.FileID = &fileID.Int64
	}
	if verifiedBy.Valid {
		c.VerifiedBy = &verifiedBy.Int64
	}
	if verifiedAt.Valid {
		c.VerifiedAt = &verifiedAt.Time
	}
	if notes.Valid {
		c.Notes = &notes.String
	}
	if updatedAt.Valid {
		c.UpdatedAt = &updatedAt.Time
	}

	return &c, nil
}

func (r *Repo) scanCertificateRow(rows *sql.Rows) (*hrmmodel.Certificate, error) {
	var c hrmmodel.Certificate
	var trainingID, fileID, verifiedBy sql.NullInt64
	var certificateNumber, notes sql.NullString
	var expiryDate, verifiedAt, updatedAt sql.NullTime

	err := rows.Scan(
		&c.ID, &c.EmployeeID, &trainingID, &c.Name, &c.Issuer,
		&certificateNumber, &c.IssuedDate, &expiryDate,
		&fileID, &c.IsVerified, &verifiedBy, &verifiedAt, &notes, &c.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if trainingID.Valid {
		c.TrainingID = &trainingID.Int64
	}
	if certificateNumber.Valid {
		c.CertificateNumber = &certificateNumber.String
	}
	if expiryDate.Valid {
		c.ExpiryDate = &expiryDate.Time
		c.IsExpired = expiryDate.Time.Before(time.Now())
	}
	if fileID.Valid {
		c.FileID = &fileID.Int64
	}
	if verifiedBy.Valid {
		c.VerifiedBy = &verifiedBy.Int64
	}
	if verifiedAt.Valid {
		c.VerifiedAt = &verifiedAt.Time
	}
	if notes.Valid {
		c.Notes = &notes.String
	}
	if updatedAt.Valid {
		c.UpdatedAt = &updatedAt.Time
	}

	return &c, nil
}

func (r *Repo) scanDevelopmentPlan(row *sql.Row) (*hrmmodel.DevelopmentPlan, error) {
	var p hrmmodel.DevelopmentPlan
	var managerID sql.NullInt64
	var approvedAt, completedAt, updatedAt sql.NullTime
	var notes sql.NullString

	err := row.Scan(
		&p.ID, &p.EmployeeID, &p.Title, &p.StartDate, &p.EndDate, &p.Status,
		&managerID, &approvedAt, &completedAt, &p.OverallProgress, &notes, &p.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if managerID.Valid {
		p.ManagerID = &managerID.Int64
	}
	if approvedAt.Valid {
		p.ApprovedAt = &approvedAt.Time
	}
	if completedAt.Valid {
		p.CompletedAt = &completedAt.Time
	}
	if notes.Valid {
		p.Notes = &notes.String
	}
	if updatedAt.Valid {
		p.UpdatedAt = &updatedAt.Time
	}

	return &p, nil
}

func (r *Repo) scanDevelopmentPlanRow(rows *sql.Rows) (*hrmmodel.DevelopmentPlan, error) {
	var p hrmmodel.DevelopmentPlan
	var managerID sql.NullInt64
	var approvedAt, completedAt, updatedAt sql.NullTime
	var notes sql.NullString

	err := rows.Scan(
		&p.ID, &p.EmployeeID, &p.Title, &p.StartDate, &p.EndDate, &p.Status,
		&managerID, &approvedAt, &completedAt, &p.OverallProgress, &notes, &p.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if managerID.Valid {
		p.ManagerID = &managerID.Int64
	}
	if approvedAt.Valid {
		p.ApprovedAt = &approvedAt.Time
	}
	if completedAt.Valid {
		p.CompletedAt = &completedAt.Time
	}
	if notes.Valid {
		p.Notes = &notes.String
	}
	if updatedAt.Valid {
		p.UpdatedAt = &updatedAt.Time
	}

	return &p, nil
}

func (r *Repo) scanDevelopmentGoal(row *sql.Row) (*hrmmodel.DevelopmentGoal, error) {
	var g hrmmodel.DevelopmentGoal
	var description, category, notes sql.NullString
	var targetDate, completedAt, updatedAt sql.NullTime
	var trainingID sql.NullInt64

	err := row.Scan(
		&g.ID, &g.PlanID, &g.EmployeeID, &g.Title, &description, &category,
		&targetDate, &g.Priority, &g.Status, &g.Progress, &completedAt,
		&trainingID, &notes, &g.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if description.Valid {
		g.Description = &description.String
	}
	if category.Valid {
		g.Category = &category.String
	}
	if targetDate.Valid {
		g.TargetDate = &targetDate.Time
	}
	if completedAt.Valid {
		g.CompletedAt = &completedAt.Time
	}
	if trainingID.Valid {
		g.TrainingID = &trainingID.Int64
	}
	if notes.Valid {
		g.Notes = &notes.String
	}
	if updatedAt.Valid {
		g.UpdatedAt = &updatedAt.Time
	}

	return &g, nil
}

func (r *Repo) scanDevelopmentGoalRow(rows *sql.Rows) (*hrmmodel.DevelopmentGoal, error) {
	var g hrmmodel.DevelopmentGoal
	var description, category, notes sql.NullString
	var targetDate, completedAt, updatedAt sql.NullTime
	var trainingID sql.NullInt64

	err := rows.Scan(
		&g.ID, &g.PlanID, &g.EmployeeID, &g.Title, &description, &category,
		&targetDate, &g.Priority, &g.Status, &g.Progress, &completedAt,
		&trainingID, &notes, &g.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if description.Valid {
		g.Description = &description.String
	}
	if category.Valid {
		g.Category = &category.String
	}
	if targetDate.Valid {
		g.TargetDate = &targetDate.Time
	}
	if completedAt.Valid {
		g.CompletedAt = &completedAt.Time
	}
	if trainingID.Valid {
		g.TrainingID = &trainingID.Int64
	}
	if notes.Valid {
		g.Notes = &notes.String
	}
	if updatedAt.Valid {
		g.UpdatedAt = &updatedAt.Time
	}

	return &g, nil
}
