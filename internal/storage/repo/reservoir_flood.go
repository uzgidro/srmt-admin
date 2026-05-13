package repo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"

	model "srmt-admin/internal/lib/model/reservoir-flood"
	"srmt-admin/internal/storage"
)

// --- Config CRUD ---

// UpsertReservoirFloodConfig inserts or updates the per-organization config row.
// Conflict key is organization_id (UNIQUE in the table).
func (r *Repo) UpsertReservoirFloodConfig(ctx context.Context, req model.UpsertConfigRequest) error {
	const op = "storage.repo.ReservoirFlood.UpsertConfig"

	const query = `
		INSERT INTO reservoir_flood_config (organization_id, sort_order, is_active)
		VALUES ($1, $2, $3)
		ON CONFLICT (organization_id) DO UPDATE SET
			sort_order = EXCLUDED.sort_order,
			is_active  = EXCLUDED.is_active,
			updated_at = NOW()`

	if _, err := r.db.ExecContext(ctx, query, req.OrganizationID, req.SortOrder, req.IsActive); err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

// GetAllReservoirFloodConfigs returns all configs joined with organization name,
// ordered by sort_order then organization name.
func (r *Repo) GetAllReservoirFloodConfigs(ctx context.Context) ([]model.Config, error) {
	const op = "storage.repo.ReservoirFlood.GetAllConfigs"

	const query = `
		SELECT c.id, c.organization_id, COALESCE(o.name, ''), c.sort_order, c.is_active, c.updated_at
		FROM reservoir_flood_config c
		LEFT JOIN organizations o ON o.id = c.organization_id
		ORDER BY c.sort_order, COALESCE(o.name, '')`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: query: %w", op, err)
	}
	defer rows.Close()

	out := make([]model.Config, 0)
	for rows.Next() {
		var c model.Config
		if err := rows.Scan(
			&c.ID, &c.OrganizationID, &c.OrganizationName,
			&c.SortOrder, &c.IsActive, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows: %w", op, err)
	}
	return out, nil
}

// DeleteReservoirFloodConfig removes the config for the given org.
// Returns storage.ErrNotFound if no row matched.
func (r *Repo) DeleteReservoirFloodConfig(ctx context.Context, organizationID int64) error {
	const op = "storage.repo.ReservoirFlood.DeleteConfig"

	res, err := r.db.ExecContext(ctx,
		`DELETE FROM reservoir_flood_config WHERE organization_id = $1`, organizationID)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: rows affected: %w", op, err)
	}
	if n == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// --- Hourly ---

// UpsertReservoirFloodHourly performs a bulk upsert of hourly observation rows
// inside a single transaction. RecordedAt on each item is the already-normalized
// hour-bound RFC3339 string from the handler; this method parses it back to
// time.Time before sending to the database. Per-field overrides use the
// optional.Optional Set flag via CASE flag parameters so that absent fields
// preserve existing values while explicit nulls/values are written through.
func (r *Repo) UpsertReservoirFloodHourly(ctx context.Context, items []model.UpsertHourlyRequest, userID int64) error {
	const op = "storage.repo.ReservoirFlood.UpsertHourly"

	if len(items) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: begin tx: %w", op, err)
	}
	defer tx.Rollback()

	const query = `
		INSERT INTO reservoir_flood_hourly (
			organization_id, recorded_at,
			water_level_m, water_volume_mln_m3, inflow_m3s, outflow_m3s,
			ges_flow_m3s, filtration_m3s, idle_discharge_m3s, duty_name,
			capacity_mwt, weather_condition, temperature_c,
			created_by_user_id, updated_by_user_id, created_at, updated_at
		)
		VALUES ($1, $2,
		        $3, $4, $5, $6,
		        $7, $8, $9, $10,
		        $11, $12, $13,
		        $14, $14, NOW(), NOW())
		ON CONFLICT (organization_id, recorded_at) DO UPDATE SET
			water_level_m       = CASE WHEN $15::boolean THEN EXCLUDED.water_level_m       ELSE reservoir_flood_hourly.water_level_m       END,
			water_volume_mln_m3 = CASE WHEN $16::boolean THEN EXCLUDED.water_volume_mln_m3 ELSE reservoir_flood_hourly.water_volume_mln_m3 END,
			inflow_m3s          = CASE WHEN $17::boolean THEN EXCLUDED.inflow_m3s          ELSE reservoir_flood_hourly.inflow_m3s          END,
			outflow_m3s         = CASE WHEN $18::boolean THEN EXCLUDED.outflow_m3s         ELSE reservoir_flood_hourly.outflow_m3s         END,
			ges_flow_m3s        = CASE WHEN $19::boolean THEN EXCLUDED.ges_flow_m3s        ELSE reservoir_flood_hourly.ges_flow_m3s        END,
			filtration_m3s      = CASE WHEN $20::boolean THEN EXCLUDED.filtration_m3s      ELSE reservoir_flood_hourly.filtration_m3s      END,
			idle_discharge_m3s  = CASE WHEN $21::boolean THEN EXCLUDED.idle_discharge_m3s  ELSE reservoir_flood_hourly.idle_discharge_m3s  END,
			duty_name           = CASE WHEN $22::boolean THEN EXCLUDED.duty_name           ELSE reservoir_flood_hourly.duty_name           END,
			capacity_mwt        = CASE WHEN $23::boolean THEN EXCLUDED.capacity_mwt        ELSE reservoir_flood_hourly.capacity_mwt        END,
			weather_condition   = CASE WHEN $24::boolean THEN EXCLUDED.weather_condition   ELSE reservoir_flood_hourly.weather_condition   END,
			temperature_c       = CASE WHEN $25::boolean THEN EXCLUDED.temperature_c       ELSE reservoir_flood_hourly.temperature_c       END,
			updated_by_user_id  = EXCLUDED.updated_by_user_id,
			updated_at          = NOW()`

	for _, it := range items {
		recordedAt, parseErr := time.Parse(time.RFC3339, it.RecordedAt)
		if parseErr != nil {
			return fmt.Errorf("%s: invalid recorded_at %q: %w", op, it.RecordedAt, parseErr)
		}

		if _, execErr := tx.ExecContext(ctx, query,
			it.OrganizationID, recordedAt, // $1, $2
			it.WaterLevelM.Value, it.WaterVolumeMlnM3.Value, // $3, $4
			it.InflowM3s.Value, it.OutflowM3s.Value, // $5, $6
			it.GESFlowM3s.Value, it.FiltrationM3s.Value, // $7, $8
			it.IdleDischargeM3s.Value, it.DutyName.Value, // $9, $10
			it.CapacityMwt.Value, it.WeatherCondition.Value, // $11, $12
			it.TemperatureC.Value, // $13
			userID, // $14
			it.WaterLevelM.Set, it.WaterVolumeMlnM3.Set, // $15, $16
			it.InflowM3s.Set, it.OutflowM3s.Set, // $17, $18
			it.GESFlowM3s.Set, it.FiltrationM3s.Set, // $19, $20
			it.IdleDischargeM3s.Set, it.DutyName.Set, // $21, $22
			it.CapacityMwt.Set, it.WeatherCondition.Set, // $23, $24
			it.TemperatureC.Set, // $25
		); execErr != nil {
			if translatedErr := r.translator.Translate(execErr, op); translatedErr != nil {
				return translatedErr
			}
			return fmt.Errorf("%s: upsert org=%d recorded_at=%s: %w",
				op, it.OrganizationID, it.RecordedAt, execErr)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: commit: %w", op, err)
	}
	return nil
}

// GetReservoirFloodHourlyRange returns hourly records for the given orgIDs in
// the half-open range [start, end). When orgIDs is empty/nil, returns records
// for ALL organizations marked is_active=true in reservoir_flood_config.
func (r *Repo) GetReservoirFloodHourlyRange(ctx context.Context, orgIDs []int64, start, end time.Time) ([]model.HourlyRecord, error) {
	const op = "storage.repo.ReservoirFlood.GetHourlyRange"

	var (
		rows *sql.Rows
		err  error
	)

	if len(orgIDs) == 0 {
		const query = `
			SELECT h.id, h.organization_id, COALESCE(o.name, ''),
			       h.recorded_at,
			       h.water_level_m, h.water_volume_mln_m3, h.inflow_m3s, h.outflow_m3s,
			       h.ges_flow_m3s, h.filtration_m3s, h.idle_discharge_m3s,
			       h.duty_name,
			       h.capacity_mwt, h.weather_condition, h.temperature_c,
			       h.created_by_user_id, h.updated_at
			FROM reservoir_flood_hourly h
			JOIN reservoir_flood_config c ON c.organization_id = h.organization_id AND c.is_active = TRUE
			LEFT JOIN organizations o ON o.id = h.organization_id
			WHERE h.recorded_at >= $1 AND h.recorded_at < $2
			ORDER BY h.organization_id, h.recorded_at`
		rows, err = r.db.QueryContext(ctx, query, start, end)
	} else {
		const query = `
			SELECT h.id, h.organization_id, COALESCE(o.name, ''),
			       h.recorded_at,
			       h.water_level_m, h.water_volume_mln_m3, h.inflow_m3s, h.outflow_m3s,
			       h.ges_flow_m3s, h.filtration_m3s, h.idle_discharge_m3s,
			       h.duty_name,
			       h.capacity_mwt, h.weather_condition, h.temperature_c,
			       h.created_by_user_id, h.updated_at
			FROM reservoir_flood_hourly h
			LEFT JOIN organizations o ON o.id = h.organization_id
			WHERE h.recorded_at >= $1 AND h.recorded_at < $2
			  AND h.organization_id = ANY($3)
			ORDER BY h.organization_id, h.recorded_at`
		rows, err = r.db.QueryContext(ctx, query, start, end, pq.Array(orgIDs))
	}
	if err != nil {
		return nil, fmt.Errorf("%s: query: %w", op, err)
	}
	defer rows.Close()

	out := make([]model.HourlyRecord, 0)
	for rows.Next() {
		rec, err := scanHourlyRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows: %w", op, err)
	}
	return out, nil
}

// scanHourlyRecord reads a single hourly row from rows. The column order must
// match the SELECT lists in GetReservoirFloodHourlyRange and
// GetReservoirFloodHourlyLatestBefore: id, organization_id, name, recorded_at,
// water_level_m, water_volume_mln_m3, inflow_m3s, outflow_m3s, ges_flow_m3s,
// filtration_m3s, idle_discharge_m3s, duty_name, capacity_mwt,
// weather_condition, temperature_c, created_by_user_id, updated_at.
func scanHourlyRecord(rows *sql.Rows) (model.HourlyRecord, error) {
	var rec model.HourlyRecord
	var (
		waterLevel, waterVolume, inflow, outflow sql.NullFloat64
		gesFlow, filtration, idleDischarge       sql.NullFloat64
		dutyName                                 sql.NullString
		capacityMwt                              sql.NullFloat64
		weatherCondition                         sql.NullString
		temperatureC                             sql.NullFloat64
		createdBy                                sql.NullInt64
	)
	if err := rows.Scan(
		&rec.ID, &rec.OrganizationID, &rec.OrganizationName,
		&rec.RecordedAt,
		&waterLevel, &waterVolume, &inflow, &outflow,
		&gesFlow, &filtration, &idleDischarge,
		&dutyName,
		&capacityMwt, &weatherCondition, &temperatureC,
		&createdBy, &rec.UpdatedAt,
	); err != nil {
		return rec, err
	}
	if waterLevel.Valid {
		v := waterLevel.Float64
		rec.WaterLevelM = &v
	}
	if waterVolume.Valid {
		v := waterVolume.Float64
		rec.WaterVolumeMlnM3 = &v
	}
	if inflow.Valid {
		v := inflow.Float64
		rec.InflowM3s = &v
	}
	if outflow.Valid {
		v := outflow.Float64
		rec.OutflowM3s = &v
	}
	if gesFlow.Valid {
		v := gesFlow.Float64
		rec.GESFlowM3s = &v
	}
	if filtration.Valid {
		v := filtration.Float64
		rec.FiltrationM3s = &v
	}
	if idleDischarge.Valid {
		v := idleDischarge.Float64
		rec.IdleDischargeM3s = &v
	}
	if dutyName.Valid {
		s := dutyName.String
		rec.DutyName = &s
	}
	if capacityMwt.Valid {
		v := capacityMwt.Float64
		rec.CapacityMwt = &v
	}
	if weatherCondition.Valid {
		s := weatherCondition.String
		rec.WeatherCondition = &s
	}
	if temperatureC.Valid {
		v := temperatureC.Float64
		rec.TemperatureC = &v
	}
	if createdBy.Valid {
		i := createdBy.Int64
		rec.CreatedByUserID = &i
	}
	return rec, nil
}

// GetReservoirFloodHourlyLatestBefore returns at most one record per org —
// the most recent row with recorded_at < before. Used by the sel report
// builder to pick the "previous snapshot" when intervals between
// observations vary (e.g. hourly at night, every 3h during day, with skips).
//
// SQL uses LATERAL + LIMIT 1 to coerce the planner into N index seeks against
// (organization_id, recorded_at DESC) rather than a Bitmap+Sort over the full
// history per org. Without the LIMIT 1, DISTINCT ON would force PG to sort
// the entire matching set first — fine for a few rows, expensive once the
// table accumulates years of history.
//
// orgIDs empty/nil → returns (nil, nil) without touching the DB. This is an
// explicit guard rather than a comment because future callers may pass an
// unfiltered slice.
func (r *Repo) GetReservoirFloodHourlyLatestBefore(ctx context.Context, orgIDs []int64, before time.Time) ([]model.HourlyRecord, error) {
	const op = "storage.repo.ReservoirFlood.GetHourlyLatestBefore"

	if len(orgIDs) == 0 {
		return nil, nil
	}

	const query = `
		SELECT h.id, h.organization_id, COALESCE(o.name, ''),
		       h.recorded_at,
		       h.water_level_m, h.water_volume_mln_m3, h.inflow_m3s, h.outflow_m3s,
		       h.ges_flow_m3s, h.filtration_m3s, h.idle_discharge_m3s,
		       h.duty_name,
		       h.capacity_mwt, h.weather_condition, h.temperature_c,
		       h.created_by_user_id, h.updated_at
		FROM unnest($1::bigint[]) AS orgs(organization_id)
		JOIN LATERAL (
		    SELECT *
		    FROM reservoir_flood_hourly hh
		    WHERE hh.organization_id = orgs.organization_id
		      AND hh.recorded_at < $2
		    ORDER BY hh.recorded_at DESC
		    LIMIT 1
		) h ON TRUE
		LEFT JOIN organizations o ON o.id = h.organization_id
		ORDER BY h.organization_id`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(orgIDs), before)
	if err != nil {
		return nil, fmt.Errorf("%s: query: %w", op, err)
	}
	defer rows.Close()

	out := make([]model.HourlyRecord, 0, len(orgIDs))
	for rows.Next() {
		rec, err := scanHourlyRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows: %w", op, err)
	}
	return out, nil
}
