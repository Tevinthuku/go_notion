package handlers_test

import (
	"go_notion/backend/handlers"
	"go_notion/backend/mocks"
	"go_notion/backend/router"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestSignIn(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	tests := []struct {
		name          string
		email         string
		userPassword  string
		passwordInput string
		expectedCode  int
	}{
		{name: "test", email: "test@test.com", userPassword: "password", passwordInput: "password", expectedCode: http.StatusOK},
		{name: "test", email: "test@test.com", userPassword: "password", passwordInput: "wrongpassword", expectedCode: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(test.userPassword), handlers.BcryptDevCost)
			if err != nil {
				t.Fatal(err)
			}
			hashedPasswordString := string(hashedPassword)

			mock.ExpectQuery(`
					select id, password from users where email=\$1
				`).
				WithArgs(test.email).
				WillReturnRows(pgxmock.NewRows([]string{"id", "password"}).AddRow(int64(1), hashedPasswordString))

			tokenGenerator := &mocks.TokenGeneratorMock{}
			signIn, err := handlers.NewSignInHandler(mock, tokenGenerator)
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
