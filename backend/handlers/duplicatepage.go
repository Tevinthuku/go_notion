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
var PageColumns = []string{"created_by", "position", "text_title", "text_content", "title", "content", "is_top_level"}

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

	pageAncestors := []PageClosure{}

	rows, err := tx.Query(ctx, `
	SELECT ancestor_id, is_parent FROM pages_closures WHERE descendant_id = $1
	`, pageID)
	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to duplicate page", err))
		return
	}
	for rows.Next() {
		var ancestorID uuid.UUID
		var isParent bool
		if err := rows.Scan(&ancestorID, &isParent); err != nil {
			c.Error(api_error.NewInternalServerError("failed to duplicate page", err))
			return
		}
		pageAncestors = append(pageAncestors, PageClosure{AncestorID: ancestorID, DescendantID: newPageID, IsParent: isParent})
	}
	rows.Close()

	if len(pageAncestors) > 0 {
		err = insertPageClosures(ctx, tx, pageAncestors)
		if err != nil {
			c.Error(api_error.NewInternalServerError("failed to duplicate page", err))
			return
		}
	}

	err = h.duplicateDescendants(ctx, tx, pageID, newPageID, position)
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

func (h *DuplicatePageHandler) duplicateDescendants(ctx context.Context, tx pgx.Tx, pageID uuid.UUID, newPageID uuid.UUID, lastPagePosition float64) error {

	var descendantsWithAllAncestors []PageClosure

	rows, err := tx.Query(ctx, `
	    WITH descendants AS (
            SELECT descendant_id 
            FROM pages_closures 
            WHERE ancestor_id = $1
        )
        SELECT DISTINCT pc.ancestor_id, pc.descendant_id, pc.is_parent
        FROM pages_closures pc
        INNER JOIN descendants d ON d.descendant_id = pc.descendant_id
	`, pageID)
	if err != nil {
		return err
	}

	for rows.Next() {
		var ancestorID uuid.UUID
		var isParent bool
		var descendantID uuid.UUID
		if err := rows.Scan(&ancestorID, &descendantID, &isParent); err != nil {
			return err
		}
		descendantsWithAllAncestors = append(descendantsWithAllAncestors, PageClosure{AncestorID: ancestorID, DescendantID: descendantID, IsParent: isParent})
	}
	rows.Close()

	if len(descendantsWithAllAncestors) == 0 {
		return nil
	}

	mappingOfDescendantsWithAllAncestors := make(map[uuid.UUID][]PageClosure)
	uniqueDescendants := make(map[uuid.UUID]struct{})
	for _, closure := range descendantsWithAllAncestors {
		uniqueDescendants[closure.DescendantID] = struct{}{}
		mappingOfDescendantsWithAllAncestors[closure.DescendantID] = append(mappingOfDescendantsWithAllAncestors[closure.DescendantID], closure)
	}

	columnsToInsert := strings.Join(PageColumns, ", ")

	columnsToSelect := []string{}
	for _, col := range PageColumns {
		if col == "position" {
			columnsToSelect = append(columnsToSelect, "$2")
		} else {
			columnsToSelect = append(columnsToSelect, col)
		}
	}

	mappingOfOldDescendantToNewDescendantId := make(map[uuid.UUID]uuid.UUID)
	for oldDescendantID := range uniqueDescendants {
		query := fmt.Sprintf(`
			INSERT INTO pages (%s)
			SELECT %s
			FROM pages 
			WHERE id = $1
			RETURNING id
		`, columnsToInsert, strings.Join(columnsToSelect, ", "))
		lastPagePosition += float64(h.pageConfig.Spacing)
		var newPageID uuid.UUID
		err = tx.QueryRow(ctx, query, oldDescendantID, lastPagePosition).Scan(&newPageID)
		if err != nil {
			return fmt.Errorf("failed to duplicate page: %w", err)
		}
		mappingOfOldDescendantToNewDescendantId[oldDescendantID] = newPageID
	}

	var newPageClosureInserts = make([]PageClosure, 0, len(descendantsWithAllAncestors))

	for descendantId, ancestors := range mappingOfDescendantsWithAllAncestors {
		newDescendantID, ok := mappingOfOldDescendantToNewDescendantId[descendantId]

		if !ok {
			return fmt.Errorf("failed to find new descendant id for descendant %s", descendantId)
		}

		for _, closure := range ancestors {
			ancestorID := closure.AncestorID

			if ancestorID == pageID {
				newPageClosureInserts = append(newPageClosureInserts, PageClosure{AncestorID: newPageID, DescendantID: newDescendantID, IsParent: closure.IsParent})
				continue
			}

			if newAncestorID, ok := mappingOfOldDescendantToNewDescendantId[ancestorID]; ok {
				newPageClosureInserts = append(newPageClosureInserts, PageClosure{AncestorID: newAncestorID, DescendantID: newDescendantID, IsParent: closure.IsParent})
				continue
			}
			newPageClosureInserts = append(newPageClosureInserts, PageClosure{AncestorID: ancestorID, DescendantID: newDescendantID, IsParent: closure.IsParent})

		}
	}

	err = insertPageClosures(ctx, tx, newPageClosureInserts)
	if err != nil {
		return fmt.Errorf("failed to insert new page closures: %w", err)
	}

	return nil
}

func (dp *DuplicatePageHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/pages/:id/duplicate", dp.DuplicatePage)
}

type PageClosure struct {
	AncestorID   uuid.UUID
	DescendantID uuid.UUID
	IsParent     bool
}

func insertPageClosures(ctx context.Context, tx pgx.Tx, pageClosures []PageClosure) error {
	valueStrings := make([]string, 0, len(pageClosures))
	valueArgs := make([]interface{}, 0, len(pageClosures)*3)
	for i, closure := range pageClosures {
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d)", i*3+1, i*3+2, i*3+3))
		valueArgs = append(valueArgs, closure.AncestorID, closure.DescendantID, closure.IsParent)
	}

	query := fmt.Sprintf(`
		INSERT INTO pages_closures (ancestor_id, descendant_id, is_parent) VALUES %s
	`, strings.Join(valueStrings, ","))

	_, err := tx.Exec(ctx, query, valueArgs...)
	return err
}
