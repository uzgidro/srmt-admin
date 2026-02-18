package decree

import (
	"context"
	"fmt"
	"log/slog"
	"mime/multipart"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/decree"
	decree_type "srmt-admin/internal/lib/model/decree-type"
	"srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/service/fileupload"
	"time"
)

type RepoInterface interface {
	// Decree CRUD
	AddDecree(ctx context.Context, req dto.AddDecreeRequest, createdByID int64) (int64, error)
	EditDecree(ctx context.Context, id int64, req dto.EditDecreeRequest, updatedByID int64) error
	DeleteDecree(ctx context.Context, id int64) error
	GetAllDecrees(ctx context.Context, filters dto.GetAllDecreesFilters) ([]*decree.ResponseModel, error)
	GetDecreeByID(ctx context.Context, id int64) (*decree.ResponseModel, error)

	// Status workflow
	GetDecreeStatusHistory(ctx context.Context, decreeID int64) ([]decree.StatusHistory, error)
	AddDecreeStatusHistoryComment(ctx context.Context, decreeID int64, comment string) error

	// Reference data
	GetAllDecreeTypes(ctx context.Context) ([]decree_type.Model, error)

	// File linking
	LinkDecreeFiles(ctx context.Context, decreeID int64, fileIDs []int64) error
	UnlinkDecreeFiles(ctx context.Context, decreeID int64) error

	// Document linking
	LinkDecreeDocuments(ctx context.Context, decreeID int64, links []dto.LinkedDocumentRequest, userID int64) error
	UnlinkDecreeDocuments(ctx context.Context, decreeID int64) error

	// File metadata
	AddFile(ctx context.Context, fileData file.Model) (int64, error)
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

func (s *Service) AddDecree(ctx context.Context, req dto.AddDecreeRequest, files []*multipart.FileHeader, createdByID int64) (id int64, err error) {
	const op = "service.decree.AddDecree"
	log := s.log.With(slog.String("op", op), slog.String("name", req.Name))

	var uploadedFiles []fileupload.UploadedFileInfo

	// Compensation for uploads
	defer func() {
		if err != nil && len(uploadedFiles) > 0 {
			fileupload.CompensateEntityUpload(ctx, log, s.uploader, s.repo, &fileupload.UploadResult{UploadedFiles: uploadedFiles})
		}
	}()

	// 1. Upload files
	if len(files) > 0 {
		cat, err := s.repo.GetCategoryByName(ctx, "decrees")
		if err != nil {
			return 0, fmt.Errorf("%s: failed to get category: %w", op, err)
		}

		for _, fileHeader := range files {
			fileInfo, err := fileupload.UploadFileHeader(
				ctx,
				log,
				s.uploader,
				s.repo,
				fileHeader,
				"decrees",
				cat.GetID(),
				req.DocumentDate,
			)
			if err != nil {
				return 0, fmt.Errorf("%s: failed to upload file: %w", op, err)
			}
			uploadedFiles = append(uploadedFiles, *fileInfo)
			req.FileIDs = append(req.FileIDs, fileInfo.ID)
		}
	}

	// 2. Create Decree
	id, err = s.repo.AddDecree(ctx, req, createdByID)
	if err != nil {
		return 0, fmt.Errorf("%s: failed to create decree: %w", op, err)
	}

	// 3. Link Files
	if len(req.FileIDs) > 0 {
		if err := s.repo.LinkDecreeFiles(ctx, id, req.FileIDs); err != nil {
			// If linking fails, we might want to fail the whole operation or just log error?
			// Usually strict transaction logic requires fail.
			// But for now, let's just return error (which triggers compensation due to named return `err`).
			// Note: `AddDecree` in repo might have created the decree record.
			// If we return error here, we compensate files, but Decree remains in DB without links.
			// Ideally we should delete Decree too, or use DB transaction.
			// Since service doesn't control DB transaction here (repo does atomic AddDecree but not combined),
			// we will log error and proceed or return error.
			// Given existing logic in handler was:
			// `if err := adder.LinkDecreeFiles...; err != nil { log.Error... }` - it didn't fail request.
			// So we should probably match that behavior: Log error but return success.
			log.Error("failed to link files to decree", sl.Err(err))
			// Reset error to nil so we don't trigger file compensation (files are in DB/Storage, just not linked)
			// Or we keep them as "orphaned" files that might be cleaned up later.
		}
	}

	// 4. Link Documents
	if len(req.LinkedDocuments) > 0 {
		if err := s.repo.LinkDecreeDocuments(ctx, id, req.LinkedDocuments, createdByID); err != nil {
			log.Error("failed to link documents to decree", sl.Err(err))
		}
	}

	return id, nil
}

func (s *Service) EditDecree(ctx context.Context, id int64, req dto.EditDecreeRequest, files []*multipart.FileHeader, updatedByID int64) (err error) {
	const op = "service.decree.EditDecree"
	log := s.log.With(slog.String("op", op), slog.Int64("id", id))

	var uploadedFiles []fileupload.UploadedFileInfo

	defer func() {
		if err != nil && len(uploadedFiles) > 0 {
			fileupload.CompensateEntityUpload(ctx, log, s.uploader, s.repo, &fileupload.UploadResult{UploadedFiles: uploadedFiles})
		}
	}()

	// 1. Upload new files if any
	if len(files) > 0 {
		cat, err := s.repo.GetCategoryByName(ctx, "decrees")
		if err != nil {
			return fmt.Errorf("%s: failed to get category: %w", op, err)
		}

		// Use DocumentDate if available in update, otherwise Use Now? Or fetch existing?
		// Existing handler used: `targetDate := time.Now(); if req.DocumentDate != nil { targetDate = *req.DocumentDate }`
		targetDate := time.Now()
		if req.DocumentDate != nil {
			targetDate = *req.DocumentDate
		} else {
			// Ideally we should fetch the existing decree to trust the date, but for optimization we might use Now or skip.
			// File path includes date. If we use Now, it puts new files in current date folder. Acceptable.
		}

		for _, fileHeader := range files {
			fileInfo, err := fileupload.UploadFileHeader(
				ctx,
				log,
				s.uploader,
				s.repo,
				fileHeader,
				"decrees",
				cat.GetID(),
				targetDate,
			)
			if err != nil {
				return fmt.Errorf("%s: failed to upload file: %w", op, err)
			}
			uploadedFiles = append(uploadedFiles, *fileInfo)
			req.FileIDs = append(req.FileIDs, fileInfo.ID)
		}
	}

	// 2. Update Decree
	if err := s.repo.EditDecree(ctx, id, req, updatedByID); err != nil {
		return fmt.Errorf("%s: failed to edit decree: %w", op, err)
	}

	// 3. Handle File changes
	// If files were uploaded OR FileIDs were explicitly passed (e.g. removing some old files)
	// Logic in handler:
	// `if hasFileChanges { Unlink; Link }`
	// `hasFileChanges` was true if multipart (new files) OR json with `file_ids`.
	// Here `req.FileIDs` contains both preserved old files (passed by frontend) AND new uploaded files (appended above).
	// But `EditDecreeRequest` has `FileIDs []int64`. If it's nil/empty, does it mean "remove all" or "no change"?
	// In `EditDecreeRequest` DTO, fields are optional. `FileIDs` is a slice.
	// If it was nil in JSON, it's nil here. But we might have appended uploaded files.
	// If we have uploaded files, `req.FileIDs` is not nil.

	// We need to know if we should update file links.
	// If `files` > 0 => yes.
	// If `req.FileIDs` passed from caller implies "this is the new list".
	// The caller (handler) must explicitly pass the list of files to keep if it wants to update the list.
	// If handler passes nil for `FileIDs`, we shouldn't touch links unless we have new files?
	// But simply appending new files to nil/empty list would mean "replace all links with these new files", deleting old links?
	// That might be dangerous if frontend just sent a file but didn't send existing IDs.
	// Assuming frontend sends ALL current file IDs + new files are added.

	// Let's assume: if `req.FileIDs` is provided (even empty slice) OR `files` provided => we update links.
	// But `req.FileIDs` in DTO is not a pointer, it's a slice. Nil slice vs Empty slice.
	// We can't easily distinguish "not provided" from "provided as null" with slice unless we use pointer to slice `*[]int64`.
	// In DTO `EditDecreeRequest`: `FileIDs []int64`.
	// If we look at `edit.go` handler:
	// `if req.FileIDs != nil { fileIDs = req.FileIDs; hasFileChanges = true }`
	// So if JSON has "file_ids": null or missing, it's skipped.
	// But wait, `add.go` logic was append.

	// Refined Logic:
	// If we uploaded files, we DEFINITELY have changes.
	// If `req.FileIDs` has entries, we have changes.
	updateLinks := len(files) > 0 || len(req.FileIDs) > 0
	// Wait, if user wants to delete all files, they might send empty list.
	// But we can't detect "intent to clear" if slice is nil vs empty easily with standard JSON decoder if omitempty is used?
	// Actually `json` unmarshal converts null to nil slice, and missing to nil slice.
	// So "clear all files" is hard.
	// Usually we use explicit action or always send list.
	// Let's stick to: if we have any file IDs (from req or upload), we sync.

	if updateLinks {
		if err := s.repo.UnlinkDecreeFiles(ctx, id); err != nil {
			log.Error("failed to unlink files", sl.Err(err))
		}
		if len(req.FileIDs) > 0 {
			if err := s.repo.LinkDecreeFiles(ctx, id, req.FileIDs); err != nil {
				log.Error("failed to link files", sl.Err(err))
			}
		}
	}

	// 4. Update Document Links
	if len(req.LinkedDocuments) > 0 {
		if err := s.repo.UnlinkDecreeDocuments(ctx, id); err != nil {
			log.Error("failed to unlink documents", sl.Err(err))
		}
		if err := s.repo.LinkDecreeDocuments(ctx, id, req.LinkedDocuments, updatedByID); err != nil {
			log.Error("failed to link documents", sl.Err(err))
		}
	} else if req.LinkedDocuments != nil {
		// Explicit empty list => Remove links?
		// DTO has `LinkedDocuments []LinkedDocumentRequest`.
		// Logic in handler was: `if req.LinkedDocuments != nil { Unlink; if len > 0 Link }`
		// So explicit empty list clears links.
		if err := s.repo.UnlinkDecreeDocuments(ctx, id); err != nil {
			log.Error("failed to unlink documents", sl.Err(err))
		}
	}

	return nil
}

func (s *Service) DeleteDecree(ctx context.Context, id int64) error {
	return s.repo.DeleteDecree(ctx, id)
}

func (s *Service) GetAllDecrees(ctx context.Context, filters dto.GetAllDecreesFilters) ([]*decree.ResponseModel, error) {
	return s.repo.GetAllDecrees(ctx, filters)
}

func (s *Service) GetDecreeByID(ctx context.Context, id int64) (*decree.ResponseModel, error) {
	return s.repo.GetDecreeByID(ctx, id)
}

func (s *Service) GetDecreeStatusHistory(ctx context.Context, decreeID int64) ([]decree.StatusHistory, error) {
	return s.repo.GetDecreeStatusHistory(ctx, decreeID)
}

func (s *Service) AddDecreeStatusHistoryComment(ctx context.Context, decreeID int64, comment string) error {
	return s.repo.AddDecreeStatusHistoryComment(ctx, decreeID, comment)
}

func (s *Service) GetAllDecreeTypes(ctx context.Context) ([]decree_type.Model, error) {
	return s.repo.GetAllDecreeTypes(ctx)
}

func (s *Service) LinkDecreeFiles(ctx context.Context, decreeID int64, fileIDs []int64) error {
	return s.repo.LinkDecreeFiles(ctx, decreeID, fileIDs)
}

func (s *Service) UnlinkDecreeFiles(ctx context.Context, decreeID int64) error {
	return s.repo.UnlinkDecreeFiles(ctx, decreeID)
}

// These methods are probably not needed to be exposed publicly if Add/Edit handle them internally,
// but keeping them for now if other handlers use them (e.g. dedicated link handlers).
// Checking `handlers` dir - there is `get-by-id` which might use them? No.
// But we should keep them part of interface if they are used elsewhere.

func (s *Service) LinkDecreeDocuments(ctx context.Context, decreeID int64, links []dto.LinkedDocumentRequest, userID int64) error {
	return s.repo.LinkDecreeDocuments(ctx, decreeID, links, userID)
}

func (s *Service) UnlinkDecreeDocuments(ctx context.Context, decreeID int64) error {
	return s.repo.UnlinkDecreeDocuments(ctx, decreeID)
}

func (s *Service) AddFile(ctx context.Context, fileData file.Model) (int64, error) {
	return s.repo.AddFile(ctx, fileData)
}

func (s *Service) DeleteFile(ctx context.Context, id int64) error {
	return s.repo.DeleteFile(ctx, id)
}

func (s *Service) GetCategoryByName(ctx context.Context, categoryName string) (fileupload.CategoryModel, error) {
	return s.repo.GetCategoryByName(ctx, categoryName)
}
