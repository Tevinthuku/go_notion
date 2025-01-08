package page

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Page struct {
	ID          uuid.UUID        `json:"id"`
	Title       *json.RawMessage `json:"title"`
	Content     *json.RawMessage `json:"content"`
	TextTitle   *string          `json:"text_title"`
	TextContent *string          `json:"text_content"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

func GetPages(ctx context.Context, db *pgxpool.Pool, whereClause string, args ...any) ([]Page, error) {

	var query = "SELECT id, title, content, text_title, text_content, created_at, updated_at FROM pages "
	query += whereClause

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []Page
	for rows.Next() {
		var p Page
		if err := rows.Scan(&p.ID, &p.Title, &p.Content, &p.TextTitle, &p.TextContent, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		pages = append(pages, p)
	}
	return pages, nil
}
