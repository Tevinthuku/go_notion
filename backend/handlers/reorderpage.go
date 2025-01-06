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
	"github.com/jackc/pgx/v5"
)

type ReorderPageHandler struct {
	db db.DB
}

func NewReorderPageHandler(db db.DB) (*ReorderPageHandler, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	return &ReorderPageHandler{db: db}, nil
}

type ReorderPageInput struct {
	NewParentId uuid.UUID `json:"new_parent_id" binding:"required,uuid"`
}

type ReorderPageUri struct {
	ID string `uri:"id" binding:"required,uuid"`
}

func (rp *ReorderPageHandler) ReorderPage(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	userID, ok := c.Get("user_id")
	if !ok {
		c.Error(api_error.NewUnauthorizedError("not authorized to reorder page", nil))
		return
	}

	userIdInt, ok := userID.(int64)
	if !ok {
		c.Error(api_error.NewUnauthorizedError("not authorized to reorder page", fmt.Errorf("user id is not an integer")))
		return
	}

	var uri ReorderPageUri
	if err := c.ShouldBindUri(&uri); err != nil {
		c.Error(api_error.NewBadRequestError(err.Error(), err))
		return
	}

	pageID, err := uuid.FromString(uri.ID)
	if err != nil {
		c.Error(api_error.NewBadRequestError(err.Error(), err))
		return
	}

	var input ReorderPageInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(api_error.NewBadRequestError(err.Error(), err))
		return
	}

	apiErr := rp.validateInput(ctx, input, pageID, userIdInt)
	if apiErr != nil {
		c.Error(apiErr)
		return
	}

	tx, err := rp.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to reorder page", fmt.Errorf("failed to begin transaction: %w", err)))
		return
	}
	defer tx.Rollback(ctx)

	ancestors, err := getAncestorIds(ctx, tx, []uuid.UUID{pageID, input.NewParentId})
	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to reorder page", fmt.Errorf("failed to get ancestors: %w", err)))
		return
	}

	_, err = tx.Exec(ctx, `
	   DELETE FROM pages_closures WHERE descendant_id = $1
	`, pageID)

	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to reorder page", fmt.Errorf("failed to delete old ancestors of page: %w", err)))
		return
	}

	descendantIds, err := getDescendants(ctx, tx, pageID)
	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to reorder page", fmt.Errorf("failed to get descendants: %w", err)))
		return
	}

	// we need to delete all the closures that are between the descendants and the current page ancestors
	// we will re-insert new ancestors for these descendants below
	existingPageAncestorIds := ancestors[pageID]
	_, err = tx.Exec(ctx, `
		    DELETE FROM pages_closures 
			WHERE (descendant_id, ancestor_id) IN (
				SELECT d, a 
				FROM unnest($1::uuid[]) d 
				CROSS JOIN unnest($2::uuid[]) a
			)
		`, descendantIds, existingPageAncestorIds)

	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to reorder page", fmt.Errorf("failed to delete old ancestors of page: %w", err)))
		return
	}

	// the plus one is for the new parent closure
	var newAncestorsForCurrentPage []PageClosure = make([]PageClosure, 0, len(ancestors[input.NewParentId])+1)
	for _, ancestor := range ancestors[input.NewParentId] {
		newAncestorsForCurrentPage = append(newAncestorsForCurrentPage, PageClosure{
			AncestorID:   ancestor,
			DescendantID: pageID,
			IsParent:     false,
		})
	}
	// since the page is being moved to a new parent, we need to add a new closure
	newAncestorsForCurrentPage = append(newAncestorsForCurrentPage, PageClosure{
		AncestorID:   input.NewParentId,
		DescendantID: pageID,
		IsParent:     true,
	})

	var newAncestorIdsForDescendants []uuid.UUID = make([]uuid.UUID, 0, len(newAncestorsForCurrentPage))
	for _, ancestor := range newAncestorsForCurrentPage {
		newAncestorIdsForDescendants = append(newAncestorIdsForDescendants, ancestor.AncestorID)
	}

	descendantClosures := generateAncestorClosuresForDescendants(newAncestorIdsForDescendants, descendantIds)
	closures := append(newAncestorsForCurrentPage, descendantClosures...)
	err = insertPageClosures(ctx, tx, closures)
	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to reorder page", fmt.Errorf("failed to insert new ancestors of page: %w", err)))
		return
	}

	err = tx.Commit(ctx)
	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to reorder page", fmt.Errorf("failed to commit transaction: %w", err)))
		return
	}

	c.Status(http.StatusOK)

}

