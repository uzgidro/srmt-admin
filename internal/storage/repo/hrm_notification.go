package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"srmt-admin/internal/lib/dto/hrm"
	hrmmodel "srmt-admin/internal/lib/model/hrm"
	"srmt-admin/internal/storage"
)

// AddNotification creates a new notification
func (r *Repo) AddNotification(ctx context.Context, req hrm.AddNotificationRequest) (int64, error) {
	const op = "storage.repo.AddNotification"

	priority := req.Priority
	if priority == "" {
		priority = hrmmodel.NotificationPriorityNormal
	}

	const query = `
		INSERT INTO hrm_notifications (
			user_id, title, message, category, entity_type, entity_id,
			priority, action_url, action_label, send_email, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.UserID, req.Title, req.Message, req.Category,
		req.EntityType, req.EntityID, priority,
		req.ActionURL, req.ActionLabel, req.SendEmail, req.ExpiresAt,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("%s: failed to insert notification: %w", op, err)
	}

	return id, nil
}

// AddNotificationsBulk creates notifications for multiple users
func (r *Repo) AddNotificationsBulk(ctx context.Context, req hrm.BulkNotificationRequest) error {
	const op = "storage.repo.AddNotificationsBulk"

	priority := req.Priority
	if priority == "" {
		priority = hrmmodel.NotificationPriorityNormal
	}

	const query = `
		INSERT INTO hrm_notifications (
			user_id, title, message, category, entity_type, entity_id,
			priority, action_url, action_label, send_email, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	for _, userID := range req.UserIDs {
		_, err := r.db.ExecContext(ctx, query,
			userID, req.Title, req.Message, req.Category,
			req.EntityType, req.EntityID, priority,
			req.ActionURL, req.ActionLabel, req.SendEmail, req.ExpiresAt,
		)
		if err != nil {
			return fmt.Errorf("%s: failed to insert notification for user %d: %w", op, userID, err)
		}
	}

	return nil
}

// GetNotificationByID retrieves notification by ID
func (r *Repo) GetNotificationByID(ctx context.Context, id int64) (*hrmmodel.Notification, error) {
	const op = "storage.repo.GetNotificationByID"

	const query = `
		SELECT id, user_id, title, message, category, entity_type, entity_id,
			priority, is_read, read_at, action_url, action_label,
			send_email, email_sent_at, expires_at, created_at
		FROM hrm_notifications
		WHERE id = $1`

	n, err := r.scanNotification(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get notification: %w", op, err)
	}

	return n, nil
}

// GetNotifications retrieves notifications with filters
func (r *Repo) GetNotifications(ctx context.Context, filter hrm.NotificationFilter) ([]*hrmmodel.Notification, error) {
	const op = "storage.repo.GetNotifications"

	var query strings.Builder
	query.WriteString(`
		SELECT id, user_id, title, message, category, entity_type, entity_id,
			priority, is_read, read_at, action_url, action_label,
			send_email, email_sent_at, expires_at, created_at
		FROM hrm_notifications
		WHERE user_id = $1
	`)

	args := []interface{}{filter.UserID}
	argIdx := 2

	if filter.Category != nil {
		query.WriteString(fmt.Sprintf(" AND category = $%d", argIdx))
		args = append(args, *filter.Category)
		argIdx++
	}
	if filter.IsRead != nil {
		query.WriteString(fmt.Sprintf(" AND is_read = $%d", argIdx))
		args = append(args, *filter.IsRead)
		argIdx++
	}
	if filter.Priority != nil {
		query.WriteString(fmt.Sprintf(" AND priority = $%d", argIdx))
		args = append(args, *filter.Priority)
		argIdx++
	}

	// Exclude expired
	query.WriteString(" AND (expires_at IS NULL OR expires_at > NOW())")

	query.WriteString(" ORDER BY created_at DESC")

	if filter.Limit > 0 {
		query.WriteString(fmt.Sprintf(" LIMIT $%d", argIdx))
		args = append(args, filter.Limit)
		argIdx++
	}
	if filter.Offset > 0 {
		query.WriteString(fmt.Sprintf(" OFFSET $%d", argIdx))
		args = append(args, filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query notifications: %w", op, err)
	}
	defer rows.Close()

	var notifications []*hrmmodel.Notification
	for rows.Next() {
		n, err := r.scanNotificationRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan notification: %w", op, err)
		}
		notifications = append(notifications, n)
	}

	if notifications == nil {
		notifications = make([]*hrmmodel.Notification, 0)
	}

	return notifications, nil
}

// MarkNotificationRead marks notification as read
func (r *Repo) MarkNotificationRead(ctx context.Context, id int64, userID int64) error {
	const op = "storage.repo.MarkNotificationRead"

	const query = `
		UPDATE hrm_notifications
		SET is_read = TRUE, read_at = $1
		WHERE id = $2 AND user_id = $3 AND is_read = FALSE`

	res, err := r.db.ExecContext(ctx, query, time.Now(), id, userID)
	if err != nil {
		return fmt.Errorf("%s: failed to mark read: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// MarkNotificationsRead marks multiple notifications as read
func (r *Repo) MarkNotificationsRead(ctx context.Context, ids []int64, userID int64) error {
	const op = "storage.repo.MarkNotificationsRead"

	if len(ids) == 0 {
		return nil
	}

	// Build placeholders
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids)+2)
	args[0] = time.Now()
	args[1] = userID
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+3)
		args[i+2] = id
	}

	query := fmt.Sprintf(`
		UPDATE hrm_notifications
		SET is_read = TRUE, read_at = $1
		WHERE user_id = $2 AND is_read = FALSE AND id IN (%s)`,
		strings.Join(placeholders, ", "))

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to mark read: %w", op, err)
	}

	return nil
}

