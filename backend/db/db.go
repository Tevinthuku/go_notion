package db

import (
	"context"
	"fmt"
	"go_notion/backend/db/migrations"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Run() *pgxpool.Pool {
	// loading of env variables is done at app startup
	dbURL, ok := os.LookupEnv("DATABASE_URL")
	if !ok {
		fmt.Fprintf(os.Stderr, "DATABASE_URL is not set\n")
		os.Exit(1)
	}

	migrations.Run(dbURL)

	dbpool, err := pgxpool.New(context.Background(), dbURL)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create connection pool: %v\n", err)
		os.Exit(1)
	}

	return dbpool
}
