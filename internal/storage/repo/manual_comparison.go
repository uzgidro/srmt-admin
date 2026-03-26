package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	manualcomparison "srmt-admin/internal/lib/model/manual-comparison"
	"srmt-admin/internal/storage"
	"strings"

	"github.com/lib/pq"
)

// --- Manual Comparison CRUD ---

func (r *Repo) UpsertManualComparison(ctx context.Context, req manualcomparison.UpsertRequest) error {
	const op = "storage.repo.ManualComparison.UpsertManualComparison"

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: begin tx: %w", op, err)
	}
	defer tx.Rollback()

	// Upsert historical date labels
	const datesQuery = `
		INSERT INTO manual_comparison_dates (organization_id, date, historical_filter_date, historical_piezo_date, created_by_user_id, updated_by_user_id, created_at, updated_at)
		VALUES ($1, $2::date, $3, $4, $5, $5, NOW(), NOW())
		ON CONFLICT (organization_id, date)
		DO UPDATE SET historical_filter_date = EXCLUDED.historical_filter_date,
		              historical_piezo_date = EXCLUDED.historical_piezo_date,
		              updated_by_user_id = EXCLUDED.updated_by_user_id,
		              updated_at = NOW()`

	if _, err := tx.ExecContext(ctx, datesQuery, req.OrganizationID, req.Date, req.HistoricalFilterDate, req.HistoricalPiezoDate, req.UserID); err != nil {
		return fmt.Errorf("%s: upsert dates: %w", op, err)
	}

	// Batch upsert filter measurements
	if len(req.Filters) > 0 {
		if err := r.batchUpsertFilters(ctx, tx, op, req); err != nil {
			return err
		}
	}

	// Batch upsert piezo measurements
	if len(req.Piezos) > 0 {
		if err := r.batchUpsertPiezos(ctx, tx, op, req); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: commit: %w", op, err)
	}
	return nil
}

func (r *Repo) batchUpsertFilters(ctx context.Context, tx *sql.Tx, op string, req manualcomparison.UpsertRequest) error {
	var sb strings.Builder
	sb.WriteString(`INSERT INTO manual_comparison_filter (organization_id, location_id, date, flow_rate, historical_flow_rate, created_by_user_id, updated_by_user_id, created_at, updated_at) VALUES `)

	args := make([]interface{}, 0, len(req.Filters)*6)
	for i, f := range req.Filters {
		if i > 0 {
			sb.WriteString(", ")
		}
		base := i * 6
		fmt.Fprintf(&sb, "($%d, $%d, $%d::date, $%d, $%d, $%d, $%d, NOW(), NOW())",
			base+1, base+2, base+3, base+4, base+5, base+6, base+6)
		args = append(args, req.OrganizationID, f.LocationID, req.Date, f.FlowRate, f.HistoricalFlowRate, req.UserID)
	}

	sb.WriteString(` ON CONFLICT (location_id, date)
		DO UPDATE SET flow_rate = EXCLUDED.flow_rate,
		              historical_flow_rate = EXCLUDED.historical_flow_rate,
		              updated_by_user_id = EXCLUDED.updated_by_user_id,
		              updated_at = NOW()`)

	if _, err := tx.ExecContext(ctx, sb.String(), args...); err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: batch upsert filters: %w", op, err)
	}
	return nil
}

