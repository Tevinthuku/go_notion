package db

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
)

func InsertTestUserFixture(conn *pgx.Conn) error {
	_, err := conn.Exec(context.Background(), `
		INSERT INTO users (email, username, password) VALUES ($1, $2, $3)
	`, "test@test.com", "test", "test")
	if err != nil {
		return fmt.Errorf("error inserting test user: %w", err)
	}
	return nil
}

func InsertTestPageFixture(page_id uuid.UUID, user_id int64) Fixture {
	return func(conn *pgx.Conn) error {
		_, err := conn.Exec(context.Background(), `
			INSERT INTO pages (id, created_by) VALUES ($1, $2)
		`, page_id, user_id)
		return err
	}
}
