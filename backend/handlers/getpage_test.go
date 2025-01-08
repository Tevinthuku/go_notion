package handlers_test

import (
	"go_notion/backend/db"
	"go_notion/backend/handlers"
	"go_notion/backend/router"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
)

func TestGetPage(t *testing.T) {
	pageId := uuid.Must(uuid.NewV4())
	pool, err := db.OpenTestDb(db.InsertTestUserFixture, db.InsertTestPageFixture(pageId, 1))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	getPage, err := handlers.NewGetPageHandler(pool)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		userID         any
		pageID         string
		expectedStatus int
	}{
		{
			name:           "successfully get page",
			userID:         int64(1),
			pageID:         pageId.String(),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid user id",
			userID:         "invalid",
			pageID:         pageId.String(),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "page not found",
			userID:         int64(1),
			pageID:         uuid.Must(uuid.NewV4()).String(),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "user not page owner",
			userID:         int64(2),
			pageID:         pageId.String(),
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := router.NewRouter()

			r.GET("/api/pages/:id", func(c *gin.Context) {
				c.Set("user_id", test.userID)
				getPage.GetPage(c)
			})

			w := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/api/pages/"+test.pageID, nil)

			r.ServeHTTP(w, c.Request)

			assert.Equal(t, test.expectedStatus, w.Code)
		})
	}
}
