package page

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
)

type Closure struct {
	AncestorID   uuid.UUID
	DescendantID uuid.UUID
	IsParent     bool
}

func InsertPageClosures(ctx context.Context, tx pgx.Tx, pageClosures []Closure) error {
	if len(pageClosures) == 0 {
		return nil
	}
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
	if err != nil {
		return fmt.Errorf("failed to insert page closures: %w", err)
	}
	return nil
}

func GetAncestors(ctx context.Context, tx pgx.Tx, pageIDs []uuid.UUID) (map[uuid.UUID][]Closure, error) {

	rows, err := tx.Query(ctx, `
	SELECT ancestor_id, descendant_id, is_parent FROM pages_closures WHERE descendant_id = ANY($1)
	`, pageIDs)

	if err != nil {
		return nil, err
	}
	ancestors := make(map[uuid.UUID][]Closure)
	for rows.Next() {
		var ancestorID uuid.UUID
		var descendantID uuid.UUID
		var isParent bool
		if err := rows.Scan(&ancestorID, &descendantID, &isParent); err != nil {
			return nil, err
		}
		ancestors[descendantID] = append(ancestors[descendantID], Closure{AncestorID: ancestorID, DescendantID: descendantID, IsParent: isParent})
	}
	rows.Close()

	return ancestors, nil
}
