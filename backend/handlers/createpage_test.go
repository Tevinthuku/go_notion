package handlers_test

import (
	"go_notion/backend/db"
	"go_notion/backend/handlers"
	"go_notion/backend/page"
	"go_notion/backend/router"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
)

func TestNewPage(t *testing.T) {

	pool, err := db.RunTestDb(db.InsertTestUserFixture)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	pageConfig := page.NewPageConfig(10)
	np, err := handlers.NewCreatePageHandler(pool, pageConfig)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		userID         any
		expectedStatus int
	}{
		{
			name:           "successfully create page",
			userID:         int64(1),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid user id",
			userID:         "invalid",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			r := router.NewRouter()
			// Setup route with middleware that would normally set user_id
			r.POST("/api/pages", func(c *gin.Context) {
				c.Set("user_id", test.userID)
				np.CreatePage(c)
			})

			w := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(w)

			req, _ := http.NewRequestWithContext(c, "POST", "/api/pages", strings.NewReader(`{}`))

			r.ServeHTTP(w, req)

			assert.Equal(t, test.expectedStatus, w.Code)
		})
	}
}

func TestCreateNestedPage(t *testing.T) {
	pageId := uuid.Must(uuid.NewV4())
	pool, err := db.RunTestDb(db.InsertTestUserFixture, db.InsertTestPageFixture(pageId, 1))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	pageConfig := page.NewPageConfig(10)
	np, err := handlers.NewCreatePageHandler(pool, pageConfig)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		userID         any
		parentID       string
		expectedStatus int
	}{
		{
			name:           "successfully create nested page",
			userID:         int64(1),
			parentID:       pageId.String(),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid parent id",
			userID:         int64(1),
			parentID:       "invalid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "parent page not found",
			userID:         int64(1),
			parentID:       uuid.Must(uuid.NewV4()).String(),
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := router.NewRouter()

			r.POST("/api/pages", func(c *gin.Context) {
				c.Set("user_id", test.userID)
				np.CreatePage(c)
			})

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("POST", "/api/pages", strings.NewReader(`{"parent_id": "`+test.parentID+`"}`))
			r.ServeHTTP(w, c.Request)

			assert.Equal(t, test.expectedStatus, w.Code)
		})
	}
}