// MarkAllNotificationsRead marks all notifications as read for user
func (r *Repo) MarkAllNotificationsRead(ctx context.Context, userID int64, category *string) error {
	const op = "storage.repo.MarkAllNotificationsRead"

	var query strings.Builder
	query.WriteString(`
		UPDATE hrm_notifications
		SET is_read = TRUE, read_at = $1
		WHERE user_id = $2 AND is_read = FALSE
	`)

	args := []interface{}{time.Now(), userID}
	if category != nil {
		query.WriteString(" AND category = $3")
		args = append(args, *category)
	}

	_, err := r.db.ExecContext(ctx, query.String(), args...)
	if err != nil {
		return fmt.Errorf("%s: failed to mark all read: %w", op, err)
	}

	return nil
}

// CountUnreadNotifications counts unread notifications for user
func (r *Repo) CountUnreadNotifications(ctx context.Context, userID int64) (int, error) {
	const op = "storage.repo.CountUnreadNotifications"

	const query = `
		SELECT COUNT(*)
		FROM hrm_notifications
		WHERE user_id = $1 AND is_read = FALSE
		AND (expires_at IS NULL OR expires_at > NOW())`

	var count int
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("%s: failed to count notifications: %w", op, err)
	}

	return count, nil
}

// GetNotificationCount returns count by read status
func (r *Repo) GetNotificationCount(ctx context.Context, userID int64) (*hrmmodel.NotificationCount, error) {
	const op = "storage.repo.GetNotificationCount"

	const query = `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE is_read = FALSE) as unread,
			COUNT(*) FILTER (WHERE is_read = TRUE) as read
		FROM hrm_notifications
		WHERE user_id = $1
		AND (expires_at IS NULL OR expires_at > NOW())`

	var count hrmmodel.NotificationCount
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&count.Total, &count.Unread, &count.Read,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get count: %w", op, err)
	}

	return &count, nil
}

