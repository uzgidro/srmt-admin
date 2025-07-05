package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"srmt-admin/internal/lib/model/user"
	"srmt-admin/internal/storage"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

type Storage struct {
	db *sql.DB
}

func New(storagePath string, migrationsPath string) (*Storage, error) {
	const op = "storage.sqlite.New"
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

	return &Storage{db: db}, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) AddUser(ctx context.Context, name, passHash string) (int64, error) {
	const op = "storage.sqlite.AddUser"

	query := `INSERT INTO users (name, pass_hash) VALUES (?, ?)`

	res, err := s.db.ExecContext(ctx, query, name, passHash)
	if err != nil {
		return 0, fmt.Errorf("%s: execute query: %w", op, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("%s: get last insert id: %w", op, err)
	}

	return id, nil
}

func (s *Storage) GetUserByName(ctx context.Context, name string) (user.Model, error) {
	const op = "storage.sqlite.GetUserByName"

	query := `SELECT id, name, pass_hash FROM users WHERE name = ?`

	row := s.db.QueryRowContext(ctx, query, name)

	var u user.Model
	err := row.Scan(&u.Id, &u.Name, &u.PassHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user.Model{}, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}
		return user.Model{}, fmt.Errorf("%s: %w", op, err)
	}

	return u, nil
}
