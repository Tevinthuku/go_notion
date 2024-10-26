package authentication

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt"
)

type TokenConfig struct {
	tokenSecret   string
	tokenLifeSpan int
}

func NewTokenConfig() (*TokenConfig, error) {
	tokenSecret, ok := os.LookupEnv("TOKEN_SECRET")
	if !ok {
		return nil, fmt.Errorf("TOKEN_SECRET is not set")
	}
	tokenLifeSpan, ok := os.LookupEnv("TOKEN_HOUR_LIFESPAN")
	if !ok {
		tokenLifeSpan = "24"
	}
	tokenLifeSpanInt, err := strconv.Atoi(tokenLifeSpan)
	if err != nil {
		return nil, fmt.Errorf("invalid token lifespan: %s. Full error: %w", tokenLifeSpan, err)
	}
	return &TokenConfig{tokenSecret: tokenSecret, tokenLifeSpan: tokenLifeSpanInt}, nil
}

func (tc *TokenConfig) GenerateToken(userID int64) (string, error) {

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * time.Duration(tc.tokenLifeSpan)).Unix(),
	})

	tokenString, err := token.SignedString([]byte(tc.tokenSecret))
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return tokenString, nil
}
