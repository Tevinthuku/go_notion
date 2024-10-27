package migrations

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"

	_ "github.com/golang-migrate/migrate/source/file"
)

func Run(dbURL string) error {

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return fmt.Errorf("unable to create database: %v", err)
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("unable to create driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://backend/db/migrations/sql",
		"go_notion",
		driver,
	)
	if err != nil {
		return fmt.Errorf("unable to create migration: %v", err)
	}
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		return fmt.Errorf("failed to apply migrations: %w", err)
	}
	return nil
}
