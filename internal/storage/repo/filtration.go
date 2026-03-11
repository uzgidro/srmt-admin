package repo

import (
	"context"
	"database/sql"
	"fmt"
	"srmt-admin/internal/lib/model/filtration"
	"srmt-admin/internal/storage"
	"strings"
)

// --- Filtration Location CRUD ---

func (r *Repo) CreateFiltrationLocation(ctx context.Context, req filtration.CreateLocationRequest, userID int64) (int64, error) {
	const op = "storage.repo.Filtration.CreateFiltrationLocation"

	const query = `
		INSERT INTO filtration_locations (organization_id, name, norm, sort_order, created_by_user_id, updated_by_user_id, created_at, updated_at)
		VALUES ($1, $2, $3, COALESCE($4, 0), $5, $5, NOW(), NOW())
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query, req.OrganizationID, req.Name, req.Norm, req.SortOrder, userID).Scan(&id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return 0, translatedErr
		}
		return 0, fmt.Errorf("%s: failed to insert filtration location: %w", op, err)
	}

	return id, nil
}

func (r *Repo) GetFiltrationLocationsByOrg(ctx context.Context, orgID int64) ([]filtration.Location, error) {
	const op = "storage.repo.Filtration.GetFiltrationLocationsByOrg"

	const query = `
		SELECT id, organization_id, name, norm, sort_order, created_at, updated_at
		FROM filtration_locations
		WHERE organization_id = $1
		ORDER BY sort_order, id`

	rows, err := r.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query locations: %w", op, err)
	}
	defer rows.Close()

	var locations []filtration.Location
	for rows.Next() {
		var loc filtration.Location
		var norm sql.NullFloat64
		if err := rows.Scan(
			&loc.ID, &loc.OrganizationID, &loc.Name, &norm,
			&loc.SortOrder, &loc.CreatedAt, &loc.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("%s: failed to scan location: %w", op, err)
		}
		if norm.Valid {
			loc.Norm = &norm.Float64
		}
		locations = append(locations, loc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if locations == nil {
		locations = make([]filtration.Location, 0)
	}

	return locations, nil
}

func (r *Repo) UpdateFiltrationLocation(ctx context.Context, id int64, req filtration.UpdateLocationRequest, userID int64) error {
	const op = "storage.repo.Filtration.UpdateFiltrationLocation"

	var updates []string
	var args []interface{}
	argID := 1

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argID))
		args = append(args, *req.Name)
		argID++
	}
	if req.Norm != nil {
		updates = append(updates, fmt.Sprintf("norm = $%d", argID))
		args = append(args, *req.Norm)
		argID++
	}
	if req.SortOrder != nil {
		updates = append(updates, fmt.Sprintf("sort_order = $%d", argID))
		args = append(args, *req.SortOrder)
		argID++
	}

	if len(updates) == 0 {
		return nil
	}

	updates = append(updates, "updated_at = NOW()")
	updates = append(updates, fmt.Sprintf("updated_by_user_id = $%d", argID))
	args = append(args, userID)
	argID++

	query := fmt.Sprintf("UPDATE filtration_locations SET %s WHERE id = $%d",
		strings.Join(updates, ", "), argID)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to update filtration location: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

func (r *Repo) DeleteFiltrationLocation(ctx context.Context, id int64) error {
	const op = "storage.repo.Filtration.DeleteFiltrationLocation"

	res, err := r.db.ExecContext(ctx, "DELETE FROM filtration_locations WHERE id = $1", id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to delete filtration location: %w", op, err)
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

// --- Piezometer CRUD ---

func (r *Repo) CreatePiezometer(ctx context.Context, req filtration.CreatePiezometerRequest, userID int64) (int64, error) {
	const op = "storage.repo.Filtration.CreatePiezometer"

	const query = `
		INSERT INTO piezometers (organization_id, name, type, norm, sort_order, created_by_user_id, updated_by_user_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, COALESCE($5, 0), $6, $6, NOW(), NOW())
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query, req.OrganizationID, req.Name, req.Type, req.Norm, req.SortOrder, userID).Scan(&id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return 0, translatedErr
		}
		return 0, fmt.Errorf("%s: failed to insert piezometer: %w", op, err)
	}

	return id, nil
}

