package handlers_test

import (
	"context"
	"fmt"
	"go_notion/backend/db"
	"go_notion/backend/handlers"
	"go_notion/backend/page"
	"go_notion/backend/router"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
)

func TestPageColumnsAreInSyncWithDb(t *testing.T) {

	pool, err := db.OpenTestDb()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	rows, err := pool.Query(context.Background(),
		"SELECT column_name FROM information_schema.columns WHERE table_name = 'pages' ORDER BY ordinal_position")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var colName string
		if err := rows.Scan(&colName); err != nil {
			t.Fatal(err)
		}
		// we ignore the auto-generated columns
		if colName == "created_at" || colName == "updated_at" {
			continue
		}
		columns = append(columns, colName)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	fmt.Println(columns)

	for _, col := range columns {
		if !slices.Contains(handlers.PageColumns, col) {
			t.Errorf("column %s is not in the PageColumns list", col)
		}
	}

}

func TestDuplicatePage(t *testing.T) {
	page_id := uuid.Must(uuid.NewV4())
	pool, err := db.OpenTestDb(db.InsertTestUserFixture, db.InsertTestPageFixture(page_id, 1))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	duplicatePage, err := handlers.NewDuplicatePageHandler(pool, page.NewPageConfig(10))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		pageId         uuid.UUID
		userID         int64
		expectedStatus int
	}{
		{
			name:           "successfully duplicate page",
			pageId:         page_id,
			userID:         1,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "page not found",
			pageId:         page_id,
			userID:         2,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "random page id",
			pageId:         uuid.Must(uuid.NewV4()),
			userID:         1,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := router.NewRouter()

			r.POST("/api/pages/:id/duplicate", func(c *gin.Context) {
				c.Set("user_id", test.userID)
				duplicatePage.DuplicatePage(c)
			})

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("POST", "/api/pages/"+test.pageId.String()+"/duplicate", nil)
			r.ServeHTTP(w, c.Request)

			assert.Equal(t, test.expectedStatus, w.Code)
		})
	}
}

func TestDuplicatePageWithNestedPages(t *testing.T) {
	parentPageId := uuid.Must(uuid.NewV4())
	childPageId := uuid.Must(uuid.NewV4())
	pool, err := db.OpenTestDb(db.InsertTestUserFixture, db.InsertTestPageFixtureWithParent(childPageId, parentPageId, 1))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	duplicatePage, err := handlers.NewDuplicatePageHandler(pool, page.NewPageConfig(10))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		pageId         uuid.UUID
		userID         int64
		expectedStatus int
	}{
		{
			name:           "successfully duplicate parent page with nested page",
			pageId:         parentPageId,
			userID:         1,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "successfully duplicate nested page",
			pageId:         childPageId,
			userID:         1,
			expectedStatus: http.StatusOK,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := router.NewRouter()

			r.POST("/api/pages/:id/duplicate", func(c *gin.Context) {
				c.Set("user_id", test.userID)
				duplicatePage.DuplicatePage(c)
			})

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("POST", "/api/pages/"+test.pageId.String()+"/duplicate", nil)
			r.ServeHTTP(w, c.Request)

			assert.Equal(t, test.expectedStatus, w.Code)
		})
	}
}
