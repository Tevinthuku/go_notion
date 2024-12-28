package migrations

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"

	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func Run(config *pgxpool.Config) error {
	dbURL := config.ConnString()
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return fmt.Errorf("unable to open database: %w", err)
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("unable to create driver: %w", err)
	}

	// runtime.Caller(0) returns the file path of the current source file (migrations.go).
	// This allows us to construct absolute paths relative to this file's location,
	// ensuring migrations can be found regardless of where the binary is executed from.
	_, filename, _, _ := runtime.Caller(0)
	// Get the project root (3 levels up from migrations.go)
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..", "..")
	migrationPath := filepath.Join(projectRoot, "backend", "db", "migrations", "sql")

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationPath),
		config.ConnConfig.Database,
		driver,
	)
	if err != nil {
		return fmt.Errorf("unable to create migration: %w", err)
	}
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		return fmt.Errorf("failed to apply migrations: %w", err)
	}
	return nil
}
