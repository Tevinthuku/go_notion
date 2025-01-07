package auth

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

const (
	// BcryptDevCost is the recommended cost for development environments
	BcryptDevCost = 10
	// BcryptProdCost is the recommended cost for production environments
	BcryptProdCost = 12
)

type HashError struct {
	passwordValidationError error
	hashError               error
}

func (e *HashError) Error() string {
	if e.passwordValidationError != nil {
		return e.passwordValidationError.Error()
	}
	return e.hashError.Error()
}

func (e *HashError) IsPasswordValidationError() bool {
	return e.passwordValidationError != nil
}

func HashPassword(password string) (string, *HashError) {

	if password == "" {
		return "", &HashError{
			passwordValidationError: fmt.Errorf("password cannot be empty"),
		}
	}

	if len(password) > 72 {
		return "", &HashError{
			passwordValidationError: fmt.Errorf("password exceeds maximum length of 72 characters"),
		}
	}

	cost := BcryptDevCost
	if os.Getenv("GO_ENV") == "production" {
		cost = BcryptProdCost
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", &HashError{
			hashError: err,
		}
	}
	return string(hashedPassword), nil
}

func ComparePassword(password, hashedPassword string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)) == nil
}
