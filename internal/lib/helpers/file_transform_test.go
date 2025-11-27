package helpers

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"srmt-admin/internal/lib/model/file"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// mockMinioRepo is a mock implementation of MinioURLGenerator for testing.
type mockMinioRepo struct {
	urls map[string]string // objectKey -> presignedURL
	err  error             // error to return
}

func (m *mockMinioRepo) GetPresignedURL(ctx context.Context, objectName string, expires time.Duration) (*url.URL, error) {
	if m.err != nil {
		return nil, m.err
	}
	urlStr, ok := m.urls[objectName]
	if !ok {
		return nil, fmt.Errorf("object not found")
	}
	return url.Parse(urlStr)
}

func TestTransformFilesWithURLs_Success(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	mockMinio := &mockMinioRepo{
		urls: map[string]string{
			"obj1": "https://minio.example.com/bucket/obj1?signature=abc123",
			"obj2": "https://minio.example.com/bucket/obj2?signature=def456",
		},
	}

	files := []file.Model{
		{
			ID:         1,
			FileName:   "test1.pdf",
			ObjectKey:  "obj1",
			CategoryID: 10,
			MimeType:   "application/pdf",
			SizeBytes:  1024,
			CreatedAt:  time.Now(),
		},
		{
			ID:         2,
			FileName:   "test2.jpg",
			ObjectKey:  "obj2",
			CategoryID: 20,
			MimeType:   "image/jpeg",
			SizeBytes:  2048,
			CreatedAt:  time.Now(),
		},
	}

	result := TransformFilesWithURLs(context.Background(), files, mockMinio, log)

	assert.Len(t, result, 2)
	assert.Equal(t, int64(1), result[0].ID)
	assert.Equal(t, "test1.pdf", result[0].FileName)
	assert.Equal(t, "https://minio.example.com/bucket/obj1?signature=abc123", result[0].URL)
	assert.Equal(t, int64(2), result[1].ID)
	assert.Equal(t, "test2.jpg", result[1].FileName)
	assert.Equal(t, "https://minio.example.com/bucket/obj2?signature=def456", result[1].URL)
}

func TestTransformFilesWithURLs_EmptyList(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	mockMinio := &mockMinioRepo{urls: map[string]string{}}

	result := TransformFilesWithURLs(context.Background(), []file.Model{}, mockMinio, log)

	assert.Empty(t, result)
}

func TestTransformFilesWithURLs_PartialFailure(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	mockMinio := &mockMinioRepo{
		urls: map[string]string{
			"obj1": "https://minio.example.com/bucket/obj1?signature=abc123",
			// obj2 is missing - will fail
		},
	}

	files := []file.Model{
		{
			ID:         1,
			FileName:   "test1.pdf",
			ObjectKey:  "obj1",
			CategoryID: 10,
			MimeType:   "application/pdf",
			SizeBytes:  1024,
			CreatedAt:  time.Now(),
		},
		{
			ID:         2,
			FileName:   "test2.jpg",
			ObjectKey:  "obj2", // This will fail
			CategoryID: 20,
			MimeType:   "image/jpeg",
			SizeBytes:  2048,
			CreatedAt:  time.Now(),
		},
	}

	result := TransformFilesWithURLs(context.Background(), files, mockMinio, log)

	// Should only have 1 file (the successful one)
	assert.Len(t, result, 1)
	assert.Equal(t, int64(1), result[0].ID)
	assert.Equal(t, "test1.pdf", result[0].FileName)
}

func TestTransformFilesWithURLs_AllFail(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	mockMinio := &mockMinioRepo{
		err: fmt.Errorf("minio unavailable"),
	}

	files := []file.Model{
		{
			ID:         1,
			FileName:   "test1.pdf",
			ObjectKey:  "obj1",
			CategoryID: 10,
			MimeType:   "application/pdf",
			SizeBytes:  1024,
			CreatedAt:  time.Now(),
		},
	}

	result := TransformFilesWithURLs(context.Background(), files, mockMinio, log)

	// All files should be skipped
	assert.Empty(t, result)
}