func (r *Repo) GetPiezometersByOrg(ctx context.Context, orgID int64) ([]filtration.Piezometer, error) {
	const op = "storage.repo.Filtration.GetPiezometersByOrg"

	const query = `
		SELECT id, organization_id, name, type, norm, sort_order, created_at, updated_at
		FROM piezometers
		WHERE organization_id = $1
		ORDER BY sort_order, id`

	rows, err := r.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query piezometers: %w", op, err)
	}
	defer rows.Close()

	var piezometers []filtration.Piezometer
	for rows.Next() {
		var p filtration.Piezometer
		var norm sql.NullFloat64
		if err := rows.Scan(
			&p.ID, &p.OrganizationID, &p.Name, &p.Type,
			&norm, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("%s: failed to scan piezometer: %w", op, err)
		}
		if norm.Valid {
			p.Norm = &norm.Float64
		}
		piezometers = append(piezometers, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if piezometers == nil {
		piezometers = make([]filtration.Piezometer, 0)
	}

	return piezometers, nil
}

func (r *Repo) GetPiezometerCountsByOrg(ctx context.Context, orgID int64) (filtration.PiezometerCounts, error) {
	const op = "storage.repo.Filtration.GetPiezometerCountsByOrg"

	const query = `
		SELECT type, COUNT(*)
		FROM piezometers
		WHERE organization_id = $1
		GROUP BY type`

	rows, err := r.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return filtration.PiezometerCounts{}, fmt.Errorf("%s: failed to query piezometer counts: %w", op, err)
	}
	defer rows.Close()

	var counts filtration.PiezometerCounts
	for rows.Next() {
		var pType filtration.PiezometerType
		var count int
		if err := rows.Scan(&pType, &count); err != nil {
			return filtration.PiezometerCounts{}, fmt.Errorf("%s: failed to scan count row: %w", op, err)
		}
		switch pType {
		case filtration.PiezometerTypePressure:
			counts.Pressure = count
		case filtration.PiezometerTypeNonPressure:
			counts.NonPressure = count
		}
	}

	if err := rows.Err(); err != nil {
		return filtration.PiezometerCounts{}, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	return counts, nil
}

func (r *Repo) UpdatePiezometer(ctx context.Context, id int64, req filtration.UpdatePiezometerRequest, userID int64) error {
	const op = "storage.repo.Filtration.UpdatePiezometer"

	var updates []string
	var args []interface{}
	argID := 1

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argID))
		args = append(args, *req.Name)
		argID++
	}
	if req.Type != nil {
		updates = append(updates, fmt.Sprintf("type = $%d", argID))
		args = append(args, *req.Type)
		argID++
	}
	if req.Norm != nil {
		updates = append(updates, fmt.Sprintf("norm = $%d", argID))
		args = append(args, *req.Norm)
		argID++
	}
	if req.SortOrder != nil {
		updates = append(updates, fmt.Sprintf("sort_order = $%d", argID))
		args = append(args, *req.SortOrder)
		argID++
	}

	if len(updates) == 0 {
		return nil
	}

	updates = append(updates, "updated_at = NOW()")
	updates = append(updates, fmt.Sprintf("updated_by_user_id = $%d", argID))
	args = append(args, userID)
	argID++

	query := fmt.Sprintf("UPDATE piezometers SET %s WHERE id = $%d",
		strings.Join(updates, ", "), argID)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to update piezometer: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

