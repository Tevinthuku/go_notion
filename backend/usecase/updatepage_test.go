package usecase_test

import (
	"encoding/json"
	"go_notion/backend/router"
	"go_notion/backend/usecase"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestUpdatePage(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	updatePage, err := usecase.NewUpdatePageUseCase(mock)
	if err != nil {
		t.Fatal(err)
	}

	pageID, err := uuid.NewV4()
	if err != nil {
		t.Fatal(err)
	}

	mock.ExpectExec(`
		UPDATE pages SET text_title = \$1, text_content = \$2, raw_title = \$3, raw_content = \$4 WHERE id = \$5 AND created_by = \$6
	`).WithArgs("title", "content", json.RawMessage(`{"data": "title"}`), json.RawMessage(`{"data": "content"}`), pageID, int64(1)).WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	r := router.NewRouter()

	r.PUT("/api/pages/:id", func(c *gin.Context) {
		c.Set("user_id", int64(1))
		updatePage.UpdatePage(c)
	})

	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)

	if err != nil {
		t.Fatal(err)
	}
	body := `{"title_text": "title", "content_text": "content", "raw_title": {"data": "title"}, "raw_content": {"data": "content"}}`
	c.Request, _ = http.NewRequest("PUT", "/api/pages/"+pageID.String(), strings.NewReader(body))

	r.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusOK, w.Code)
}
