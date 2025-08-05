package repo

import (
	"context"
	"fmt"
)

func (s *Repo) AddReservoir(ctx context.Context, name string) (int64, error) {
	const op = "storage.reservoir.AddReservoir"

	stmt, err := s.Driver.Prepare("INSERT INTO reservoirs(name) VALUES($1) RETURNING id")
	if err != nil {
		return 0, fmt.Errorf("%s: failed to prepare statement: %w", op, err)
	}
	defer stmt.Close()

	var id int64
	if err := stmt.QueryRowContext(ctx, name).Scan(&id); err != nil {
		if err := s.ErrorHandler.Translate(err, op); err != nil {
			return 0, err
		}
		return 0, fmt.Errorf("%s: failed to execute statement: %w", op, err)
	}

	return id, nil
}

func (s *Repo) SetIndicator(ctx context.Context, resID int64, height float64) (int64, error) {
	const op = "storage.reservoir.SetIndicator"

	const query = `
		INSERT INTO indicator_height (res_id, height)
		VALUES ($1, $2)
		ON CONFLICT (res_id)
		DO UPDATE SET height = EXCLUDED.height
		RETURNING id
	`

	stmt, err := s.Driver.Prepare(query)
	if err != nil {
		return 0, fmt.Errorf("%s: failed to prepare statement: %w", op, err)
	}
	defer stmt.Close()

	var id int64
	if err := stmt.QueryRowContext(ctx, resID, height).Scan(&id); err != nil {
		if translatedErr := s.ErrorHandler.Translate(err, op); translatedErr != nil {
			return 0, translatedErr
		}
		return 0, fmt.Errorf("%s: failed to execute statement: %w", op, err)
	}

	return id, nil
}
