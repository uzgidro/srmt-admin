package fileupload

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"srmt-admin/internal/lib/logger/sl"
	filemodel "srmt-admin/internal/lib/model/file"
	"time"
)

// FileUploader defines interface for uploading files to storage (MinIO)
type FileUploader interface {
	UploadFile(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) error
	DeleteFile(ctx context.Context, objectName string) error
}

// FileMetaSaver defines interface for saving file metadata to database
type FileMetaSaver interface {
	AddFile(ctx context.Context, fileData filemodel.Model) (int64, error)
	DeleteFile(ctx context.Context, id int64) error
}

// UploadedFileInfo contains information about a successfully uploaded file
type UploadedFileInfo struct {
	ID        int64  `json:"id"`
	FileName  string `json:"file_name"`
	ObjectKey string `json:"object_key"`
	SizeBytes int64  `json:"size_bytes"`
	MimeType  string `json:"mime_type"`
}

// UploadResult contains the result of file upload operation
type UploadResult struct {
	UploadedFiles []UploadedFileInfo
	FileIDs       []int64
}

const (
	MaxUploadSize       = 50 * 1024 * 1024 // 50 MB
	FormFieldFiles      = "files"
	FormFieldCategoryID = "category_id"
)

// ProcessFormFiles handles file uploads from multipart form data
// Returns file IDs of uploaded files
// Implements compensation (cleanup) on failure
func ProcessFormFiles(
	ctx context.Context,
	r *http.Request,
	log *slog.Logger,
	uploader FileUploader,
	saver FileMetaSaver,
	categoryName string,
	uploadDate time.Time,
) (*UploadResult, error) {
	const op = "fileupload.ProcessFormFiles"

	// Parse multipart form
	if err := r.ParseMultipartForm(MaxUploadSize); err != nil {
		return nil, fmt.Errorf("%s: failed to parse multipart form: %w", op, err)
	}

	// Get files from form
	files := r.MultipartForm.File[FormFieldFiles]
	if len(files) == 0 {
		// No files to upload, return empty result
		return &UploadResult{
			UploadedFiles: []UploadedFileInfo{},
			FileIDs:       []int64{},
		}, nil
	}

	// Track uploaded files for compensation
	uploadedFiles := []UploadedFileInfo{}
	uploadedFileIDs := []int64{}
	uploadedObjectKeys := []string{}

	// Upload each file
	for _, fileHeader := range files {
		fileInfo, err := uploadSingleFile(
			ctx,
			log,
			uploader,
			saver,
			fileHeader,
			categoryName,
			uploadDate,
		)
		if err != nil {
			// Compensation: cleanup all previously uploaded files
			log.Error("file upload failed, starting compensation", sl.Err(err))
			compensateUploads(ctx, log, uploader, saver, uploadedFileIDs, uploadedObjectKeys)
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		uploadedFiles = append(uploadedFiles, *fileInfo)
		uploadedFileIDs = append(uploadedFileIDs, fileInfo.ID)
		uploadedObjectKeys = append(uploadedObjectKeys, fileInfo.ObjectKey)

		log.Info("file uploaded successfully",
			slog.Int64("file_id", fileInfo.ID),
			slog.String("file_name", fileInfo.FileName),
		)
	}

	return &UploadResult{
		UploadedFiles: uploadedFiles,
		FileIDs:       uploadedFileIDs,
	}, nil
}

// uploadSingleFile handles uploading a single file
func uploadSingleFile(
	ctx context.Context,
	log *slog.Logger,
	uploader FileUploader,
	saver FileMetaSaver,
	fileHeader *multipart.FileHeader,
	categoryName string,
	uploadDate time.Time,
) (*UploadedFileInfo, error) {
	const op = "fileupload.uploadSingleFile"

	// Open file
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("%s: failed to open file: %w", op, err)
	}
	defer file.Close()

	// Generate object key for MinIO
	// Format: <category>/<date>/<uuid>.<ext>
	datePrefix := uploadDate.Format("2006/01/02")
	objectKey := fmt.Sprintf("%s/%s/%s%s",
		categoryName,
		datePrefix,
		uuid.New().String(),
		filepath.Ext(fileHeader.Filename),
	)

	// Upload to MinIO
	err = uploader.UploadFile(
		ctx,
		objectKey,
		file,
		fileHeader.Size,
		fileHeader.Header.Get("Content-Type"),
	)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to upload to storage: %w", op, err)
	}

	// Save metadata to database
	fileData := filemodel.Model{
		FileName:   fileHeader.Filename,
		ObjectKey:  objectKey,
		CategoryID: 0, // Will be set by caller if needed
		MimeType:   fileHeader.Header.Get("Content-Type"),
		SizeBytes:  fileHeader.Size,
		CreatedAt:  uploadDate,
	}

	fileID, err := saver.AddFile(ctx, fileData)
	if err != nil {
		// Cleanup: delete file from MinIO
		if delErr := uploader.DeleteFile(ctx, objectKey); delErr != nil {
			log.Error("failed to delete orphaned file from storage",
				sl.Err(delErr),
				slog.String("object_key", objectKey),
			)
		}
		return nil, fmt.Errorf("%s: failed to save file metadata: %w", op, err)
	}

	return &UploadedFileInfo{
		ID:        fileID,
		FileName:  fileHeader.Filename,
		ObjectKey: objectKey,
		SizeBytes: fileHeader.Size,
		MimeType:  fileHeader.Header.Get("Content-Type"),
	}, nil
}

// compensateUploads rolls back file uploads by deleting them from both MinIO and database
func compensateUploads(
	ctx context.Context,
	log *slog.Logger,
	uploader FileUploader,
	saver FileMetaSaver,
	fileIDs []int64,
	objectKeys []string,
) {
	const op = "fileupload.compensateUploads"

	log.Warn("starting upload compensation",
		slog.Int("files_to_delete", len(fileIDs)),
	)

	// Delete from MinIO
	for i, objectKey := range objectKeys {
		if err := uploader.DeleteFile(ctx, objectKey); err != nil {
			log.Error("compensation: failed to delete file from storage",
				sl.Err(err),
				slog.String("object_key", objectKey),
				slog.Int("index", i),
			)
		} else {
			log.Info("compensation: deleted file from storage",
				slog.String("object_key", objectKey),
			)
		}
	}

	// Delete from database
	for i, fileID := range fileIDs {
		if err := saver.DeleteFile(ctx, fileID); err != nil {
			log.Error("compensation: failed to delete file metadata from database",
				sl.Err(err),
				slog.Int64("file_id", fileID),
				slog.Int("index", i),
			)
		} else {
			log.Info("compensation: deleted file metadata from database",
				slog.Int64("file_id", fileID),
			)
		}
	}

	log.Info("upload compensation completed", slog.Int("files_processed", len(fileIDs)))
}

// CompensateEntityUpload should be called if entity creation fails after files were uploaded
// This is a public function for handlers to use
func CompensateEntityUpload(
	ctx context.Context,
	log *slog.Logger,
	uploader FileUploader,
	saver FileMetaSaver,
	uploadResult *UploadResult,
) {
	if uploadResult == nil || len(uploadResult.UploadedFiles) == 0 {
		return
	}

	objectKeys := make([]string, len(uploadResult.UploadedFiles))
	for i, f := range uploadResult.UploadedFiles {
		objectKeys[i] = f.ObjectKey
	}

	compensateUploads(ctx, log, uploader, saver, uploadResult.FileIDs, objectKeys)
}
