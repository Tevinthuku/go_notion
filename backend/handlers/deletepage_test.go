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
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestDeletePage(t *testing.T) {

	pageId := uuid.Must(uuid.NewV4())
	pool, err := db.RunTestDb(db.InsertTestUserFixture, db.InsertTestPageFixture(pageId, 1))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	deletePage, err := handlers.NewDeletePageHandler(pool)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		userID         any
		pageId         string
		setupMock      func(mock pgxmock.PgxPoolIface)
		expectedStatus int
	}{
		{
			name:           "successfully delete page",
			userID:         int64(1),
			pageId:         pageId.String(),
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "invalid user id",
			userID:         "invalid",
			pageId:         pageId.String(),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid page id",
			userID:         int64(1),
			pageId:         "invalid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "page not found",
			userID:         int64(1),
			pageId:         "123e4567-e89b-12d3-a456-426614174000",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			r := router.NewRouter()

			r.DELETE("/api/pages/:id", func(c *gin.Context) {
				c.Set("user_id", test.userID)
				deletePage.DeletePage(c)
			})

			w := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(w)

			c.Request, _ = http.NewRequest("DELETE", "/api/pages/"+test.pageId, nil)

			r.ServeHTTP(w, c.Request)

			assert.Equal(t, test.expectedStatus, w.Code)
		})
	}
}
