package db

import (
	"context"
	"errors"
	"fmt"
	"go_notion/backend/db/migrations"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Run() (*pgxpool.Pool, error) {
	// loading of env variables is done at app startup
	dbURL, ok := os.LookupEnv("DATABASE_URL")
	if !ok {
		return nil, errors.New("DATABASE_URL is not set")
	}

	err := migrations.Run(dbURL)
	if err != nil {
		return nil, fmt.Errorf("unable to run migrations: %v", err)
	}

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %v", err)
	}
	// setting connection pool limits to prevent resource exhaustion
	config.MaxConns = 20
	config.MinConns = 2
	config.MaxConnLifetime = 1 * time.Hour
	config.MaxConnIdleTime = 30 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbpool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %v", err)
	}
	// we need to verify connection before returning the pool so we don't return a pool with broken connections that can't be used
	if err := dbpool.Ping(ctx); err != nil {
		dbpool.Close()
		return nil, fmt.Errorf("failed to verify database connection: %v", err)
	}

	return dbpool, nil
}

type DB interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}
