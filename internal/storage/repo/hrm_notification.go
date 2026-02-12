package repo

import (
	"context"
	"fmt"
	"srmt-admin/internal/lib/model/hrm/dashboard"
	"srmt-admin/internal/storage"
)

func (r *Repo) GetHRMNotifications(ctx context.Context, userID int64) ([]*dashboard.Notification, error) {
	const op = "repo.GetHRMNotifications"

	query := `
		SELECT id, title, message, type, read, read_at::text, created_at::text, link
		FROM hrm_notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 50`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var notifications []*dashboard.Notification
	for rows.Next() {
		var n dashboard.Notification
		var link *string
		if err := rows.Scan(&n.ID, &n.Title, &n.Message, &n.Type, &n.Read, &n.ReadAt, &n.CreatedAt, &link); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		n.Link = link
		notifications = append(notifications, &n)
	}
	return notifications, rows.Err()
}

func (r *Repo) MarkHRMNotificationRead(ctx context.Context, notificationID int64, userID int64) error {
	const op = "repo.MarkHRMNotificationRead"

	result, err := r.db.ExecContext(ctx,
		"UPDATE hrm_notifications SET read = TRUE, read_at = NOW() WHERE id = $1 AND user_id = $2",
		notificationID, userID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrNotificationNotFound
	}
	return nil
}

func (r *Repo) MarkAllHRMNotificationsRead(ctx context.Context, userID int64) error {
	const op = "repo.MarkAllHRMNotificationsRead"

	_, err := r.db.ExecContext(ctx,
		"UPDATE hrm_notifications SET read = TRUE, read_at = NOW() WHERE user_id = $1 AND read = FALSE",
		userID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}
