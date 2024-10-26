package authentication

import (
	"go_notion/backend/api_error"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type AuthenticationContext struct {
	db          *pgxpool.Pool
	tokenConfig *TokenConfig
}

func NewAuthenticationContext(db *pgxpool.Pool, tokenConfig *TokenConfig) *AuthenticationContext {
	return &AuthenticationContext{db: db, tokenConfig: tokenConfig}
}

func (ac *AuthenticationContext) RegisterRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	auth.POST("/sign-up", ac.signUp)
	auth.POST("/sign-in", ac.signIn)
}

type SignUpInput struct {
	Email    string `json:"email" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (ac *AuthenticationContext) signUp(c *gin.Context) {
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
	err = ac.db.QueryRow(c, `
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
	err = ac.db.QueryRow(c, `
		INSERT INTO users (email, username, password) 
		VALUES ($1, $2, $3) 
		RETURNING id
	`, input.Email, input.Username, input.Password).Scan(&userID)

	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to create user.", err))
		return
	}
	token, err := ac.tokenConfig.GenerateToken(userID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})

}

type SignInInput struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (ac *AuthenticationContext) signIn(c *gin.Context) {
	var input SignInInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(api_error.NewBadRequestError(err.Error(), err))
		return
	}

	var userID int64
	var hashedPassword string

	err := ac.db.QueryRow(c, "select id, password from users where email=$1", input.Email).Scan(&userID, &hashedPassword)

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

	token, err := ac.tokenConfig.GenerateToken(userID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})

}
