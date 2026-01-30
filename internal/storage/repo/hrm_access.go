package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"

	"srmt-admin/internal/lib/dto/hrm"
	hrmmodel "srmt-admin/internal/lib/model/hrm"
	"srmt-admin/internal/storage"
)

// --- Access Zones ---

// AddAccessZone creates a new access zone
func (r *Repo) AddAccessZone(ctx context.Context, req hrm.AddAccessZoneRequest) (int, error) {
	const op = "storage.repo.AddAccessZone"

	const query = `
		INSERT INTO hrm_access_zones (
			name, code, description, building, floor, security_level, access_schedule
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	var id int
	err := r.db.QueryRowContext(ctx, query,
		req.Name, req.Code, req.Description, req.Building, req.Floor, req.SecurityLevel, req.AccessSchedule,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("%s: failed to insert access zone: %w", op, err)
	}

	return id, nil
}

// GetAccessZoneByID retrieves access zone by ID
func (r *Repo) GetAccessZoneByID(ctx context.Context, id int) (*hrmmodel.AccessZone, error) {
	const op = "storage.repo.GetAccessZoneByID"

	const query = `
		SELECT id, name, code, description, building, floor, security_level, access_schedule, is_active, created_at, updated_at
		FROM hrm_access_zones
		WHERE id = $1`

	z, err := r.scanAccessZone(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get access zone: %w", op, err)
	}

	return z, nil
}

// GetAccessZones retrieves access zones with filters
func (r *Repo) GetAccessZones(ctx context.Context, filter hrm.AccessZoneFilter) ([]*hrmmodel.AccessZone, error) {
	const op = "storage.repo.GetAccessZones"

	var query strings.Builder
	query.WriteString(`
		SELECT id, name, code, description, building, floor, security_level, access_schedule, is_active, created_at, updated_at
		FROM hrm_access_zones
		WHERE 1=1
	`)

	args := []interface{}{}
	argIdx := 1

	if filter.Building != nil {
		query.WriteString(fmt.Sprintf(" AND building = $%d", argIdx))
		args = append(args, *filter.Building)
		argIdx++
	}
	if filter.SecurityLevel != nil {
		query.WriteString(fmt.Sprintf(" AND security_level = $%d", argIdx))
		args = append(args, *filter.SecurityLevel)
		argIdx++
	}
	if filter.IsActive != nil {
		query.WriteString(fmt.Sprintf(" AND is_active = $%d", argIdx))
		args = append(args, *filter.IsActive)
		argIdx++
	}

	query.WriteString(" ORDER BY building, floor, name")

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query access zones: %w", op, err)
	}
	defer rows.Close()

	var zones []*hrmmodel.AccessZone
	for rows.Next() {
		z, err := r.scanAccessZoneRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan access zone: %w", op, err)
		}
		zones = append(zones, z)
	}

	if zones == nil {
		zones = make([]*hrmmodel.AccessZone, 0)
	}

	return zones, nil
}

// EditAccessZone updates access zone
func (r *Repo) EditAccessZone(ctx context.Context, id int, req hrm.EditAccessZoneRequest) error {
	const op = "storage.repo.EditAccessZone"

	var updates []string
	var args []interface{}
	argIdx := 1

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Code != nil {
		updates = append(updates, fmt.Sprintf("code = $%d", argIdx))
		args = append(args, *req.Code)
		argIdx++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.Building != nil {
		updates = append(updates, fmt.Sprintf("building = $%d", argIdx))
		args = append(args, *req.Building)
		argIdx++
	}
	if req.Floor != nil {
		updates = append(updates, fmt.Sprintf("floor = $%d", argIdx))
		args = append(args, *req.Floor)
		argIdx++
	}
	if req.SecurityLevel != nil {
		updates = append(updates, fmt.Sprintf("security_level = $%d", argIdx))
		args = append(args, *req.SecurityLevel)
		argIdx++
	}
	if req.AccessSchedule != nil {
		updates = append(updates, fmt.Sprintf("access_schedule = $%d", argIdx))
		args = append(args, req.AccessSchedule)
		argIdx++
	}
	if req.IsActive != nil {
		updates = append(updates, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *req.IsActive)
		argIdx++
	}

	if len(updates) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE hrm_access_zones SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to update access zone: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteAccessZone deletes access zone
func (r *Repo) DeleteAccessZone(ctx context.Context, id int) error {
	const op = "storage.repo.DeleteAccessZone"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_access_zones WHERE id = $1", id)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			return storage.ErrForeignKeyViolation
		}
		return fmt.Errorf("%s: failed to delete access zone: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- Access Cards ---

// AddAccessCard creates a new access card
func (r *Repo) AddAccessCard(ctx context.Context, req hrm.AddAccessCardRequest) (int64, error) {
	const op = "storage.repo.AddAccessCard"

	const query = `
		INSERT INTO hrm_access_cards (
			employee_id, card_number, card_type, issued_date, expiry_date, is_active, notes
		) VALUES ($1, $2, $3, $4, $5, TRUE, $6)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.EmployeeID, req.CardNumber, req.CardType, req.IssuedDate, req.ExpiryDate, req.Notes,
	).Scan(&id)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return 0, storage.ErrUniqueViolation
		}
		return 0, fmt.Errorf("%s: failed to insert access card: %w", op, err)
	}

	return id, nil
}

