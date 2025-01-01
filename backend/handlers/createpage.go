package handlers

import (
	"context"
	"fmt"
	"go_notion/backend/api_error"
	"go_notion/backend/db"
	"go_notion/backend/page"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"github.com/gofrs/uuid/v5"
)

type CreatePageHandler struct {
	db         db.DB
	pageConfig *page.PageConfig
}

func NewCreatePageHandler(db db.DB, pageConfig *page.PageConfig) (*CreatePageHandler, error) {
	if db == nil {
		return nil, fmt.Errorf("db cannot be nil")
	}
	if pageConfig == nil {
		return nil, fmt.Errorf("page config cannot be nil")
	}
	return &CreatePageHandler{db, pageConfig}, nil
}

type CreatePageInput struct {
	ParentID *uuid.UUID `json:"parent_id"`
}

func (np *CreatePageHandler) CreatePage(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	userID, ok := c.Get("user_id")
	if !ok {
		c.Error(api_error.NewUnauthorizedError("not authorized to create page", nil))
		return
	}
	userIdInt, ok := userID.(int64)
	if !ok {
		c.Error(api_error.NewUnauthorizedError("not authorized to create page", fmt.Errorf("user id is not an integer")))
		return
	}

	var input CreatePageInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(api_error.NewBadRequestError(err.Error(), err))
		return
	}

	var isTopLevel bool = true
	if input.ParentID != nil {
		isTopLevel = false
	}

	tx, err := np.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to create page", err))
		return
	}

	defer tx.Rollback(ctx)

	var position float64

	err = tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(position), 0) FROM pages WHERE created_by = $1
	`, userIdInt).Scan(&position)
	if err != nil {
		c.Error(api_error.NewInternalServerError("internal server error", err))
		return
	}

	var pageID uuid.UUID
	position += float64(np.pageConfig.Spacing)
	err = tx.QueryRow(ctx, `
		INSERT INTO pages (created_by, position, is_top_level) VALUES ($1, $2, $3) RETURNING id
	`, userIdInt, position, isTopLevel).Scan(&pageID)

	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to create page", err))
		return
	}

	if input.ParentID != nil {
		_, err = tx.Exec(ctx, `
			INSERT INTO pages_closures (ancestor_id, descendant_id, is_parent) 
			SELECT ancestor_id, $2 as descendant_id,
			CASE WHEN ancestor_id = $2 THEN true ELSE false END as is_parent
			FROM pages_closures
			WHERE descendant_id = $1

			UNION ALL

			SELECT $1 as ancestor_id, $2 as descendant_id, true as is_parent
		`, input.ParentID, pageID)
		if err != nil {
			c.Error(api_error.NewInternalServerError("failed to link page to parent", err))
			return
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to create page", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"page_id": pageID})

}

func (np *CreatePageHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/pages", np.CreatePage)
}
