package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/mattn/go-sqlite3"
	"srmt-admin/internal/lib/model/role"
	"srmt-admin/internal/lib/model/user"
	"srmt-admin/internal/storage"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
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

// AddRole создает новую роль.
func (s *Storage) AddRole(ctx context.Context, name string) (int64, error) {
	stmt, err := s.db.Prepare("INSERT INTO roles(name) VALUES(?)")
	if err != nil {
		return 0, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(ctx, name)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return 0, storage.ErrRoleExists
		}
		return 0, fmt.Errorf("failed to execute statement: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}
	return id, nil
}

// AssignRoleToUser назначает роль пользователю.
func (s *Storage) AssignRoleToUser(ctx context.Context, userID, roleID int64) error {
	stmt, err := s.db.Prepare("INSERT INTO user_roles(user_id, role_id) VALUES(?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	if _, err := stmt.ExecContext(ctx, userID, roleID); err != nil {
		// Можно добавить обработку ошибки FOREIGN KEY, если нужно
		return fmt.Errorf("failed to execute statement: %w", err)
	}
	return nil
}

// GetUserRoles возвращает все роли, назначенные пользователю.
func (s *Storage) GetUserRoles(ctx context.Context, userID int64) ([]role.Model, error) {
	const query = `
		SELECT r.id, r.name FROM roles r
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = ?
	`
	stmt, err := s.db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query roles: %w", err)
	}
	defer rows.Close()

	var roles []role.Model
	for rows.Next() {
		var r role.Model
		if err := rows.Scan(&r.ID, &r.Name); err != nil {
			return nil, fmt.Errorf("failed to scan role row: %w", err)
		}
		roles = append(roles, r)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return roles, nil
}