// GetAccessCardByID retrieves access card by ID
func (r *Repo) GetAccessCardByID(ctx context.Context, id int64) (*hrmmodel.AccessCard, error) {
	const op = "storage.repo.GetAccessCardByID"

	const query = `
		SELECT id, employee_id, card_number, card_type, issued_date, expiry_date, is_active,
			deactivated_at, deactivation_reason, notes, created_at, updated_at
		FROM hrm_access_cards
		WHERE id = $1`

	c, err := r.scanAccessCard(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get access card: %w", op, err)
	}

	return c, nil
}

// GetAccessCards retrieves access cards with filters
func (r *Repo) GetAccessCards(ctx context.Context, filter hrm.AccessCardFilter) ([]*hrmmodel.AccessCard, error) {
	const op = "storage.repo.GetAccessCards"

	var query strings.Builder
	query.WriteString(`
		SELECT id, employee_id, card_number, card_type, issued_date, expiry_date, is_active,
			deactivated_at, deactivation_reason, notes, created_at, updated_at
		FROM hrm_access_cards
		WHERE 1=1
	`)

	args := []interface{}{}
	argIdx := 1

	if filter.EmployeeID != nil {
		query.WriteString(fmt.Sprintf(" AND employee_id = $%d", argIdx))
		args = append(args, *filter.EmployeeID)
		argIdx++
	}
	if filter.CardType != nil {
		query.WriteString(fmt.Sprintf(" AND card_type = $%d", argIdx))
		args = append(args, *filter.CardType)
		argIdx++
	}
	if filter.IsActive != nil {
		query.WriteString(fmt.Sprintf(" AND is_active = $%d", argIdx))
		args = append(args, *filter.IsActive)
		argIdx++
	}
	if filter.Search != nil {
		query.WriteString(fmt.Sprintf(" AND card_number ILIKE $%d", argIdx))
		args = append(args, "%"+*filter.Search+"%")
		argIdx++
	}

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
		return nil, fmt.Errorf("%s: failed to query access cards: %w", op, err)
	}
	defer rows.Close()

	var cards []*hrmmodel.AccessCard
	for rows.Next() {
		c, err := r.scanAccessCardRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan access card: %w", op, err)
		}
		cards = append(cards, c)
	}

	if cards == nil {
		cards = make([]*hrmmodel.AccessCard, 0)
	}

	return cards, nil
}

