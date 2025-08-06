package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"srmt-admin/internal/lib/model/data"
	"srmt-admin/internal/storage"
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

func (s *Repo) GetIndicator(ctx context.Context, resID int64) (float64, error) {
	const op = "storage.reservoir.GetIndicator"

	stmt, err := s.Driver.Prepare("SELECT height FROM indicator_height WHERE res_id = $1")
	if err != nil {
		return 0, fmt.Errorf("%s: failed to prepare statement: %w", op, err)
	}
	defer stmt.Close()

	var height float64
	if err := stmt.QueryRowContext(ctx, resID).Scan(&height); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, storage.ErrIndicatorNotFound
		}
		if translatedErr := s.ErrorHandler.Translate(err, op); translatedErr != nil {
			return 0, translatedErr
		}
		return 0, fmt.Errorf("%s: failed to execute statement: %w", op, err)
	}

	return height, nil
}

func (s *Repo) GetVolumeByLevel(ctx context.Context, resID int64, level float64) (float64, error) {
	const op = "storage.reservoir.GetVolumeByLevel"

	queryBelow := `SELECT level, volume FROM level_volume WHERE res_id = $1 AND level <= $2 ORDER BY level DESC LIMIT 1`
	queryAbove := `SELECT level, volume FROM level_volume WHERE res_id = $1 AND level >= $2 ORDER BY level LIMIT 1`

	var p1, p2 data.Model

	rowBelow := s.Driver.QueryRowContext(ctx, queryBelow, resID, level)
	err1 := rowBelow.Scan(&p1.Level, &p1.Volume)

	rowAbove := s.Driver.QueryRowContext(ctx, queryAbove, resID, level)
	err2 := rowAbove.Scan(&p2.Level, &p2.Volume)

	if err1 != nil || err2 != nil {
		if errors.Is(err1, sql.ErrNoRows) || errors.Is(err2, sql.ErrNoRows) {
			return 0, storage.ErrLevelOutOfCurveRange
		}
		if err1 != nil {
			return 0, fmt.Errorf("%s: failed to get lower point: %w", op, err1)
		}
		return 0, fmt.Errorf("%s: failed to get upper point: %w", op, err2)
	}

	if p1.Level == p2.Level {
		return p1.Volume, nil
	}

	interpolatedVolume := p1.Volume + (level-p1.Level)*(p2.Volume-p1.Volume)/(p2.Level-p1.Level)

	return interpolatedVolume, nil
}
