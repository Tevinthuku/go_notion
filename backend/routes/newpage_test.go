package routes_test

import (
	"go_notion/backend/page"
	"go_notion/backend/router"
	"go_notion/backend/routes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestNewPage(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	pageConfig := page.NewPageConfig(10)
	np, err := routes.NewPage(mock, pageConfig)
	if err != nil {
		t.Fatal(err)
	}

	mock.ExpectQuery(`
		SELECT COALESCE\(MAX\(position\), 0\) FROM pages WHERE created_by = \$1
	`).WithArgs(int64(1)).WillReturnRows(pgxmock.NewRows([]string{"coalesce"}).AddRow(float64(0)))

	uuid, err := uuid.NewV4()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery(`
		INSERT INTO pages \(created_by, position\) VALUES \(\$1, \$2\) RETURNING id
	`).WithArgs(int64(1), float64(pageConfig.Spacing)).WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(uuid))

	r := router.NewRouter()

	// Setup route with middleware that would normally set user_id
	r.POST("/api/pages", func(c *gin.Context) {
		// Simulate middleware
		c.Set("user_id", int64(1))

		np.CreatePage(c)
	})

	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequestWithContext(c, "POST", "/api/pages", nil)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
