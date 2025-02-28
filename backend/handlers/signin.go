package handlers

import (
	"context"
	"fmt"
	"go_notion/backend/api_error"
	"go_notion/backend/auth"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SignInHandler struct {
	db             *pgxpool.Pool
	tokenGenerator auth.TokenGenerator
}

func NewSignInHandler(db *pgxpool.Pool, tokenGenerator auth.TokenGenerator) (*SignInHandler, error) {
	if db == nil {
		return nil, fmt.Errorf("db cannot be nil")
	}
	if tokenGenerator == nil {
		return nil, fmt.Errorf("tokenGenerator cannot be nil")
	}
	return &SignInHandler{db, tokenGenerator}, nil
}

type SignInInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=5"`
}

func (s *SignInHandler) SignIn(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var input SignInInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(api_error.NewBadRequestError(err.Error(), err))
		return
	}

	var userID int64
	var hashedPassword string

	err := s.db.QueryRow(ctx, "select id, password from users where email=$1", input.Email).Scan(&userID, &hashedPassword)

	if err != nil {
		c.Error(api_error.NewBadRequestError("wrong email or password", err))
		return
	}

	if !auth.ComparePassword(input.Password, hashedPassword) {
		// as a security practice, we should not be too direct about what exactly went wrong because
		// "hackers" could try and brute force the password once we let them know its the password that's wrong
		c.Error(api_error.NewBadRequestError("wrong email or password", err))
		return
	}

	token, err := s.tokenGenerator.Generate(userID)
	if err != nil {
		c.Error(api_error.NewInternalServerError("authentication failed", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

func (s *SignInHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/auth/signin", s.SignIn)
}
