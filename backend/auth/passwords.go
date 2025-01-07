package auth

import (
	"os"

	"golang.org/x/crypto/bcrypt"
)

const (
	// BcryptDevCost is the recommended cost for development environments
	BcryptDevCost = 10
	// BcryptProdCost is the recommended cost for production environments
	BcryptProdCost = 12
)

func HashPassword(password string) (string, error) {
	cost := BcryptDevCost
	if os.Getenv("GO_ENV") == "production" {
		cost = BcryptProdCost
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func ComparePassword(password, hashedPassword string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)) == nil
}