// GetActiveCardByEmployee retrieves active card for employee
func (r *Repo) GetActiveCardByEmployee(ctx context.Context, employeeID int64) (*hrmmodel.AccessCard, error) {
	const op = "storage.repo.GetActiveCardByEmployee"

	const query = `
		SELECT id, employee_id, card_number, card_type, issued_date, expiry_date, is_active,
			deactivated_at, deactivation_reason, notes, created_at, updated_at
		FROM hrm_access_cards
		WHERE employee_id = $1 AND is_active = TRUE
		ORDER BY created_at DESC
		LIMIT 1`

	c, err := r.scanAccessCard(r.db.QueryRowContext(ctx, query, employeeID))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get active card: %w", op, err)
	}

	return c, nil
}

// EditAccessCard updates access card
func (r *Repo) EditAccessCard(ctx context.Context, id int64, req hrm.EditAccessCardRequest) error {
	const op = "storage.repo.EditAccessCard"

	var updates []string
	var args []interface{}
	argIdx := 1

	if req.CardNumber != nil {
		updates = append(updates, fmt.Sprintf("card_number = $%d", argIdx))
		args = append(args, *req.CardNumber)
		argIdx++
	}
	if req.CardType != nil {
		updates = append(updates, fmt.Sprintf("card_type = $%d", argIdx))
		args = append(args, *req.CardType)
		argIdx++
	}
	if req.ExpiryDate != nil {
		updates = append(updates, fmt.Sprintf("expiry_date = $%d", argIdx))
		args = append(args, *req.ExpiryDate)
		argIdx++
	}
	if req.IsActive != nil {
		updates = append(updates, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *req.IsActive)
		argIdx++
	}
	if req.Notes != nil {
		updates = append(updates, fmt.Sprintf("notes = $%d", argIdx))
		args = append(args, *req.Notes)
		argIdx++
	}

	if len(updates) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE hrm_access_cards SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return storage.ErrUniqueViolation
		}
		return fmt.Errorf("%s: failed to update access card: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeactivateAccessCard deactivates access card
func (r *Repo) DeactivateAccessCard(ctx context.Context, id int64, reason string) error {
	const op = "storage.repo.DeactivateAccessCard"

	const query = `
		UPDATE hrm_access_cards
		SET is_active = FALSE, deactivated_at = $1, deactivation_reason = $2
		WHERE id = $3`

	res, err := r.db.ExecContext(ctx, query, time.Now(), reason, id)
	if err != nil {
		return fmt.Errorf("%s: failed to deactivate card: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteAccessCard deletes access card
func (r *Repo) DeleteAccessCard(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteAccessCard"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_access_cards WHERE id = $1", id)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			return storage.ErrForeignKeyViolation
		}
		return fmt.Errorf("%s: failed to delete access card: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- Card Zone Access ---

// AddCardZoneAccess grants zone access to card
func (r *Repo) AddCardZoneAccess(ctx context.Context, req hrm.AddCardZoneAccessRequest, grantedBy *int64) (int64, error) {
	const op = "storage.repo.AddCardZoneAccess"

	const query = `
		INSERT INTO hrm_card_zone_access (
			card_id, zone_id, custom_schedule, valid_from, valid_until, granted_by, granted_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.CardID, req.ZoneID, req.CustomSchedule, req.ValidFrom, req.ValidUntil, grantedBy, time.Now(),
	).Scan(&id)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return 0, storage.ErrUniqueViolation
		}
		return 0, fmt.Errorf("%s: failed to add zone access: %w", op, err)
	}

	return id, nil
}

// BulkAddCardZoneAccess grants multiple zone access to card
func (r *Repo) BulkAddCardZoneAccess(ctx context.Context, req hrm.BulkZoneAccessRequest, grantedBy *int64) error {
	const op = "storage.repo.BulkAddCardZoneAccess"

	const query = `
		INSERT INTO hrm_card_zone_access (
			card_id, zone_id, valid_from, valid_until, granted_by, granted_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (card_id, zone_id) DO UPDATE SET valid_from = $3, valid_until = $4`

	for _, zoneID := range req.ZoneIDs {
		_, err := r.db.ExecContext(ctx, query,
			req.CardID, zoneID, req.ValidFrom, req.ValidUntil, grantedBy, time.Now(),
		)
		if err != nil {
			return fmt.Errorf("%s: failed to add zone %d: %w", op, zoneID, err)
		}
	}

	return nil
}

// GetCardZoneAccess retrieves zone access for card
func (r *Repo) GetCardZoneAccess(ctx context.Context, filter hrm.CardZoneAccessFilter) ([]*hrmmodel.CardZoneAccess, error) {
	const op = "storage.repo.GetCardZoneAccess"

	var query strings.Builder
	query.WriteString(`
		SELECT id, card_id, zone_id, custom_schedule, valid_from, valid_until, granted_by, granted_at, created_at, updated_at
		FROM hrm_card_zone_access
		WHERE 1=1
	`)

	args := []interface{}{}
	argIdx := 1

	if filter.CardID != nil {
		query.WriteString(fmt.Sprintf(" AND card_id = $%d", argIdx))
		args = append(args, *filter.CardID)
		argIdx++
	}
	if filter.ZoneID != nil {
		query.WriteString(fmt.Sprintf(" AND zone_id = $%d", argIdx))
		args = append(args, *filter.ZoneID)
		argIdx++
	}

	query.WriteString(" ORDER BY zone_id")

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query zone access: %w", op, err)
	}
	defer rows.Close()

	var access []*hrmmodel.CardZoneAccess
	for rows.Next() {
		var a hrmmodel.CardZoneAccess
		var customSchedule []byte
		var validUntil, updatedAt sql.NullTime
		var grantedBy sql.NullInt64

		err := rows.Scan(
			&a.ID, &a.CardID, &a.ZoneID, &customSchedule, &a.ValidFrom, &validUntil, &grantedBy, &a.GrantedAt, &a.CreatedAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan zone access: %w", op, err)
		}

		if customSchedule != nil {
			a.CustomSchedule = customSchedule
		}
		if validUntil.Valid {
			a.ValidUntil = &validUntil.Time
		}
		if grantedBy.Valid {
			a.GrantedBy = &grantedBy.Int64
		}
		if updatedAt.Valid {
			a.UpdatedAt = &updatedAt.Time
		}

		access = append(access, &a)
	}

	if access == nil {
		access = make([]*hrmmodel.CardZoneAccess, 0)
	}

	return access, nil
}

// EditCardZoneAccess updates zone access
func (r *Repo) EditCardZoneAccess(ctx context.Context, id int64, req hrm.EditCardZoneAccessRequest) error {
	const op = "storage.repo.EditCardZoneAccess"

	var updates []string
	var args []interface{}
	argIdx := 1

	if req.CustomSchedule != nil {
		updates = append(updates, fmt.Sprintf("custom_schedule = $%d", argIdx))
		args = append(args, req.CustomSchedule)
		argIdx++
	}
	if req.ValidFrom != nil {
		updates = append(updates, fmt.Sprintf("valid_from = $%d", argIdx))
		args = append(args, *req.ValidFrom)
		argIdx++
	}
	if req.ValidUntil != nil {
		updates = append(updates, fmt.Sprintf("valid_until = $%d", argIdx))
		args = append(args, *req.ValidUntil)
		argIdx++
	}

	if len(updates) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE hrm_card_zone_access SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to update zone access: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteCardZoneAccess removes zone access
func (r *Repo) DeleteCardZoneAccess(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteCardZoneAccess"

	res, err := r.db.ExecContext(ctx, "DELETE FROM hrm_card_zone_access WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("%s: failed to delete zone access: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// --- Access Logs ---

// AddAccessLog records access event
func (r *Repo) AddAccessLog(ctx context.Context, req hrm.AddAccessLogRequest) (int64, error) {
	const op = "storage.repo.AddAccessLog"

	const query = `
		INSERT INTO hrm_access_logs (
			card_id, employee_id, zone_id, event_type, event_time,
			device_id, device_name, denial_reason, card_number, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.CardID, req.EmployeeID, req.ZoneID, req.EventType, req.EventTime,
		req.DeviceID, req.DeviceName, req.DenialReason, req.CardNumber, req.Metadata,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("%s: failed to insert access log: %w", op, err)
	}

	return id, nil
}

// GetAccessLogs retrieves access logs with filters
func (r *Repo) GetAccessLogs(ctx context.Context, filter hrm.AccessLogFilter) ([]*hrmmodel.AccessLog, error) {
	const op = "storage.repo.GetAccessLogs"

	var query strings.Builder
	query.WriteString(`
		SELECT id, card_id, employee_id, zone_id, event_type, event_time,
			device_id, device_name, denial_reason, card_number, metadata
		FROM hrm_access_logs
		WHERE 1=1
	`)

	args := []interface{}{}
	argIdx := 1

	if filter.EmployeeID != nil {
		query.WriteString(fmt.Sprintf(" AND employee_id = $%d", argIdx))
		args = append(args, *filter.EmployeeID)
		argIdx++
	}
	if filter.CardID != nil {
		query.WriteString(fmt.Sprintf(" AND card_id = $%d", argIdx))
		args = append(args, *filter.CardID)
		argIdx++
	}
	if filter.ZoneID != nil {
		query.WriteString(fmt.Sprintf(" AND zone_id = $%d", argIdx))
		args = append(args, *filter.ZoneID)
		argIdx++
	}
	if filter.EventType != nil {
		query.WriteString(fmt.Sprintf(" AND event_type = $%d", argIdx))
		args = append(args, *filter.EventType)
		argIdx++
	}
	if filter.FromTime != nil {
		query.WriteString(fmt.Sprintf(" AND event_time >= $%d", argIdx))
		args = append(args, *filter.FromTime)
		argIdx++
	}
	if filter.ToTime != nil {
		query.WriteString(fmt.Sprintf(" AND event_time <= $%d", argIdx))
		args = append(args, *filter.ToTime)
		argIdx++
	}

	query.WriteString(" ORDER BY event_time DESC")

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
		return nil, fmt.Errorf("%s: failed to query access logs: %w", op, err)
	}
	defer rows.Close()

	var logs []*hrmmodel.AccessLog
	for rows.Next() {
		var l hrmmodel.AccessLog
		var cardID, employeeID sql.NullInt64
		var deviceID, deviceName, denialReason, cardNumber sql.NullString
		var metadata []byte

		err := rows.Scan(
			&l.ID, &cardID, &employeeID, &l.ZoneID, &l.EventType, &l.EventTime,
			&deviceID, &deviceName, &denialReason, &cardNumber, &metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan access log: %w", op, err)
		}

		if cardID.Valid {
			l.CardID = &cardID.Int64
		}
		if employeeID.Valid {
			l.EmployeeID = &employeeID.Int64
		}
		if deviceID.Valid {
			l.DeviceID = &deviceID.String
		}
		if deviceName.Valid {
			l.DeviceName = &deviceName.String
		}
		if denialReason.Valid {
			l.DenialReason = &denialReason.String
		}
		if cardNumber.Valid {
			l.CardNumber = &cardNumber.String
		}
		if metadata != nil {
			l.Metadata = metadata
		}

		logs = append(logs, &l)
	}

	if logs == nil {
		logs = make([]*hrmmodel.AccessLog, 0)
	}

	return logs, nil
}

// GetAccessStats retrieves access statistics
func (r *Repo) GetAccessStats(ctx context.Context) (*hrmmodel.AccessStats, error) {
	const op = "storage.repo.GetAccessStats"

	stats := &hrmmodel.AccessStats{}
	today := time.Now().Truncate(24 * time.Hour)

	// Active cards
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM hrm_access_cards WHERE is_active = TRUE").Scan(&stats.TotalActiveCards)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to count active cards: %w", op, err)
	}

	// Total zones
	err = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM hrm_access_zones WHERE is_active = TRUE").Scan(&stats.TotalZones)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to count zones: %w", op, err)
	}

	// Today's entries
	err = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM hrm_access_logs WHERE event_type = 'entry' AND event_time >= $1", today).Scan(&stats.TodayEntries)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to count entries: %w", op, err)
	}

	// Today's exits
	err = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM hrm_access_logs WHERE event_type = 'exit' AND event_time >= $1", today).Scan(&stats.TodayExits)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to count exits: %w", op, err)
	}

	// Today's denied
	err = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM hrm_access_logs WHERE event_type = 'denied' AND event_time >= $1", today).Scan(&stats.TodayDenied)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to count denied: %w", op, err)
	}

	return stats, nil
}

