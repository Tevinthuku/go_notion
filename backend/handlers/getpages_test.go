package handlers_test

import (
	"context"
	"encoding/json"
	"go_notion/backend/db"
	"go_notion/backend/handlers"
	"go_notion/backend/router"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
)

func TestGetPages(t *testing.T) {
	page1 := uuid.Must(uuid.NewV4())
	page2 := uuid.Must(uuid.NewV4())

	pool, err := db.OpenTestDb(db.InsertTestUserFixture, db.InsertTestPageFixtureWithParent(page1, page2, 1))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	getPages, err := handlers.NewGetPagesHandler(pool)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		userID         any
		size           int
		expectedStatus int
	}{
		{
			name:           "successfully get pages",
			userID:         int64(1),
			size:           10,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid user id",
			userID:         "invalid",
			size:           10,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid size",
			userID:         int64(1),
			size:           0,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			r := router.NewRouter()

			r.GET("/api/pages", func(c *gin.Context) {
				c.Set("user_id", test.userID)
				getPages.GetPages(c)
			})

			w := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/api/pages", nil)

			q := c.Request.URL.Query()
			q.Add("size", strconv.Itoa(test.size))

			c.Request.URL.RawQuery = q.Encode()

			r.ServeHTTP(w, c.Request)

			assert.Equal(t, test.expectedStatus, w.Code)
		})
	}
}

func TestGetPagesParams(t *testing.T) {
	page1 := uuid.Must(uuid.NewV4())
	page2 := uuid.Must(uuid.NewV4())
	page3 := uuid.Must(uuid.NewV4())

	pool, err := db.OpenTestDb(db.InsertTestUserFixture, db.InsertTestPageFixtureWithParent(page1, page2, 1), db.InsertTestPageFixtureWithPosition(page3, 1, 10010))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	page2CreatedAt := "2024-01-01 00:00:00"

	_, err = pool.Exec(context.Background(), `
		UPDATE pages SET created_at = $1 WHERE id = $2
	`, page2CreatedAt, page2)
	if err != nil {
		t.Fatal(err)
	}

	page3CreatedAt := "2023-01-01 00:00:00"

	_, err = pool.Exec(context.Background(), `
		UPDATE pages SET created_at = $1 WHERE id = $2
	`, page3CreatedAt, page3)
	if err != nil {
		t.Fatal(err)
	}

	getPages, err := handlers.NewGetPagesHandler(pool)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		userID         int64
		size           int
		createdBefore  string
		expectedStatus int
		expectedPages  []uuid.UUID
	}{
		{
			name:           "pages are returned in the size specified",
			userID:         int64(1),
			size:           1,
			expectedStatus: http.StatusOK,
			expectedPages:  []uuid.UUID{page2},
		},
		{
			name:           "pages returned respect the created_before param",
			userID:         int64(1),
			size:           1,
			createdBefore:  time.Date(2023, 5, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
			expectedStatus: http.StatusOK,
			expectedPages:  []uuid.UUID{page3},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := router.NewRouter()

			r.GET("/api/pages", func(c *gin.Context) {
				c.Set("user_id", test.userID)
				getPages.GetPages(c)
			})

			w := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(w)

			c.Request, _ = http.NewRequest("GET", "/api/pages", nil)
			q := c.Request.URL.Query()
			q.Add("size", strconv.Itoa(test.size))
			if test.createdBefore != "" {
				q.Add("created_before", test.createdBefore)
			}
			c.Request.URL.RawQuery = q.Encode()

			r.ServeHTTP(w, c.Request)

			assert.Equal(t, test.expectedStatus, w.Code)

			var response handlers.PagesResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			if err != nil {
				t.Fatal(err)
			}

			responsePages := make([]uuid.UUID, len(response.Pages))
			for i, page := range response.Pages {
				responsePages[i] = page.Page.ID
			}

			assert.Equal(t, test.expectedPages, responsePages)
		})
	}
}
