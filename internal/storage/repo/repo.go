package repo

import (
	"context"
	"database/sql"
	"srmt-admin/internal/lib/model/data"
	"srmt-admin/internal/storage"
)

type Repo struct {
	db         *sql.DB
	translator storage.ErrorTranslator
}

func New(driver *storage.Driver) *Repo {
	return &Repo{driver.DB, driver.Translator}
}

func (r *Repo) Close() error {
	return r.db.Close()
}

func (r *Repo) SaveData(ctx context.Context, data data.Model) error {
	//TODO implement me
	panic("implement me")
}
