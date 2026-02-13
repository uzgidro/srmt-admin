package competency

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/hrm/competency"
	"srmt-admin/internal/storage"
)

type RepoInterface interface {
	// Competencies
	CreateCompetency(ctx context.Context, req dto.CreateCompetencyRequest, positionIDs []int64) (int64, error)
	GetCompetencyByID(ctx context.Context, id int64) (*competency.Competency, error)
	GetAllCompetencies(ctx context.Context, filters dto.CompetencyFilters) ([]*competency.Competency, error)
	UpdateCompetency(ctx context.Context, id int64, req dto.UpdateCompetencyRequest) error
	DeleteCompetency(ctx context.Context, id int64) error

	// Assessment Sessions
	CreateAssessment(ctx context.Context, req dto.CreateAssessmentRequest, createdBy int64) (int64, error)
	GetAssessmentByID(ctx context.Context, id int64) (*competency.AssessmentSession, error)
	GetAllAssessments(ctx context.Context, filters dto.AssessmentFilters) ([]*competency.AssessmentSession, error)
	UpdateAssessment(ctx context.Context, id int64, req dto.UpdateAssessmentRequest) error
	UpdateAssessmentStatus(ctx context.Context, id int64, status string) error

	// Employee Assessments
	GetEmployeeAssessments(ctx context.Context, employeeID int64) ([]*competency.AssessmentSession, error)

	// Scores
	GetAssessorBySessionAndEmployee(ctx context.Context, sessionID, employeeID int64) (*competency.AssessmentAssessor, error)
	SubmitScores(ctx context.Context, sessionID, assessorID int64, scores []dto.ScoreInput) error

	// Matrices
	GetAllCompetencyMatrices(ctx context.Context) ([]*competency.CompetencyMatrix, error)
	GetPositionMatrix(ctx context.Context, positionID int64) (*competency.CompetencyMatrix, error)

	// GAP Analysis
	GetEmployeeGapAnalysis(ctx context.Context, employeeID int64) (*competency.GapAnalysis, error)

	// Reports
	GetCompetencyReport(ctx context.Context) (*competency.CompetencyReport, error)
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

// ==================== Competencies ====================

func (s *Service) CreateCompetency(ctx context.Context, req dto.CreateCompetencyRequest) (int64, error) {
	return s.repo.CreateCompetency(ctx, req, req.PositionIDs)
}

func (s *Service) GetCompetencyByID(ctx context.Context, id int64) (*competency.Competency, error) {
	return s.repo.GetCompetencyByID(ctx, id)
}

func (s *Service) GetAllCompetencies(ctx context.Context, filters dto.CompetencyFilters) ([]*competency.Competency, error) {
	result, err := s.repo.GetAllCompetencies(ctx, filters)
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = []*competency.Competency{}
	}
	return result, nil
}

func (s *Service) UpdateCompetency(ctx context.Context, id int64, req dto.UpdateCompetencyRequest) error {
	return s.repo.UpdateCompetency(ctx, id, req)
}

func (s *Service) DeleteCompetency(ctx context.Context, id int64) error {
	return s.repo.DeleteCompetency(ctx, id)
}

// ==================== Assessment Sessions ====================

func (s *Service) CreateAssessment(ctx context.Context, req dto.CreateAssessmentRequest, createdBy int64) (int64, error) {
	return s.repo.CreateAssessment(ctx, req, createdBy)
}

func (s *Service) GetAssessmentByID(ctx context.Context, id int64) (*competency.AssessmentSession, error) {
	session, err := s.repo.GetAssessmentByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if session.Competencies == nil {
		session.Competencies = []*competency.AssessmentCompetency{}
	}
	if session.Candidates == nil {
		session.Candidates = []*competency.AssessmentCandidate{}
	}
	if session.Assessors == nil {
		session.Assessors = []*competency.AssessmentAssessor{}
	}
	return session, nil
}

