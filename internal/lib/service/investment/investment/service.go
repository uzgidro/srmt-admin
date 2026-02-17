package investment

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	filemodel "srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/model/investment"
	investment_status "srmt-admin/internal/lib/model/investment-status"
	investment_type "srmt-admin/internal/lib/model/investment-type"
	"srmt-admin/internal/lib/service/fileupload"
)

type RepoInterface interface {
	// Investment CRUD
	AddInvestment(ctx context.Context, req dto.AddInvestmentRequest, createdByID int64) (int64, error)
	EditInvestment(ctx context.Context, id int64, req dto.EditInvestmentRequest) error
	DeleteInvestment(ctx context.Context, id int64) error
	GetAllInvestments(ctx context.Context, filters dto.GetAllInvestmentsFilters) ([]*investment.ResponseModel, error)
	GetInvestmentByID(ctx context.Context, id int64) (*investment.ResponseModel, error)

	// Investment types
	AddInvestmentType(ctx context.Context, req dto.AddInvestmentTypeRequest) (int, error)
	EditInvestmentType(ctx context.Context, id int, req dto.EditInvestmentTypeRequest) error
	DeleteInvestmentType(ctx context.Context, id int) error
	GetAllInvestmentTypes(ctx context.Context) ([]investment_type.Model, error)

	// Investment statuses
	AddInvestmentStatus(ctx context.Context, req dto.AddInvestmentStatusRequest) (int, error)
	EditInvestmentStatus(ctx context.Context, id int, req dto.EditInvestmentStatusRequest) error
	DeleteInvestmentStatus(ctx context.Context, id int) error
	GetAllInvestmentStatuses(ctx context.Context) ([]investment_status.Model, error)
	GetInvestmentStatusesByType(ctx context.Context, typeID int) ([]investment_status.Model, error)

	// File linking
	LinkInvestmentFiles(ctx context.Context, investmentID int64, fileIDs []int64) error
	UnlinkInvestmentFiles(ctx context.Context, investmentID int64) error

	// File metadata
	AddFile(ctx context.Context, fileData filemodel.Model) (int64, error)
	DeleteFile(ctx context.Context, id int64) error
	GetCategoryByName(ctx context.Context, categoryName string) (fileupload.CategoryModel, error)
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) AddInvestment(ctx context.Context, req dto.AddInvestmentRequest, createdByID int64) (int64, error) {
	return s.repo.AddInvestment(ctx, req, createdByID)
}

func (s *Service) EditInvestment(ctx context.Context, id int64, req dto.EditInvestmentRequest) error {
	return s.repo.EditInvestment(ctx, id, req)
}

func (s *Service) DeleteInvestment(ctx context.Context, id int64) error {
	return s.repo.DeleteInvestment(ctx, id)
}

func (s *Service) GetAllInvestments(ctx context.Context, filters dto.GetAllInvestmentsFilters) ([]*investment.ResponseModel, error) {
	return s.repo.GetAllInvestments(ctx, filters)
}

func (s *Service) GetInvestmentByID(ctx context.Context, id int64) (*investment.ResponseModel, error) {
	return s.repo.GetInvestmentByID(ctx, id)
}

func (s *Service) AddInvestmentType(ctx context.Context, req dto.AddInvestmentTypeRequest) (int, error) {
	return s.repo.AddInvestmentType(ctx, req)
}

func (s *Service) EditInvestmentType(ctx context.Context, id int, req dto.EditInvestmentTypeRequest) error {
	return s.repo.EditInvestmentType(ctx, id, req)
}

func (s *Service) DeleteInvestmentType(ctx context.Context, id int) error {
	return s.repo.DeleteInvestmentType(ctx, id)
}

func (s *Service) GetAllInvestmentTypes(ctx context.Context) ([]investment_type.Model, error) {
	return s.repo.GetAllInvestmentTypes(ctx)
}

func (s *Service) AddInvestmentStatus(ctx context.Context, req dto.AddInvestmentStatusRequest) (int, error) {
	return s.repo.AddInvestmentStatus(ctx, req)
}

func (s *Service) EditInvestmentStatus(ctx context.Context, id int, req dto.EditInvestmentStatusRequest) error {
	return s.repo.EditInvestmentStatus(ctx, id, req)
}

func (s *Service) DeleteInvestmentStatus(ctx context.Context, id int) error {
	return s.repo.DeleteInvestmentStatus(ctx, id)
}

func (s *Service) GetAllInvestmentStatuses(ctx context.Context) ([]investment_status.Model, error) {
	return s.repo.GetAllInvestmentStatuses(ctx)
}

func (s *Service) GetInvestmentStatusesByType(ctx context.Context, typeID int) ([]investment_status.Model, error) {
	return s.repo.GetInvestmentStatusesByType(ctx, typeID)
}

func (s *Service) LinkInvestmentFiles(ctx context.Context, investmentID int64, fileIDs []int64) error {
	return s.repo.LinkInvestmentFiles(ctx, investmentID, fileIDs)
}

func (s *Service) UnlinkInvestmentFiles(ctx context.Context, investmentID int64) error {
	return s.repo.UnlinkInvestmentFiles(ctx, investmentID)
}

func (s *Service) AddFile(ctx context.Context, fileData filemodel.Model) (int64, error) {
	return s.repo.AddFile(ctx, fileData)
}

func (s *Service) DeleteFile(ctx context.Context, id int64) error {
	return s.repo.DeleteFile(ctx, id)
}

func (s *Service) GetCategoryByName(ctx context.Context, categoryName string) (fileupload.CategoryModel, error) {
	return s.repo.GetCategoryByName(ctx, categoryName)
}
