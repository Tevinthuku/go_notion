package handlers

import (
	"context"
	"fmt"
	"go_notion/backend/api_error"
	"go_notion/backend/db"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
)

type DeletePageHandler struct {
	db db.DB
}

func NewDeletePageHandler(db db.DB) (*DeletePageHandler, error) {
	if db == nil {
		return nil, fmt.Errorf("db cannot be nil")
	}
	return &DeletePageHandler{db}, nil
}

type DeletePageUri struct {
	ID string `uri:"id" binding:"required,uuid"`
}

func (dp *DeletePageHandler) DeletePage(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	userID, ok := c.Get("user_id")
	if !ok {
		c.Error(api_error.NewUnauthorizedError("not authorized to delete page", nil))
		return
	}

	var uri DeletePageUri
	if err := c.ShouldBindUri(&uri); err != nil {
		c.Error(api_error.NewBadRequestError(err.Error(), err))
		return
	}

	pageID, err := uuid.FromString(uri.ID)
	if err != nil {
		c.Error(api_error.NewBadRequestError(err.Error(), err))
		return
	}

	cmd, err := dp.db.Exec(ctx, `
		DELETE FROM pages WHERE id = $1 AND created_by = $2
	`, pageID, userID)
	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to delete page", err))
		return
	}

	if cmd.RowsAffected() == 0 {
		c.Error(api_error.NewNotFoundError("page not found", nil))
		return
	}

	c.Status(http.StatusNoContent)
}

func (dp *DeletePageHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.DELETE("/pages/:id", dp.DeletePage)
}