func (r *Repo) batchUpsertPiezos(ctx context.Context, tx *sql.Tx, op string, req manualcomparison.UpsertRequest) error {
	var sb strings.Builder
	sb.WriteString(`INSERT INTO manual_comparison_piezo (organization_id, piezometer_id, date, level, anomaly, historical_level, created_by_user_id, updated_by_user_id, created_at, updated_at) VALUES `)

	args := make([]interface{}, 0, len(req.Piezos)*7)
	for i, p := range req.Piezos {
		if i > 0 {
			sb.WriteString(", ")
		}
		base := i * 7
		fmt.Fprintf(&sb, "($%d, $%d, $%d::date, $%d, COALESCE($%d, false), $%d, $%d, $%d, NOW(), NOW())",
			base+1, base+2, base+3, base+4, base+5, base+6, base+7, base+7)
		args = append(args, req.OrganizationID, p.PiezometerID, req.Date, p.Level, p.Anomaly, p.HistoricalLevel, req.UserID)
	}

	sb.WriteString(` ON CONFLICT (piezometer_id, date)
		DO UPDATE SET level = EXCLUDED.level,
		              anomaly = EXCLUDED.anomaly,
		              historical_level = EXCLUDED.historical_level,
		              updated_by_user_id = EXCLUDED.updated_by_user_id,
		              updated_at = NOW()`)

	if _, err := tx.ExecContext(ctx, sb.String(), args...); err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: batch upsert piezos: %w", op, err)
	}
	return nil
}

// GetManualComparison returns manual comparison data for a single org+date.
// Used by the GET /measurements handler for a single org.
func (r *Repo) GetManualComparison(ctx context.Context, orgID int64, date string) (*manualcomparison.OrgManualComparison, error) {
	const op = "storage.repo.ManualComparison.GetManualComparison"

	var orgName string
	err := r.db.QueryRowContext(ctx, "SELECT name FROM organizations WHERE id = $1", orgID).Scan(&orgName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: get org name: %w", op, err)
	}

	result := &manualcomparison.OrgManualComparison{
		OrganizationID:   orgID,
		OrganizationName: orgName,
		Date:             date,
	}

	err = r.db.QueryRowContext(ctx,
		"SELECT historical_filter_date, historical_piezo_date FROM manual_comparison_dates WHERE organization_id = $1 AND date = $2::date",
		orgID, date,
	).Scan(&result.HistoricalFilterDate, &result.HistoricalPiezoDate)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%s: get dates: %w", op, err)
	}

	if err := r.scanManualFilters(ctx, result, orgID, date, op); err != nil {
		return nil, err
	}
	if err := r.scanManualPiezos(ctx, result, orgID, date, op); err != nil {
		return nil, err
	}

	return result, nil
}

