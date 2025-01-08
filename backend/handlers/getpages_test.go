package handlers_test

import (
	"go_notion/backend/db"
	"go_notion/backend/handlers"
	"go_notion/backend/router"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

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
