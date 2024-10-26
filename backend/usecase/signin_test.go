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
	"golang.org/x/crypto/bcrypt"
)

func TestSignInWorks(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	password := "password"
	// we need to hash the password to compare it with the hashed password in the database
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	hashedPasswordString := string(hashedPassword)
	mock.ExpectQuery("select").
		WithArgs("test@test.com").
		WillReturnRows(pgxmock.NewRows([]string{"id", "password"}).AddRow(int64(1), hashedPasswordString))

	tokenGenerator := &mocks.TokenGeneratorMock{}
	signIn := usecase.NewSignIn(mock, tokenGenerator)

	r := mocks.NewRouter()
	signIn.RegisterRoutes(r.Group("/api"))

	w := httptest.NewRecorder()

	json := `{"email": "test@test.com", "password": "password"}`
	req, _ := http.NewRequest("POST", "/api/auth/signin", strings.NewReader(json))
	r.ServeHTTP(w, req)

	data := w.Body.String()
	println(data)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWrongPasswordReturnsAnError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	password := "password"
	// we need to hash the password to compare it with the hashed password in the database
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	hashedPasswordString := string(hashedPassword)

	mock.ExpectQuery("select").
		WithArgs("test@test.com").
		WillReturnRows(pgxmock.NewRows([]string{"id", "password"}).AddRow(1, hashedPasswordString))

	tokenGenerator := &mocks.TokenGeneratorMock{}
	signIn := usecase.NewSignIn(mock, tokenGenerator)

	r := mocks.NewRouter()
	signIn.RegisterRoutes(r.Group("/api"))

	w := httptest.NewRecorder()

	// pass wrong password
	json := `{"email": "test@test.com", "password": "wrongpassword"}`
	req, _ := http.NewRequest("POST", "/api/auth/signin", strings.NewReader(json))
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
