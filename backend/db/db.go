package db

import (
	"context"
	"errors"
	"fmt"
	"go_notion/backend/db/migrations"
	"os"

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

	dbpool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %v", err)
	}

	return dbpool, nil
}

type DB interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}
