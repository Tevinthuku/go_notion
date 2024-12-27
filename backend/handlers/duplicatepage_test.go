package handlers_test

import (
	"context"
	"fmt"
	"go_notion/backend/db/migrations"
	"go_notion/backend/handlers"
	"os"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPageColumnsAreInSyncWithDb(t *testing.T) {
	dbURL, _ := os.LookupEnv("DATABASE_URL")
	fmt.Println(dbURL)
	err := migrations.RunInner(dbURL, "../db/migrations/sql", "notion_test")
	if err != nil {
		t.Fatal(err)
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	rows, err := pool.Query(context.Background(),
		"SELECT column_name FROM information_schema.columns WHERE table_name = 'pages' ORDER BY ordinal_position")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var colName string
		if err := rows.Scan(&colName); err != nil {
			t.Fatal(err)
		}
		// we ignore the auto-generated columns
		if colName == "id" || colName == "created_at" || colName == "updated_at" {
			continue
		}
		columns = append(columns, colName)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	fmt.Println(columns)
	for _, col := range handlers.PageColumns {
		if !slices.Contains(columns, col) {
			t.Errorf("column %s is missing from the pages table", col)
		}
	}

}
