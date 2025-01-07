package db

import (
	"context"
	"encoding/json"
	"fmt"
	"go_notion/backend/auth"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
)

func InsertTestUserFixture(conn *pgx.Conn) error {
	return InsertTestUserWithData("test@test.com", "test", "test")(conn)
}

func InsertTestUserWithData(email, username, password string) Fixture {
	return func(conn *pgx.Conn) error {
		hashedPassword, err := auth.HashPassword(password)
		if err != nil {
			return fmt.Errorf("error hashing password: %w", err)
		}
		_, err = conn.Exec(context.Background(), `
		INSERT INTO users (email, username, password) VALUES ($1, $2, $3)
	`, email, username, hashedPassword)
		return err
	}
}

func InsertTestPageFixture(page_id uuid.UUID, user_id int64) Fixture {
	return func(conn *pgx.Conn) error {
		err := insertPageFixture(conn, page_id, user_id, 1, true)
		return err
	}
}

func InsertTestPageFixtureWithParent(page_id uuid.UUID, parent_id uuid.UUID, user_id int64) Fixture {
	return func(conn *pgx.Conn) error {
		err := InsertTestPageFixture(parent_id, user_id)(conn)
		if err != nil {
			return err
		}

		err = insertPageFixture(conn, page_id, user_id, 2, false)
		if err != nil {
			return err
		}

		_, err = conn.Exec(context.Background(), `
			INSERT INTO pages_closures (ancestor_id, descendant_id, is_parent) VALUES ($1, $2, true)
		`, parent_id, page_id)
		return err
	}
}

func InsertTestPageFixtureWithPosition(page_id uuid.UUID, user_id int64, position int) Fixture {
	return func(conn *pgx.Conn) error {
		return insertPageFixture(conn, page_id, user_id, position, true)
	}
}

func insertPageFixture(conn *pgx.Conn, page_id uuid.UUID, user_id int64, position int, is_top_level bool) error {
	_, err := conn.Exec(context.Background(), `
	INSERT INTO pages (id, created_by, position, text_title, text_content, title, content, is_top_level) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`, page_id, user_id, position, "test", "test", json.RawMessage(`{"data": "test"}`), json.RawMessage(`{"data": "test"}`), is_top_level)
	return err
}
