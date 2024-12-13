package handlers_test

import (
	"go_notion/backend/handlers"
	"go_notion/backend/mocks"
	"go_notion/backend/router"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestDeletePage(t *testing.T) {

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	deletePage, err := handlers.NewDeletePageHandler(mock)
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
			name:   "successfully delete page",
			userID: int64(1),
			pageId: "d23d0a84-3260-4670-aa1f-5d316ba6325b",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM pages WHERE id = \$1 AND created_by = \$2`).WithArgs(mocks.AnyUUID{}, int64(1)).WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:   "invalid user id",
			userID: "invalid",
			pageId: "d23d0a84-3260-4670-aa1f-5d316ba6325b",
			setupMock: func(mock pgxmock.PgxPoolIface) {
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "invalid page id",
			userID: int64(1),
			pageId: "invalid",
			setupMock: func(mock pgxmock.PgxPoolIface) {
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "page not found",
			userID: int64(1),
			pageId: "123e4567-e89b-12d3-a456-426614174000",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM pages WHERE id = \$1 AND created_by = \$2`).WithArgs(mocks.AnyUUID{}, int64(1)).WillReturnResult(pgxmock.NewResult("DELETE", 0))
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mock.Reset()
			test.setupMock(mock)

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
