package usecase

import (
	"go_notion/backend/api_error"
	"go_notion/backend/authtoken"
	"go_notion/backend/db"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type SignIn struct {
	db             db.DB
	tokenGenerator authtoken.TokenGenerator
}

func NewSignIn(db db.DB, tokenGenerator authtoken.TokenGenerator) *SignIn {
	return &SignIn{db, tokenGenerator}
}

type SignInInput struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (s *SignIn) SignIn(c *gin.Context) {
	var input SignInInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(api_error.NewBadRequestError(err.Error(), err))
		return
	}

	var userID int64
	var hashedPassword string

	err := s.db.QueryRow(c, "select id, password from users where email=$1", input.Email).Scan(&userID, &hashedPassword)

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
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

func (s *SignIn) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/auth/signin", s.SignIn)
}