// DeleteNotification deletes a notification
func (r *Repo) DeleteNotification(ctx context.Context, id int64, userID int64) error {
	const op = "storage.repo.DeleteNotification"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_notifications WHERE id = $1 AND user_id = $2", id, userID)
	if err != nil {
		return fmt.Errorf("%s: failed to delete notification: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteExpiredNotifications removes expired notifications
func (r *Repo) DeleteExpiredNotifications(ctx context.Context) (int64, error) {
	const op = "storage.repo.DeleteExpiredNotifications"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_notifications WHERE expires_at IS NOT NULL AND expires_at < NOW()")
	if err != nil {
		return 0, fmt.Errorf("%s: failed to delete expired: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	return rowsAffected, nil
}

// MarkEmailSent marks notification email as sent
func (r *Repo) MarkEmailSent(ctx context.Context, id int64) error {
	const op = "storage.repo.MarkEmailSent"

	const query = `
		UPDATE hrm_notifications
		SET email_sent_at = $1
		WHERE id = $2`

	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("%s: failed to mark email sent: %w", op, err)
	}

	return nil
}

// GetPendingEmailNotifications retrieves notifications that need email
func (r *Repo) GetPendingEmailNotifications(ctx context.Context, limit int) ([]*hrmmodel.Notification, error) {
	const op = "storage.repo.GetPendingEmailNotifications"

	const query = `
		SELECT id, user_id, title, message, category, entity_type, entity_id,
			priority, is_read, read_at, action_url, action_label,
			send_email, email_sent_at, expires_at, created_at
		FROM hrm_notifications
		WHERE send_email = TRUE AND email_sent_at IS NULL
		AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY created_at
		LIMIT $1`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query: %w", op, err)
	}
	defer rows.Close()

	var notifications []*hrmmodel.Notification
	for rows.Next() {
		n, err := r.scanNotificationRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan: %w", op, err)
		}
		notifications = append(notifications, n)
	}

	if notifications == nil {
		notifications = make([]*hrmmodel.Notification, 0)
	}

	return notifications, nil
}

// Helper to scan notification
func (r *Repo) scanNotification(row *sql.Row) (*hrmmodel.Notification, error) {
	var n hrmmodel.Notification
	var entityType, actionURL, actionLabel sql.NullString
	var entityIDVal sql.NullInt64
	var readAt, emailSentAt, expiresAt sql.NullTime

	err := row.Scan(
		&n.ID, &n.UserID, &n.Title, &n.Message, &n.Category,
		&entityType, &entityIDVal, &n.Priority, &n.IsRead, &readAt,
		&actionURL, &actionLabel, &n.SendEmail, &emailSentAt, &expiresAt, &n.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if entityType.Valid {
		n.EntityType = &entityType.String
	}
	if entityIDVal.Valid {
		n.EntityID = &entityIDVal.Int64
	}
	if readAt.Valid {
		n.ReadAt = &readAt.Time
	}
	if actionURL.Valid {
		n.ActionURL = &actionURL.String
	}
	if actionLabel.Valid {
		n.ActionLabel = &actionLabel.String
	}
	if emailSentAt.Valid {
		n.EmailSentAt = &emailSentAt.Time
	}
	if expiresAt.Valid {
		n.ExpiresAt = &expiresAt.Time
	}

	return &n, nil
}

// Helper to scan notification from rows
func (r *Repo) scanNotificationRow(rows *sql.Rows) (*hrmmodel.Notification, error) {
	var n hrmmodel.Notification
	var entityType sql.NullString
	var entityIDVal sql.NullInt64
	var actionURL, actionLabel sql.NullString
	var readAt, emailSentAt, expiresAt sql.NullTime

	err := rows.Scan(
		&n.ID, &n.UserID, &n.Title, &n.Message, &n.Category,
		&entityType, &entityIDVal, &n.Priority, &n.IsRead, &readAt,
		&actionURL, &actionLabel, &n.SendEmail, &emailSentAt, &expiresAt, &n.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if entityType.Valid {
		n.EntityType = &entityType.String
	}
	if entityIDVal.Valid {
		n.EntityID = &entityIDVal.Int64
	}
	if readAt.Valid {
		n.ReadAt = &readAt.Time
	}
	if actionURL.Valid {
		n.ActionURL = &actionURL.String
	}
	if actionLabel.Valid {
		n.ActionLabel = &actionLabel.String
	}
	if emailSentAt.Valid {
		n.EmailSentAt = &emailSentAt.Time
	}
	if expiresAt.Valid {
		n.ExpiresAt = &expiresAt.Time
	}

	return &n, nil
}
