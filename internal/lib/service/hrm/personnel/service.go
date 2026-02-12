package personnel

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/hrm/personnel"
)

type RepoInterface interface {
	CreatePersonnelRecord(ctx context.Context, req dto.CreatePersonnelRecordRequest) (int64, error)
	GetPersonnelRecordByID(ctx context.Context, id int64) (*personnel.Record, error)
	GetPersonnelRecordByEmployeeID(ctx context.Context, employeeID int64) (*personnel.Record, error)
	GetAllPersonnelRecords(ctx context.Context, filters dto.PersonnelRecordFilters) ([]*personnel.Record, error)
	UpdatePersonnelRecord(ctx context.Context, id int64, req dto.EditPersonnelRecordRequest) error
	DeletePersonnelRecord(ctx context.Context, id int64) error
	GetPersonnelDocuments(ctx context.Context, recordID int64) ([]*personnel.Document, error)
	GetPersonnelTransfers(ctx context.Context, recordID int64) ([]*personnel.Transfer, error)
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) Create(ctx context.Context, req dto.CreatePersonnelRecordRequest) (int64, error) {
	return s.repo.CreatePersonnelRecord(ctx, req)
}

func (s *Service) GetByID(ctx context.Context, id int64) (*personnel.Record, error) {
	return s.repo.GetPersonnelRecordByID(ctx, id)
}

func (s *Service) GetByEmployeeID(ctx context.Context, employeeID int64) (*personnel.Record, error) {
	return s.repo.GetPersonnelRecordByEmployeeID(ctx, employeeID)
}

func (s *Service) GetAll(ctx context.Context, filters dto.PersonnelRecordFilters) ([]*personnel.Record, error) {
	return s.repo.GetAllPersonnelRecords(ctx, filters)
}

func (s *Service) Update(ctx context.Context, id int64, req dto.EditPersonnelRecordRequest) error {
	return s.repo.UpdatePersonnelRecord(ctx, id, req)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.DeletePersonnelRecord(ctx, id)
}

func (s *Service) GetDocuments(ctx context.Context, recordID int64) ([]*personnel.Document, error) {
	return s.repo.GetPersonnelDocuments(ctx, recordID)
}

func (s *Service) GetTransfers(ctx context.Context, recordID int64) ([]*personnel.Transfer, error) {
	return s.repo.GetPersonnelTransfers(ctx, recordID)
}
