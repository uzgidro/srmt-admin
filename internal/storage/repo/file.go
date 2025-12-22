package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/storage"
)

func (r *Repo) AddFile(ctx context.Context, fileData file.Model) (int64, error) {
	const op = "repo.file.AddFile"
	const query = `
		INSERT INTO files(file_name, object_key, category_id, mime_type, size_bytes, created_at, target_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	var id int64
	err := r.db.QueryRowContext(ctx, query,
		fileData.FileName,
		fileData.ObjectKey,
		fileData.CategoryID,
		fileData.MimeType,
		fileData.SizeBytes,
		fileData.CreatedAt,
		fileData.TargetDate,
	).Scan(&id)
	if err != nil {
		return 0, r.translator.Translate(err, op)
	}

	return id, nil
}

func (r *Repo) GetLatestFiles(ctx context.Context) ([]file.LatestFile, error) {
	const op = "repo.file.GetLatestFiles"
	const query = `
		WITH ranked_files AS (
			SELECT
				f.id,
				f.file_name,
				f.object_key,
				f.size_bytes,
				f.created_at,
				c.display_name as category_name,
				ROW_NUMBER() OVER(PARTITION BY f.category_id ORDER BY f.created_at DESC) as rn
			FROM files f
			JOIN categories c ON f.category_id = c.id
		)
		SELECT id, file_name, object_key, size_bytes, category_name, created_at
		FROM ranked_files
		WHERE rn = 1
		ORDER BY category_name;
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	// Get Presigned URL needs object key
	var latestFiles []file.LatestFile
	for rows.Next() {
		var f file.LatestFile
		if err := rows.Scan(&f.ID, &f.FileName, &f.ObjectKey, &f.SizeBytes, &f.CategoryName, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("%s: failed to scan row: %w", op, err)
		}
		latestFiles = append(latestFiles, f)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return latestFiles, nil
}

func (r *Repo) GetFileByID(ctx context.Context, id int64) (file.Model, error) {
	const op = "repo.file.GetFileByID"

	const query = `
		SELECT id, file_name, object_key, category_id, mime_type, size_bytes, created_at, target_date
		FROM files
		WHERE id = $1
	`

	var f file.Model
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&f.ID,
		&f.FileName,
		&f.ObjectKey,
		&f.CategoryID,
		&f.MimeType,
		&f.SizeBytes,
		&f.CreatedAt,
		&f.TargetDate,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return file.Model{}, storage.ErrNotFound // Используем кастомную ошибку
		}
		return file.Model{}, fmt.Errorf("%s: failed to scan row: %w", op, err)
	}

	return f, nil
}

func (r *Repo) DeleteFile(ctx context.Context, id int64) error {
	const op = "repo.file.DeleteFile"
	const query = "DELETE FROM files WHERE id = $1"

	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("%s: failed to execute statement: %w", op, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: failed to get affected rows: %w", op, err)
	}

	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// GetLatestFileByCategoryAndDate retrieves the latest file for a given category name and target date
func (r *Repo) GetLatestFileByCategoryAndDate(ctx context.Context, categoryName string, targetDate string) (file.Model, error) {
	const op = "repo.file.GetLatestFileByCategoryAndDate"

	const query = `
		SELECT f.id, f.file_name, f.object_key, f.category_id, f.mime_type, f.size_bytes, f.created_at, f.target_date
		FROM files f
		JOIN categories c ON f.category_id = c.id
		WHERE c.name = $1 AND f.target_date = $2
		ORDER BY f.created_at DESC
		LIMIT 1
	`

	var f file.Model
	err := r.db.QueryRowContext(ctx, query, categoryName, targetDate).Scan(
		&f.ID,
		&f.FileName,
		&f.ObjectKey,
		&f.CategoryID,
		&f.MimeType,
		&f.SizeBytes,
		&f.CreatedAt,
		&f.TargetDate,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return file.Model{}, storage.ErrNotFound
		}
		return file.Model{}, fmt.Errorf("%s: failed to scan row: %w", op, err)
	}

	return f, nil
}
