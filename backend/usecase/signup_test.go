package usecase_test

import (
	"go_notion/backend/mocks"
	"go_notion/backend/router"
	"go_notion/backend/usecase"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestSignUp(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	tests := []struct {
		name          string
		email         string
		username      string
		password      string
		emailCount    int
		usernameCount int
		expectedCode  int
	}{
		{name: "test", email: "test@test.com", username: "test", password: "password", emailCount: 0, usernameCount: 0, expectedCode: http.StatusOK},
		{name: "test", email: "test@test.com", username: "test", password: "password", emailCount: 1, usernameCount: 0, expectedCode: http.StatusBadRequest},
		{name: "test", email: "test@test.com", username: "test", password: "password", emailCount: 0, usernameCount: 1, expectedCode: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mock.ExpectBeginTx(pgx.TxOptions{})
			// mock the query to check if the email or username already exists
			mock.ExpectQuery(`SELECT 
			COUNT\(\*\) FILTER \(WHERE email = \$1\) as email_count,
			COUNT\(\*\) FILTER \(WHERE username = \$2\) as username_count
				FROM users 
				WHERE email = \$1 OR username = \$2`).
				WithArgs(test.email, test.name).
				WillReturnRows(pgxmock.NewRows([]string{"email_count", "user_name_count"}).AddRow(test.emailCount, test.usernameCount))

			if test.emailCount == 0 && test.usernameCount == 0 {
				// mock the query to insert the user
				mock.ExpectQuery(`
			INSERT INTO users \(email, username, password\) 
			VALUES \(\$1, \$2, \$3\) 
			RETURNING id
		`).
					WithArgs(test.email, test.username, mocks.AnyPassword{}).
					WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(1)))
				mock.ExpectCommit()
			} else {
				mock.ExpectRollback()
			}

			tokenGenerator := &mocks.TokenGeneratorMock{}
			signUp, err := usecase.NewSignUp(mock, tokenGenerator)
			if err != nil {
				t.Fatal(err)
			}

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