// GetManualComparisonBatch returns manual comparison data for multiple orgs in 4 queries (not N*3).
func (r *Repo) GetManualComparisonBatch(ctx context.Context, orgIDs []int64, date string) (map[int64]*manualcomparison.OrgManualComparison, error) {
	const op = "storage.repo.ManualComparison.GetManualComparisonBatch"

	result := make(map[int64]*manualcomparison.OrgManualComparison, len(orgIDs))
	if len(orgIDs) == 0 {
		return result, nil
	}

	// Query 1: org names
	nameRows, err := r.db.QueryContext(ctx, "SELECT id, name FROM organizations WHERE id = ANY($1)", pq.Array(orgIDs))
	if err != nil {
		return nil, fmt.Errorf("%s: query org names: %w", op, err)
	}
	defer nameRows.Close()

	for nameRows.Next() {
		var id int64
		var name string
		if err := nameRows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("%s: scan org name: %w", op, err)
		}
		result[id] = &manualcomparison.OrgManualComparison{
			OrganizationID:   id,
			OrganizationName: name,
			Date:             date,
			Filters:          make([]manualcomparison.FilterReading, 0),
			Piezometers:      make([]manualcomparison.PiezoReading, 0),
		}
	}
	if err := nameRows.Err(); err != nil {
		return nil, fmt.Errorf("%s: org name rows: %w", op, err)
	}
	nameRows.Close()

	// Query 2: historical date labels
	dateRows, err := r.db.QueryContext(ctx,
		"SELECT organization_id, historical_filter_date, historical_piezo_date FROM manual_comparison_dates WHERE organization_id = ANY($1) AND date = $2::date",
		pq.Array(orgIDs), date,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: query dates: %w", op, err)
	}
	defer dateRows.Close()

	for dateRows.Next() {
		var orgID int64
		var filterDate, piezoDate string
		if err := dateRows.Scan(&orgID, &filterDate, &piezoDate); err != nil {
			return nil, fmt.Errorf("%s: scan dates: %w", op, err)
		}
		if mc, ok := result[orgID]; ok {
			mc.HistoricalFilterDate = filterDate
			mc.HistoricalPiezoDate = piezoDate
		}
	}
	if err := dateRows.Err(); err != nil {
		return nil, fmt.Errorf("%s: date rows: %w", op, err)
	}
	dateRows.Close()

	// Query 3: filter readings for all orgs
	const filterQuery = `
		SELECT l.organization_id, l.id, l.name, l.norm, l.sort_order,
		       m.flow_rate, m.historical_flow_rate
		FROM filtration_locations l
		LEFT JOIN manual_comparison_filter m ON m.location_id = l.id AND m.date = $2::date
		WHERE l.organization_id = ANY($1)
		ORDER BY l.organization_id, l.sort_order, l.id`

	filterRows, err := r.db.QueryContext(ctx, filterQuery, pq.Array(orgIDs), date)
	if err != nil {
		return nil, fmt.Errorf("%s: query filters: %w", op, err)
	}
	defer filterRows.Close()

	for filterRows.Next() {
		var orgID int64
		var fr manualcomparison.FilterReading
		var norm, flowRate, histFlowRate sql.NullFloat64
		if err := filterRows.Scan(
			&orgID, &fr.LocationID, &fr.LocationName, &norm, &fr.SortOrder,
			&flowRate, &histFlowRate,
		); err != nil {
			return nil, fmt.Errorf("%s: scan filter: %w", op, err)
		}
		if norm.Valid {
			fr.Norm = &norm.Float64
		}
		if flowRate.Valid {
			fr.FlowRate = &flowRate.Float64
		}
		if histFlowRate.Valid {
			fr.HistoricalFlowRate = &histFlowRate.Float64
		}
		if mc, ok := result[orgID]; ok {
			mc.Filters = append(mc.Filters, fr)
		}
	}
	if err := filterRows.Err(); err != nil {
		return nil, fmt.Errorf("%s: filter rows: %w", op, err)
	}
	filterRows.Close()

	// Query 4: piezo readings for all orgs
	const piezoQuery = `
		SELECT p.organization_id, p.id, p.name, p.norm, p.sort_order,
		       m.level, COALESCE(m.anomaly, false), m.historical_level
		FROM piezometers p
		LEFT JOIN manual_comparison_piezo m ON m.piezometer_id = p.id AND m.date = $2::date
		WHERE p.organization_id = ANY($1)
		ORDER BY p.organization_id, p.sort_order, p.id`

	piezoRows, err := r.db.QueryContext(ctx, piezoQuery, pq.Array(orgIDs), date)
	if err != nil {
		return nil, fmt.Errorf("%s: query piezos: %w", op, err)
	}
	defer piezoRows.Close()

	for piezoRows.Next() {
		var orgID int64
		var pr manualcomparison.PiezoReading
		var norm, level, histLevel sql.NullFloat64
		if err := piezoRows.Scan(
			&orgID, &pr.PiezometerID, &pr.PiezometerName, &norm, &pr.SortOrder,
			&level, &pr.Anomaly, &histLevel,
		); err != nil {
			return nil, fmt.Errorf("%s: scan piezo: %w", op, err)
		}
		if norm.Valid {
			pr.Norm = &norm.Float64
		}
		if level.Valid {
			pr.Level = &level.Float64
		}
		if histLevel.Valid {
			pr.HistoricalLevel = &histLevel.Float64
		}
		if mc, ok := result[orgID]; ok {
			mc.Piezometers = append(mc.Piezometers, pr)
		}
	}
	if err := piezoRows.Err(); err != nil {
		return nil, fmt.Errorf("%s: piezo rows: %w", op, err)
	}

	return result, nil
}

