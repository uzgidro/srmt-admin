package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/contact"
	"srmt-admin/internal/lib/model/event"
	"srmt-admin/internal/lib/model/event_status"
	"srmt-admin/internal/lib/model/event_type"
	"srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/model/organization"
	"srmt-admin/internal/lib/model/user"
	"srmt-admin/internal/storage"
	"strings"
)

// --- 1. CREATE ---

// AddEvent creates a new event in the database
func (r *Repo) AddEvent(ctx context.Context, req dto.AddEventRequest) (int64, error) {
	const op = "storage.repo.AddEvent"

	const query = `
		INSERT INTO events (
			name, description, location, event_date,
			responsible_contact_id, event_status_id, event_type_id,
			organization_id, created_by_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.Name, req.Description, req.Location, req.EventDate,
		req.ResponsibleContactID, req.EventStatusID, req.EventTypeID,
		req.OrganizationID, req.CreatedByID,
	).Scan(&id)

	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return 0, translatedErr
		}
		return 0, fmt.Errorf("%s: failed to insert event: %w", op, err)
	}

	// Link files if provided
	if len(req.FileIDs) > 0 {
		if err := r.LinkEventFiles(ctx, id, req.FileIDs); err != nil {
			return 0, fmt.Errorf("%s: failed to link files: %w", op, err)
		}
	}

	return id, nil
}

// LinkEventFiles links multiple files to an event
func (r *Repo) LinkEventFiles(ctx context.Context, eventID int64, fileIDs []int64) error {
	const op = "storage.repo.LinkEventFiles"

	if len(fileIDs) == 0 {
		return nil
	}

	// Build bulk insert query
	var valueStrings []string
	var valueArgs []interface{}
	argID := 1

	for _, fileID := range fileIDs {
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d)", argID, argID+1))
		valueArgs = append(valueArgs, eventID, fileID)
		argID += 2
	}

	query := fmt.Sprintf(
		"INSERT INTO event_file_links (event_id, file_id) VALUES %s ON CONFLICT DO NOTHING",
		strings.Join(valueStrings, ", "),
	)

	_, err := r.db.ExecContext(ctx, query, valueArgs...)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to insert file links: %w", op, err)
	}

	return nil
}

// UnlinkEventFiles removes all file links for an event
func (r *Repo) UnlinkEventFiles(ctx context.Context, eventID int64) error {
	const op = "storage.repo.UnlinkEventFiles"

	_, err := r.db.ExecContext(ctx, "DELETE FROM event_file_links WHERE event_id = $1", eventID)
	if err != nil {
		return fmt.Errorf("%s: failed to delete file links: %w", op, err)
	}

	return nil
}

// --- 2. READ (Helper functions) ---

const (
	selectEventFields = `
		SELECT
			e.id, e.name, e.description, e.location, e.event_date,
			e.responsible_contact_id, e.event_status_id, e.event_type_id,
			e.organization_id, e.created_by_user_id, e.updated_by_user_id,
			e.created_at, e.updated_at,

			es.id as status_id, es.name as status_name, es.description as status_desc,

			et.id as type_id, et.name as type_name, et.description as type_desc,

			rc.id as contact_id, rc.fio as contact_fio, rc.phone as contact_phone,

			o.id as org_id, o.name as org_name,

			cu.id as created_user_id, cu_contact.fio as created_username,

			uu.id as updated_user_id, uu_contact.fio as updated_username
	`
	fromEventJoins = `
		FROM events e
		INNER JOIN event_status es ON e.event_status_id = es.id
		INNER JOIN event_type et ON e.event_type_id = et.id
		INNER JOIN contacts rc ON e.responsible_contact_id = rc.id
		LEFT JOIN organizations o ON e.organization_id = o.id
		LEFT JOIN users cu ON e.created_by_user_id = cu.id
		LEFT JOIN contacts cu_contact ON cu.contact_id = cu_contact.id
		LEFT JOIN users uu ON e.updated_by_user_id = uu.id
		LEFT JOIN contacts uu_contact ON uu.contact_id = uu_contact.id
	`
)

// scanEventRow scans an enriched event model from a database row
func scanEventRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*event.Model, error) {
	var e event.Model
	var (
		description, location sql.NullString
		orgID                 sql.NullInt64
		updatedByUserID       sql.NullInt64
		updatedAt             sql.NullTime
		orgName               sql.NullString
		statusDesc, typeDesc  sql.NullString
		contactPhone          sql.NullString
		createdUserID         sql.NullInt64
		createdUsername       sql.NullString
		updatedUserID         sql.NullInt64
		updatedUsername       sql.NullString
	)

	// Status and type IDs and names
	var statusID, typeID int
	var statusName, typeName string

	// Contact info
	var contactID int64
	var contactFIO string

	err := scanner.Scan(
		&e.ID, &e.Name, &description, &location, &e.EventDate,
		&e.ResponsibleContactID, &e.EventStatusID, &e.EventTypeID,
		&orgID, &e.CreatedByID, &updatedByUserID,
		&e.CreatedAt, &updatedAt,
		&statusID, &statusName, &statusDesc,
		&typeID, &typeName, &typeDesc,
		&contactID, &contactFIO, &contactPhone,
		&orgID, &orgName,
		&createdUserID, &createdUsername,
		&updatedUserID, &updatedUsername,
	)
	if err != nil {
		return nil, err
	}

	// Nullable fields
	if description.Valid {
		e.Description = &description.String
	}
	if location.Valid {
		e.Location = &location.String
	}
	if orgID.Valid {
		e.OrganizationID = &orgID.Int64
	}
	if updatedByUserID.Valid {
		e.UpdatedByID = &updatedByUserID.Int64
	}
	if updatedAt.Valid {
		e.UpdatedAt = &updatedAt.Time
	}

	// Nested structures
	e.EventStatus = &event_status.Model{
		ID:          statusID,
		Name:        statusName,
		Description: statusDesc.String,
	}

	e.EventType = &event_type.Model{
		ID:          typeID,
		Name:        typeName,
		Description: typeDesc.String,
	}

	e.ResponsibleContact = &contact.Model{
		ID:  contactID,
		FIO: contactFIO,
	}
	if contactPhone.Valid {
		e.ResponsibleContact.Phone = &contactPhone.String
	}

	if orgID.Valid && orgName.Valid {
		e.Organization = &organization.Model{
			ID:   orgID.Int64,
			Name: orgName.String,
		}
	}

	if createdUserID.Valid && createdUsername.Valid {
		e.CreatedBy = &user.Model{
			ID:    createdUserID.Int64,
			Login: createdUsername.String,
		}
	}

	if updatedUserID.Valid && updatedUsername.Valid {
		e.UpdatedBy = &user.Model{
			ID:    updatedUserID.Int64,
			Login: updatedUsername.String,
		}
	}

	return &e, nil
}

// loadEventFiles loads all files linked to a specific event
func (r *Repo) loadEventFiles(ctx context.Context, eventID int64) ([]file.Model, error) {
	const query = `
		SELECT f.id, f.file_name, f.object_key, f.category_id, f.mime_type, f.size_bytes, f.created_at
		FROM files f
		INNER JOIN event_file_links efl ON f.id = efl.file_id
		WHERE efl.event_id = $1
		ORDER BY f.created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []file.Model
	for rows.Next() {
		var f file.Model
		var mimeType sql.NullString
		var sizeBytes sql.NullInt64

		err := rows.Scan(
			&f.ID, &f.FileName, &f.ObjectKey, &f.CategoryID,
			&mimeType, &sizeBytes, &f.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if mimeType.Valid {
			f.MimeType = mimeType.String
		}
		if sizeBytes.Valid {
			f.SizeBytes = sizeBytes.Int64
		}

		files = append(files, f)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	if files == nil {
		files = make([]file.Model, 0)
	}

	return files, nil
}

// GetEventByID retrieves a single event by its ID with all related data
func (r *Repo) GetEventByID(ctx context.Context, id int64) (*event.Model, error) {
	const op = "storage.repo.GetEventByID"

	query := selectEventFields + fromEventJoins + " WHERE e.id = $1"

	row := r.db.QueryRowContext(ctx, query, id)
	e, err := scanEventRow(row)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to scan event: %w", op, err)
	}

	// Load linked files
	files, err := r.loadEventFiles(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to load event files: %w", op, err)
	}
	e.Files = files

	return e, nil
}

// GetAllEvents retrieves events with optional filters
func (r *Repo) GetAllEvents(ctx context.Context, filters dto.GetAllEventsFilters) ([]*event.Model, error) {
	const op = "storage.repo.GetAllEvents"

	var query strings.Builder
	query.WriteString(selectEventFields)
	query.WriteString(fromEventJoins)

	var whereClauses []string
	var args []interface{}
	argID := 1

	// Filter by event status IDs
	if len(filters.EventStatusIDs) > 0 {
		var placeholders []string
		for _, statusID := range filters.EventStatusIDs {
			placeholders = append(placeholders, fmt.Sprintf("$%d", argID))
			args = append(args, statusID)
			argID++
		}
		whereClauses = append(whereClauses, fmt.Sprintf("e.event_status_id IN (%s)", strings.Join(placeholders, ", ")))
	}

	// Filter by event type IDs
	if len(filters.EventTypeIDs) > 0 {
		var placeholders []string
		for _, typeID := range filters.EventTypeIDs {
			placeholders = append(placeholders, fmt.Sprintf("$%d", argID))
			args = append(args, typeID)
			argID++
		}
		whereClauses = append(whereClauses, fmt.Sprintf("e.event_type_id IN (%s)", strings.Join(placeholders, ", ")))
	}

	// Filter by date range
	if filters.StartDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("e.event_date >= $%d", argID))
		args = append(args, *filters.StartDate)
		argID++
	}
	if filters.EndDate != nil {
		// Add one day to end_date to make it inclusive
		endDate := filters.EndDate.AddDate(0, 0, 1)
		whereClauses = append(whereClauses, fmt.Sprintf("e.event_date < $%d", argID))
		args = append(args, endDate)
		argID++
	}

	// Filter by organization
	if filters.OrganizationID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("e.organization_id = $%d", argID))
		args = append(args, *filters.OrganizationID)
		argID++
	}

	if len(whereClauses) > 0 {
		query.WriteString(" WHERE " + strings.Join(whereClauses, " AND "))
	}

	// Order by event_date descending (most recent first)
	query.WriteString(" ORDER BY e.event_date DESC")

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query events: %w", op, err)
	}
	defer rows.Close()

	var events []*event.Model
	for rows.Next() {
		e, err := scanEventRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan event row: %w", op, err)
		}
		events = append(events, e)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	// Load files for each event (could be optimized with a single query, but this is clearer)
	for _, e := range events {
		files, err := r.loadEventFiles(ctx, e.ID)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to load event files for event %d: %w", op, e.ID, err)
		}
		e.Files = files
	}

	if events == nil {
		events = make([]*event.Model, 0)
	}

	return events, nil
}

