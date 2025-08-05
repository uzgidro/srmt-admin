package repo

import (
	"context"
	"database/sql"
	"srmt-admin/internal/storage"
	"time"
)

type Repo struct {
	Driver       *sql.DB
	ErrorHandler storage.ErrorTranslator
}

func New(driver *storage.Driver) *Repo {
	return &Repo{driver.DB, driver.Translator}
}

func (s *Repo) Close() error {
	return s.Driver.Close()
}

func (s *Repo) SaveAndijanData(ctx context.Context, t time.Time, current, resistance float64) error {
	//TODO implement me
	panic("implement me")
}
