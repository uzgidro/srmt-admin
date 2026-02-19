package shutdown

import (
	"context"
	"fmt"
	"log/slog"
	"mime/multipart"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	filemodel "srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/model/shutdown"
	"srmt-admin/internal/lib/service/fileupload"
	"time"
)

type RepoInterface interface {
	AddShutdown(ctx context.Context, req dto.AddShutdownRequest) (int64, error)
	EditShutdown(ctx context.Context, id int64, req dto.EditShutdownRequest) error
	DeleteShutdown(ctx context.Context, id int64) error
	GetShutdowns(ctx context.Context, day time.Time) ([]*shutdown.ResponseModel, error)
	GetOrganizationTypesMap(ctx context.Context) (map[int64][]string, error)
	MarkShutdownAsViewed(ctx context.Context, id int64) error
	LinkShutdownFiles(ctx context.Context, shutdownID int64, fileIDs []int64) error
	UnlinkShutdownFiles(ctx context.Context, shutdownID int64) error
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

func (s *Service) AddShutdown(ctx context.Context, req dto.AddShutdownRequest, files []*multipart.FileHeader) (id int64, uploadedFiles []fileupload.UploadedFileInfo, err error) {
	const op = "service.shutdown.AddShutdown"
	log := s.log.With(slog.String("op", op))

	var uploadResult *fileupload.UploadResult

	// Process file uploads
	if len(files) > 0 {
		uploadResult, err = fileupload.ProcessFileHeaders(
			ctx,
			log,
			s.uploader,
			s.repo, // acts as saver
			s.repo, // acts as categoryGetter
			files,
			"shutdowns",
			req.StartTime,
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
			log.Warn("shutdown creation failed, compensating uploaded files")
			fileupload.CompensateEntityUpload(ctx, log, s.uploader, s.repo, uploadResult)
		}
	}()

	id, err = s.repo.AddShutdown(ctx, req)
	if err != nil {
		return 0, nil, fmt.Errorf("%s: failed to add shutdown: %w", op, err)
	}

	// Link files
	if len(req.FileIDs) > 0 {
		if linkErr := s.repo.LinkShutdownFiles(ctx, id, req.FileIDs); linkErr != nil {
			log.Error("failed to link files", sl.Err(linkErr))
		}
	}

	return id, uploadedFiles, nil
}

func (s *Service) EditShutdown(ctx context.Context, id int64, req dto.EditShutdownRequest, files []*multipart.FileHeader) (uploadedFiles []fileupload.UploadedFileInfo, err error) {
	const op = "service.shutdown.EditShutdown"
	log := s.log.With(slog.String("op", op), slog.Int64("id", id))

	var uploadResult *fileupload.UploadResult

	// Process file uploads
	if len(files) > 0 {
		uploadResult, err = fileupload.ProcessFileHeaders(
			ctx,
			log,
			s.uploader,
			s.repo, // acts as saver
			s.repo, // acts as categoryGetter
			files,
			"shutdowns",
			time.Now(),
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to upload files: %w", op, err)
		}

		uploadedFiles = uploadResult.UploadedFiles

		// If FileIDs was nil (not provided), initialize it to empty slice so we can append
		if req.FileIDs == nil {
			req.FileIDs = []int64{}
		}
		req.FileIDs = append(req.FileIDs, uploadResult.FileIDs...)
	}

	// Defer compensation
	defer func() {
		if err != nil && uploadResult != nil {
			log.Warn("shutdown update failed, compensating uploaded files")
			fileupload.CompensateEntityUpload(ctx, log, s.uploader, s.repo, uploadResult)
		}
	}()

	if err := s.repo.EditShutdown(ctx, id, req); err != nil {
		return nil, fmt.Errorf("%s: failed to edit shutdown: %w", op, err)
	}

	// Update file links if FileIDs is provided (non-nil)
	if req.FileIDs != nil {
		if err := s.repo.UnlinkShutdownFiles(ctx, id); err != nil {
			log.Error("failed to unlink old files", sl.Err(err))
		}

		if len(req.FileIDs) > 0 {
			if err := s.repo.LinkShutdownFiles(ctx, id, req.FileIDs); err != nil {
				log.Error("failed to link files", sl.Err(err))
			}
		}
	}

	return uploadedFiles, nil
}

func (s *Service) DeleteShutdown(ctx context.Context, id int64) error {
	return s.repo.DeleteShutdown(ctx, id)
}

func (s *Service) GetShutdowns(ctx context.Context, day time.Time) ([]*shutdown.ResponseModel, error) {
	return s.repo.GetShutdowns(ctx, day)
}

func (s *Service) GetOrganizationTypesMap(ctx context.Context) (map[int64][]string, error) {
	return s.repo.GetOrganizationTypesMap(ctx)
}

func (s *Service) MarkShutdownAsViewed(ctx context.Context, id int64) error {
	return s.repo.MarkShutdownAsViewed(ctx, id)
}

func (s *Service) LinkShutdownFiles(ctx context.Context, shutdownID int64, fileIDs []int64) error {
	return s.repo.LinkShutdownFiles(ctx, shutdownID, fileIDs)
}

func (s *Service) UnlinkShutdownFiles(ctx context.Context, shutdownID int64) error {
	return s.repo.UnlinkShutdownFiles(ctx, shutdownID)
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
