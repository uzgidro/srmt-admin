package visit

import (
	"context"
	"fmt"
	"log/slog"
	"mime/multipart"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	filemodel "srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/model/visit"
	"srmt-admin/internal/lib/service/fileupload"
	"time"
)

type RepoInterface interface {
	AddVisit(ctx context.Context, req dto.AddVisitRequest) (int64, error)
	EditVisit(ctx context.Context, id int64, req dto.EditVisitRequest) error
	DeleteVisit(ctx context.Context, id int64) error
	GetVisits(ctx context.Context, day time.Time) ([]*visit.ResponseModel, error)
	LinkVisitFiles(ctx context.Context, visitID int64, fileIDs []int64) error
	UnlinkVisitFiles(ctx context.Context, visitID int64) error
	AddFile(ctx context.Context, fileData filemodel.Model) (int64, error)
	DeleteFile(ctx context.Context, id int64) error
	GetCategoryByName(ctx context.Context, categoryName string) (fileupload.CategoryModel, error)
}

type Service struct {
	repo     RepoInterface
	uploader fileupload.FileUploader
	log      *slog.Logger
}

func NewService(repo RepoInterface, uploader fileupload.FileUploader, log *slog.Logger) *Service {
	return &Service{repo: repo, uploader: uploader, log: log}
}

func (s *Service) AddVisit(ctx context.Context, req dto.AddVisitRequest, files []*multipart.FileHeader) (id int64, uploadedFiles []fileupload.UploadedFileInfo, err error) {
	const op = "service.visit.AddVisit"
	log := s.log.With(slog.String("op", op))

	var uploadResult *fileupload.UploadResult

	// Process file uploads
	if len(files) > 0 {
		uploadResult, err = fileupload.ProcessFileHeaders(
			ctx,
			log,
			s.uploader,
			s.repo,
			s.repo,
			files,
			"visits",
			req.VisitDate,
		)
		if err != nil {
			return 0, nil, fmt.Errorf("%s: failed to upload files: %w", op, err)
		}

		uploadedFiles = uploadResult.UploadedFiles
		req.FileIDs = append(req.FileIDs, uploadResult.FileIDs...)
	}

	// Defer compensation
	defer func() {
		if err != nil && uploadResult != nil {
			log.Warn("visit creation failed, compensating uploaded files")
			fileupload.CompensateEntityUpload(ctx, log, s.uploader, s.repo, uploadResult)
		}
	}()

	id, err = s.repo.AddVisit(ctx, req)
	if err != nil {
		return 0, nil, fmt.Errorf("%s: failed to add visit: %w", op, err)
	}

	// Link files
	if len(req.FileIDs) > 0 {
		if linkErr := s.repo.LinkVisitFiles(ctx, id, req.FileIDs); linkErr != nil {
			log.Error("failed to link files", sl.Err(linkErr))
		}
	}

	return id, uploadedFiles, nil
}

func (s *Service) EditVisit(ctx context.Context, id int64, req dto.EditVisitRequest, files []*multipart.FileHeader) (uploadedFiles []fileupload.UploadedFileInfo, err error) {
	const op = "service.visit.EditVisit"
	log := s.log.With(slog.String("op", op), slog.Int64("id", id))

	var uploadResult *fileupload.UploadResult

	// Process file uploads
	if len(files) > 0 {
		uploadResult, err = fileupload.ProcessFileHeaders(
			ctx,
			log,
			s.uploader,
			s.repo,
			s.repo,
			files,
			"visits",
			time.Now(),
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to upload files: %w", op, err)
		}

		uploadedFiles = uploadResult.UploadedFiles

		if req.FileIDs == nil {
			req.FileIDs = []int64{}
		}
		req.FileIDs = append(req.FileIDs, uploadResult.FileIDs...)
	}

	// Defer compensation
	defer func() {
		if err != nil && uploadResult != nil {
			log.Warn("visit update failed, compensating uploaded files")
			fileupload.CompensateEntityUpload(ctx, log, s.uploader, s.repo, uploadResult)
		}
	}()

	if err := s.repo.EditVisit(ctx, id, req); err != nil {
		return nil, fmt.Errorf("%s: failed to edit visit: %w", op, err)
	}

	// Update file links if FileIDs is provided (non-nil)
	if req.FileIDs != nil {
		if err := s.repo.UnlinkVisitFiles(ctx, id); err != nil {
			log.Error("failed to unlink old files", sl.Err(err))
		}

		if len(req.FileIDs) > 0 {
			if err := s.repo.LinkVisitFiles(ctx, id, req.FileIDs); err != nil {
				log.Error("failed to link files", sl.Err(err))
			}
		}
	}

	return uploadedFiles, nil
}

func (s *Service) DeleteVisit(ctx context.Context, id int64) error {
	return s.repo.DeleteVisit(ctx, id)
}

func (s *Service) GetVisits(ctx context.Context, day time.Time) ([]*visit.ResponseModel, error) {
	return s.repo.GetVisits(ctx, day)
}

func (s *Service) LinkVisitFiles(ctx context.Context, visitID int64, fileIDs []int64) error {
	return s.repo.LinkVisitFiles(ctx, visitID, fileIDs)
}

func (s *Service) UnlinkVisitFiles(ctx context.Context, visitID int64) error {
	return s.repo.UnlinkVisitFiles(ctx, visitID)
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
