package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"go_notion/backend/db/migrations"
	"net/url"
	"os"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	pgxuuid "github.com/jackc/pgx-gofrs-uuid"
)

func RunTestDb() (*pgxpool.Pool, error) {
	return runInner(true)
}

func Run() (*pgxpool.Pool, error) {
	return runInner(false)
}

// Run initializes a database connection pool and applies pending migrations.
// When is_test_mode is true, it creates a test-specific database and applies migrations to it.
//
// Test mode is useful for isolated testing environments where each test can
// have its own database instance.
//
// Returns a connection pool and any error encountered during setup.
func runInner(is_test_mode bool) (*pgxpool.Pool, error) {
	// loading of env variables is done at app startup
	dbURL, ok := os.LookupEnv("DATABASE_URL")
	if !ok {
		return nil, errors.New("DATABASE_URL is not set")
	}

	if is_test_mode {
		db, err := sql.Open("pgx", dbURL)
		if err != nil {
			return nil, fmt.Errorf("unable to open database: %w", err)
		}
		dbName := uuid.Must(uuid.NewV4()).String()
		_, err = db.Exec(fmt.Sprintf("CREATE DATABASE \"%s\"", dbName))
		if err != nil {
			return nil, fmt.Errorf("unable to create database: %w", err)
		}

		err = db.Close()
		if err != nil {
			return nil, fmt.Errorf("unable to close database: %w", err)
		}

		u, err := url.Parse(dbURL)
		if err != nil {
			return nil, fmt.Errorf("parsing database URL: %w", err)
		}
		println("dbName", dbName)
		u.Path = "/" + dbName
		dbURL = u.String()
	}

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}
	// setting connection pool limits to prevent resource exhaustion
	config.MaxConns = 20
	config.MinConns = 2
	config.MaxConnLifetime = 1 * time.Hour
	config.MaxConnIdleTime = 30 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute

	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		pgxuuid.Register(conn.TypeMap())
		return nil
	}

	err = migrations.Run(config)
	if err != nil {
		return nil, fmt.Errorf("unable to run migrations: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbpool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}
	// we need to verify connection before returning the pool so we don't return a pool with broken connections that can't be used
	if err := dbpool.Ping(ctx); err != nil {
		dbpool.Close()
		return nil, fmt.Errorf("failed to verify database connection: %w", err)
	}

	return dbpool, nil
}

type DB interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error)
}
