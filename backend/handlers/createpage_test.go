package handlers_test

import (
	"go_notion/backend/handlers"
	"go_notion/backend/page"
	"go_notion/backend/router"
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
	np, err := handlers.NewCreatePageHandler(mock, pageConfig)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		userID         any
		setupMock      func(mock pgxmock.PgxPoolIface)
		expectedStatus int
	}{
		{
			name:   "successfully create page",
			userID: int64(1),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				uuid, err := uuid.NewV4()
				if err != nil {
					t.Fatal(err)
				}
				mock.ExpectQuery(`
				SELECT COALESCE\(MAX\(position\), 0\) FROM pages WHERE created_by = \$1
			`).WithArgs(int64(1)).WillReturnRows(pgxmock.NewRows([]string{"coalesce"}).AddRow(float64(0)))

				mock.ExpectQuery(`
				INSERT INTO pages \(created_by, position\) VALUES \(\$1, \$2\) RETURNING id
			`).WithArgs(int64(1), float64(pageConfig.Spacing)).WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(uuid))
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "invalid user id",
			userID: "invalid",
			setupMock: func(mock pgxmock.PgxPoolIface) {
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mock.Reset()
			test.setupMock(mock)

			r := router.NewRouter()
			// Setup route with middleware that would normally set user_id
			r.POST("/api/pages", func(c *gin.Context) {
				c.Set("user_id", test.userID)
				np.CreatePage(c)
			})

			w := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(w)

			req, _ := http.NewRequestWithContext(c, "POST", "/api/pages", nil)

			r.ServeHTTP(w, req)

			assert.Equal(t, test.expectedStatus, w.Code)
		})
	}
}
