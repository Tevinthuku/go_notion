package handlers_test

import (
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

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	deletePage, err := handlers.NewDeletePageHandler(mock)
	if err != nil {
		t.Fatal(err)
	}

	pageID, err := uuid.NewV4()
	if err != nil {
		t.Fatal(err)
	}

	mock.ExpectExec(`DELETE FROM pages WHERE id = \$1 AND created_by = \$2`).WithArgs(pageID, int64(1)).WillReturnResult(pgxmock.NewResult("DELETE", 1))

	r := router.NewRouter()

	r.DELETE("/api/pages/:id", func(c *gin.Context) {
		c.Set("user_id", int64(1))
		deletePage.DeletePage(c)
	})

	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)

	c.Request, _ = http.NewRequest("DELETE", "/api/pages/"+pageID.String(), nil)

	r.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusNoContent, w.Code)
}
