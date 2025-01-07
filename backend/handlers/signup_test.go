package handlers_test

import (
	"go_notion/backend/db"
	"go_notion/backend/handlers"
	"go_notion/backend/mocks"
	"go_notion/backend/router"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSignUp(t *testing.T) {

	pool, err := db.RunTestDb()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	tokenGenerator := &mocks.TokenGeneratorMock{}
	signUp, err := handlers.NewSignUpHandler(pool, tokenGenerator)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name         string
		email        string
		username     string
		password     string
		expectedCode int
	}{
		{name: "test", email: "test@test.com", username: "test", password: "password", expectedCode: http.StatusOK},
		// email already exists
		{name: "test12", email: "test@test.com", username: "test", password: "password", expectedCode: http.StatusBadRequest},
		// username already exists
		{name: "test", email: "test2@test.com", username: "test", password: "password", expectedCode: http.StatusBadRequest},
		// email and username already exists
		{name: "test", email: "test@test.com", username: "test", password: "password", expectedCode: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			r := router.NewRouter()
			signUp.RegisterRoutes(r.Group("/api"))

			w := httptest.NewRecorder()

			json := `{"email": "` + test.email + `", "password": "` + test.password + `", "username": "` + test.username + `"}`
			req, _ := http.NewRequest("POST", "/api/auth/signup", strings.NewReader(json))
			r.ServeHTTP(w, req)

			assert.Equal(t, test.expectedCode, w.Code)
		})
	}
}
