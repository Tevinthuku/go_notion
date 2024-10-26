package authentication

import (
	"go_notion/backend/api_error"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type AuthenticationContext struct {
	db *pgxpool.Pool
}

func NewAuthenticationContext(db *pgxpool.Pool) *AuthenticationContext {
	return &AuthenticationContext{db: db}
}

func (ac *AuthenticationContext) RegisterRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	auth.POST("/sign-up", ac.signUp)
}

type SignUpInput struct {
	Email    string `json:"email" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (ac *AuthenticationContext) signUp(c *gin.Context) {
	var input SignUpInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.Error(api_error.NewInternalServerError("Failed to hash password. Try a different password", err))
		return
	}

	input.Password = string(hashedPassword)

	var existingEmailCount, existingUserNameCount int
	err = ac.db.QueryRow(c, `
		SELECT 
			(SELECT COUNT(*) FROM users WHERE email = $1),
			(SELECT COUNT(*) FROM users WHERE username = $2)
	`, input.Email, input.Username).Scan(&existingEmailCount, &existingUserNameCount)

	if err != nil {
		c.Error(api_error.NewInternalServerError("User validation check failed. Please try again.", err))
		return
	}

	if existingEmailCount > 0 {
		c.Error(api_error.NewBadRequestError("Email already in use", nil))
		return
	}

	if existingUserNameCount > 0 {
		c.Error(api_error.NewBadRequestError("Username already taken", nil))
		return
	}

	_, err = ac.db.Exec(c, "INSERT INTO users (email, username, password) VALUES ($1, $2, $3)", input.Email, input.Username, input.Password)
	if err != nil {
		c.Error(api_error.NewInternalServerError("Failed to create user", err))
		return
	}

}