// --- 3. UPDATE ---

// EditEvent updates an existing event
func (r *Repo) EditEvent(ctx context.Context, eventID int64, req dto.EditEventRequest) error {
	const op = "storage.repo.EditEvent"

	// Build dynamic UPDATE query
	var updates []string
	var args []interface{}
	argID := 1

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argID))
		args = append(args, *req.Name)
		argID++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argID))
		args = append(args, *req.Description)
		argID++
	}
	if req.Location != nil {
		updates = append(updates, fmt.Sprintf("location = $%d", argID))
		args = append(args, *req.Location)
		argID++
	}
	if req.EventDate != nil {
		updates = append(updates, fmt.Sprintf("event_date = $%d", argID))
		args = append(args, *req.EventDate)
		argID++
	}
	if req.ResponsibleContactID != nil {
		updates = append(updates, fmt.Sprintf("responsible_contact_id = $%d", argID))
		args = append(args, *req.ResponsibleContactID)
		argID++
	}
	if req.EventStatusID != nil {
		updates = append(updates, fmt.Sprintf("event_status_id = $%d", argID))
		args = append(args, *req.EventStatusID)
		argID++
	}
	if req.EventTypeID != nil {
		updates = append(updates, fmt.Sprintf("event_type_id = $%d", argID))
		args = append(args, *req.EventTypeID)
		argID++
	}
	if req.OrganizationID != nil {
		updates = append(updates, fmt.Sprintf("organization_id = $%d", argID))
		args = append(args, *req.OrganizationID)
		argID++
	}

	// Always set updated_by and updated_at
	updates = append(updates, fmt.Sprintf("updated_by_user_id = $%d", argID))
	args = append(args, req.UpdatedByID)
	argID++

	updates = append(updates, "updated_at = NOW()")

	if len(updates) == 1 && len(req.FileIDs) == 0 {
		// Only updated_by was set and no file changes - nothing substantial to update
		return nil
	}

	// Execute update query
	query := fmt.Sprintf("UPDATE events SET %s WHERE id = $%d",
		strings.Join(updates, ", "),
		argID,
	)
	args = append(args, eventID)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to update event: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	// Update file links if provided
	if len(req.FileIDs) > 0 {
		// Remove old links
		if err := r.UnlinkEventFiles(ctx, eventID); err != nil {
			return fmt.Errorf("%s: failed to unlink old files: %w", op, err)
		}
		// Add new links
		if err := r.LinkEventFiles(ctx, eventID, req.FileIDs); err != nil {
			return fmt.Errorf("%s: failed to link new files: %w", op, err)
		}
	}

	return nil
}

