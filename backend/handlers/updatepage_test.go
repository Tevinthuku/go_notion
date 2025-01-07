package handlers_test

import (
	"go_notion/backend/db"
	"go_notion/backend/handlers"
	"go_notion/backend/router"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
)

func TestUpdatePage(t *testing.T) {

	pageId := uuid.Must(uuid.NewV4())
	pool, err := db.RunTestDb(db.InsertTestUserFixture, db.InsertTestPageFixture(pageId, 1))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	updatePage, err := handlers.NewUpdatePageHandler(pool)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		userID         any
		body           string
		getPageID      func() string
		expectedStatus int
	}{
		{
			name:   "successfully update page",
			userID: int64(1),
			body:   `{"title_text": "title", "content_text": "content", "raw_title": {"data": "title"}, "raw_content": {"data": "content"}}`,
			getPageID: func() string {
				return pageId.String()
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
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "invalid page id",
			userID: int64(1),
			body:   `{"title_text": "title", "content_text": "content", "raw_title": {"data": "title"}, "raw_content": {"data": "content"}}`,
			getPageID: func() string {
				return "invalid"
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
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := router.NewRouter()

			r.PUT("/api/pages/:id", func(c *gin.Context) {
				c.Set("user_id", test.userID)
				updatePage.UpdatePage(c)
			})

			w := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("PUT", "/api/pages/"+test.getPageID(), strings.NewReader(test.body))

			r.ServeHTTP(w, c.Request)

			assert.Equal(t, test.expectedStatus, w.Code)
		})
	}
}
