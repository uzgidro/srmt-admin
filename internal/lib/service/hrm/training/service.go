package training

import (
	"context"
	"errors"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/hrm/training"
	"srmt-admin/internal/storage"
	"time"
)

type RepoInterface interface {
	// Trainings
	CreateTraining(ctx context.Context, req dto.CreateTrainingRequest, createdBy int64) (int64, error)
	GetTrainingByID(ctx context.Context, id int64) (*training.Training, error)
	GetAllTrainings(ctx context.Context, filters dto.TrainingFilters) ([]*training.Training, error)
	UpdateTraining(ctx context.Context, id int64, req dto.UpdateTrainingRequest) error
	DeleteTraining(ctx context.Context, id int64) error

	// Participants
	AddParticipant(ctx context.Context, trainingID, employeeID int64) (int64, error)
	GetParticipantByID(ctx context.Context, id int64) (*training.Participant, error)
	GetTrainingParticipants(ctx context.Context, trainingID int64) ([]*training.Participant, error)
	CompleteParticipant(ctx context.Context, id int64, score *int, certificateID *int64, notes *string) error
	GetEmployeeTrainings(ctx context.Context, employeeID int64) ([]*training.Training, error)

	// Certificates
	CreateCertificate(ctx context.Context, employeeID int64, trainingID *int64, title string, issuer *string, issueDate string) (int64, error)
	GetEmployeeCertificates(ctx context.Context, employeeID int64) ([]*training.Certificate, error)

	// Development Plans
	CreateDevelopmentPlan(ctx context.Context, req dto.CreateDevelopmentPlanRequest, createdBy int64) (int64, error)
	GetDevelopmentPlanByID(ctx context.Context, id int64) (*training.DevelopmentPlan, error)
	GetAllDevelopmentPlans(ctx context.Context, employeeID *int64) ([]*training.DevelopmentPlan, error)

	// Development Goals
	AddDevelopmentGoal(ctx context.Context, planID int64, req dto.AddDevelopmentGoalRequest) (int64, error)
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

// ==================== Trainings ====================

func (s *Service) CreateTraining(ctx context.Context, req dto.CreateTrainingRequest, createdBy int64) (int64, error) {
	return s.repo.CreateTraining(ctx, req, createdBy)
}

func (s *Service) GetTrainingByID(ctx context.Context, id int64) (*training.Training, error) {
	return s.repo.GetTrainingByID(ctx, id)
}

func (s *Service) GetAllTrainings(ctx context.Context, filters dto.TrainingFilters) ([]*training.Training, error) {
	trainings, err := s.repo.GetAllTrainings(ctx, filters)
	if err != nil {
		return nil, err
	}
	if trainings == nil {
		trainings = []*training.Training{}
	}
	return trainings, nil
}

func (s *Service) UpdateTraining(ctx context.Context, id int64, req dto.UpdateTrainingRequest) error {
	return s.repo.UpdateTraining(ctx, id, req)
}

func (s *Service) DeleteTraining(ctx context.Context, id int64) error {
	tr, err := s.repo.GetTrainingByID(ctx, id)
	if err != nil {
		return err
	}
	if tr.Status == "in_progress" || tr.Status == "completed" {
		return storage.ErrInvalidStatus
	}
	return s.repo.DeleteTraining(ctx, id)
}

// ==================== Participants ====================

func (s *Service) AddParticipant(ctx context.Context, trainingID, employeeID int64) (int64, error) {
	tr, err := s.repo.GetTrainingByID(ctx, trainingID)
	if err != nil {
		return 0, err
	}

	if tr.MaxParticipants > 0 && tr.CurrentParticipants >= tr.MaxParticipants {
		return 0, storage.ErrTrainingFull
	}

	id, err := s.repo.AddParticipant(ctx, trainingID, employeeID)
	if err != nil {
		if errors.Is(err, storage.ErrUniqueViolation) {
			return 0, storage.ErrAlreadyEnrolled
		}
		return 0, err
	}
	return id, nil
}

func (s *Service) GetTrainingParticipants(ctx context.Context, trainingID int64) ([]*training.Participant, error) {
	participants, err := s.repo.GetTrainingParticipants(ctx, trainingID)
	if err != nil {
		return nil, err
	}
	if participants == nil {
		participants = []*training.Participant{}
	}
	return participants, nil
}

func (s *Service) CompleteParticipant(ctx context.Context, participantID int64, req dto.CompleteParticipantRequest) error {
	p, err := s.repo.GetParticipantByID(ctx, participantID)
	if err != nil {
		return err
	}

	// Get training to create certificate
	tr, err := s.repo.GetTrainingByID(ctx, p.TrainingID)
	if err != nil {
		return err
	}

	// Create certificate
	today := time.Now().Format("2006-01-02")
	certID, err := s.repo.CreateCertificate(ctx, p.EmployeeID, &p.TrainingID, tr.Title, tr.Provider, today)
	if err != nil {
		s.log.Error("failed to create certificate", "error", err, "participant_id", participantID)
		// Still complete participant even if certificate creation fails
		return s.repo.CompleteParticipant(ctx, participantID, req.Score, nil, req.Notes)
	}

	return s.repo.CompleteParticipant(ctx, participantID, req.Score, &certID, req.Notes)
}

// ==================== Employee ====================

func (s *Service) GetEmployeeTrainings(ctx context.Context, employeeID int64) ([]*training.Training, error) {
	trainings, err := s.repo.GetEmployeeTrainings(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if trainings == nil {
		trainings = []*training.Training{}
	}
	return trainings, nil
}

func (s *Service) GetEmployeeCertificates(ctx context.Context, employeeID int64) ([]*training.Certificate, error) {
	certs, err := s.repo.GetEmployeeCertificates(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if certs == nil {
		certs = []*training.Certificate{}
	}
	return certs, nil
}

// ==================== Development Plans ====================

func (s *Service) CreateDevelopmentPlan(ctx context.Context, req dto.CreateDevelopmentPlanRequest, createdBy int64) (int64, error) {
	return s.repo.CreateDevelopmentPlan(ctx, req, createdBy)
}

func (s *Service) GetAllDevelopmentPlans(ctx context.Context, employeeID *int64) ([]*training.DevelopmentPlan, error) {
	plans, err := s.repo.GetAllDevelopmentPlans(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if plans == nil {
		plans = []*training.DevelopmentPlan{}
	}
	return plans, nil
}

// ==================== Development Goals ====================

func (s *Service) AddDevelopmentGoal(ctx context.Context, planID int64, req dto.AddDevelopmentGoalRequest) (int64, error) {
	_, err := s.repo.GetDevelopmentPlanByID(ctx, planID)
	if err != nil {
		return 0, err
	}
	return s.repo.AddDevelopmentGoal(ctx, planID, req)
}
