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

	tx, err := rp.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to reorder page", fmt.Errorf("failed to begin transaction: %w", err)))
		return
	}
	defer tx.Rollback(ctx)
	var pagesBelongsToUser bool
	err = tx.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM pages 
			WHERE id IN ($1, $2) 
			GROUP BY created_by 
			HAVING created_by = $3 
			AND COUNT(*) = 2
		)`, uri.ID, input.NewParentId, userID).Scan(&pagesBelongsToUser)
	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to reorder page", fmt.Errorf("failed to check if page belongs to user: %w", err)))
		return
	}

	if !pagesBelongsToUser {
		c.Error(api_error.NewUnauthorizedError("not authorized to reorder page", nil))
		return
	}

	ancestors, err := getAncestors(ctx, tx, []uuid.UUID{pageID, input.NewParentId})
	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to reorder page", fmt.Errorf("failed to get ancestors: %w", err)))
		return
	}

	_, err = tx.Exec(ctx, `
		DELETE FROM pages_closures WHERE descendant_id = $1
	`, pageID)
	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to reorder page", fmt.Errorf("failed to delete ancestors of page: %w", err)))
		return
	}
	var newPageIdAncestors = make([]PageClosure, len(ancestors[input.NewParentId]))
	copy(newPageIdAncestors, ancestors[input.NewParentId])
	newPageIdAncestors = append(newPageIdAncestors, PageClosure{
		AncestorID:   input.NewParentId,
		DescendantID: pageID,
		IsParent:     true,
	})
	for i, _ := range newPageIdAncestors {
		newPageIdAncestors[i].DescendantID = pageID
	}

	err = insertPageClosures(ctx, tx, newPageIdAncestors)
	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to reorder page", fmt.Errorf("failed to insert new ancestors of page: %w", err)))
		return
	}

	descendants, err := getDescendants(ctx, tx, pageID)
	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to reorder page", fmt.Errorf("failed to get descendants: %w", err)))
		return
	}

	oldAncestors := ancestors[pageID]
	oldAncestorIds := make([]uuid.UUID, len(oldAncestors))
	for _, ancestor := range oldAncestors {
		oldAncestorIds = append(oldAncestorIds, ancestor.AncestorID)
	}

	_, err = tx.Exec(ctx, `
		    DELETE FROM pages_closures 
			WHERE (descendant_id, ancestor_id) IN (
				SELECT d, a 
				FROM unnest($1::uuid[]) d 
				CROSS JOIN unnest($2::uuid[]) a
			)
	`, descendants, oldAncestorIds)

	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to reorder page", fmt.Errorf("failed to delete old ancestors of page: %w", err)))
		return
	}

	var newDescendantClosuresToInsert = make([]PageClosure, len(descendants))
	for _, descendant := range descendants {
		var newAncestors = make([]PageClosure, len(ancestors[input.NewParentId]))
		copy(newAncestors, ancestors[input.NewParentId])
		for i := range newAncestors {
			newAncestors[i].DescendantID = descendant
		}
		newDescendantClosuresToInsert = append(newDescendantClosuresToInsert, newAncestors...)
	}

	err = insertPageClosures(ctx, tx, newDescendantClosuresToInsert)
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