func (rp *ReorderPageHandler) validateInput(ctx context.Context, input ReorderPageInput, pageID uuid.UUID, userID int64) *api_error.ApiError {
	// ensure new parent is not a descendant of the current page
	var willGenerateCyclicClosure bool
	err := rp.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM pages_closures WHERE ancestor_id = $1 AND descendant_id = $2
		)
	`, pageID, input.NewParentId).Scan(&willGenerateCyclicClosure)
	if err != nil {
		return api_error.NewInternalServerError("failed to reorder page", fmt.Errorf("failed to check if new parent is a descendant: %w", err))
	}

	if willGenerateCyclicClosure {
		return api_error.NewBadRequestError("cannot add page to nested page", nil)
	}

	var pagesBelongsToUser bool
	err = rp.db.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM pages 
			WHERE id IN ($1, $2) 
			GROUP BY created_by 
			HAVING created_by = $3 
			AND COUNT(*) = 2
		)`, pageID, input.NewParentId, userID).Scan(&pagesBelongsToUser)

	if err != nil {
		return api_error.NewInternalServerError("failed to reorder page", fmt.Errorf("failed to check if page belongs to user: %w", err))
	}

	if !pagesBelongsToUser {
		return api_error.NewUnauthorizedError("not authorized to reorder page", nil)
	}

	return nil
}

func getDescendants(ctx context.Context, tx pgx.Tx, pageID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := tx.Query(ctx, `
		SELECT descendant_id FROM pages_closures WHERE ancestor_id = $1
	`, pageID)

	if err != nil {
		return nil, fmt.Errorf("failed to get descendants: %w", err)
	}
	defer rows.Close()

	var descendants []uuid.UUID
	for rows.Next() {
		var descendant uuid.UUID
		if err := rows.Scan(&descendant); err != nil {
			return nil, fmt.Errorf("failed to scan descendant: %w", err)
		}
		descendants = append(descendants, descendant)
	}

	return descendants, nil
}

func getAncestorIds(ctx context.Context, tx pgx.Tx, pageIDs []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) {
	ancestors, err := getAncestors(ctx, tx, pageIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get ancestors: %w", err)
	}
	ancestorIds := make(map[uuid.UUID][]uuid.UUID, len(ancestors))
	for ancestor, closures := range ancestors {
		ancestor_ids := make([]uuid.UUID, 0, len(closures))
		for _, closure := range closures {
			ancestor_ids = append(ancestor_ids, closure.AncestorID)
		}
		ancestorIds[ancestor] = ancestor_ids
	}

	return ancestorIds, nil
}

func generateAncestorClosuresForDescendants(newAncestorIds []uuid.UUID, descendants []uuid.UUID) []PageClosure {
	var newClosureInserts = make([]PageClosure, 0, len(descendants)*len(newAncestorIds))
	for _, descendantId := range descendants {
		var ancestors = make([]PageClosure, 0, len(newAncestorIds))
		for _, ancestorId := range newAncestorIds {
			ancestors = append(ancestors, PageClosure{
				AncestorID:   ancestorId,
				DescendantID: descendantId,
				IsParent:     false,
			})
		}
		newClosureInserts = append(newClosureInserts, ancestors...)
	}
	return newClosureInserts
}

func (rp *ReorderPageHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/pages/:id/reorder", rp.ReorderPage)
}
