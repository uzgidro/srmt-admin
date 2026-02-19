package incident

import (
	"context"
	"fmt"
	"log/slog"
	"mime/multipart"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	filemodel "srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/model/incident"
	"srmt-admin/internal/lib/service/fileupload"
	"time"
)

type RepoInterface interface {
	AddIncident(ctx context.Context, orgID *int64, incidentTime time.Time, description string, createdByID int64) (int64, error)
	EditIncident(ctx context.Context, id int64, req dto.EditIncidentRequest) error
	DeleteIncident(ctx context.Context, id int64) error
	GetIncidents(ctx context.Context, day time.Time) ([]*incident.ResponseModel, error)
	LinkIncidentFiles(ctx context.Context, incidentID int64, fileIDs []int64) error
	UnlinkIncidentFiles(ctx context.Context, incidentID int64) error
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

func (s *Service) AddIncident(ctx context.Context, req dto.AddIncidentRequest, files []*multipart.FileHeader) (id int64, uploadedFiles []fileupload.UploadedFileInfo, err error) {
	const op = "service.incident.AddIncident"
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
			"incidents",
			req.IncidentTime,
		)
		if err != nil {
			return 0, nil, fmt.Errorf("%s: failed to upload files: %w", op, err)
		}

		// Append uploaded file IDs to request
		req.FileIDs = append(req.FileIDs, uploadResult.FileIDs...)
		uploadedFiles = uploadResult.UploadedFiles
	}

	// Defer compensation
	defer func() {
		if err != nil && uploadResult != nil {
			log.Warn("incident creation failed, compensating uploaded files")
			fileupload.CompensateEntityUpload(ctx, log, s.uploader, s.repo, uploadResult)
		}
	}()

	id, err = s.repo.AddIncident(ctx, req.OrganizationID, req.IncidentTime, req.Description, req.CreatedByUserID)
	if err != nil {
		return 0, nil, fmt.Errorf("%s: failed to add incident: %w", op, err)
	}

	// Link files
	if len(req.FileIDs) > 0 {
		if linkErr := s.repo.LinkIncidentFiles(ctx, id, req.FileIDs); linkErr != nil {
			log.Error("failed to link files", sl.Err(linkErr))
		}
	}

	return id, uploadedFiles, nil
}

func (s *Service) EditIncident(ctx context.Context, id int64, req dto.EditIncidentRequest, files []*multipart.FileHeader) (uploadedFiles []fileupload.UploadedFileInfo, err error) {
	const op = "service.incident.EditIncident"
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
			"incidents",
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
			log.Warn("incident update failed, compensating uploaded files")
			fileupload.CompensateEntityUpload(ctx, log, s.uploader, s.repo, uploadResult)
		}
	}()

	if err := s.repo.EditIncident(ctx, id, req); err != nil {
		return nil, fmt.Errorf("%s: failed to edit incident: %w", op, err)
	}

	// Update file links if FileIDs is provided (non-nil)
	// This covers cases:
	// 1. Files uploaded (req.FileIDs became non-nil above)
	// 2. Client sent "file_ids": [] (to clear files)
	// 3. Client sent "file_ids": [1, 2] (to update)
	if req.FileIDs != nil {
		if err := s.repo.UnlinkIncidentFiles(ctx, id); err != nil {
			log.Error("failed to unlink old files", sl.Err(err))
			// Continue to link new ones? Or return error?
			// Handler just logged error.
		}

		if len(req.FileIDs) > 0 {
			if err := s.repo.LinkIncidentFiles(ctx, id, req.FileIDs); err != nil {
				log.Error("failed to link files", sl.Err(err))
			}
		}
	}

	return uploadedFiles, nil
}

func (s *Service) DeleteIncident(ctx context.Context, id int64) error {
	return s.repo.DeleteIncident(ctx, id)
}

func (s *Service) GetIncidents(ctx context.Context, day time.Time) ([]*incident.ResponseModel, error) {
	return s.repo.GetIncidents(ctx, day)
}

func (s *Service) LinkIncidentFiles(ctx context.Context, incidentID int64, fileIDs []int64) error {
	return s.repo.LinkIncidentFiles(ctx, incidentID, fileIDs)
}

func (s *Service) UnlinkIncidentFiles(ctx context.Context, incidentID int64) error {
	return s.repo.UnlinkIncidentFiles(ctx, incidentID)
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