func (r *Repo) DeleteManualComparison(ctx context.Context, orgID int64, date string) error {
	const op = "storage.repo.ManualComparison.DeleteManualComparison"

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: begin tx: %w", op, err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "DELETE FROM manual_comparison_filter WHERE organization_id = $1 AND date = $2::date", orgID, date); err != nil {
		return fmt.Errorf("%s: delete filters: %w", op, err)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM manual_comparison_piezo WHERE organization_id = $1 AND date = $2::date", orgID, date); err != nil {
		return fmt.Errorf("%s: delete piezos: %w", op, err)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM manual_comparison_dates WHERE organization_id = $1 AND date = $2::date", orgID, date); err != nil {
		return fmt.Errorf("%s: delete dates: %w", op, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: commit: %w", op, err)
	}
	return nil
}

// --- Internal helpers ---

func (r *Repo) scanManualFilters(ctx context.Context, result *manualcomparison.OrgManualComparison, orgID int64, date, op string) error {
	const filterQuery = `
		SELECT l.id, l.name, l.norm, l.sort_order,
		       m.flow_rate, m.historical_flow_rate
		FROM filtration_locations l
		LEFT JOIN manual_comparison_filter m ON m.location_id = l.id AND m.date = $2::date
		WHERE l.organization_id = $1
		ORDER BY l.sort_order, l.id`

	filterRows, err := r.db.QueryContext(ctx, filterQuery, orgID, date)
	if err != nil {
		return fmt.Errorf("%s: query filters: %w", op, err)
	}
	defer filterRows.Close()

	for filterRows.Next() {
		var fr manualcomparison.FilterReading
		var norm, flowRate, histFlowRate sql.NullFloat64
		if err := filterRows.Scan(
			&fr.LocationID, &fr.LocationName, &norm, &fr.SortOrder,
			&flowRate, &histFlowRate,
		); err != nil {
			return fmt.Errorf("%s: scan filter: %w", op, err)
		}
		if norm.Valid {
			fr.Norm = &norm.Float64
		}
		if flowRate.Valid {
			fr.FlowRate = &flowRate.Float64
		}
		if histFlowRate.Valid {
			fr.HistoricalFlowRate = &histFlowRate.Float64
		}
		result.Filters = append(result.Filters, fr)
	}
	if err := filterRows.Err(); err != nil {
		return fmt.Errorf("%s: filter rows error: %w", op, err)
	}
	if result.Filters == nil {
		result.Filters = make([]manualcomparison.FilterReading, 0)
	}
	return nil
}

func (r *Repo) scanManualPiezos(ctx context.Context, result *manualcomparison.OrgManualComparison, orgID int64, date, op string) error {
	const piezoQuery = `
		SELECT p.id, p.name, p.norm, p.sort_order,
		       m.level, COALESCE(m.anomaly, false), m.historical_level
		FROM piezometers p
		LEFT JOIN manual_comparison_piezo m ON m.piezometer_id = p.id AND m.date = $2::date
		WHERE p.organization_id = $1
		ORDER BY p.sort_order, p.id`

	piezoRows, err := r.db.QueryContext(ctx, piezoQuery, orgID, date)
	if err != nil {
		return fmt.Errorf("%s: query piezos: %w", op, err)
	}
	defer piezoRows.Close()

	for piezoRows.Next() {
		var pr manualcomparison.PiezoReading
		var norm, level, histLevel sql.NullFloat64
		if err := piezoRows.Scan(
			&pr.PiezometerID, &pr.PiezometerName, &norm, &pr.SortOrder,
			&level, &pr.Anomaly, &histLevel,
		); err != nil {
			return fmt.Errorf("%s: scan piezo: %w", op, err)
		}
		if norm.Valid {
			pr.Norm = &norm.Float64
		}
		if level.Valid {
			pr.Level = &level.Float64
		}
		if histLevel.Valid {
			pr.HistoricalLevel = &histLevel.Float64
		}
		result.Piezometers = append(result.Piezometers, pr)
	}
	if err := piezoRows.Err(); err != nil {
		return fmt.Errorf("%s: piezo rows error: %w", op, err)
	}
	if result.Piezometers == nil {
		result.Piezometers = make([]manualcomparison.PiezoReading, 0)
	}
	return nil
}
