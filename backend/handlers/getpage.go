package handlers

import (
	"context"
	"fmt"
	"go_notion/backend/api_error"
	"go_notion/backend/page"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GetPageHandler struct {
	db *pgxpool.Pool
}

func NewGetPageHandler(db *pgxpool.Pool) (*GetPageHandler, error) {
	if db == nil {
		return nil, fmt.Errorf("db cannot be nil")
	}
	return &GetPageHandler{db}, nil
}

type GetPageUri struct {
	ID string `uri:"id" binding:"required,uuid"`
}

func (gp *GetPageHandler) GetPage(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	userID, ok := c.Get("user_id")
	if !ok {
		c.Error(api_error.NewUnauthorizedError("not authorized to update page", nil))
		return
	}

	userIdInt, ok := userID.(int64)
	if !ok {
		c.Error(api_error.NewUnauthorizedError("not authorized to update page", fmt.Errorf("user id is not an integer")))
		return
	}

	var uri GetPageUri
	if err := c.ShouldBindUri(&uri); err != nil {
		c.Error(api_error.NewBadRequestError(err.Error(), err))
		return
	}

	pageID, err := uuid.FromString(uri.ID)
	if err != nil {
		c.Error(api_error.NewBadRequestError(err.Error(), err))
		return
	}

	var page page.Page

	err = gp.db.QueryRow(ctx, `
		SELECT id, title, content, text_title, text_content, created_at, updated_at FROM pages WHERE id = $1 AND created_by = $2
	`, pageID, userIdInt).Scan(&page.ID, &page.Title, &page.Content, &page.TextTitle, &page.TextContent, &page.CreatedAt, &page.UpdatedAt)

	if err == pgx.ErrNoRows {
		c.Error(api_error.NewNotFoundError("page not found", nil))
		return
	} else if err != nil {
		c.Error(api_error.NewInternalServerError("error getting page", err))
		return
	}

	c.JSON(http.StatusOK, PageResponse{Data: page})
}

type PageResponse struct {
	Data page.Page `json:"data"`
}

func (gp *GetPageHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/pages/:id", gp.GetPage)
}
