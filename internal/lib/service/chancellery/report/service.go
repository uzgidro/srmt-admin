package report

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	filemodel "srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/model/report"
	report_type "srmt-admin/internal/lib/model/report-type"
	"srmt-admin/internal/lib/service/fileupload"
)

type RepoInterface interface {
	// Report CRUD
	AddReport(ctx context.Context, req dto.AddReportRequest, createdByID int64) (int64, error)
	EditReport(ctx context.Context, id int64, req dto.EditReportRequest, updatedByID int64) error
	DeleteReport(ctx context.Context, id int64) error
	GetAllReports(ctx context.Context, filters dto.GetAllReportsFilters) ([]*report.ResponseModel, error)
	GetReportByID(ctx context.Context, id int64) (*report.ResponseModel, error)

	// Status workflow
	GetReportStatusHistory(ctx context.Context, reportID int64) ([]report.StatusHistory, error)
	AddReportStatusHistoryComment(ctx context.Context, reportID int64, comment string) error

	// Reference data
	GetAllReportTypes(ctx context.Context) ([]report_type.Model, error)

	// File linking
	LinkReportFiles(ctx context.Context, reportID int64, fileIDs []int64) error
	UnlinkReportFiles(ctx context.Context, reportID int64) error

	// Document linking
	LinkReportDocuments(ctx context.Context, reportID int64, links []dto.LinkedDocumentRequest, userID int64) error
	UnlinkReportDocuments(ctx context.Context, reportID int64) error

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

func (s *Service) AddReport(ctx context.Context, req dto.AddReportRequest, createdByID int64) (int64, error) {
	return s.repo.AddReport(ctx, req, createdByID)
}

func (s *Service) EditReport(ctx context.Context, id int64, req dto.EditReportRequest, updatedByID int64) error {
	return s.repo.EditReport(ctx, id, req, updatedByID)
}

func (s *Service) DeleteReport(ctx context.Context, id int64) error {
	return s.repo.DeleteReport(ctx, id)
}

func (s *Service) GetAllReports(ctx context.Context, filters dto.GetAllReportsFilters) ([]*report.ResponseModel, error) {
	return s.repo.GetAllReports(ctx, filters)
}

func (s *Service) GetReportByID(ctx context.Context, id int64) (*report.ResponseModel, error) {
	return s.repo.GetReportByID(ctx, id)
}

func (s *Service) GetReportStatusHistory(ctx context.Context, reportID int64) ([]report.StatusHistory, error) {
	return s.repo.GetReportStatusHistory(ctx, reportID)
}

func (s *Service) AddReportStatusHistoryComment(ctx context.Context, reportID int64, comment string) error {
	return s.repo.AddReportStatusHistoryComment(ctx, reportID, comment)
}

func (s *Service) GetAllReportTypes(ctx context.Context) ([]report_type.Model, error) {
	return s.repo.GetAllReportTypes(ctx)
}

func (s *Service) LinkReportFiles(ctx context.Context, reportID int64, fileIDs []int64) error {
	return s.repo.LinkReportFiles(ctx, reportID, fileIDs)
}

func (s *Service) UnlinkReportFiles(ctx context.Context, reportID int64) error {
	return s.repo.UnlinkReportFiles(ctx, reportID)
}

func (s *Service) LinkReportDocuments(ctx context.Context, reportID int64, links []dto.LinkedDocumentRequest, userID int64) error {
	return s.repo.LinkReportDocuments(ctx, reportID, links, userID)
}

func (s *Service) UnlinkReportDocuments(ctx context.Context, reportID int64) error {
	return s.repo.UnlinkReportDocuments(ctx, reportID)
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
