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

func TestReorderPageWorks(t *testing.T) {
	parentId := uuid.Must(uuid.NewV4())
	childId := uuid.Must(uuid.NewV4())
	upComingParentId := uuid.Must(uuid.NewV4())
	pool, err := db.RunTestDb(db.InsertTestUserFixture, db.InsertTestPageFixtureWithParent(parentId, childId, 1), db.InsertTestPageFixtureWithPosition(upComingParentId, 1, 200))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	rp, err := handlers.NewReorderPageHandler(pool)
	if err != nil {
		t.Fatal(err)
	}

	r := router.NewRouter()
	r.POST("/api/pages/:id/reorder", func(c *gin.Context) {
		c.Set("user_id", int64(1))
		rp.ReorderPage(c)
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	json := `{"new_parent_id": "` + upComingParentId.String() + `"}`
	c.Request, _ = http.NewRequest("POST", "/api/pages/"+childId.String()+"/reorder", strings.NewReader(json))
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusOK, w.Code)
}
