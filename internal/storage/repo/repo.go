package repo

import (
	"context"
	"database/sql"
	"srmt-admin/internal/lib/model/data"
	"srmt-admin/internal/storage"
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

func (s *Repo) SaveData(ctx context.Context, data data.Model) error {
	//TODO implement me
	panic("implement me")
}
