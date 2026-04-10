package discharge

import (
	"context"
	"errors"
	"fmt"
	"srmt-admin/internal/storage"
	"time"
)

// Repository defines data-access methods for ongoing discharge checks.
type Repository interface {
	CheckOngoingDischarge(ctx context.Context, orgID int64) (id int64, exists bool, err error)
	CloseDischarge(ctx context.Context, id int64, endTime time.Time) error
}

// Service handles discharge business logic.
type Service struct {
	repo Repository
}

// NewService creates a new discharge Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// EnsureNoOngoingDischarge checks if an ongoing idle discharge exists for the organization.
// If force is true and one exists, it closes the existing discharge with end_time = newStartTime.
// If force is false and one exists, it returns ErrOngoingDischargeExists.
func (s *Service) EnsureNoOngoingDischarge(ctx context.Context, orgID int64, force bool, newStartTime time.Time) error {
	const op = "service.discharge.EnsureNoOngoingDischarge"

	id, exists, err := s.repo.CheckOngoingDischarge(ctx, orgID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if !exists {
		return nil
	}

	if !force {
		return storage.ErrOngoingDischargeExists
	}

	if err := s.repo.CloseDischarge(ctx, id, newStartTime); err != nil {
		if errors.Is(err, storage.ErrCheckConstraintViolation) {
			return storage.ErrDischargeEndBeforeStart
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
