package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/hrm/access"
	"srmt-admin/internal/storage"
	"strings"
)

// ==================== Access Cards ====================

func (r *Repo) CreateAccessCard(ctx context.Context, req dto.CreateAccessCardRequest) (int64, error) {
	const op = "repo.CreateAccessCard"

	zones := req.AccessZones
	if zones == nil {
		zones = json.RawMessage("[]")
	}

	query := `
		INSERT INTO access_cards (employee_id, card_number, issued_date, expiry_date, access_zones, access_level)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.EmployeeID, req.CardNumber, req.IssuedDate, req.ExpiryDate, zones, req.AccessLevel,
	).Scan(&id)
	if err != nil {
		if translated := r.translator.Translate(err, op); translated != nil {
			return 0, translated
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

func (r *Repo) GetAccessCardByID(ctx context.Context, id int64) (*access.AccessCard, error) {
	const op = "repo.GetAccessCardByID"

	query := `
		SELECT ac.id, ac.employee_id, COALESCE(c.name, ''),
			   ac.card_number, ac.status, ac.issued_date, ac.expiry_date,
			   ac.access_zones, ac.access_level, ac.created_at, ac.updated_at
		FROM access_cards ac
		LEFT JOIN contacts c ON ac.employee_id = c.id
		WHERE ac.id = $1`

	card, err := scanAccessCard(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrAccessCardNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return card, nil
}

func (r *Repo) GetAllAccessCards(ctx context.Context, filters dto.AccessCardFilters) ([]*access.AccessCard, error) {
	const op = "repo.GetAllAccessCards"

	query := `
		SELECT ac.id, ac.employee_id, COALESCE(c.name, ''),
			   ac.card_number, ac.status, ac.issued_date, ac.expiry_date,
			   ac.access_zones, ac.access_level, ac.created_at, ac.updated_at
		FROM access_cards ac
		LEFT JOIN contacts c ON ac.employee_id = c.id`

	var conditions []string
	var args []interface{}
	argIdx := 1

	if filters.EmployeeID != nil {
		conditions = append(conditions, fmt.Sprintf("ac.employee_id = $%d", argIdx))
		args = append(args, *filters.EmployeeID)
		argIdx++
	}
	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("ac.status = $%d", argIdx))
		args = append(args, *filters.Status)
		argIdx++
	}
	if filters.Search != nil {
		conditions = append(conditions, fmt.Sprintf("(ac.card_number ILIKE $%d OR c.name ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+*filters.Search+"%")
		argIdx++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY ac.created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var cards []*access.AccessCard
	for rows.Next() {
		card, err := scanAccessCard(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		cards = append(cards, card)
	}
	return cards, nil
}

func (r *Repo) UpdateAccessCard(ctx context.Context, id int64, req dto.UpdateAccessCardRequest) error {
	const op = "repo.UpdateAccessCard"

	var setClauses []string
	var args []interface{}
	argIdx := 1

	if req.CardNumber != nil {
		setClauses = append(setClauses, fmt.Sprintf("card_number = $%d", argIdx))
		args = append(args, *req.CardNumber)
		argIdx++
	}
	if req.IssuedDate != nil {
		setClauses = append(setClauses, fmt.Sprintf("issued_date = $%d", argIdx))
		args = append(args, *req.IssuedDate)
		argIdx++
	}
	if req.ExpiryDate != nil {
		setClauses = append(setClauses, fmt.Sprintf("expiry_date = $%d", argIdx))
		args = append(args, *req.ExpiryDate)
		argIdx++
	}
	if req.AccessZones != nil {
		setClauses = append(setClauses, fmt.Sprintf("access_zones = $%d", argIdx))
		args = append(args, *req.AccessZones)
		argIdx++
	}
	if req.AccessLevel != nil {
		setClauses = append(setClauses, fmt.Sprintf("access_level = $%d", argIdx))
		args = append(args, *req.AccessLevel)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE access_cards SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if translated := r.translator.Translate(err, op); translated != nil {
			return translated
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrAccessCardNotFound
	}
	return nil
}

func (r *Repo) UpdateAccessCardStatus(ctx context.Context, id int64, status string) error {
	const op = "repo.UpdateAccessCardStatus"

	result, err := r.db.ExecContext(ctx,
		"UPDATE access_cards SET status = $1 WHERE id = $2", status, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrAccessCardNotFound
	}
	return nil
}

// ==================== Access Zones ====================

func (r *Repo) CreateAccessZone(ctx context.Context, req dto.CreateAccessZoneRequest) (int64, error) {
	const op = "repo.CreateAccessZone"

	readers := req.Readers
	if readers == nil {
		readers = json.RawMessage("[]")
	}
	schedules := req.Schedules
	if schedules == nil {
		schedules = json.RawMessage("[]")
	}

	maxOccupancy := 0
	if req.MaxOccupancy != nil {
		maxOccupancy = *req.MaxOccupancy
	}

	query := `
		INSERT INTO access_zones (name, description, security_level, building, floor, max_occupancy, readers, schedules)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.Name, req.Description, req.SecurityLevel, req.Building, req.Floor,
		maxOccupancy, readers, schedules,
	).Scan(&id)
	if err != nil {
		if translated := r.translator.Translate(err, op); translated != nil {
			return 0, translated
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

func (r *Repo) GetAccessZoneByID(ctx context.Context, id int64) (*access.AccessZone, error) {
	const op = "repo.GetAccessZoneByID"

	query := `
		SELECT az.id, az.name, az.description, az.security_level, az.building, az.floor,
			   az.max_occupancy,
			   (SELECT COUNT(*) FROM access_logs al WHERE al.zone_id = az.id AND al.direction = 'entry' AND al.status = 'granted')
			   - (SELECT COUNT(*) FROM access_logs al WHERE al.zone_id = az.id AND al.direction = 'exit' AND al.status = 'granted'),
			   az.readers, az.schedules, az.created_at, az.updated_at
		FROM access_zones az
		WHERE az.id = $1`

	zone, err := scanAccessZone(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrAccessZoneNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return zone, nil
}

func (r *Repo) GetAllAccessZones(ctx context.Context) ([]*access.AccessZone, error) {
	const op = "repo.GetAllAccessZones"

	query := `
		SELECT az.id, az.name, az.description, az.security_level, az.building, az.floor,
			   az.max_occupancy, 0,
			   az.readers, az.schedules, az.created_at, az.updated_at
		FROM access_zones az
		ORDER BY az.name`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var zones []*access.AccessZone
	for rows.Next() {
		zone, err := scanAccessZone(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		zones = append(zones, zone)
	}
	return zones, nil
}

func (r *Repo) UpdateAccessZone(ctx context.Context, id int64, req dto.UpdateAccessZoneRequest) error {
	const op = "repo.UpdateAccessZone"

	var setClauses []string
	var args []interface{}
	argIdx := 1

	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.SecurityLevel != nil {
		setClauses = append(setClauses, fmt.Sprintf("security_level = $%d", argIdx))
		args = append(args, *req.SecurityLevel)
		argIdx++
	}
	if req.Building != nil {
		setClauses = append(setClauses, fmt.Sprintf("building = $%d", argIdx))
		args = append(args, *req.Building)
		argIdx++
	}
	if req.Floor != nil {
		setClauses = append(setClauses, fmt.Sprintf("floor = $%d", argIdx))
		args = append(args, *req.Floor)
		argIdx++
	}
	if req.MaxOccupancy != nil {
		setClauses = append(setClauses, fmt.Sprintf("max_occupancy = $%d", argIdx))
		args = append(args, *req.MaxOccupancy)
		argIdx++
	}
	if req.Readers != nil {
		setClauses = append(setClauses, fmt.Sprintf("readers = $%d", argIdx))
		args = append(args, *req.Readers)
		argIdx++
	}
	if req.Schedules != nil {
		setClauses = append(setClauses, fmt.Sprintf("schedules = $%d", argIdx))
		args = append(args, *req.Schedules)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE access_zones SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if translated := r.translator.Translate(err, op); translated != nil {
			return translated
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrAccessZoneNotFound
	}
	return nil
}

// ==================== Access Logs ====================

func (r *Repo) GetAccessLogs(ctx context.Context, filters dto.AccessLogFilters) ([]*access.AccessLog, error) {
	const op = "repo.GetAccessLogs"

	query := `
		SELECT al.id, al.employee_id, COALESCE(c.name, ''), COALESCE(ac.card_number, ''),
			   al.zone_id, COALESCE(az.name, ''),
			   al.reader_id, al.direction, al.timestamp, al.status, al.denial_reason
		FROM access_logs al
		LEFT JOIN contacts c ON al.employee_id = c.id
		LEFT JOIN access_cards ac ON al.card_id = ac.id
		LEFT JOIN access_zones az ON al.zone_id = az.id`

	var conditions []string
	var args []interface{}
	argIdx := 1

	if filters.EmployeeID != nil {
		conditions = append(conditions, fmt.Sprintf("al.employee_id = $%d", argIdx))
		args = append(args, *filters.EmployeeID)
		argIdx++
	}
	if filters.ZoneID != nil {
		conditions = append(conditions, fmt.Sprintf("al.zone_id = $%d", argIdx))
		args = append(args, *filters.ZoneID)
		argIdx++
	}
	if filters.Direction != nil {
		conditions = append(conditions, fmt.Sprintf("al.direction = $%d", argIdx))
		args = append(args, *filters.Direction)
		argIdx++
	}
	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("al.status = $%d", argIdx))
		args = append(args, *filters.Status)
		argIdx++
	}
	if filters.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("al.timestamp >= $%d", argIdx))
		args = append(args, *filters.DateFrom)
		argIdx++
	}
	if filters.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("al.timestamp <= $%d", argIdx))
		args = append(args, *filters.DateTo)
		argIdx++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY al.timestamp DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var logs []*access.AccessLog
	for rows.Next() {
		l, err := scanAccessLog(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// ==================== Access Requests ====================

func (r *Repo) CreateAccessRequest(ctx context.Context, employeeID int64, req dto.CreateAccessRequestReq) (int64, error) {
	const op = "repo.CreateAccessRequest"

	query := `
		INSERT INTO access_requests (employee_id, zone_id, reason)
		VALUES ($1, $2, $3)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query, employeeID, req.ZoneID, req.Reason).Scan(&id)
	if err != nil {
		if translated := r.translator.Translate(err, op); translated != nil {
			return 0, translated
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

func (r *Repo) GetAllAccessRequests(ctx context.Context, employeeID *int64) ([]*access.AccessRequest, error) {
	const op = "repo.GetAllAccessRequests"

	query := `
		SELECT ar.id, ar.employee_id, COALESCE(c.name, ''),
			   ar.zone_id, az.name,
			   ar.reason, ar.status, ar.approved_by, ar.rejection_reason,
			   ar.created_at, ar.updated_at
		FROM access_requests ar
		LEFT JOIN contacts c ON ar.employee_id = c.id
		LEFT JOIN access_zones az ON ar.zone_id = az.id`

	var args []interface{}
	if employeeID != nil {
		query += " WHERE ar.employee_id = $1"
		args = append(args, *employeeID)
	}
	query += " ORDER BY ar.created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var requests []*access.AccessRequest
	for rows.Next() {
		req, err := scanAccessRequest(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		requests = append(requests, req)
	}
	return requests, nil
}

func (r *Repo) GetAccessRequestByID(ctx context.Context, id int64) (*access.AccessRequest, error) {
	const op = "repo.GetAccessRequestByID"

	query := `
		SELECT ar.id, ar.employee_id, COALESCE(c.name, ''),
			   ar.zone_id, az.name,
			   ar.reason, ar.status, ar.approved_by, ar.rejection_reason,
			   ar.created_at, ar.updated_at
		FROM access_requests ar
		LEFT JOIN contacts c ON ar.employee_id = c.id
		LEFT JOIN access_zones az ON ar.zone_id = az.id
		WHERE ar.id = $1`

	req, err := scanAccessRequest(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrAccessRequestNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return req, nil
}

func (r *Repo) UpdateAccessRequestStatus(ctx context.Context, id int64, status string, approvedBy *int64, rejectionReason *string) error {
	const op = "repo.UpdateAccessRequestStatus"

	query := `UPDATE access_requests SET status = $1, approved_by = $2, rejection_reason = $3 WHERE id = $4`
	result, err := r.db.ExecContext(ctx, query, status, approvedBy, rejectionReason, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrAccessRequestNotFound
	}
	return nil
}

// ==================== Scanners ====================

func scanAccessCard(s scannable) (*access.AccessCard, error) {
	var ac access.AccessCard
	var zones []byte
	err := s.Scan(
		&ac.ID, &ac.EmployeeID, &ac.EmployeeName,
		&ac.CardNumber, &ac.Status, &ac.IssuedDate, &ac.ExpiryDate,
		&zones, &ac.AccessLevel, &ac.CreatedAt, &ac.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	ac.AccessZones = json.RawMessage(zones)
	return &ac, nil
}

func scanAccessZone(s scannable) (*access.AccessZone, error) {
	var az access.AccessZone
	var readers, schedules []byte
	err := s.Scan(
		&az.ID, &az.Name, &az.Description, &az.SecurityLevel, &az.Building, &az.Floor,
		&az.MaxOccupancy, &az.CurrentOccupancy,
		&readers, &schedules, &az.CreatedAt, &az.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	az.Readers = json.RawMessage(readers)
	az.Schedules = json.RawMessage(schedules)
	if az.CurrentOccupancy < 0 {
		az.CurrentOccupancy = 0
	}
	return &az, nil
}

func scanAccessLog(s scannable) (*access.AccessLog, error) {
	var al access.AccessLog
	err := s.Scan(
		&al.ID, &al.EmployeeID, &al.EmployeeName, &al.CardNumber,
		&al.ZoneID, &al.ZoneName,
		&al.ReaderID, &al.Direction, &al.Timestamp, &al.Status, &al.DenialReason,
	)
	if err != nil {
		return nil, err
	}
	return &al, nil
}

func scanAccessRequest(s scannable) (*access.AccessRequest, error) {
	var ar access.AccessRequest
	err := s.Scan(
		&ar.ID, &ar.EmployeeID, &ar.EmployeeName,
		&ar.ZoneID, &ar.ZoneName,
		&ar.Reason, &ar.Status, &ar.ApprovedBy, &ar.RejectionReason,
		&ar.CreatedAt, &ar.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &ar, nil
}