func (r *Repo) DeletePiezometer(ctx context.Context, id int64) error {
	const op = "storage.repo.Filtration.DeletePiezometer"

	res, err := r.db.ExecContext(ctx, "DELETE FROM piezometers WHERE id = $1", id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to delete piezometer: %w", op, err)
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

// --- Measurements ---

func (r *Repo) UpsertFiltrationMeasurements(ctx context.Context, date string, items []filtration.FiltrationMeasurementInput, userID int64) error {
	const op = "storage.repo.Filtration.UpsertFiltrationMeasurements"

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: begin tx: %w", op, err)
	}
	defer tx.Rollback()

	const query = `
		INSERT INTO filtration_measurements (location_id, date, flow_rate, created_by_user_id, updated_by_user_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4, NOW(), NOW())
		ON CONFLICT (location_id, date)
		DO UPDATE SET flow_rate = EXCLUDED.flow_rate, updated_by_user_id = EXCLUDED.updated_by_user_id, updated_at = NOW()`

	for _, item := range items {
		if _, err := tx.ExecContext(ctx, query, item.LocationID, date, item.FlowRate, userID); err != nil {
			if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
				return translatedErr
			}
			return fmt.Errorf("%s: upsert location_id=%d: %w", op, item.LocationID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: commit: %w", op, err)
	}

	return nil
}

func (r *Repo) GetFiltrationMeasurements(ctx context.Context, orgID int64, date string) ([]filtration.FiltrationMeasurement, error) {
	const op = "storage.repo.Filtration.GetFiltrationMeasurements"

	const query = `
		SELECT m.id, m.location_id, m.date::text, m.flow_rate
		FROM filtration_measurements m
		JOIN filtration_locations l ON l.id = m.location_id
		WHERE l.organization_id = $1 AND m.date = $2::date
		ORDER BY l.sort_order, l.id`

	rows, err := r.db.QueryContext(ctx, query, orgID, date)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query measurements: %w", op, err)
	}
	defer rows.Close()

	var measurements []filtration.FiltrationMeasurement
	for rows.Next() {
		var m filtration.FiltrationMeasurement
		var flowRate sql.NullFloat64
		if err := rows.Scan(&m.ID, &m.LocationID, &m.Date, &flowRate); err != nil {
			return nil, fmt.Errorf("%s: failed to scan measurement: %w", op, err)
		}
		if flowRate.Valid {
			m.FlowRate = &flowRate.Float64
		}
		measurements = append(measurements, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if measurements == nil {
		measurements = make([]filtration.FiltrationMeasurement, 0)
	}

	return measurements, nil
}

func (r *Repo) UpsertPiezometerMeasurements(ctx context.Context, date string, items []filtration.PiezometerMeasurementInput, userID int64) error {
	const op = "storage.repo.Filtration.UpsertPiezometerMeasurements"

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: begin tx: %w", op, err)
	}
	defer tx.Rollback()

	const query = `
		INSERT INTO piezometer_measurements (piezometer_id, date, level, created_by_user_id, updated_by_user_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4, NOW(), NOW())
		ON CONFLICT (piezometer_id, date)
		DO UPDATE SET level = EXCLUDED.level, updated_by_user_id = EXCLUDED.updated_by_user_id, updated_at = NOW()`

	for _, item := range items {
		if _, err := tx.ExecContext(ctx, query, item.PiezometerID, date, item.Level, userID); err != nil {
			if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
				return translatedErr
			}
			return fmt.Errorf("%s: upsert piezometer_id=%d: %w", op, item.PiezometerID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: commit: %w", op, err)
	}

	return nil
}

func (r *Repo) GetPiezometerMeasurements(ctx context.Context, orgID int64, date string) ([]filtration.PiezometerMeasurement, error) {
	const op = "storage.repo.Filtration.GetPiezometerMeasurements"

	const query = `
		SELECT m.id, m.piezometer_id, m.date::text, m.level
		FROM piezometer_measurements m
		JOIN piezometers p ON p.id = m.piezometer_id
		WHERE p.organization_id = $1 AND m.date = $2::date
		ORDER BY p.sort_order, p.id`

	rows, err := r.db.QueryContext(ctx, query, orgID, date)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query measurements: %w", op, err)
	}
	defer rows.Close()

	var measurements []filtration.PiezometerMeasurement
	for rows.Next() {
		var m filtration.PiezometerMeasurement
		var level sql.NullFloat64
		if err := rows.Scan(&m.ID, &m.PiezometerID, &m.Date, &level); err != nil {
			return nil, fmt.Errorf("%s: failed to scan measurement: %w", op, err)
		}
		if level.Valid {
			m.Level = &level.Float64
		}
		measurements = append(measurements, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if measurements == nil {
		measurements = make([]filtration.PiezometerMeasurement, 0)
	}

	return measurements, nil
}

// --- Summary ---

func (r *Repo) GetOrgFiltrationSummary(ctx context.Context, orgID int64, date string) (*filtration.OrgFiltrationSummary, error) {
	const op = "storage.repo.Filtration.GetOrgFiltrationSummary"

	// Get organization name
	var orgName string
	err := r.db.QueryRowContext(ctx, "SELECT name FROM organizations WHERE id = $1", orgID).Scan(&orgName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get organization: %w", op, err)
	}

	summary := &filtration.OrgFiltrationSummary{
		OrganizationID:   orgID,
		OrganizationName: orgName,
	}

	// Get locations with measurements
	const locQuery = `
		SELECT l.id, l.organization_id, l.name, l.norm, l.sort_order, l.created_at, l.updated_at,
		       m.flow_rate
		FROM filtration_locations l
		LEFT JOIN filtration_measurements m ON m.location_id = l.id AND m.date = $2::date
		WHERE l.organization_id = $1
		ORDER BY l.sort_order, l.id`

	locRows, err := r.db.QueryContext(ctx, locQuery, orgID, date)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query locations: %w", op, err)
	}
	defer locRows.Close()

	for locRows.Next() {
		var lr filtration.LocationReading
		var norm, flowRate sql.NullFloat64
		if err := locRows.Scan(
			&lr.ID, &lr.OrganizationID, &lr.Name, &norm,
			&lr.SortOrder, &lr.CreatedAt, &lr.UpdatedAt,
			&flowRate,
		); err != nil {
			return nil, fmt.Errorf("%s: failed to scan location reading: %w", op, err)
		}
		if norm.Valid {
			lr.Norm = &norm.Float64
		}
		if flowRate.Valid {
			lr.FlowRate = &flowRate.Float64
		}
		summary.Locations = append(summary.Locations, lr)
	}

	if err := locRows.Err(); err != nil {
		return nil, fmt.Errorf("%s: location rows iteration error: %w", op, err)
	}

	if summary.Locations == nil {
		summary.Locations = make([]filtration.LocationReading, 0)
	}

	// Get piezometers with measurements
	const piezoQuery = `
		SELECT p.id, p.organization_id, p.name, p.type, p.norm, p.sort_order, p.created_at, p.updated_at,
		       m.level
		FROM piezometers p
		LEFT JOIN piezometer_measurements m ON m.piezometer_id = p.id AND m.date = $2::date
		WHERE p.organization_id = $1
		ORDER BY p.sort_order, p.id`

	piezoRows, err := r.db.QueryContext(ctx, piezoQuery, orgID, date)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query piezometers: %w", op, err)
	}
	defer piezoRows.Close()

	for piezoRows.Next() {
		var pr filtration.PiezoReading
		var norm, level sql.NullFloat64
		if err := piezoRows.Scan(
			&pr.ID, &pr.OrganizationID, &pr.Name, &pr.Type,
			&norm, &pr.SortOrder, &pr.CreatedAt, &pr.UpdatedAt,
			&level,
		); err != nil {
			return nil, fmt.Errorf("%s: failed to scan piezometer reading: %w", op, err)
		}
		if norm.Valid {
			pr.Norm = &norm.Float64
		}
		if level.Valid {
			pr.Level = &level.Float64
		}
		summary.Piezometers = append(summary.Piezometers, pr)
	}

	if err := piezoRows.Err(); err != nil {
		return nil, fmt.Errorf("%s: piezometer rows iteration error: %w", op, err)
	}

	if summary.Piezometers == nil {
		summary.Piezometers = make([]filtration.PiezoReading, 0)
	}

	// Compute piezometer counts from already-fetched data
	for _, pr := range summary.Piezometers {
		switch pr.Type {
		case filtration.PiezometerTypePressure:
			summary.PiezoCounts.Pressure++
		case filtration.PiezometerTypeNonPressure:
			summary.PiezoCounts.NonPressure++
		}
	}

	return summary, nil
}
