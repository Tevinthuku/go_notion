package usecase

import (
	"go_notion/backend/api_error"
	"go_notion/backend/authtoken"
	"go_notion/backend/db"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type SignUp struct {
	db             db.DB
	tokenGenerator authtoken.TokenGenerator
}

func NewSignUp(db db.DB, tokenGenerator authtoken.TokenGenerator) *SignUp {
	return &SignUp{db, tokenGenerator}
}

type SignUpInput struct {
	Email    string `json:"email" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (s *SignUp) SignUp(c *gin.Context) {
	var input SignUpInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(api_error.NewBadRequestError(err.Error(), err))
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.Error(api_error.NewInternalServerError("Failed to hash password. Try a different password", err))
		return
	}

	input.Password = string(hashedPassword)

	var existingEmailCount, existingUserNameCount int
	err = s.db.QueryRow(c, `
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
	err = s.db.QueryRow(c, `
		INSERT INTO users (email, username, password) 
		VALUES ($1, $2, $3) 
		RETURNING id
	`, input.Email, input.Username, input.Password).Scan(&userID)

	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to create user.", err))
		return
	}
	token, err := s.tokenGenerator.GenerateToken(userID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

func (s *SignUp) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/auth/signup", s.SignUp)
}
