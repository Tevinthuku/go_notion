package auth

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	TokenSecretEnvVar    = "TOKEN_SECRET"
	TokenLifeSpanEnvVar  = "TOKEN_HOUR_LIFESPAN"
	DefaultTokenLifeSpan = "24"
)

type TokenGenerator interface {
	Generate(userID int64) (string, error)
}

type TokenConfig struct {
	tokenSecret   string
	tokenLifeSpan int
}

func NewTokenConfig() (*TokenConfig, error) {
	// loading of env variables is done at app startup
	tokenSecret, ok := os.LookupEnv(TokenSecretEnvVar)
	if !ok {
		return nil, fmt.Errorf("authentication configuration error: %s environment variable is not set", TokenSecretEnvVar)
	}
	tokenLifeSpan, ok := os.LookupEnv(TokenLifeSpanEnvVar)
	if !ok {
		log.Printf("authentication configuration error: %s environment variable is not set, defaulting to %s hours", TokenLifeSpanEnvVar, DefaultTokenLifeSpan)
		tokenLifeSpan = DefaultTokenLifeSpan
	}
	tokenLifeSpanInt, err := strconv.Atoi(tokenLifeSpan)
	if err != nil || tokenLifeSpanInt <= 0 {
		return nil, fmt.Errorf("invalid token lifespan: %s. Full error: %w", tokenLifeSpan, err)
	}
	return &TokenConfig{tokenSecret: tokenSecret, tokenLifeSpan: tokenLifeSpanInt}, nil
}

func (tc *TokenConfig) Generate(userID int64) (string, error) {

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

func (tc *TokenConfig) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := tc.extractUserID(c)
		if err != nil {
			log.Printf("userId extraction error: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token",
			})
			c.Abort()
			return
		}
		c.Set("user_id", userID)

		c.Next()
	}
}

func (tc *TokenConfig) extractUserID(c *gin.Context) (int64, error) {
	token := c.GetHeader("Authorization")
	if token == "" {
		return 0, fmt.Errorf("no token provided")
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(token, prefix) {
		return 0, fmt.Errorf("invalid token format")
	}
	token = token[len(prefix):]

	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(tc.tokenSecret), nil
	})

	if err != nil {
		return 0, fmt.Errorf("error parsing token: %w", err)
	}

	if !parsedToken.Valid {
		return 0, fmt.Errorf("invalid token")
	}

	claims := parsedToken.Claims.(jwt.MapClaims)
	// When parsing JSON numbers, the default type for numbers in Go's map[string]interface{} is float64, not int64
	userID, ok := claims["user_id"].(float64)
	if !ok {
		return 0, fmt.Errorf("invalid token. user_id claim not found")
	}
	return int64(userID), nil
}
