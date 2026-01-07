package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"srmt-admin/internal/lib/dto"
	investActiveProject "srmt-admin/internal/lib/model/invest-active-project"
	"srmt-admin/internal/storage"
)

// AddInvestActiveProject creates a new active project record
func (r *Repo) AddInvestActiveProject(ctx context.Context, req dto.AddInvestActiveProjectRequest) (int64, error) {
	const op = "storage.repo.AddInvestActiveProject"

	const query = `
		INSERT INTO invest_active_projects (category, project_name, foreign_partner, implementation_period, capacity_mw, production_mln_kwh, cost_mln_usd, status_text)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		req.Category,
		req.ProjectName,
		req.ForeignPartner,
		req.ImplementationPeriod,
		req.CapacityMW,
		req.ProductionMlnKWh,
		req.CostMlnUSD,
		req.StatusText,
	).Scan(&id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return 0, translatedErr
		}
		return 0, fmt.Errorf("%s: failed to insert invest_active_project: %w", op, err)
	}

	return id, nil
}

// GetInvestActiveProjectByID retrieves a single active project
func (r *Repo) GetInvestActiveProjectByID(ctx context.Context, id int64) (*investActiveProject.Model, error) {
	const op = "storage.repo.GetInvestActiveProjectByID"

	const query = `
		SELECT
			id, category, project_name, foreign_partner, implementation_period, capacity_mw, production_mln_kwh, cost_mln_usd, status_text, created_at
		FROM invest_active_projects
		WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, id)
	return scanInvestActiveProjectRow(row)
}

// GetAllInvestActiveProjects retrieves all active projects
func (r *Repo) GetAllInvestActiveProjects(ctx context.Context) ([]*investActiveProject.Model, error) {
	const op = "storage.repo.GetAllInvestActiveProjects"

	const query = `
		SELECT
			id, category, project_name, foreign_partner, implementation_period, capacity_mw, production_mln_kwh, cost_mln_usd, status_text, created_at
		FROM invest_active_projects
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query invest_active_projects: %w", op, err)
	}
	defer rows.Close()

	var projects []*investActiveProject.Model
	for rows.Next() {
		p, err := scanInvestActiveProjectRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan invest_active_project row: %w", op, err)
		}
		projects = append(projects, p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration error: %w", op, err)
	}

	if projects == nil {
		projects = make([]*investActiveProject.Model, 0)
	}

	return projects, nil
}

// EditInvestActiveProject updates an active project record
func (r *Repo) EditInvestActiveProject(ctx context.Context, id int64, req dto.EditInvestActiveProjectRequest) error {
	const op = "storage.repo.EditInvestActiveProject"

	var updates []string
	var args []interface{}
	argID := 1

	if req.Category != nil {
		updates = append(updates, fmt.Sprintf("category = $%d", argID))
		args = append(args, *req.Category)
		argID++
	}
	if req.ProjectName != nil {
		updates = append(updates, fmt.Sprintf("project_name = $%d", argID))
		args = append(args, *req.ProjectName)
		argID++
	}
	if req.ForeignPartner != nil {
		updates = append(updates, fmt.Sprintf("foreign_partner = $%d", argID))
		args = append(args, *req.ForeignPartner)
		argID++
	}
	if req.ImplementationPeriod != nil {
		updates = append(updates, fmt.Sprintf("implementation_period = $%d", argID))
		args = append(args, *req.ImplementationPeriod)
		argID++
	}
	if req.CapacityMW != nil {
		updates = append(updates, fmt.Sprintf("capacity_mw = $%d", argID))
		args = append(args, *req.CapacityMW)
		argID++
	}
	if req.ProductionMlnKWh != nil {
		updates = append(updates, fmt.Sprintf("production_mln_kwh = $%d", argID))
		args = append(args, *req.ProductionMlnKWh)
		argID++
	}
	if req.CostMlnUSD != nil {
		updates = append(updates, fmt.Sprintf("cost_mln_usd = $%d", argID))
		args = append(args, *req.CostMlnUSD)
		argID++
	}
	if req.StatusText != nil {
		updates = append(updates, fmt.Sprintf("status_text = $%d", argID))
		args = append(args, *req.StatusText)
		argID++
	}

	if len(updates) == 0 {
		return nil // Nothing to update
	}

	query := fmt.Sprintf("UPDATE invest_active_projects SET %s WHERE id = $%d",
		strings.Join(updates, ", "),
		argID,
	)
	args = append(args, id)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to update invest_active_project: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteInvestActiveProject deletes an active project record
func (r *Repo) DeleteInvestActiveProject(ctx context.Context, id int64) error {
	const op = "storage.repo.DeleteInvestActiveProject"

	res, err := r.db.ExecContext(ctx, "DELETE FROM invest_active_projects WHERE id = $1", id)
	if err != nil {
		if translatedErr := r.translator.Translate(err, op); translatedErr != nil {
			return translatedErr
		}
		return fmt.Errorf("%s: failed to delete invest_active_project: %w", op, err)
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

func scanInvestActiveProjectRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*investActiveProject.Model, error) {
	var p investActiveProject.Model
	var foreignPartner, implementationPeriod, statusText sql.NullString
	var capacityMW, productionMlnKWh, costMlnUSD sql.NullFloat64

	err := scanner.Scan(
		&p.ID,
		&p.Category,
		&p.ProjectName,
		&foreignPartner,
		&implementationPeriod,
		&capacityMW,
		&productionMlnKWh,
		&costMlnUSD,
		&statusText,
		&p.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, err
	}

	if foreignPartner.Valid {
		p.ForeignPartner = &foreignPartner.String
	}
	if implementationPeriod.Valid {
		p.ImplementationPeriod = &implementationPeriod.String
	}
	if statusText.Valid {
		p.StatusText = &statusText.String
	}
	if capacityMW.Valid {
		p.CapacityMW = &capacityMW.Float64
	}
	if productionMlnKWh.Valid {
		p.ProductionMlnKWh = &productionMlnKWh.Float64
	}
	if costMlnUSD.Valid {
		p.CostMlnUSD = &costMlnUSD.Float64
	}

	return &p, nil
}