func (s *Service) GetAllAssessments(ctx context.Context, filters dto.AssessmentFilters) ([]*competency.AssessmentSession, error) {
	result, err := s.repo.GetAllAssessments(ctx, filters)
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = []*competency.AssessmentSession{}
	}
	return result, nil
}

func (s *Service) UpdateAssessment(ctx context.Context, id int64, req dto.UpdateAssessmentRequest) error {
	session, err := s.repo.GetAssessmentByID(ctx, id)
	if err != nil {
		return err
	}
	if session.Status != "draft" && session.Status != "planned" {
		return storage.ErrInvalidStatus
	}
	return s.repo.UpdateAssessment(ctx, id, req)
}

func (s *Service) UpdateAssessmentStatus(ctx context.Context, id int64, status string) error {
	session, err := s.repo.GetAssessmentByID(ctx, id)
	if err != nil {
		return err
	}

	valid := map[string][]string{
		"draft":       {"planned", "cancelled"},
		"planned":     {"in_progress", "cancelled"},
		"in_progress": {"completed", "cancelled"},
	}

	allowed, ok := valid[session.Status]
	if !ok {
		return storage.ErrInvalidStatus
	}
	for _, a := range allowed {
		if a == status {
			return s.repo.UpdateAssessmentStatus(ctx, id, status)
		}
	}
	return storage.ErrInvalidStatus
}

func (s *Service) CompleteAssessment(ctx context.Context, id int64) error {
	return s.UpdateAssessmentStatus(ctx, id, "completed")
}

// ==================== Employee Assessments ====================

func (s *Service) GetEmployeeAssessments(ctx context.Context, employeeID int64) ([]*competency.AssessmentSession, error) {
	result, err := s.repo.GetEmployeeAssessments(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = []*competency.AssessmentSession{}
	}
	return result, nil
}

// ==================== Scores ====================

func (s *Service) SubmitScores(ctx context.Context, sessionID, assessorEmployeeID int64, req dto.SubmitScoresRequest) error {
	session, err := s.repo.GetAssessmentByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if session.Status != "in_progress" {
		return storage.ErrInvalidStatus
	}

	assessor, err := s.repo.GetAssessorBySessionAndEmployee(ctx, sessionID, assessorEmployeeID)
	if err != nil {
		return err
	}

	return s.repo.SubmitScores(ctx, sessionID, assessor.ID, req.Scores)
}

// ==================== Matrices ====================

func (s *Service) GetAllCompetencyMatrices(ctx context.Context) ([]*competency.CompetencyMatrix, error) {
	result, err := s.repo.GetAllCompetencyMatrices(ctx)
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = []*competency.CompetencyMatrix{}
	}
	for _, m := range result {
		if m.Items == nil {
			m.Items = []*competency.CompetencyMatrixItem{}
		}
	}
	return result, nil
}

func (s *Service) GetPositionMatrix(ctx context.Context, positionID int64) (*competency.CompetencyMatrix, error) {
	m, err := s.repo.GetPositionMatrix(ctx, positionID)
	if err != nil {
		return nil, err
	}
	if m.Items == nil {
		m.Items = []*competency.CompetencyMatrixItem{}
	}
	return m, nil
}

// ==================== GAP Analysis ====================

func (s *Service) GetEmployeeGapAnalysis(ctx context.Context, employeeID int64) (*competency.GapAnalysis, error) {
	gap, err := s.repo.GetEmployeeGapAnalysis(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if gap.Items == nil {
		gap.Items = []*competency.GapItem{}
	}
	return gap, nil
}

// ==================== Reports ====================

func (s *Service) GetCompetencyReport(ctx context.Context) (*competency.CompetencyReport, error) {
	report, err := s.repo.GetCompetencyReport(ctx)
	if err != nil {
		return nil, err
	}
	if report.ByCategory == nil {
		report.ByCategory = []*competency.CategoryScore{}
	}
	if report.TopGaps == nil {
		report.TopGaps = []*competency.GapItem{}
	}
	return report, nil
}
