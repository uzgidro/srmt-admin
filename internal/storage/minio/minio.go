package minio

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"srmt-admin/internal/config"
)

type ImageURL struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Repo handles communication with a MinIO server.
type Repo struct {
	client *minio.Client
	log    *slog.Logger
	bucket string
}

var fileNameRegex = regexp.MustCompile(`^(\d+)(.*)`)

// New creates a new MinIO repository.
func New(cfg config.Minio, log *slog.Logger, bucket string) (*Repo, error) {
	const op = "storage.minio.New"

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Repo{client: client, log: log, bucket: bucket}, nil
}

// ListImageURLs lists all objects in a bucket and returns them as presigned URLs.
// It also sorts them based on the numeric prefix in the filename.
func (r *Repo) ListImageURLs(ctx context.Context, bucketName string) ([]ImageURL, error) {
	const op = "storage.minio.ListImageURLs"

	objectsCh := r.client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
		Recursive: true,
	})

	// This internal struct now also holds the clean name.
	type objectInfo struct {
		key   string
		order int
		name  string
	}

	var objects []objectInfo

	for object := range objectsCh {
		if object.Err != nil {
			return nil, fmt.Errorf("%s: failed to list object: %w", op, object.Err)
		}

		var order int
		// Default to the original key if the pattern doesn't match.
		cleanName := object.Key

		matches := fileNameRegex.FindStringSubmatch(object.Key)
		if len(matches) == 3 {
			// The first submatch (index 1) is our captured number.
			if n, err := strconv.Atoi(matches[1]); err == nil {
				order = n
			}
			// The second submatch (index 2) is the rest of the filename.
			nameWithExt := matches[2]

			// 3. Remove the extension from the name.
			// For a name like "Chorvoq.jpg", this will result in "Chorvoq".
			cleanName = strings.TrimSuffix(nameWithExt, path.Ext(nameWithExt))
		}

		objects = append(objects, objectInfo{key: object.Key, order: order, name: cleanName})
	}

	// Sort objects based on the extracted number (no changes here).
	sort.Slice(objects, func(i, j int) bool {
		return objects[i].order < objects[j].order
	})

	// Create a slice of the new ImageURL struct.
	var result []ImageURL
	for _, obj := range objects {
		// Generate a presigned URL that is valid for 1 hour.
		presignedURL, err := r.client.PresignedGetObject(ctx, bucketName, obj.key, time.Hour, nil)
		if err != nil {
			// Log the error but continue, so one failed URL doesn't break the whole list.
			r.log.Error("failed to generate presigned URL", "bucket", bucketName, "key", obj.key, "error", err)
			continue
		}

		// Append the structured object to the result slice.
		result = append(result, ImageURL{
			Name: obj.name,
			URL:  presignedURL.String(),
		})
	}

	return result, nil
}

func (r *Repo) UploadFile(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) error {
	const op = "repo.minio.UploadFile"

	_, err := r.client.PutObject(ctx, r.bucket, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

// GetPresignedURL генерирует временную ссылку для скачивания файла.
func (r *Repo) GetPresignedURL(ctx context.Context, objectName string, expires time.Duration) (*url.URL, error) {
	const op = "repo.minio.GetPresignedURL"

	presignedURL, err := r.client.PresignedGetObject(ctx, r.bucket, objectName, expires, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return presignedURL, nil
}

// DeleteFile удаляет файл из бакета.
func (r *Repo) DeleteFile(ctx context.Context, objectName string) error {
	const op = "repo.minio.DeleteFile"

	err := r.client.RemoveObject(ctx, r.bucket, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}
