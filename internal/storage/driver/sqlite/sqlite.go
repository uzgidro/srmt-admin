package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/mattn/go-sqlite3"
	"srmt-admin/internal/storage"
	"time"

	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func New(storagePath string, migrationsPath string) (*storage.Driver, error) {
	const op = "storage.driver.sqlite.New"
	const driver = "sqlite3"

	db, err := sql.Open(driver, storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	pingErr := db.Ping()
	if pingErr != nil {
		return nil, fmt.Errorf("%s: ping failed: %w", op, pingErr)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(5 * time.Minute)

	m, err := migrate.New(
		migrationsPath,
		driver+"://"+storagePath,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to initialize migrations: %w", op, err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return nil, fmt.Errorf("%s: failed to apply migrations: %w", op, err)
	}

	return &storage.Driver{DB: db, Translator: &Translator{}}, nil
}

type Translator struct{}

func (t *Translator) Translate(err error, op string) error {
	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) {
		switch sqliteErr.ExtendedCode {

		case sqlite3.ErrConstraintUnique:
			return fmt.Errorf("%s: %w", op, storage.ErrDuplicate)

		case sqlite3.ErrConstraintForeignKey:
			return fmt.Errorf("%s: %w", op, storage.ErrForeignKeyViolation)
		}
	}
	return fmt.Errorf("%s: %w", op, err)
}
