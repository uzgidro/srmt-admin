package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgconn"
	"srmt-admin/internal/storage"

	_ "github.com/lib/pq"
)

func New(storagePath string, migrationsPath string) (*storage.Driver, error) {
	const op = "storage.driver.postgres.New"
	const driver = "postgres"

	db, err := sql.Open(driver, storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	pingErr := db.Ping()
	if pingErr != nil {
		return nil, fmt.Errorf("%s", pingErr)
	}

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
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return fmt.Errorf("%s: %w", op, storage.ErrDuplicate)
		case "23503":
			return fmt.Errorf("%s: %w", op, storage.ErrForeignKeyViolation)
		}
	}
	return fmt.Errorf("%s: %w", op, err)
}
