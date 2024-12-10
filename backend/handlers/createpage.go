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

	var position float64

	err := np.db.QueryRow(ctx, `
		SELECT COALESCE(MAX(position), 0) FROM pages WHERE created_by = $1
	`, userIdInt).Scan(&position)
	if err != nil {
		c.Error(api_error.NewInternalServerError("internal server error", err))
		return
	}

	var pageID uuid.UUID
	position += float64(np.pageConfig.Spacing)
	err = np.db.QueryRow(ctx, `
		INSERT INTO pages (created_by, position) VALUES ($1, $2) RETURNING id
	`, userID, position).Scan(&pageID)

	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to create page", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"page_id": pageID})

}

func (np *CreatePageHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/pages", np.CreatePage)
}
