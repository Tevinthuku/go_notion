package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"go_notion/backend/api_error"
	"go_notion/backend/page"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// maintaining this list is important to ensure that the query is updated when the schema changes.
// this is a copy of the columns in the pages table, excluding created_at, and updated_at because these will be auto generated
// Note: TestPageColumnsMatchSchema in backend/handlers/duplicatepage_test.go ensures this list stays in sync with the database schema
var PageColumns = []string{"id", "created_by", "position", "text_title", "text_content", "title", "content", "is_top_level"}

type DuplicatePageHandler struct {
	db         *pgxpool.Pool
	pageConfig *page.PageConfig
}

func NewDuplicatePageHandler(db *pgxpool.Pool, pageConfig *page.PageConfig) (*DuplicatePageHandler, error) {
	if db == nil {
		return nil, fmt.Errorf("db cannot be nil")
	}
	if pageConfig == nil {
		return nil, fmt.Errorf("pageConfig cannot be nil")
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

	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to begin transaction", err))
		return
	}
	defer tx.Rollback(ctx)

	targetPage, apiErr := h.duplicateTargetPage(ctx, tx, pageID, userIdInt)
	if apiErr != nil {
		c.Error(apiErr)
		return
	}

	err = h.duplicateDescendants(ctx, tx, pageID, targetPage.ID, targetPage.Position+float64(h.pageConfig.Spacing))
	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to duplicate page", err))
		return
	}

	if err := tx.Commit(ctx); err != nil {
		c.Error(api_error.NewInternalServerError("failed to duplicate page", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "page duplicated successfully", "id": targetPage.ID})

}

type DuplicatedPage struct {
	ID       uuid.UUID
	Position float64
}

