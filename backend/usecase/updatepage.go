package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"go_notion/backend/api_error"
	"go_notion/backend/db"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
)

type UpdatePage struct {
	db db.DB
}

func NewUpdatePageUseCase(db db.DB) (*UpdatePage, error) {
	if db == nil {
		return nil, fmt.Errorf("db cannot be nil")
	}
	return &UpdatePage{db}, nil
}

type UpdatePageUri struct {
	ID string `uri:"id" binding:"required,uuid"`
}

type UpdatePageInput struct {
	TitleText   string          `json:"title_text"`
	ContentText string          `json:"content_text"`
	RawTitle    json.RawMessage `json:"raw_title"`
	RawContent  json.RawMessage `json:"raw_content"`
}

func (up *UpdatePage) UpdatePage(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	userID, ok := c.Get("user_id")
	if !ok {
		c.Error(api_error.NewUnauthorizedError("not authorized to update page", nil))
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
		UPDATE pages SET text_title = $1, text_content = $2, raw_title = $3, raw_content = $4 WHERE id = $5 AND created_by = $6
	`, input.TitleText, input.ContentText, input.RawTitle, input.RawContent, pageID, userID)
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

func (up *UpdatePage) RegisterRoutes(router *gin.RouterGroup) {
	router.PUT("/pages/:id", up.UpdatePage)
}