// --- Helpers ---

func (r *Repo) scanAccessZone(row *sql.Row) (*hrmmodel.AccessZone, error) {
	var z hrmmodel.AccessZone
	var code, description, building, floor sql.NullString
	var accessSchedule []byte
	var updatedAt sql.NullTime

	err := row.Scan(
		&z.ID, &z.Name, &code, &description, &building, &floor, &z.SecurityLevel, &accessSchedule, &z.IsActive, &z.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if code.Valid {
		z.Code = &code.String
	}
	if description.Valid {
		z.Description = &description.String
	}
	if building.Valid {
		z.Building = &building.String
	}
	if floor.Valid {
		z.Floor = &floor.String
	}
	if accessSchedule != nil {
		z.AccessSchedule = accessSchedule
	}
	if updatedAt.Valid {
		z.UpdatedAt = &updatedAt.Time
	}

	return &z, nil
}

func (r *Repo) scanAccessZoneRow(rows *sql.Rows) (*hrmmodel.AccessZone, error) {
	var z hrmmodel.AccessZone
	var code, description, building, floor sql.NullString
	var accessSchedule []byte
	var updatedAt sql.NullTime

	err := rows.Scan(
		&z.ID, &z.Name, &code, &description, &building, &floor, &z.SecurityLevel, &accessSchedule, &z.IsActive, &z.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if code.Valid {
		z.Code = &code.String
	}
	if description.Valid {
		z.Description = &description.String
	}
	if building.Valid {
		z.Building = &building.String
	}
	if floor.Valid {
		z.Floor = &floor.String
	}
	if accessSchedule != nil {
		z.AccessSchedule = accessSchedule
	}
	if updatedAt.Valid {
		z.UpdatedAt = &updatedAt.Time
	}

	return &z, nil
}

func (r *Repo) scanAccessCard(row *sql.Row) (*hrmmodel.AccessCard, error) {
	var c hrmmodel.AccessCard
	var expiryDate, deactivatedAt, updatedAt sql.NullTime
	var deactivationReason, notes sql.NullString

	err := row.Scan(
		&c.ID, &c.EmployeeID, &c.CardNumber, &c.CardType, &c.IssuedDate, &expiryDate, &c.IsActive,
		&deactivatedAt, &deactivationReason, &notes, &c.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if expiryDate.Valid {
		c.ExpiryDate = &expiryDate.Time
	}
	if deactivatedAt.Valid {
		c.DeactivatedAt = &deactivatedAt.Time
	}
	if deactivationReason.Valid {
		c.DeactivationReason = &deactivationReason.String
	}
	if notes.Valid {
		c.Notes = &notes.String
	}
	if updatedAt.Valid {
		c.UpdatedAt = &updatedAt.Time
	}

	return &c, nil
}

func (r *Repo) scanAccessCardRow(rows *sql.Rows) (*hrmmodel.AccessCard, error) {
	var c hrmmodel.AccessCard
	var expiryDate, deactivatedAt, updatedAt sql.NullTime
	var deactivationReason, notes sql.NullString

	err := rows.Scan(
		&c.ID, &c.EmployeeID, &c.CardNumber, &c.CardType, &c.IssuedDate, &expiryDate, &c.IsActive,
		&deactivatedAt, &deactivationReason, &notes, &c.CreatedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if expiryDate.Valid {
		c.ExpiryDate = &expiryDate.Time
	}
	if deactivatedAt.Valid {
		c.DeactivatedAt = &deactivatedAt.Time
	}
	if deactivationReason.Valid {
		c.DeactivationReason = &deactivationReason.String
	}
	if notes.Valid {
		c.Notes = &notes.String
	}
	if updatedAt.Valid {
		c.UpdatedAt = &updatedAt.Time
	}

	return &c, nil
}
