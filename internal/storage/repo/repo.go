package repo

import (
	"database/sql"
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