func (h *DuplicatePageHandler) duplicateTargetPage(ctx context.Context, tx pgx.Tx, pageID uuid.UUID, userIdInt int64) (*DuplicatedPage, *api_error.ApiError) {
	var pageTitle sql.NullString
	err := tx.QueryRow(ctx, `
		SELECT text_title FROM pages WHERE id = $1 AND created_by = $2
	`, pageID, userIdInt).Scan(&pageTitle)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, api_error.NewNotFoundError("page not found", nil)
	}

	if err != nil {
		return nil, api_error.NewInternalServerError("error getting page to duplicate", err)
	}
	var position float64

	err = tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(position), 0) FROM pages WHERE created_by = $1
	`, userIdInt).Scan(&position)
	if err != nil {
		return nil, api_error.NewInternalServerError("failed to duplicate page", err)
	}

	m, err := h.duplicatePages(ctx, tx, []uuid.UUID{pageID}, position)
	if err != nil {
		return nil, api_error.NewInternalServerError("failed to duplicate page", err)
	}
	newPageID, ok := m[pageID]
	if !ok {
		return nil, api_error.NewInternalServerError("failed to duplicate page", fmt.Errorf("failed to find new page id for page %s", pageID))
	}

	var newPageTitle string
	if pageTitle.Valid {
		newPageTitle = fmt.Sprintf("Copy of - %s", pageTitle.String)
	} else {
		newPageTitle = "Copy"
	}

	_, err = tx.Exec(ctx, `
		UPDATE pages SET text_title = $1 WHERE id = $2
	`, newPageTitle, newPageID)
	if err != nil {
		return nil, api_error.NewInternalServerError("failed to change page title", err)
	}

	ancestors, err := page.GetAncestors(ctx, tx, []uuid.UUID{pageID})
	if err != nil {
		return nil, api_error.NewInternalServerError("failed to duplicate page", err)
	}
	pageAncestors := ancestors[pageID]

	// we need to replace pageId with newPageId in the pageAncestors
	for i := range pageAncestors {
		pageAncestors[i].DescendantID = newPageID
	}

	err = page.InsertPageClosures(ctx, tx, pageAncestors)
	if err != nil {
		return nil, api_error.NewInternalServerError("failed to duplicate page", err)
	}

	return &DuplicatedPage{ID: newPageID, Position: position}, nil
}

func (h *DuplicatePageHandler) duplicateDescendants(ctx context.Context, tx pgx.Tx, pageID uuid.UUID, newPageID uuid.UUID, lastPagePosition float64) error {

	mappingOfDescendantsWithAllAncestors, err := page.GetAllDescendants(ctx, tx.Conn(), []uuid.UUID{pageID})
	if err != nil {
		return fmt.Errorf("failed to get all descendants: %w", err)
	}
	uniqueDescendants := make(map[uuid.UUID]struct{})
	for descendantID, _ := range mappingOfDescendantsWithAllAncestors {
		uniqueDescendants[descendantID] = struct{}{}
	}

	var descendantIds []uuid.UUID
	for descendantID := range uniqueDescendants {
		descendantIds = append(descendantIds, descendantID)
	}

	mappingOfOldDescendantToNewDescendantId, err := h.duplicatePages(ctx, tx, descendantIds, lastPagePosition)
	if err != nil {
		return fmt.Errorf("failed to duplicate descendant pages: %w", err)
	}

	var newPageClosureInserts []page.Closure

	for descendantId, ancestors := range mappingOfDescendantsWithAllAncestors {
		newDescendantID, ok := mappingOfOldDescendantToNewDescendantId[descendantId]

		if !ok {
			return fmt.Errorf("failed to find new descendant id for descendant %s", descendantId)
		}

		for _, closure := range ancestors {
			ancestorID := closure.AncestorID

			// pageID is the page that we are duplicating, so we need to update the page closure to point to the new page
			if ancestorID == pageID {
				newPageClosureInserts = append(newPageClosureInserts, page.Closure{AncestorID: newPageID, DescendantID: newDescendantID, IsParent: closure.IsParent})
				continue
			}

			// if the ancestor is a descendant of the page we are duplicating, we need to update the page closure to point to the new descendant page
			if newAncestorID, ok := mappingOfOldDescendantToNewDescendantId[ancestorID]; ok {
				newPageClosureInserts = append(newPageClosureInserts, page.Closure{AncestorID: newAncestorID, DescendantID: newDescendantID, IsParent: closure.IsParent})
				continue
			}
			newPageClosureInserts = append(newPageClosureInserts, page.Closure{AncestorID: ancestorID, DescendantID: newDescendantID, IsParent: closure.IsParent})

		}
	}

	err = page.InsertPageClosures(ctx, tx, newPageClosureInserts)
	if err != nil {
		return fmt.Errorf("failed to insert new page closures: %w", err)
	}

	return nil
}

func (h *DuplicatePageHandler) duplicatePages(ctx context.Context, tx pgx.Tx, pageIds []uuid.UUID, lastPagePosition float64) (map[uuid.UUID]uuid.UUID, error) {
	if len(pageIds) == 0 {
		return nil, nil
	}
	valueStrings := make([]string, 0, len(pageIds))
	valueArgs := make([]interface{}, 0, len(pageIds))
	mappingOfOldPageIdToNewPageId := make(map[uuid.UUID]uuid.UUID, len(pageIds))
	for i, pageId := range pageIds {
		newPageId, err := uuid.NewV4()
		if err != nil {
			return nil, fmt.Errorf("failed to generate new page id: %w", err)
		}
		mappingOfOldPageIdToNewPageId[pageId] = newPageId
		lastPagePosition += float64(h.pageConfig.Spacing)
		valueStrings = append(valueStrings, fmt.Sprintf("($%d::uuid, $%d::uuid, $%d::float8)", i*3+1, i*3+2, i*3+3))
		valueArgs = append(valueArgs, pageId, newPageId, lastPagePosition)
	}

	columnsToInsert := []string{}
	columnsToSelect := []string{}
	for _, col := range PageColumns {
		if col == "id" {
			columnsToSelect = append(columnsToSelect, "v.new_page_id as id")
		} else if col == "position" {
			columnsToSelect = append(columnsToSelect, "v.new_position as position")
		} else {
			columnsToSelect = append(columnsToSelect, col)
		}
		columnsToInsert = append(columnsToInsert, col)
	}

	query := fmt.Sprintf(`
		INSERT INTO pages (%s)
		SELECT %s
		FROM pages
		CROSS JOIN (VALUES %s) AS v(id, new_page_id, new_position)
		WHERE pages.id = v.id
	`, strings.Join(columnsToInsert, ", "), strings.Join(columnsToSelect, ", "), strings.Join(valueStrings, ","))

	_, err := tx.Exec(ctx, query, valueArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to duplicate pages in bulk: %w", err)
	}

	return mappingOfOldPageIdToNewPageId, nil

}

func (dp *DuplicatePageHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/pages/:id/duplicate", dp.DuplicatePage)
}
