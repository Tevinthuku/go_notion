package usecase

import (
	"context"
	"fmt"
	"go_notion/backend/api_error"
	"go_notion/backend/authtoken"
	"go_notion/backend/db"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type SignIn struct {
	db             db.DB
	tokenGenerator authtoken.TokenGenerator
}

func NewSignIn(db db.DB, tokenGenerator authtoken.TokenGenerator) (*SignIn, error) {
	if db == nil || tokenGenerator == nil {
		return nil, fmt.Errorf("db and tokenGenerator cannot be nil")
	}
	return &SignIn{db, tokenGenerator}, nil
}

type SignInInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=5"`
}

func (s *SignIn) SignIn(c *gin.Context) {
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
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(input.Password)); err != nil {
		// as a security practice, we should not be too direct about what exactly went wrong because
		// "hackers" could try and brute force the password once we let them know its the password that's wrong
		c.Error(api_error.NewBadRequestError("wrong email or password", err))
		return
	}

	token, err := s.tokenGenerator.GenerateToken(userID)
	if err != nil {
		c.Error(api_error.NewInternalServerError("authentication failed", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

func (s *SignIn) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/auth/signin", s.SignIn)
}
