package handlers_test

import (
	"encoding/json"
	"go_notion/backend/handlers"
	"go_notion/backend/mocks"
	"go_notion/backend/router"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestUpdatePage(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	updatePage, err := handlers.NewUpdatePageHandler(mock)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		userID         any
		body           string
		getPageID      func() string
		setupMock      func(mock pgxmock.PgxPoolIface)
		expectedStatus int
	}{
		{
			name:   "successfully update page",
			userID: int64(1),
			body:   `{"title_text": "title", "content_text": "content", "raw_title": {"data": "title"}, "raw_content": {"data": "content"}}`,
			getPageID: func() string {
				return "d23d0a84-3260-4670-aa1f-5d316ba6325b"
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`
				UPDATE pages SET text_title = \$1, text_content = \$2, title = \$3, content = \$4 WHERE id = \$5 AND created_by = \$6
			`).WithArgs("title", "content", json.RawMessage(`{"data": "title"}`), json.RawMessage(`{"data": "content"}`), mocks.AnyUUID{}, int64(1)).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "invalid user id",
			userID: "invalid",
			body:   `{"title_text": "title", "content_text": "content", "raw_title": {"data": "title"}, "raw_content": {"data": "content"}}`,
			getPageID: func() string {
				return "123e4567-e89b-12d3-a456-426614174000"
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "invalid page id",
			userID: int64(1),
			body:   `{"title_text": "title", "content_text": "content", "raw_title": {"data": "title"}, "raw_content": {"data": "content"}}`,
			getPageID: func() string {
				return "invalid"
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "invalid body",
			userID: int64(1),
			body:   `{"title_texts": "title"}`,
			getPageID: func() string {
				return "d23d0a84-3260-4670-aa1f-5d316ba6325b"
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mock.Reset()
			test.setupMock(mock)
			r := router.NewRouter()

			r.PUT("/api/pages/:id", func(c *gin.Context) {
				c.Set("user_id", test.userID)
				updatePage.UpdatePage(c)
			})

			w := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(w)

			if err != nil {
				t.Fatal(err)
			}
			c.Request, _ = http.NewRequest("PUT", "/api/pages/"+test.getPageID(), strings.NewReader(test.body))

			r.ServeHTTP(w, c.Request)

			assert.Equal(t, test.expectedStatus, w.Code)
		})
	}
}
