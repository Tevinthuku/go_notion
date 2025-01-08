package page

import (
	"encoding/json"
	"time"

	"github.com/gofrs/uuid/v5"
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
