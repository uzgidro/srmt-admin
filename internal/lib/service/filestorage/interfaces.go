package filestorage

import (
	"context"
	"io"
	"net/url"
	"time"
)

// FileStorage abstracts object-storage operations (MinIO).
// Both *minio.Repo methods satisfy this interface.
type FileStorage interface {
	UploadFile(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) error
	DeleteFile(ctx context.Context, objectName string) error
	GetPresignedURL(ctx context.Context, objectName string, expires time.Duration) (*url.URL, error)
}