// --- 4. DELETE ---

// DeleteEvent deletes an event by its ID
func (r *Repo) DeleteEvent(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteEvent"

	// File links will be automatically deleted due to CASCADE
	res, err := r.db.ExecContext(ctx, "DELETE FROM events WHERE id = $1", id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to delete event: %w", op, err)
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

// GetEventStatuses retrieves all available event statuses
func (r *Repo) GetEventStatuses(ctx context.Context) ([]event_status.Model, error) {
	const op = "storage.repo.GetEventStatuses"

	rows, err := r.db.QueryContext(ctx, "SELECT id, name, description FROM event_status ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query event statuses: %w", op, err)
	}
	defer rows.Close()

	var statuses []event_status.Model
	for rows.Next() {
		var s event_status.Model
		var desc sql.NullString

		if err := rows.Scan(&s.ID, &s.Name, &desc); err != nil {
			return nil, fmt.Errorf("%s: failed to scan status: %w", op, err)
		}

		if desc.Valid {
			s.Description = desc.String
		}

		statuses = append(statuses, s)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if statuses == nil {
		statuses = make([]event_status.Model, 0)
	}

	return statuses, nil
}

// GetEventTypes retrieves all available event types
func (r *Repo) GetEventTypes(ctx context.Context) ([]event_type.Model, error) {
	const op = "storage.repo.GetEventTypes"

	rows, err := r.db.QueryContext(ctx, "SELECT id, name, description FROM event_type ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query event types: %w", op, err)
	}
	defer rows.Close()

	var types []event_type.Model
	for rows.Next() {
		var t event_type.Model
		var desc sql.NullString

		if err := rows.Scan(&t.ID, &t.Name, &desc); err != nil {
			return nil, fmt.Errorf("%s: failed to scan type: %w", op, err)
		}

		if desc.Valid {
			t.Description = desc.String
		}

		types = append(types, t)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if types == nil {
		types = make([]event_type.Model, 0)
	}

	return types, nil
}

// GetAllEventsShort retrieves events in compact format with only id, name, type, and status
func (r *Repo) GetAllEventsShort(ctx context.Context, filters dto.GetAllEventsFilters) ([]dto.EventShort, error) {
	const op = "storage.repo.GetAllEventsShort"

	var query strings.Builder
	query.WriteString(`
		SELECT
			e.id, e.name, e.event_date,
			es.id as status_id, es.name as status_name,
			et.id as type_id, et.name as type_name
		FROM events e
		INNER JOIN event_status es ON e.event_status_id = es.id
		INNER JOIN event_type et ON e.event_type_id = et.id
	`)

	var whereClauses []string
	var args []interface{}
	argID := 1

	// Filter by event status IDs
	if len(filters.EventStatusIDs) > 0 {
		var placeholders []string
		for _, statusID := range filters.EventStatusIDs {
			placeholders = append(placeholders, fmt.Sprintf("$%d", argID))
			args = append(args, statusID)
			argID++
		}
		whereClauses = append(whereClauses, fmt.Sprintf("e.event_status_id IN (%s)", strings.Join(placeholders, ", ")))
	}

	// Filter by event type IDs
	if len(filters.EventTypeIDs) > 0 {
		var placeholders []string
		for _, typeID := range filters.EventTypeIDs {
			placeholders = append(placeholders, fmt.Sprintf("$%d", argID))
			args = append(args, typeID)
			argID++
		}
		whereClauses = append(whereClauses, fmt.Sprintf("e.event_type_id IN (%s)", strings.Join(placeholders, ", ")))
	}

	// Filter by date range
	if filters.StartDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("e.event_date >= $%d", argID))
		args = append(args, *filters.StartDate)
		argID++
	}
	if filters.EndDate != nil {
		endDate := filters.EndDate.AddDate(0, 0, 1)
		whereClauses = append(whereClauses, fmt.Sprintf("e.event_date < $%d", argID))
		args = append(args, endDate)
		argID++
	}

	// Filter by organization
	if filters.OrganizationID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("e.organization_id = $%d", argID))
		args = append(args, *filters.OrganizationID)
		argID++
	}

	if len(whereClauses) > 0 {
		query.WriteString(" WHERE " + strings.Join(whereClauses, " AND "))
	}

	query.WriteString(" ORDER BY e.event_date DESC")

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query events: %w", op, err)
	}
	defer rows.Close()

	var events []dto.EventShort
	for rows.Next() {
		var e dto.EventShort
		var statusID, typeID int
		var statusName, typeName string

		err := rows.Scan(
			&e.ID, &e.Name, &e.EventDate,
			&statusID, &statusName,
			&typeID, &typeName,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan event row: %w", op, err)
		}

		e.Status.ID = statusID
		e.Status.Name = statusName
		e.Type.ID = typeID
		e.Type.Name = typeName

		events = append(events, e)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if events == nil {
		events = make([]dto.EventShort, 0)
	}

	return events, nil
}
