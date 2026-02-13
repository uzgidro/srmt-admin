package performance

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/hrm/performance"
	"srmt-admin/internal/storage"
)

type RepoInterface interface {
	// Reviews
	CreateReview(ctx context.Context, req dto.CreateReviewRequest) (int64, error)
	GetReviewByID(ctx context.Context, id int64) (*performance.PerformanceReview, error)
	GetAllReviews(ctx context.Context, filters dto.ReviewFilters) ([]*performance.PerformanceReview, error)
	UpdateReview(ctx context.Context, id int64, req dto.UpdateReviewRequest) error
	UpdateReviewFields(ctx context.Context, id int64, fields map[string]interface{}) error
	UpdateReviewStatus(ctx context.Context, id int64, status string) error

	// Goals
	CreateGoal(ctx context.Context, req dto.CreateGoalRequest) (int64, error)
	GetGoalByID(ctx context.Context, id int64) (*performance.PerformanceGoal, error)
	GetAllGoals(ctx context.Context, filters dto.GoalFilters) ([]*performance.PerformanceGoal, error)
	GetGoalsByReviewID(ctx context.Context, reviewID int64) ([]*performance.PerformanceGoal, error)
	UpdateGoal(ctx context.Context, id int64, req dto.UpdateGoalRequest) error
	UpdateGoalProgress(ctx context.Context, id int64, currentValue float64, progress int, status string) error
	DeleteGoal(ctx context.Context, id int64) error

	// Analytics
	GetKPIs(ctx context.Context) ([]*performance.KPI, error)
	GetAllRatings(ctx context.Context) ([]*performance.EmployeeRating, error)
	GetEmployeeRating(ctx context.Context, employeeID int64) (*performance.EmployeeRating, error)
	GetPerformanceDashboard(ctx context.Context) (*performance.PerformanceDashboard, error)
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

// ==================== Reviews ====================

func (s *Service) CreateReview(ctx context.Context, req dto.CreateReviewRequest) (int64, error) {
	return s.repo.CreateReview(ctx, req)
}

func (s *Service) GetReviewByID(ctx context.Context, id int64) (*performance.PerformanceReview, error) {
	review, err := s.repo.GetReviewByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if review.Goals == nil {
		review.Goals = []*performance.PerformanceGoal{}
	}
	return review, nil
}

func (s *Service) GetAllReviews(ctx context.Context, filters dto.ReviewFilters) ([]*performance.PerformanceReview, error) {
	result, err := s.repo.GetAllReviews(ctx, filters)
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = []*performance.PerformanceReview{}
	}
	return result, nil
}

func (s *Service) UpdateReview(ctx context.Context, id int64, req dto.UpdateReviewRequest) error {
	review, err := s.repo.GetReviewByID(ctx, id)
	if err != nil {
		return err
	}
	if review.Status != "draft" {
		return storage.ErrInvalidStatus
	}
	return s.repo.UpdateReview(ctx, id, req)
}

func (s *Service) SelfReview(ctx context.Context, id int64, req dto.SelfReviewRequest) error {
	review, err := s.repo.GetReviewByID(ctx, id)
	if err != nil {
		return err
	}
	if review.Status != "self_review" {
		return storage.ErrInvalidStatus
	}

	fields := map[string]interface{}{
		"self_rating": req.SelfRating,
	}
	if req.SelfComment != nil {
		fields["self_comment"] = *req.SelfComment
	}

	if err := s.repo.UpdateReviewFields(ctx, id, fields); err != nil {
		return err
	}

	return s.repo.UpdateReviewStatus(ctx, id, "manager_review")
}

func (s *Service) ManagerReview(ctx context.Context, id int64, req dto.ManagerReviewRequest) error {
	review, err := s.repo.GetReviewByID(ctx, id)
	if err != nil {
		return err
	}
	if review.Status != "manager_review" {
		return storage.ErrInvalidStatus
	}

	fields := map[string]interface{}{
		"manager_rating": req.ManagerRating,
	}
	if req.ManagerComment != nil {
		fields["manager_comment"] = *req.ManagerComment
	}
	if req.FinalRating != nil {
		fields["final_rating"] = *req.FinalRating
	}
	if req.Strengths != nil {
		fields["strengths"] = *req.Strengths
	}
	if req.Improvements != nil {
		fields["improvements"] = *req.Improvements
	}

	if err := s.repo.UpdateReviewFields(ctx, id, fields); err != nil {
		return err
	}

	return s.repo.UpdateReviewStatus(ctx, id, "completed")
}

