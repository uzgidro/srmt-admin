package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

type Storage struct {
	DB *sql.DB
}

func New(storagePath string) (*Storage, error) {
	const op = "storage.sqlite.New"

	db, err := sql.Open("sqlite3", storagePath)
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

	fmt.Printf("Запуск миграций из %q...\n", "file://./migrations")
	m, err := migrate.New(
		"file://./migrations",       // Путь к файлам миграций
		"sqlite3"+"://"+storagePath, // Строка подключения для мигратора
	)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to initialize migrations: %w", op, err)
	}
	defer m.Close() // Важно закрыть мигратор

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return nil, fmt.Errorf("%s: failed to apply migrations: %w", op, err)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		fmt.Println("Миграции не требуются, база данных уже актуальна.")
	} else {
		fmt.Println("Миграции успешно выполнены.")
	}

	return &Storage{DB: db}, nil
}
