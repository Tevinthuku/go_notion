package handlers

import (
	"context"
	"fmt"
	"go_notion/backend/api_error"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DeletePageHandler struct {
	db *pgxpool.Pool
}

func NewDeletePageHandler(db *pgxpool.Pool) (*DeletePageHandler, error) {
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

	userIdInt, ok := userID.(int64)
	if !ok {
		c.Error(api_error.NewUnauthorizedError("not authorized to delete page", fmt.Errorf("user id is not an integer")))
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

	tx, err := dp.db.BeginTx(ctx, pgx.TxOptions{})
	defer tx.Rollback(ctx)
	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to delete page", err))
		return
	}

	// Delete nested pages first. If we delete the parent page first, its pages_closures records
	// will be deleted, losing the information about which pages were nested under it. This would
	// leave the child pages orphaned in the database.
	_, err = tx.Exec(ctx, `
		DELETE FROM pages WHERE id IN (
			SELECT descendant_id FROM pages_closures WHERE ancestor_id = $1
		)
	`, pageID)

	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to delete nested pages", err))
		return
	}

	cmd, err := tx.Exec(ctx, `
		DELETE FROM pages WHERE id = $1::uuid AND created_by = $2
	`, pageID, userIdInt)
	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to delete page", err))
		return
	}

	if cmd.RowsAffected() == 0 {
		c.Error(api_error.NewNotFoundError("page not found", nil))
		return
	}

	err = tx.Commit(ctx)
	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to delete page", err))
		return
	}

	c.Status(http.StatusNoContent)
}

func (dp *DeletePageHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.DELETE("/pages/:id", dp.DeletePage)
}
