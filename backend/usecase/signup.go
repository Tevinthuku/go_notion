package usecase

import (
	"context"
	"fmt"
	"go_notion/backend/api_error"
	"go_notion/backend/authtoken"
	"go_notion/backend/db"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	// Recommended cost for development
	MinBcryptCost = 10
	// Recommended cost for production
	ProdBcryptCost = 12
)

type SignUp struct {
	db             db.DB
	tokenGenerator authtoken.TokenGenerator
}

func NewSignUp(db db.DB, tokenGenerator authtoken.TokenGenerator) (*SignUp, error) {
	if db == nil || tokenGenerator == nil {
		return nil, fmt.Errorf("db and tokenGenerator cannot be nil")
	}
	return &SignUp{db, tokenGenerator}, nil
}

type SignUpInput struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required,min=3,max=30"`
	Password string `json:"password" binding:"required,min=5"`
}

func (s *SignUp) SignUp(c *gin.Context) {

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	var input SignUpInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(api_error.NewBadRequestError(err.Error(), err))
		return
	}
	cost := MinBcryptCost
	if os.Getenv("GO_ENV") == "production" {
		cost = ProdBcryptCost
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), cost)
	if err != nil {
		c.Error(api_error.NewInternalServerError("Failed to hash password. Try a different password", err))
		return
	}

	input.Password = string(hashedPassword)

	// the check and insert should be in a transaction to prevent race conditions in case of concurrent requests
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		c.Error(api_error.NewInternalServerError("internal server error", err))
		return
	}
	defer tx.Rollback(ctx)

	var existingEmailCount, existingUserNameCount int
	err = tx.QueryRow(ctx, `
		SELECT 
			(SELECT COUNT(*) FROM users WHERE email = $1),
			(SELECT COUNT(*) FROM users WHERE username = $2)
	`, input.Email, input.Username).Scan(&existingEmailCount, &existingUserNameCount)

	if err != nil {
		c.Error(api_error.NewInternalServerError("user validation check failed. Please try again.", err))
		return
	}

	if existingEmailCount > 0 {
		c.Error(api_error.NewBadRequestError("email already in use", nil))
		return
	}

	if existingUserNameCount > 0 {
		c.Error(api_error.NewBadRequestError("username already taken", nil))
		return
	}

	var userID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO users (email, username, password) 
		VALUES ($1, $2, $3) 
		RETURNING id
	`, input.Email, input.Username, input.Password).Scan(&userID)

	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to create user.", err))
		return
	}
	token, err := s.tokenGenerator.Generate(userID)
	if err != nil {
		c.Error(api_error.NewInternalServerError("authentication failed", err))
		return
	}
	if err := tx.Commit(ctx); err != nil {
		c.Error(api_error.NewInternalServerError("internal server error", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

func (s *SignUp) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/auth/signup", s.SignUp)
}