func (s *Service) CompleteReview(ctx context.Context, id int64) error {
	review, err := s.repo.GetReviewByID(ctx, id)
	if err != nil {
		return err
	}
	if review.Status != "calibration" {
		return storage.ErrInvalidStatus
	}
	return s.repo.UpdateReviewStatus(ctx, id, "completed")
}

func (s *Service) UpdateReviewStatus(ctx context.Context, id int64, status string) error {
	review, err := s.repo.GetReviewByID(ctx, id)
	if err != nil {
		return err
	}

	valid := map[string][]string{
		"draft":          {"self_review"},
		"self_review":    {"manager_review"},
		"manager_review": {"calibration", "completed"},
		"calibration":    {"completed"},
		"completed":      {"acknowledged"},
	}

	allowed, ok := valid[review.Status]
	if !ok {
		return storage.ErrInvalidStatus
	}
	for _, a := range allowed {
		if a == status {
			return s.repo.UpdateReviewStatus(ctx, id, status)
		}
	}
	return storage.ErrInvalidStatus
}

// ==================== Goals ====================

func (s *Service) CreateGoal(ctx context.Context, req dto.CreateGoalRequest) (int64, error) {
	return s.repo.CreateGoal(ctx, req)
}

func (s *Service) GetAllGoals(ctx context.Context, filters dto.GoalFilters) ([]*performance.PerformanceGoal, error) {
	result, err := s.repo.GetAllGoals(ctx, filters)
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = []*performance.PerformanceGoal{}
	}
	return result, nil
}

func (s *Service) UpdateGoal(ctx context.Context, id int64, req dto.UpdateGoalRequest) error {
	return s.repo.UpdateGoal(ctx, id, req)
}

func (s *Service) UpdateGoalProgress(ctx context.Context, id int64, req dto.UpdateGoalProgressRequest) error {
	goal, err := s.repo.GetGoalByID(ctx, id)
	if err != nil {
		return err
	}

	currentValue := goal.CurrentValue
	if req.CurrentValue != nil {
		currentValue = *req.CurrentValue
	}

	progress := goal.Progress
	if req.Progress != nil {
		progress = *req.Progress
	}

	// Auto-calculate progress if target_value > 0 and current_value provided
	if req.CurrentValue != nil && goal.TargetValue > 0 {
		calculated := int(currentValue / goal.TargetValue * 100)
		if calculated > 100 {
			calculated = 100
		}
		progress = calculated
	}

	status := goal.Status
	if progress == 100 {
		status = "completed"
	} else if progress > 0 && status == "not_started" {
		status = "in_progress"
	}

	return s.repo.UpdateGoalProgress(ctx, id, currentValue, progress, status)
}

func (s *Service) DeleteGoal(ctx context.Context, id int64) error {
	return s.repo.DeleteGoal(ctx, id)
}

// ==================== Analytics ====================

func (s *Service) GetKPIs(ctx context.Context) ([]*performance.KPI, error) {
	result, err := s.repo.GetKPIs(ctx)
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = []*performance.KPI{}
	}
	return result, nil
}

func (s *Service) GetAllRatings(ctx context.Context) ([]*performance.EmployeeRating, error) {
	result, err := s.repo.GetAllRatings(ctx)
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = []*performance.EmployeeRating{}
	}
	return result, nil
}

func (s *Service) GetEmployeeRating(ctx context.Context, employeeID int64) (*performance.EmployeeRating, error) {
	rating, err := s.repo.GetEmployeeRating(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if rating.Details == nil {
		rating.Details = []*performance.EmployeeRatingDetail{}
	}
	return rating, nil
}

func (s *Service) GetPerformanceDashboard(ctx context.Context) (*performance.PerformanceDashboard, error) {
	return s.repo.GetPerformanceDashboard(ctx)
}
