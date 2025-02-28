package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"go_notion/backend/api_error"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UpdatePageHandler struct {
	db *pgxpool.Pool
}

func NewUpdatePageHandler(db *pgxpool.Pool) (*UpdatePageHandler, error) {
	if db == nil {
		return nil, fmt.Errorf("db cannot be nil")
	}
	return &UpdatePageHandler{db}, nil
}

type UpdatePageUri struct {
	ID string `uri:"id" binding:"required,uuid"`
}

type UpdatePageInput struct {
	TitleText   string          `json:"title_text" binding:"required"`
	ContentText string          `json:"content_text" binding:"required"`
	RawTitle    json.RawMessage `json:"raw_title" binding:"required"`
	RawContent  json.RawMessage `json:"raw_content" binding:"required"`
}

func (up *UpdatePageHandler) UpdatePage(c *gin.Context) {
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

	var uri UpdatePageUri
	if err := c.ShouldBindUri(&uri); err != nil {
		c.Error(api_error.NewBadRequestError(err.Error(), err))
		return
	}

	var input UpdatePageInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(api_error.NewBadRequestError(err.Error(), err))
		return
	}

	pageID, err := uuid.FromString(uri.ID)
	if err != nil {
		c.Error(api_error.NewBadRequestError(err.Error(), err))
		return
	}

	cmd, err := up.db.Exec(ctx, `
		UPDATE pages SET text_title = $1, text_content = $2, title = $3, content = $4 WHERE id = $5 AND created_by = $6
	`, input.TitleText, input.ContentText, input.RawTitle, input.RawContent, pageID, userIdInt)
	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to update page", err))
		return
	}

	if cmd.RowsAffected() == 0 {
		c.Error(api_error.NewNotFoundError("page not found", nil))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "page updated successfully"})
}

func (up *UpdatePageHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.PUT("/pages/:id", up.UpdatePage)
}
