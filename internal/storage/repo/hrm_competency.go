package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/hrm/competency"
	"srmt-admin/internal/storage"
	"strings"
)

// ==================== Competencies ====================

func (r *Repo) CreateCompetency(ctx context.Context, req dto.CreateCompetencyRequest, positionIDs []int64) (int64, error) {
	const op = "repo.CreateCompetency"

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("%s: begin tx: %w", op, err)
	}
	defer tx.Rollback()

	levels := req.Levels
	if levels == nil {
		levels = json.RawMessage("[]")
	}

	query := `
		INSERT INTO competencies (name, description, category, levels)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	var id int64
	err = tx.QueryRowContext(ctx, query, req.Name, req.Description, req.Category, levels).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	for _, posID := range positionIDs {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO competency_positions (competency_id, position_id) VALUES ($1, $2)`,
			id, posID)
		if err != nil {
			return 0, fmt.Errorf("%s: insert position: %w", op, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("%s: commit: %w", op, err)
	}
	return id, nil
}

func (r *Repo) GetCompetencyByID(ctx context.Context, id int64) (*competency.Competency, error) {
	const op = "repo.GetCompetencyByID"

	query := `
		SELECT c.id, c.name, c.description, c.category, c.levels,
			   COALESCE((SELECT jsonb_agg(cp.position_id) FROM competency_positions cp WHERE cp.competency_id = c.id), '[]'::jsonb),
			   c.created_at, c.updated_at
		FROM competencies c
		WHERE c.id = $1`

	comp, err := scanCompetency(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrCompetencyNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return comp, nil
}

func (r *Repo) GetAllCompetencies(ctx context.Context, filters dto.CompetencyFilters) ([]*competency.Competency, error) {
	const op = "repo.GetAllCompetencies"

	query := `
		SELECT c.id, c.name, c.description, c.category, c.levels,
			   COALESCE((SELECT jsonb_agg(cp.position_id) FROM competency_positions cp WHERE cp.competency_id = c.id), '[]'::jsonb),
			   c.created_at, c.updated_at
		FROM competencies c`

	var conditions []string
	var args []interface{}
	argIdx := 1

	if filters.Category != nil {
		conditions = append(conditions, fmt.Sprintf("c.category = $%d", argIdx))
		args = append(args, *filters.Category)
		argIdx++
	}
	if filters.Search != nil {
		conditions = append(conditions, fmt.Sprintf("c.name ILIKE $%d", argIdx))
		args = append(args, "%"+*filters.Search+"%")
		argIdx++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY c.name"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var result []*competency.Competency
	for rows.Next() {
		c, err := scanCompetency(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		result = append(result, c)
	}
	return result, nil
}

func (r *Repo) UpdateCompetency(ctx context.Context, id int64, req dto.UpdateCompetencyRequest) error {
	const op = "repo.UpdateCompetency"

	var setClauses []string
	var args []interface{}
	argIdx := 1

	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.Category != nil {
		setClauses = append(setClauses, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, *req.Category)
		argIdx++
	}
	if req.Levels != nil {
		setClauses = append(setClauses, fmt.Sprintf("levels = $%d", argIdx))
		args = append(args, *req.Levels)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE competencies SET %s WHERE id = $%d", strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return storage.ErrCompetencyNotFound
	}
	return nil
}

func (r *Repo) DeleteCompetency(ctx context.Context, id int64) error {
	const op = "repo.DeleteCompetency"

	res, err := r.db.ExecContext(ctx, "DELETE FROM competencies WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return storage.ErrCompetencyNotFound
	}
	return nil
}

// ==================== Assessment Sessions ====================

func (r *Repo) CreateAssessment(ctx context.Context, req dto.CreateAssessmentRequest, createdBy int64) (int64, error) {
	const op = "repo.CreateAssessment"

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("%s: begin tx: %w", op, err)
	}
	defer tx.Rollback()

	var id int64
	err = tx.QueryRowContext(ctx,
		`INSERT INTO assessment_sessions (name, description, start_date, end_date, created_by)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		req.Name, req.Description, req.StartDate, req.EndDate, createdBy,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("%s: insert session: %w", op, err)
	}

	for _, c := range req.Competencies {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO assessment_competencies (session_id, competency_id, weight, required_level)
			 VALUES ($1, $2, $3, $4)`,
			id, c.CompetencyID, c.Weight, c.RequiredLevel)
		if err != nil {
			return 0, fmt.Errorf("%s: insert competency: %w", op, err)
		}
	}

	for _, candidateID := range req.Candidates {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO assessment_candidates (session_id, employee_id) VALUES ($1, $2)`,
			id, candidateID)
		if err != nil {
			return 0, fmt.Errorf("%s: insert candidate: %w", op, err)
		}
	}

	for _, a := range req.Assessors {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO assessment_assessors (session_id, employee_id, role) VALUES ($1, $2, $3)`,
			id, a.EmployeeID, a.Role)
		if err != nil {
			return 0, fmt.Errorf("%s: insert assessor: %w", op, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("%s: commit: %w", op, err)
	}
	return id, nil
}

func (r *Repo) GetAssessmentByID(ctx context.Context, id int64) (*competency.AssessmentSession, error) {
	const op = "repo.GetAssessmentByID"

	query := `
		SELECT id, name, description, status, start_date, end_date, created_by, created_at, updated_at
		FROM assessment_sessions
		WHERE id = $1`

	session, err := scanAssessmentSession(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrAssessmentNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Load competencies
	compRows, err := r.db.QueryContext(ctx,
		`SELECT ac.id, ac.session_id, ac.competency_id, COALESCE(c.name, ''), ac.weight, ac.required_level
		 FROM assessment_competencies ac
		 LEFT JOIN competencies c ON ac.competency_id = c.id
		 WHERE ac.session_id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("%s: load competencies: %w", op, err)
	}
	defer compRows.Close()
	for compRows.Next() {
		ac, err := scanAssessmentCompetency(compRows)
		if err != nil {
			return nil, fmt.Errorf("%s: scan competency: %w", op, err)
		}
		session.Competencies = append(session.Competencies, ac)
	}

	// Load candidates
	candRows, err := r.db.QueryContext(ctx,
		`SELECT ac.id, ac.session_id, ac.employee_id,
				COALESCE(c.first_name || ' ' || c.last_name, ''),
				COALESCE(p.name, ''),
				COALESCE(d.name, ''),
				ac.status
		 FROM assessment_candidates ac
		 LEFT JOIN contacts c ON ac.employee_id = c.id
		 LEFT JOIN personnel_records pr ON pr.contact_id = ac.employee_id
		 LEFT JOIN positions p ON pr.position_id = p.id
		 LEFT JOIN departments d ON pr.department_id = d.id
		 WHERE ac.session_id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("%s: load candidates: %w", op, err)
	}
	defer candRows.Close()
	for candRows.Next() {
		ac, err := scanAssessmentCandidate(candRows)
		if err != nil {
			return nil, fmt.Errorf("%s: scan candidate: %w", op, err)
		}
		session.Candidates = append(session.Candidates, ac)
	}

	// Load assessors
	assRows, err := r.db.QueryContext(ctx,
		`SELECT aa.id, aa.session_id, aa.employee_id,
				COALESCE(c.first_name || ' ' || c.last_name, ''),
				aa.role
		 FROM assessment_assessors aa
		 LEFT JOIN contacts c ON aa.employee_id = c.id
		 WHERE aa.session_id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("%s: load assessors: %w", op, err)
	}
	defer assRows.Close()
	for assRows.Next() {
		aa, err := scanAssessmentAssessor(assRows)
		if err != nil {
			return nil, fmt.Errorf("%s: scan assessor: %w", op, err)
		}
		session.Assessors = append(session.Assessors, aa)
	}

	return session, nil
}

func (r *Repo) GetAllAssessments(ctx context.Context, filters dto.AssessmentFilters) ([]*competency.AssessmentSession, error) {
	const op = "repo.GetAllAssessments"

	query := `
		SELECT id, name, description, status, start_date, end_date, created_by, created_at, updated_at
		FROM assessment_sessions`

	var conditions []string
	var args []interface{}
	argIdx := 1

	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filters.Status)
		argIdx++
	}
	if filters.Search != nil {
		conditions = append(conditions, fmt.Sprintf("name ILIKE $%d", argIdx))
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

	var result []*competency.AssessmentSession
	for rows.Next() {
		s, err := scanAssessmentSession(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		result = append(result, s)
	}
	return result, nil
}

func (r *Repo) UpdateAssessment(ctx context.Context, id int64, req dto.UpdateAssessmentRequest) error {
	const op = "repo.UpdateAssessment"

	var setClauses []string
	var args []interface{}
	argIdx := 1

	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
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

	if len(setClauses) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE assessment_sessions SET %s WHERE id = $%d", strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return storage.ErrAssessmentNotFound
	}
	return nil
}

func (r *Repo) UpdateAssessmentStatus(ctx context.Context, id int64, status string) error {
	const op = "repo.UpdateAssessmentStatus"

	res, err := r.db.ExecContext(ctx, "UPDATE assessment_sessions SET status = $1 WHERE id = $2", status, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return storage.ErrAssessmentNotFound
	}
	return nil
}

// ==================== Employee Assessments ====================

func (r *Repo) GetEmployeeAssessments(ctx context.Context, employeeID int64) ([]*competency.AssessmentSession, error) {
	const op = "repo.GetEmployeeAssessments"

	query := `
		SELECT s.id, s.name, s.description, s.status, s.start_date, s.end_date, s.created_by, s.created_at, s.updated_at
		FROM assessment_sessions s
		JOIN assessment_candidates ac ON ac.session_id = s.id
		WHERE ac.employee_id = $1
		ORDER BY s.created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, employeeID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var result []*competency.AssessmentSession
	for rows.Next() {
		s, err := scanAssessmentSession(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		result = append(result, s)
	}
	return result, nil
}

// ==================== Scores ====================

func (r *Repo) GetAssessorBySessionAndEmployee(ctx context.Context, sessionID, employeeID int64) (*competency.AssessmentAssessor, error) {
	const op = "repo.GetAssessorBySessionAndEmployee"

	query := `
		SELECT aa.id, aa.session_id, aa.employee_id,
			   COALESCE(c.first_name || ' ' || c.last_name, ''),
			   aa.role
		FROM assessment_assessors aa
		LEFT JOIN contacts c ON aa.employee_id = c.id
		WHERE aa.session_id = $1 AND aa.employee_id = $2`

	aa, err := scanAssessmentAssessor(r.db.QueryRowContext(ctx, query, sessionID, employeeID))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return aa, nil
}

func (r *Repo) SubmitScores(ctx context.Context, sessionID, assessorID int64, scores []dto.ScoreInput) error {
	const op = "repo.SubmitScores"

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: begin tx: %w", op, err)
	}
	defer tx.Rollback()

	for _, s := range scores {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO assessment_scores (session_id, candidate_id, assessor_id, competency_id, score, comment)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 ON CONFLICT (session_id, candidate_id, assessor_id, competency_id)
			 DO UPDATE SET score = EXCLUDED.score, comment = EXCLUDED.comment`,
			sessionID, s.CandidateID, assessorID, s.CompetencyID, s.Score, s.Comment)
		if err != nil {
			return fmt.Errorf("%s: upsert score: %w", op, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("%s: commit: %w", op, err)
	}
	return nil
}

// ==================== Competency Matrices ====================

func (r *Repo) GetAllCompetencyMatrices(ctx context.Context) ([]*competency.CompetencyMatrix, error) {
	const op = "repo.GetAllCompetencyMatrices"

	query := `
		SELECT cp.position_id, COALESCE(p.name, ''),
			   c.id, c.name, c.category, cp.required_level
		FROM competency_positions cp
		JOIN competencies c ON cp.competency_id = c.id
		JOIN positions p ON cp.position_id = p.id
		ORDER BY cp.position_id, c.name`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	matrixMap := make(map[int64]*competency.CompetencyMatrix)
	var order []int64

	for rows.Next() {
		var posID int64
		var posName string
		var item competency.CompetencyMatrixItem
		if err := rows.Scan(&posID, &posName, &item.CompetencyID, &item.CompetencyName, &item.Category, &item.RequiredLevel); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		m, ok := matrixMap[posID]
		if !ok {
			m = &competency.CompetencyMatrix{PositionID: posID, PositionName: posName}
			matrixMap[posID] = m
			order = append(order, posID)
		}
		m.Items = append(m.Items, &item)
	}

	var result []*competency.CompetencyMatrix
	for _, id := range order {
		result = append(result, matrixMap[id])
	}
	return result, nil
}

func (r *Repo) GetPositionMatrix(ctx context.Context, positionID int64) (*competency.CompetencyMatrix, error) {
	const op = "repo.GetPositionMatrix"

	var posName string
	err := r.db.QueryRowContext(ctx, "SELECT name FROM positions WHERE id = $1", positionID).Scan(&posName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: get position: %w", op, err)
	}

	query := `
		SELECT c.id, c.name, c.category, cp.required_level
		FROM competency_positions cp
		JOIN competencies c ON cp.competency_id = c.id
		WHERE cp.position_id = $1
		ORDER BY c.name`

	rows, err := r.db.QueryContext(ctx, query, positionID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	matrix := &competency.CompetencyMatrix{
		PositionID:   positionID,
		PositionName: posName,
	}
	for rows.Next() {
		var item competency.CompetencyMatrixItem
		if err := rows.Scan(&item.CompetencyID, &item.CompetencyName, &item.Category, &item.RequiredLevel); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		matrix.Items = append(matrix.Items, &item)
	}
	return matrix, nil
}

// ==================== GAP Analysis ====================

func (r *Repo) GetEmployeeGapAnalysis(ctx context.Context, employeeID int64) (*competency.GapAnalysis, error) {
	const op = "repo.GetEmployeeGapAnalysis"

	// Get employee info
	var empName, position string
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(c.first_name || ' ' || c.last_name, ''), COALESCE(p.name, '')
		 FROM contacts c
		 LEFT JOIN personnel_records pr ON pr.contact_id = c.id
		 LEFT JOIN positions p ON pr.position_id = p.id
		 WHERE c.id = $1`, employeeID).Scan(&empName, &position)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: get employee: %w", op, err)
	}

	query := `
		SELECT c.id, c.name, c.category, cp.required_level,
			   COALESCE(AVG(s.score), 0) as current_level,
			   cp.required_level - COALESCE(AVG(s.score), 0) as gap
		FROM competency_positions cp
		JOIN competencies c ON cp.competency_id = c.id
		LEFT JOIN assessment_scores s ON s.competency_id = c.id
			AND s.candidate_id IN (SELECT ac.id FROM assessment_candidates ac WHERE ac.employee_id = $1)
		WHERE cp.position_id = (SELECT position_id FROM personnel_records WHERE contact_id = $2 LIMIT 1)
		GROUP BY c.id, c.name, c.category, cp.required_level
		ORDER BY gap DESC`

	rows, err := r.db.QueryContext(ctx, query, employeeID, employeeID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	gap := &competency.GapAnalysis{
		EmployeeID:   employeeID,
		EmployeeName: empName,
		Position:     position,
	}
	for rows.Next() {
		var item competency.GapItem
		if err := rows.Scan(&item.CompetencyID, &item.CompetencyName, &item.Category,
			&item.RequiredLevel, &item.CurrentLevel, &item.Gap); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		gap.Items = append(gap.Items, &item)
	}
	return gap, nil
}

// ==================== Reports ====================

func (r *Repo) GetCompetencyReport(ctx context.Context) (*competency.CompetencyReport, error) {
	const op = "repo.GetCompetencyReport"

	report := &competency.CompetencyReport{}

	// Total and completed assessments
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*), COUNT(*) FILTER (WHERE status = 'completed')
		 FROM assessment_sessions`).Scan(&report.TotalAssessments, &report.CompletedCount)
	if err != nil {
		return nil, fmt.Errorf("%s: count sessions: %w", op, err)
	}

	// Average score
	var avgScore sql.NullFloat64
	err = r.db.QueryRowContext(ctx, "SELECT AVG(score) FROM assessment_scores").Scan(&avgScore)
	if err != nil {
		return nil, fmt.Errorf("%s: avg score: %w", op, err)
	}
	if avgScore.Valid {
		report.AverageScore = avgScore.Float64
	}

	// By category
	catRows, err := r.db.QueryContext(ctx,
		`SELECT c.category, AVG(s.score), COUNT(DISTINCT ac.employee_id)
		 FROM assessment_scores s
		 JOIN competencies c ON s.competency_id = c.id
		 JOIN assessment_candidates ac ON s.candidate_id = ac.id
		 GROUP BY c.category
		 ORDER BY c.category`)
	if err != nil {
		return nil, fmt.Errorf("%s: by category: %w", op, err)
	}
	defer catRows.Close()
	for catRows.Next() {
		var cs competency.CategoryScore
		if err := catRows.Scan(&cs.Category, &cs.AverageScore, &cs.EmployeeCount); err != nil {
			return nil, fmt.Errorf("%s: scan category: %w", op, err)
		}
		report.ByCategory = append(report.ByCategory, &cs)
	}

	// Top gaps (across all employees, top 10)
	gapRows, err := r.db.QueryContext(ctx,
		`SELECT c.id, c.name, c.category, cp.required_level,
				COALESCE(AVG(s.score), 0) as current_level,
				cp.required_level - COALESCE(AVG(s.score), 0) as gap
		 FROM competency_positions cp
		 JOIN competencies c ON cp.competency_id = c.id
		 LEFT JOIN assessment_scores s ON s.competency_id = c.id
		 GROUP BY c.id, c.name, c.category, cp.required_level
		 HAVING cp.required_level - COALESCE(AVG(s.score), 0) > 0
		 ORDER BY gap DESC
		 LIMIT 10`)
	if err != nil {
		return nil, fmt.Errorf("%s: top gaps: %w", op, err)
	}
	defer gapRows.Close()
	for gapRows.Next() {
		var item competency.GapItem
		if err := gapRows.Scan(&item.CompetencyID, &item.CompetencyName, &item.Category,
			&item.RequiredLevel, &item.CurrentLevel, &item.Gap); err != nil {
			return nil, fmt.Errorf("%s: scan gap: %w", op, err)
		}
		report.TopGaps = append(report.TopGaps, &item)
	}

	return report, nil
}

// ==================== Scanners ====================

func scanCompetency(s scannable) (*competency.Competency, error) {
	var c competency.Competency
	var levels, positions []byte
	err := s.Scan(&c.ID, &c.Name, &c.Description, &c.Category, &levels, &positions, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	c.Levels = json.RawMessage(levels)
	c.RequiredForPositions = json.RawMessage(positions)
	return &c, nil
}

func scanAssessmentSession(s scannable) (*competency.AssessmentSession, error) {
	var sess competency.AssessmentSession
	err := s.Scan(&sess.ID, &sess.Name, &sess.Description, &sess.Status,
		&sess.StartDate, &sess.EndDate, &sess.CreatedBy, &sess.CreatedAt, &sess.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func scanAssessmentCompetency(s scannable) (*competency.AssessmentCompetency, error) {
	var ac competency.AssessmentCompetency
	err := s.Scan(&ac.ID, &ac.SessionID, &ac.CompetencyID, &ac.CompetencyName, &ac.Weight, &ac.RequiredLevel)
	if err != nil {
		return nil, err
	}
	return &ac, nil
}

func scanAssessmentCandidate(s scannable) (*competency.AssessmentCandidate, error) {
	var ac competency.AssessmentCandidate
	err := s.Scan(&ac.ID, &ac.SessionID, &ac.EmployeeID, &ac.Name, &ac.Position, &ac.Department, &ac.Status)
	if err != nil {
		return nil, err
	}
	return &ac, nil
}

func scanAssessmentAssessor(s scannable) (*competency.AssessmentAssessor, error) {
	var aa competency.AssessmentAssessor
	err := s.Scan(&aa.ID, &aa.SessionID, &aa.EmployeeID, &aa.Name, &aa.Role)
	if err != nil {
		return nil, err
	}
	return &aa, nil
}
