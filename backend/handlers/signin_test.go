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

func TestSignIn(t *testing.T) {

	email, username, password := "test@test.com", "test", "password"

	pool, err := db.RunTestDb(db.InsertTestUserWithData(email, username, password))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	tests := []struct {
		name          string
		email         string
		userPassword  string
		passwordInput string
		expectedCode  int
	}{
		{name: "test", email: email, userPassword: password, passwordInput: password, expectedCode: http.StatusOK},
		{name: "test", email: email, userPassword: password, passwordInput: "wrongpassword", expectedCode: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			tokenGenerator := &mocks.TokenGeneratorMock{}
			signIn, err := handlers.NewSignInHandler(pool, tokenGenerator)
			if err != nil {
				t.Fatal(err)
			}

			r := router.NewRouter()
			signIn.RegisterRoutes(r.Group("/api"))

			w := httptest.NewRecorder()

			json := `{"email": "` + test.email + `", "password": "` + test.passwordInput + `"}`
			req, _ := http.NewRequest("POST", "/api/auth/signin", strings.NewReader(json))
			r.ServeHTTP(w, req)

			assert.Equal(t, test.expectedCode, w.Code)
		})
	}

}
