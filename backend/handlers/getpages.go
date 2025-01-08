package handlers

import (
	"context"
	"fmt"
	"go_notion/backend/api_error"
	"go_notion/backend/page"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GetPagesHandler struct {
	db *pgxpool.Pool
}

func NewGetPagesHandler(db *pgxpool.Pool) (*GetPagesHandler, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	return &GetPagesHandler{db: db}, nil
}

func (gp *GetPagesHandler) GetPages(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	userID, ok := c.Get("user_id")
	if !ok {
		c.Error(api_error.NewUnauthorizedError("not authorized to get page", nil))
		return
	}

	userIdInt, ok := userID.(int64)
	if !ok {
		c.Error(api_error.NewUnauthorizedError("not authorized to update page", fmt.Errorf("user id is not an integer")))
		return
	}

	params, err := getPagesParamsFromQuery(c)
	if err != nil {
		c.Error(api_error.NewBadRequestError(err.Error(), err))
		return
	}

	pages, err := gp.getTopLevelPages(ctx, params, userIdInt)
	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to get pages", err))
		return
	}

	pageIds := make([]uuid.UUID, 0, len(pages))
	for _, page := range pages {
		pageIds = append(pageIds, page.ID)
	}

	mapOfPageIdToSubPages, err := gp.generateSubPagesForTopLevelPages(ctx, pageIds)
	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to get pages", err))
		return
	}

	var pagesWithSubPages = make([]PageWithSubPages, 0, len(pages))

	for _, page := range pages {
		pagesWithSubPages = append(pagesWithSubPages, PageWithSubPages{Page: page, SubPages: mapOfPageIdToSubPages[page.ID]})
	}

	c.JSON(http.StatusOK, PagesResponse{
		Pages: pagesWithSubPages,
	})
}

func (gp *GetPagesHandler) getTopLevelPages(ctx context.Context, params *GetPagesParams, userId int64) ([]page.Page, error) {

	size := 10
	if params.Size != nil {
		size = *params.Size
	}

	var pages []page.Page
	var err error
	if params.CreatedAfter != nil {
		pages, err = page.GetPages(ctx, gp.db, "WHERE user_id = $1 AND is_top_level = true AND created_at > $2 ORDER BY created_at DESC LIMIT $3", userId, params.CreatedAfter, size)
		if err != nil {
			return nil, err
		}
	} else {
		pages, err = page.GetPages(ctx, gp.db, "WHERE user_id = $1 AND is_top_level = true ORDER BY created_at DESC LIMIT $2", userId, size)
		if err != nil {
			return nil, err
		}
	}

	return pages, nil
}

type GetPagesParams struct {
	Size         *int       `form:"size" binding:"min=1,max=100"`
	CreatedAfter *time.Time `form:"created_after"`
}

func getPagesParamsFromQuery(c *gin.Context) (*GetPagesParams, error) {
	var params GetPagesParams
	if err := c.ShouldBindQuery(&params); err != nil {
		return nil, err
	}
	return &params, nil
}

type PagesResponse struct {
	Pages []PageWithSubPages `json:"pages"`
}

type PageWithSubPages struct {
	Page     page.Page `json:"page"`
	SubPages []SubPage `json:"sub_pages"`
}

type SubPage struct {
	ID        uuid.UUID `json:"id"`
	TextTitle *string   `json:"text_title"`
	SubPages  []SubPage `json:"sub_pages"`
}

func (gp *GetPagesHandler) generateSubPagesForTopLevelPages(ctx context.Context, pageIds []uuid.UUID) (map[uuid.UUID][]SubPage, error) {

	conn, err := gp.db.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	descendantsWithAllAncestors, err := page.GetAllDescendants(ctx, conn.Conn(), pageIds)
	if err != nil {
		return nil, err
	}

	descendantIds := make([]uuid.UUID, 0, len(descendantsWithAllAncestors))
	for descendantId := range descendantsWithAllAncestors {
		descendantIds = append(descendantIds, descendantId)
	}

	type bareDescendant struct {
		ID        uuid.UUID
		TextTitle *string
	}
	rows, err := gp.db.Query(ctx, `
		SELECT id, text_title
		FROM pages
		WHERE id = ANY($1)
	`, descendantIds)

	if err != nil {
		return nil, err
	}

	var mappingOfDescendantIdToBareDescendant = make(map[uuid.UUID]bareDescendant)
	for rows.Next() {
		var bareDescendant bareDescendant
		if err := rows.Scan(&bareDescendant.ID, &bareDescendant.TextTitle); err != nil {
			return nil, err
		}
		mappingOfDescendantIdToBareDescendant[bareDescendant.ID] = bareDescendant
	}

	var mappingOfPageIdToSubPages = make(map[uuid.UUID][]SubPage)

	var buildSubPageTree func(pageId uuid.UUID) []SubPage
	buildSubPageTree = func(pageId uuid.UUID) []SubPage {
		descendants := descendantsWithAllAncestors[pageId]
		subPages := []SubPage{}

		for _, closure := range descendants {
			if closure.IsParent {
				subPage := SubPage{
					ID:        closure.DescendantID,
					TextTitle: mappingOfDescendantIdToBareDescendant[closure.DescendantID].TextTitle,
					SubPages:  buildSubPageTree(closure.DescendantID),
				}
				subPages = append(subPages, subPage)
			}
		}
		return subPages
	}

	for _, pageId := range pageIds {
		mappingOfPageIdToSubPages[pageId] = buildSubPageTree(pageId)
	}

	return mappingOfPageIdToSubPages, nil
}

func (gp *GetPagesHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/pages", gp.GetPages)
}
