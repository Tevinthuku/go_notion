package handlers_test

import (
	"context"
	"fmt"
	"go_notion/backend/db"
	"go_notion/backend/handlers"
	"slices"
	"testing"
)

func TestPageColumnsAreInSyncWithDb(t *testing.T) {

	pool, err := db.RunTestDb()
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
