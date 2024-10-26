package usecase_test

import (
	"go_notion/backend/mocks"
	"go_notion/backend/usecase"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestSignUpWorks(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	// mock the query to check if the email or username already exists
	mock.ExpectQuery("SELECT").
		WithArgs("test@test.com", "test").
		WillReturnRows(pgxmock.NewRows([]string{"email_count", "user_name_count"}).AddRow(0, 0))

	// mock the query to insert the user
	mock.ExpectQuery("INSERT").
		WithArgs("test@test.com", "test", mocks.AnyPassword{}).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(1)))

	tokenGenerator := &mocks.TokenGeneratorMock{}
	signUp := usecase.NewSignUp(mock, tokenGenerator)

	r := mocks.NewRouter()
	signUp.RegisterRoutes(r.Group("/api"))

	w := httptest.NewRecorder()

	json := `{"email": "test@test.com", "password": "password", "username": "test"}`
	req, _ := http.NewRequest("POST", "/api/auth/signup", strings.NewReader(json))
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSignUpReturnsBadRequestWhenEmailAlreadyExists(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// mock the query to check if the email or username already exists
	mock.ExpectQuery("SELECT").
		WithArgs("test@test.com", "test").
		WillReturnRows(pgxmock.NewRows([]string{"email_count", "user_name_count"}).AddRow(1, 1))

	tokenGenerator := &mocks.TokenGeneratorMock{}
	signUp := usecase.NewSignUp(mock, tokenGenerator)

	r := mocks.NewRouter()
	signUp.RegisterRoutes(r.Group("/api"))

	w := httptest.NewRecorder()

	json := `{"email": "test@test.com", "password": "password", "username": "test"}`
	req, _ := http.NewRequest("POST", "/api/auth/signup", strings.NewReader(json))
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
