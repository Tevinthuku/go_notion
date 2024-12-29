package db

import (
	"context"
	"encoding/json"
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
			INSERT INTO pages (id, created_by, position, text_title, text_content, title, content) VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, page_id, user_id, 1, "test", "test", json.RawMessage(`{"data": "test"}`), json.RawMessage(`{"data": "test"}`))
		return err
	}
}
