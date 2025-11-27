package helpers

import (
	"context"
	"log/slog"
	"net/url"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/file"
	"time"
)

// MinioURLGenerator defines the interface for generating presigned URLs from MinIO.
type MinioURLGenerator interface {
	GetPresignedURL(ctx context.Context, objectName string, expires time.Duration) (*url.URL, error)
}

// TransformFilesWithURLs converts file models to file responses with presigned URLs.
// If URL generation fails for a file, it logs the error and skips that file (graceful degradation).
func TransformFilesWithURLs(
	ctx context.Context,
	files []file.Model,
	minioRepo MinioURLGenerator,
	log *slog.Logger,
) []dto.FileResponse {
	if len(files) == 0 {
		return []dto.FileResponse{}
	}

	fileResponses := make([]dto.FileResponse, 0, len(files))

	for _, f := range files {
		presignedURL, err := minioRepo.GetPresignedURL(ctx, f.ObjectKey, time.Hour)
		if err != nil {
			log.Error("failed to generate presigned URL",
				slog.Int64("file_id", f.ID),
				slog.String("file_name", f.FileName),
				slog.String("object_key", f.ObjectKey),
				sl.Err(err))
			continue // Skip this file
		}

		fileResponses = append(fileResponses, dto.FileResponse{
			ID:         f.ID,
			FileName:   f.FileName,
			CategoryID: f.CategoryID,
			MimeType:   f.MimeType,
			SizeBytes:  f.SizeBytes,
			CreatedAt:  f.CreatedAt,
			URL:        presignedURL.String(),
		})
	}

	return fileResponses
}
