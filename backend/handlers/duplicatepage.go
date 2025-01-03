package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"go_notion/backend/api_error"
	"go_notion/backend/db"
	"go_notion/backend/page"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
)

// maintaining this list is important to ensure that the query is updated when the schema changes.
// we also have a test to ensure it is updated when the schema changes
// this is a copy of the columns in the pages table, excluding id, created_at, and updated_at
var PageColumns = []string{"created_by", "position", "text_title", "text_content", "title", "content"}

type DuplicatePageHandler struct {
	db         db.DB
	pageConfig *page.PageConfig
}

func NewDuplicatePageHandler(db db.DB, pageConfig *page.PageConfig) (*DuplicatePageHandler, error) {
	if db == nil || pageConfig == nil {
		return nil, fmt.Errorf("db and pageConfig cannot be nil")
	}
	return &DuplicatePageHandler{db, pageConfig}, nil
}

type DuplicatePageUrlInput struct {
	ID string `uri:"id" binding:"required,uuid"`
}

func (h *DuplicatePageHandler) DuplicatePage(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	userID, ok := c.Get("user_id")
	if !ok {
		c.Error(api_error.NewUnauthorizedError("not authorized to duplicate page", nil))
		return
	}
	userIdInt, ok := userID.(int64)
	if !ok {
		c.Error(api_error.NewUnauthorizedError("not authorized to duplicate page", fmt.Errorf("user id is not an integer")))
		return
	}

	var uri DuplicatePageUrlInput
	if err := c.ShouldBindUri(&uri); err != nil {
		c.Error(api_error.NewBadRequestError(err.Error(), err))
		return
	}

	pageID, err := uuid.FromString(uri.ID)
	if err != nil {
		c.Error(api_error.NewBadRequestError("invalid page id", err))
		return
	}

	var pageTitle sql.NullString
	var pageCreatedBy int64

	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to begin transaction", err))
		return
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		SELECT created_by, text_title FROM pages WHERE id = $1 AND created_by = $2
	`, pageID, userIdInt).Scan(&pageCreatedBy, &pageTitle)

	if errors.Is(err, pgx.ErrNoRows) {
		c.Error(api_error.NewNotFoundError("page not found", nil))
		return
	}

	if err != nil {
		c.Error(api_error.NewInternalServerError("error getting page to duplicate", err))
		return
	}
	var position float64

	err = tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(position), 0) FROM pages WHERE created_by = $1
	`, userIdInt).Scan(&position)
	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to duplicate page", err))
		return
	}

	position += float64(h.pageConfig.Spacing)

	columnsToSelect := []string{}
	for _, col := range PageColumns {
		if col == "position" {
			columnsToSelect = append(columnsToSelect, "$2")
		} else if col == "text_title" {
			columnsToSelect = append(columnsToSelect, "$3")
		} else {
			columnsToSelect = append(columnsToSelect, col)
		}
	}
	columnsToInsert := strings.Join(PageColumns, ", ")
	var newPageID uuid.UUID
	query := fmt.Sprintf(`
		INSERT INTO pages (%s)
		SELECT %s
		FROM pages 
		WHERE id = $1
		RETURNING id
	`, columnsToInsert, strings.Join(columnsToSelect, ", "))

	var newPageTitle string
	if pageTitle.Valid {
		newPageTitle = fmt.Sprintf("Copy of - %s", pageTitle.String)
	} else {
		newPageTitle = "Copy"
	}

	err = tx.QueryRow(ctx, query, pageID, position, newPageTitle).Scan(&newPageID)
	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to duplicate page", err))
		return
	}

	if err := tx.Commit(ctx); err != nil {
		c.Error(api_error.NewInternalServerError("failed to duplicate page", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "page duplicated successfully", "id": newPageID})

}

func (dp *DuplicatePageHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/pages/:id/duplicate", dp.DuplicatePage)
}
