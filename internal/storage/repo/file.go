package repo

import (
	"context"
	"fmt"
	"srmt-admin/internal/lib/model/file"
)

func (r *Repo) AddFile(ctx context.Context, fileData file.Model) (int64, error) {
	const op = "repo.AddFile"
	const query = `
		INSERT INTO files(file_name, object_key, category_id, mime_type, size_bytes)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	stmt, err := r.db.Prepare(query)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	var id int64
	err = stmt.QueryRowContext(ctx,
		fileData.FileName,
		fileData.ObjectKey,
		fileData.CategoryID,
		fileData.MimeType,
		fileData.SizeBytes,
	).Scan(&id)
	if err != nil {
		return 0, r.translator.Translate(err, op)
	}

	return id, nil
}

func (r *Repo) GetLatestFiles(ctx context.Context) ([]file.LatestFile, error) {
	const op = "repo.GetLatestFiles"
	const query = `
		WITH ranked_files AS (
			SELECT
				f.id,
				f.file_name,
				f.object_key,
				f.created_at,
				c.display_name as category_name,
				ROW_NUMBER() OVER(PARTITION BY f.category_id ORDER BY f.created_at DESC) as rn
			FROM files f
			JOIN categories c ON f.category_id = c.id
		)
		SELECT id, file_name, object_key, created_at, category_name
		FROM ranked_files
		WHERE rn = 1
		ORDER BY category_name;
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var latestFiles []file.LatestFile // Используем специальную модель для результата
	for rows.Next() {
		var f file.LatestFile
		if err := rows.Scan(&f.ID, &f.FileName, &f.ObjectKey, &f.CreatedAt, &f.CategoryName); err != nil {
			return nil, fmt.Errorf("%s: failed to scan row: %w", op, err)
		}
		latestFiles = append(latestFiles, f)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return latestFiles, nil
}
