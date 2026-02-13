package recruiting

import (
	"context"
	"fmt"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/hrm/recruiting"
	"srmt-admin/internal/storage"
	"time"
)

type RepoInterface interface {
	// Vacancies
	CreateVacancy(ctx context.Context, req dto.CreateVacancyRequest, createdBy int64) (int64, error)
	GetVacancyByID(ctx context.Context, id int64) (*recruiting.Vacancy, error)
	GetAllVacancies(ctx context.Context, filters dto.VacancyFilters) ([]*recruiting.Vacancy, error)
	UpdateVacancy(ctx context.Context, id int64, req dto.UpdateVacancyRequest) error
	DeleteVacancy(ctx context.Context, id int64) error
	UpdateVacancyStatus(ctx context.Context, id int64, status string) error

	// Candidates
	CreateCandidate(ctx context.Context, req dto.CreateCandidateRequest) (int64, error)
	GetCandidateByID(ctx context.Context, id int64) (*recruiting.CandidateListItem, error)
	GetAllCandidates(ctx context.Context, filters dto.CandidateFilters) ([]*recruiting.CandidateListItem, error)
	UpdateCandidate(ctx context.Context, id int64, req dto.UpdateCandidateRequest) error
	DeleteCandidate(ctx context.Context, id int64) error
	UpdateCandidateStatus(ctx context.Context, id int64, status, stage string) error

	// Education / Experience
	CreateCandidateEducation(ctx context.Context, candidateID int64, items []dto.EducationInput) error
	GetCandidateEducation(ctx context.Context, candidateID int64) ([]recruiting.Education, error)
	DeleteCandidateEducation(ctx context.Context, candidateID int64) error
	CreateCandidateExperience(ctx context.Context, candidateID int64, items []dto.ExperienceInput) error
	GetCandidateExperience(ctx context.Context, candidateID int64) ([]recruiting.Experience, error)
	DeleteCandidateExperience(ctx context.Context, candidateID int64) error

	// Interviews
	CreateInterview(ctx context.Context, req dto.CreateInterviewRequest, scheduledAt time.Time) (int64, error)
	GetInterviewByID(ctx context.Context, id int64) (*recruiting.Interview, error)
	GetAllInterviews(ctx context.Context, filters dto.InterviewFilters) ([]*recruiting.Interview, error)
	UpdateInterview(ctx context.Context, id int64, req dto.UpdateInterviewRequest) error
	GetInterviewsByCandidate(ctx context.Context, candidateID int64) ([]*recruiting.Interview, error)
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

// ==================== Vacancies ====================

func (s *Service) CreateVacancy(ctx context.Context, req dto.CreateVacancyRequest, createdBy int64) (int64, error) {
	return s.repo.CreateVacancy(ctx, req, createdBy)
}

func (s *Service) GetVacancyByID(ctx context.Context, id int64) (*recruiting.Vacancy, error) {
	return s.repo.GetVacancyByID(ctx, id)
}

func (s *Service) GetAllVacancies(ctx context.Context, filters dto.VacancyFilters) ([]*recruiting.Vacancy, error) {
	vacancies, err := s.repo.GetAllVacancies(ctx, filters)
	if err != nil {
		return nil, err
	}
	if vacancies == nil {
		vacancies = []*recruiting.Vacancy{}
	}
	return vacancies, nil
}

func (s *Service) UpdateVacancy(ctx context.Context, id int64, req dto.UpdateVacancyRequest) error {
	vacancy, err := s.repo.GetVacancyByID(ctx, id)
	if err != nil {
		return err
	}
	if vacancy.Status != "draft" && vacancy.Status != "on_hold" {
		return storage.ErrInvalidStatus
	}
	return s.repo.UpdateVacancy(ctx, id, req)
}

func (s *Service) DeleteVacancy(ctx context.Context, id int64) error {
	vacancy, err := s.repo.GetVacancyByID(ctx, id)
	if err != nil {
		return err
	}
	if vacancy.Status != "draft" {
		return storage.ErrInvalidStatus
	}
	return s.repo.DeleteVacancy(ctx, id)
}

func (s *Service) PublishVacancy(ctx context.Context, id int64) error {
	vacancy, err := s.repo.GetVacancyByID(ctx, id)
	if err != nil {
		return err
	}
	if vacancy.Status != "approved" {
		return storage.ErrInvalidStatus
	}
	return s.repo.UpdateVacancyStatus(ctx, id, "published")
}

func (s *Service) CloseVacancy(ctx context.Context, id int64) error {
	vacancy, err := s.repo.GetVacancyByID(ctx, id)
	if err != nil {
		return err
	}
	if vacancy.Status != "published" {
		return storage.ErrVacancyNotPublished
	}
	return s.repo.UpdateVacancyStatus(ctx, id, "closed")
}

// ==================== Candidates ====================

func (s *Service) CreateCandidate(ctx context.Context, req dto.CreateCandidateRequest) (int64, error) {
	id, err := s.repo.CreateCandidate(ctx, req)
	if err != nil {
		return 0, err
	}

	if len(req.Education) > 0 {
		if err := s.repo.CreateCandidateEducation(ctx, id, req.Education); err != nil {
			return 0, fmt.Errorf("create education: %w", err)
		}
	}

	if len(req.WorkExperience) > 0 {
		if err := s.repo.CreateCandidateExperience(ctx, id, req.WorkExperience); err != nil {
			return 0, fmt.Errorf("create experience: %w", err)
		}
	}

	return id, nil
}

func (s *Service) GetCandidateByID(ctx context.Context, id int64) (*recruiting.Candidate, error) {
	item, err := s.repo.GetCandidateByID(ctx, id)
	if err != nil {
		return nil, err
	}

	education, err := s.repo.GetCandidateEducation(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get education: %w", err)
	}
	if education == nil {
		education = []recruiting.Education{}
	}

	experience, err := s.repo.GetCandidateExperience(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get experience: %w", err)
	}
	if experience == nil {
		experience = []recruiting.Experience{}
	}

	return &recruiting.Candidate{
		CandidateListItem: *item,
		Education:         education,
		WorkExperience:    experience,
	}, nil
}

func (s *Service) GetAllCandidates(ctx context.Context, filters dto.CandidateFilters) ([]*recruiting.CandidateListItem, error) {
	candidates, err := s.repo.GetAllCandidates(ctx, filters)
	if err != nil {
		return nil, err
	}
	if candidates == nil {
		candidates = []*recruiting.CandidateListItem{}
	}
	return candidates, nil
}

func (s *Service) UpdateCandidate(ctx context.Context, id int64, req dto.UpdateCandidateRequest) error {
	if err := s.repo.UpdateCandidate(ctx, id, req); err != nil {
		return err
	}

	if req.Education != nil {
		if err := s.repo.DeleteCandidateEducation(ctx, id); err != nil {
			return fmt.Errorf("delete education: %w", err)
		}
		if len(*req.Education) > 0 {
			if err := s.repo.CreateCandidateEducation(ctx, id, *req.Education); err != nil {
				return fmt.Errorf("create education: %w", err)
			}
		}
	}

	if req.WorkExperience != nil {
		if err := s.repo.DeleteCandidateExperience(ctx, id); err != nil {
			return fmt.Errorf("delete experience: %w", err)
		}
		if len(*req.WorkExperience) > 0 {
			if err := s.repo.CreateCandidateExperience(ctx, id, *req.WorkExperience); err != nil {
				return fmt.Errorf("create experience: %w", err)
			}
		}
	}

	return nil
}

func (s *Service) DeleteCandidate(ctx context.Context, id int64) error {
	return s.repo.DeleteCandidate(ctx, id)
}

func (s *Service) ChangeCandidateStatus(ctx context.Context, id int64, req dto.ChangeCandidateStatusRequest) error {
	_, err := s.repo.GetCandidateByID(ctx, id)
	if err != nil {
		return err
	}
	return s.repo.UpdateCandidateStatus(ctx, id, req.Status, req.Stage)
}

// ==================== Interviews ====================

func (s *Service) CreateInterview(ctx context.Context, req dto.CreateInterviewRequest) (int64, error) {
	scheduledAt, err := time.Parse(time.RFC3339, req.ScheduledAt)
	if err != nil {
		return 0, fmt.Errorf("invalid scheduled_at format: %w", err)
	}
	return s.repo.CreateInterview(ctx, req, scheduledAt)
}

func (s *Service) GetInterviewByID(ctx context.Context, id int64) (*recruiting.Interview, error) {
	return s.repo.GetInterviewByID(ctx, id)
}

func (s *Service) GetAllInterviews(ctx context.Context, filters dto.InterviewFilters) ([]*recruiting.Interview, error) {
	interviews, err := s.repo.GetAllInterviews(ctx, filters)
	if err != nil {
		return nil, err
	}
	if interviews == nil {
		interviews = []*recruiting.Interview{}
	}
	return interviews, nil
}

func (s *Service) UpdateInterview(ctx context.Context, id int64, req dto.UpdateInterviewRequest) error {
	return s.repo.UpdateInterview(ctx, id, req)
}

func (s *Service) GetCandidateInterviews(ctx context.Context, candidateID int64) ([]*recruiting.Interview, error) {
	_, err := s.repo.GetCandidateByID(ctx, candidateID)
	if err != nil {
		return nil, err
	}
	interviews, err := s.repo.GetInterviewsByCandidate(ctx, candidateID)
	if err != nil {
		return nil, err
	}
	if interviews == nil {
		interviews = []*recruiting.Interview{}
	}
	return interviews, nil
}
