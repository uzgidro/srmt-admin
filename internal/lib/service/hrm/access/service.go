package access

import (
	"context"
	"errors"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/hrm/access"
	"srmt-admin/internal/storage"
)

type RepoInterface interface {
	// Cards
	CreateAccessCard(ctx context.Context, req dto.CreateAccessCardRequest) (int64, error)
	GetAccessCardByID(ctx context.Context, id int64) (*access.AccessCard, error)
	GetAllAccessCards(ctx context.Context, filters dto.AccessCardFilters) ([]*access.AccessCard, error)
	UpdateAccessCard(ctx context.Context, id int64, req dto.UpdateAccessCardRequest) error
	UpdateAccessCardStatus(ctx context.Context, id int64, status string) error

	// Zones
	CreateAccessZone(ctx context.Context, req dto.CreateAccessZoneRequest) (int64, error)
	GetAccessZoneByID(ctx context.Context, id int64) (*access.AccessZone, error)
	GetAllAccessZones(ctx context.Context) ([]*access.AccessZone, error)
	UpdateAccessZone(ctx context.Context, id int64, req dto.UpdateAccessZoneRequest) error

	// Logs
	GetAccessLogs(ctx context.Context, filters dto.AccessLogFilters) ([]*access.AccessLog, error)

	// Requests
	CreateAccessRequest(ctx context.Context, employeeID int64, req dto.CreateAccessRequestReq) (int64, error)
	GetAllAccessRequests(ctx context.Context, employeeID *int64) ([]*access.AccessRequest, error)
	GetAccessRequestByID(ctx context.Context, id int64) (*access.AccessRequest, error)
	UpdateAccessRequestStatus(ctx context.Context, id int64, status string, approvedBy *int64, rejectionReason *string) error
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

// ==================== Cards ====================

func (s *Service) CreateCard(ctx context.Context, req dto.CreateAccessCardRequest) (int64, error) {
	return s.repo.CreateAccessCard(ctx, req)
}

func (s *Service) GetAllCards(ctx context.Context, filters dto.AccessCardFilters) ([]*access.AccessCard, error) {
	cards, err := s.repo.GetAllAccessCards(ctx, filters)
	if err != nil {
		return nil, err
	}
	if cards == nil {
		cards = []*access.AccessCard{}
	}
	return cards, nil
}

func (s *Service) UpdateCard(ctx context.Context, id int64, req dto.UpdateAccessCardRequest) error {
	return s.repo.UpdateAccessCard(ctx, id, req)
}

func (s *Service) BlockCard(ctx context.Context, id int64) error {
	card, err := s.repo.GetAccessCardByID(ctx, id)
	if err != nil {
		return err
	}
	if card.Status != "active" {
		return storage.ErrInvalidStatus
	}
	return s.repo.UpdateAccessCardStatus(ctx, id, "blocked")
}

func (s *Service) UnblockCard(ctx context.Context, id int64) error {
	card, err := s.repo.GetAccessCardByID(ctx, id)
	if err != nil {
		return err
	}
	if card.Status != "blocked" {
		return storage.ErrInvalidStatus
	}
	return s.repo.UpdateAccessCardStatus(ctx, id, "active")
}

// ==================== Zones ====================

func (s *Service) CreateZone(ctx context.Context, req dto.CreateAccessZoneRequest) (int64, error) {
	return s.repo.CreateAccessZone(ctx, req)
}

func (s *Service) GetAllZones(ctx context.Context) ([]*access.AccessZone, error) {
	zones, err := s.repo.GetAllAccessZones(ctx)
	if err != nil {
		return nil, err
	}
	if zones == nil {
		zones = []*access.AccessZone{}
	}
	return zones, nil
}

func (s *Service) UpdateZone(ctx context.Context, id int64, req dto.UpdateAccessZoneRequest) error {
	return s.repo.UpdateAccessZone(ctx, id, req)
}

// ==================== Logs ====================

func (s *Service) GetLogs(ctx context.Context, filters dto.AccessLogFilters) ([]*access.AccessLog, error) {
	logs, err := s.repo.GetAccessLogs(ctx, filters)
	if err != nil {
		return nil, err
	}
	if logs == nil {
		logs = []*access.AccessLog{}
	}
	return logs, nil
}

// ==================== Requests ====================

func (s *Service) CreateRequest(ctx context.Context, employeeID int64, req dto.CreateAccessRequestReq) (int64, error) {
	return s.repo.CreateAccessRequest(ctx, employeeID, req)
}

func (s *Service) GetAllRequests(ctx context.Context, employeeID *int64) ([]*access.AccessRequest, error) {
	requests, err := s.repo.GetAllAccessRequests(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if requests == nil {
		requests = []*access.AccessRequest{}
	}
	return requests, nil
}

func (s *Service) ApproveRequest(ctx context.Context, id int64, approvedBy int64) error {
	req, err := s.repo.GetAccessRequestByID(ctx, id)
	if err != nil {
		return err
	}
	if req.Status != "pending" {
		return errors.New("request is not in pending status")
	}
	return s.repo.UpdateAccessRequestStatus(ctx, id, "approved", &approvedBy, nil)
}

func (s *Service) RejectRequest(ctx context.Context, id int64, reason string) error {
	req, err := s.repo.GetAccessRequestByID(ctx, id)
	if err != nil {
		return err
	}
	if req.Status != "pending" {
		return errors.New("request is not in pending status")
	}
	return s.repo.UpdateAccessRequestStatus(ctx, id, "rejected", nil, &reason)
}
